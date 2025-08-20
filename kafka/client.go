package kafka

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
)

type KafkaClient struct {
	Client      sarama.Client
	Admin       sarama.ClusterAdmin
	ClusterName string
}

func NewKafkaClient(config *sarama.Config, brokers []string, clusterName string) (*KafkaClient, error) {
	client, err := sarama.NewClient(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka client for %s: %v", clusterName, err)
	}

	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka admin for %s: %v", clusterName, err)
	}

	return &KafkaClient{
		Client:      client,
		Admin:       admin,
		ClusterName: clusterName,
	}, nil
}

func (kc *KafkaClient) Close() error {
	if kc.Admin != nil {
		kc.Admin.Close()
	}
	if kc.Client != nil {
		return kc.Client.Close()
	}
	return nil
}

func (kc *KafkaClient) GetClusterMetrics() (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	// Получаем список брокеров
	brokers := kc.Client.Brokers()
	metrics["brokers_count"] = len(brokers)

	// Получаем список топиков
	topics, err := kc.Client.Topics()
	if err != nil {
		return nil, err
	}
	metrics["topics_count"] = len(topics)

	// Получаем информацию о consumer groups
	groups, err := kc.Admin.ListConsumerGroups()
	if err != nil {
		log.Printf("Failed to list consumer groups for cluster %s: %v", kc.ClusterName, err)
	} else {
		metrics["consumer_groups_count"] = len(groups)
	}

	return metrics, nil
}

func (kc *KafkaClient) GetDetailedMetrics() (map[string]interface{}, error) {
	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	metrics := make(map[string]interface{})

	// Проверяем, поддерживает ли Admin интерфейс с контекстом
	// Для sarama может потребоваться отдельная реализация
	// В данном случае используем обычный вызов, но в фоне

	// Детальная информация о топиках
	topics, err := kc.Admin.ListTopics()
	if err != nil {
		return nil, err
	}

	// Проверяем контекст на таймаут
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout exceeded while processing topics for cluster %s", kc.ClusterName)
	default:
		// Продолжаем выполнение
	}

	topicDetails := make(map[string]interface{})
	for topic, detail := range topics {
		// Периодически проверяем контекст
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout exceeded while processing topic %s", topic)
		default:
			topicDetails[topic] = map[string]interface{}{
				"partitions":         detail.NumPartitions,
				"replication_factor": detail.ReplicationFactor,
			}
		}
	}
	metrics["topics"] = topicDetails

	return metrics, nil
}

// Вспомогательная функция для парсинга версии Kafka
func ParseKafkaVersion(version string) sarama.KafkaVersion {
	parsedVersion, err := sarama.ParseKafkaVersion(version)
	if err != nil {
		// Возвращаем версию по умолчанию
		return sarama.V2_8_0_0
	}
	return parsedVersion
}
