package collector

import (
	"crypto/tls"
	"fmt"
	"log"
	"sync"

	"your-project/config"
	"your-project/kafka"

	"github.com/IBM/sarama"
	"github.com/prometheus/client_golang/prometheus"
)

type ClusterCollector struct {
	clients map[string]*kafka.KafkaClient
	metrics map[string]*ClusterMetrics
	mutex   sync.Mutex
	config  *config.ExporterConfig
}

func NewClusterCollector(cfg *config.ExporterConfig) (*ClusterCollector, error) {
	collector := &ClusterCollector{
		clients: make(map[string]*kafka.KafkaClient),
		metrics: make(map[string]*ClusterMetrics),
		config:  cfg,
	}

	// Инициализируем клиенты для всех кластеров
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

	// Создаем конфиг Sarama
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = kafka.ParseKafkaVersion(cfg.Version)

	// Настройка TLS если нужно
	if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		tlsConfig, err := createTLSConfig(cfg.TLSCertFile, cfg.TLSKeyFile, cfg.TLSCAFile)
		if err != nil {
			return fmt.Errorf("failed to create TLS config: %v", err)
		}
		saramaConfig.Net.TLS.Enable = true
		saramaConfig.Net.TLS.Config = tlsConfig
	}

	// Настройка SASL если нужно
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

	var wg sync.WaitGroup

	for clusterName, client := range c.clients {
		wg.Add(1)
		go func(name string, cl *kafka.KafkaClient) {
			defer wg.Done()
			c.collectForCluster(name, cl, ch)
		}(clusterName, client)
	}

	wg.Wait()
}

func (c *ClusterCollector) collectForCluster(clusterName string, client *kafka.KafkaClient, ch chan<- prometheus.Metric) {
	metrics := c.metrics[clusterName]

	// Собираем базовые метрики
	clusterMetrics, err := client.GetClusterMetrics()
	if err != nil {
		log.Printf("Failed to get metrics for cluster %s: %v", clusterName, err)
		return
	}

	// Обновляем метрики
	if brokers, ok := clusterMetrics["brokers_count"].(int); ok {
		metrics.BrokersTotal.Set(float64(brokers))
	}

	if topics, ok := clusterMetrics["topics_count"].(int); ok {
		metrics.TopicsTotal.Set(float64(topics))
	}

	// Собираем детальную информацию
	detailedMetrics, err := client.GetDetailedMetrics()
	if err != nil {
		log.Printf("Failed to get detailed metrics for cluster %s: %v", clusterName, err)
		return
	}

	// Обновляем метрики топиков
	if topics, ok := detailedMetrics["topics"].(map[string]interface{}); ok {
		for topic, details := range topics {
			if topicDetails, ok := details.(map[string]interface{}); ok {
				if partitions, ok := topicDetails["partitions"].(int32); ok {
					metrics.TopicPartitions.WithLabelValues(topic).Set(float64(partitions))
				}
				if replication, ok := topicDetails["replication_factor"].(int16); ok {
					metrics.TopicReplication.WithLabelValues(topic).Set(float64(replication))
				}
			}
		}
	}

	// Собираем метрики
	metrics.Collect(ch)
}

func (c *ClusterCollector) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, client := range c.clients {
		client.Close()
	}
}

// Вспомогательная функция для создания TLS конфигурации
func createTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	// Упрощенная реализация - в production нужно загружать реальные сертификаты
	return &tls.Config{
		InsecureSkipVerify: true, // Для тестирования
	}, nil
}
