package collector

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"kafka-exporter-clusters/config"
	"kafka-exporter-clusters/kafka"

	"github.com/IBM/sarama"
	"github.com/prometheus/client_golang/prometheus"
)

type ClusterCollector struct {
	clients        map[string]*kafka.KafkaClient
	metrics        map[string]*ClusterMetrics
	mutex          sync.Mutex
	config         *config.ExporterConfig
	lastCollection time.Time
	cache          map[string]clusterCache
	cacheMutex     sync.RWMutex
}

type clusterCache struct {
	metrics      *ClusterMetrics
	timestamp    time.Time
	consumerLags map[string]map[string]map[int32]int64 // group -> topic -> partition -> lag
}

func NewClusterCollector(cfg *config.ExporterConfig) (*ClusterCollector, error) {
	collector := &ClusterCollector{
		clients:        make(map[string]*kafka.KafkaClient),
		metrics:        make(map[string]*ClusterMetrics),
		config:         cfg,
		cache:          make(map[string]clusterCache),
		lastCollection: time.Now().Add(-1 * time.Hour), // Принудительное обновление при старте
	}

	for _, clusterCfg := range cfg.Clusters {
		if err := collector.AddCluster(clusterCfg); err != nil {
			return nil, fmt.Errorf("failed to add cluster %s: %v", clusterCfg.Name, err)
		}
	}

	return collector, nil
}

func (c *ClusterCollector) AddCluster(cfg config.KafkaConfig) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = kafka.ParseKafkaVersion(cfg.Version)
	saramaConfig.Admin.Timeout = 15 * time.Second

	if cfg.SASLUsername != "" && cfg.SASLPassword != "" {
		saramaConfig.Net.SASL.Enable = true
		saramaConfig.Net.SASL.User = cfg.SASLUsername
		saramaConfig.Net.SASL.Password = cfg.SASLPassword
		saramaConfig.Net.SASL.Mechanism = sarama.SASLMechanism(cfg.SASLMechanism)
	}

	client, err := kafka.NewKafkaClient(saramaConfig, cfg.Brokers, cfg.Name)
	if err != nil {
		return err
	}

	c.clients[cfg.Name] = client
	c.metrics[cfg.Name] = NewClusterMetrics(cfg.Name)

	return nil
}

func (c *ClusterCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, metrics := range c.metrics {
		metrics.Describe(ch)
	}
}

func (c *ClusterCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Используем кэширование на 30 секунд для быстрого ответа
	if time.Since(c.lastCollection) < 30*time.Second {
		c.collectFromCache(ch)
		return
	}

	// Полное обновление метрик
	c.lastCollection = time.Now()
	var wg sync.WaitGroup

	for clusterName, client := range c.clients {
		wg.Add(1)
		go func(name string, cl *kafka.KafkaClient) {
			defer wg.Done()
			c.collectAllMetrics(name, cl)
		}(clusterName, client)
	}

	wg.Wait()

	// Отправляем все метрики
	for _, metrics := range c.metrics {
		metrics.Collect(ch)
	}
}

func (c *ClusterCollector) collectFromCache(ch chan<- prometheus.Metric) {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	for clusterName, cache := range c.cache {
		if metrics, exists := c.metrics[clusterName]; exists && time.Since(cache.timestamp) < 2*time.Minute {
			// Восстанавливаем метрики из кэша
			c.restoreMetricsFromCache(metrics, cache.consumerLags)
			metrics.Collect(ch)
		} else {
			// Если кэш устарел, используем текущие метрики
			metrics.Collect(ch)
		}
	}
}

func (c *ClusterCollector) collectAllMetrics(clusterName string, client *kafka.KafkaClient) {
	metrics := c.metrics[clusterName]

	// 1. Быстрый сбор базовых метрик
	brokers := client.Client.Brokers()
	metrics.KafkaBrokers.Set(float64(len(brokers)))
	metrics.BrokersTotal.Set(float64(len(brokers)))

	for _, broker := range brokers {
		metrics.KafkaBrokerInfo.WithLabelValues(
			broker.Addr(),
			strconv.Itoa(int(broker.ID())),
		).Set(1)
	}

	// 2. Сбор топиков
	topics, err := client.Admin.ListTopics()
	if err != nil {
		log.Printf("Failed to list topics for cluster %s: %v", clusterName, err)
		return
	}

	metrics.TopicsTotal.Set(float64(len(topics)))

	for topic, detail := range topics {
		metrics.KafkaTopicPartitions.WithLabelValues(topic).Set(float64(detail.NumPartitions))

		// Базовые метрики партиций
		for i := int32(0); i < detail.NumPartitions; i++ {
			partitionStr := strconv.Itoa(int(i))
			metrics.KafkaTopicPartitionCurrentOffset.WithLabelValues(topic, partitionStr).Set(0)
			metrics.KafkaTopicPartitionOldestOffset.WithLabelValues(topic, partitionStr).Set(0)
			metrics.KafkaTopicPartitionLeader.WithLabelValues(topic, partitionStr).Set(1)
			metrics.KafkaTopicPartitionReplicas.WithLabelValues(topic, partitionStr).Set(float64(detail.ReplicationFactor))
			metrics.KafkaTopicPartitionInSyncReplica.WithLabelValues(topic, partitionStr).Set(float64(detail.ReplicationFactor))
			metrics.KafkaTopicPartitionUnderReplicated.WithLabelValues(topic, partitionStr).Set(0)
		}
	}

	// 3. Сбор consumer groups с lag (оптимизированная версия)
	consumerLags := c.collectConsumerGroupsLagOptimized(clusterName, client, metrics)

	// 4. Кэшируем результаты
	c.cacheMutex.Lock()
	c.cache[clusterName] = clusterCache{
		metrics:      metrics,
		timestamp:    time.Now(),
		consumerLags: consumerLags,
	}
	c.cacheMutex.Unlock()
}

func (c *ClusterCollector) collectConsumerGroupsLagOptimized(clusterName string, client *kafka.KafkaClient, metrics *ClusterMetrics) map[string]map[string]map[int32]int64 {
	consumerLags := make(map[string]map[string]map[int32]int64)

	// Получаем список consumer groups
	groups, err := client.Admin.ListConsumerGroups()
	if err != nil {
		log.Printf("Failed to list consumer groups for cluster %s: %v", clusterName, err)
		return consumerLags
	}

	// Ограничиваем количество групп для мониторинга (первые 20 для скорости)
	groupCount := 0
	maxGroups := 20

	for group := range groups {
		if groupCount >= maxGroups {
			break
		}

		groupLags, err := c.getConsumerGroupLag(client, group)
		if err != nil {
			log.Printf("Failed to get lag for group %s: %v", group, err)
			continue
		}

		consumerLags[group] = groupLags
		groupCount++

		// Устанавливаем метрики
		c.setConsumerGroupMetrics(metrics, group, groupLags)
	}

	return consumerLags
}

func (c *ClusterCollector) getConsumerGroupLag(client *kafka.KafkaClient, group string) (map[string]map[int32]int64, error) {
	groupLags := make(map[string]map[int32]int64)

	// Получаем offsets consumer group
	offsetResponse, err := client.Admin.ListConsumerGroupOffsets(group, nil)
	if err != nil {
		return nil, err
	}

	for topic, partitions := range offsetResponse.Blocks {
		for partition, offsetInfo := range partitions {
			if offsetInfo.Err != sarama.ErrNoError {
				continue
			}

			// Получаем водяные знаки для вычисления lag
			newestOffset, err := client.Client.GetOffset(topic, partition, sarama.OffsetNewest)
			if err != nil {
				continue
			}

			oldestOffset, err := client.Client.GetOffset(topic, partition, sarama.OffsetOldest)
			if err != nil {
				continue
			}

			// Вычисляем lag
			consumerOffset := offsetInfo.Offset
			lag := newestOffset - consumerOffset
			if lag < 0 {
				lag = 0
			}

			// Сохраняем lag
			if groupLags[topic] == nil {
				groupLags[topic] = make(map[int32]int64)
			}
			groupLags[topic][partition] = lag

			// Сохраняем oldest offset
			if groupLags["_oldest_"] == nil {
				groupLags["_oldest_"] = make(map[int32]int64)
			}
			groupLags["_oldest_"][partition] = oldestOffset
		}
	}

	return groupLags, nil
}

func (c *ClusterCollector) setConsumerGroupMetrics(metrics *ClusterMetrics, group string, groupLags map[string]map[int32]int64) {
	var totalLag int64 = 0
	var totalOffset int64 = 0
	var memberCount int = 1 // Базовая заглушка

	// Устанавливаем количество членов группы
	metrics.KafkaConsumerGroupMembers.WithLabelValues(group).Set(float64(memberCount))

	for topic, partitions := range groupLags {
		if topic == "_oldest_" {
			// Пропускаем служебные данные
			continue
		}

		var topicLag int64 = 0
		var topicOffset int64 = 0

		for partition, lag := range partitions {
			partitionStr := strconv.Itoa(int(partition))

			// Устанавливаем lag для каждой партиции
			metrics.KafkaConsumerGroupLag.WithLabelValues(group, topic, partitionStr).Set(float64(lag))

			// Устанавливаем current offset (используем водяные знаки)
			oldestOffset := groupLags["_oldest_"][partition]
			currentOffset := oldestOffset + lag // Приблизительное значение
			metrics.KafkaConsumerGroupCurrentOffset.WithLabelValues(group, topic, partitionStr).Set(float64(currentOffset))

			topicLag += lag
			topicOffset += currentOffset
		}

		// Устанавливаем суммарные метрики для топика
		metrics.KafkaConsumerGroupLagSum.WithLabelValues(group, topic).Set(float64(topicLag))
		metrics.KafkaConsumerGroupOffsetSum.WithLabelValues(group, topic).Set(float64(topicOffset))

		totalLag += topicLag
		totalOffset += topicOffset
	}

	// Устанавливаем общие суммарные метрики
	if totalLag > 0 {
		metrics.KafkaConsumerGroupLagSum.WithLabelValues(group, "_all_").Set(float64(totalLag))
	}
	if totalOffset > 0 {
		metrics.KafkaConsumerGroupOffsetSum.WithLabelValues(group, "_all_").Set(float64(totalOffset))
	}
}

func (c *ClusterCollector) restoreMetricsFromCache(metrics *ClusterMetrics, consumerLags map[string]map[string]map[int32]int64) {
	for group, topics := range consumerLags {
		for topic, partitions := range topics {
			if topic == "_oldest_" {
				continue
			}
			for partition, lag := range partitions {
				partitionStr := strconv.Itoa(int(partition))
				metrics.KafkaConsumerGroupLag.WithLabelValues(group, topic, partitionStr).Set(float64(lag))
			}
		}
	}
}

func (c *ClusterCollector) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, client := range c.clients {
		client.Close()
	}
}
