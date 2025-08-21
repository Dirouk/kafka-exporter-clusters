package collector

import (
	"github.com/prometheus/client_golang/prometheus"
)

type ClusterMetrics struct {
	ClusterName string

	// Метрики как в стандартном kafka-exporter
	KafkaBrokers                       prometheus.Gauge
	KafkaBrokerInfo                    *prometheus.GaugeVec
	KafkaTopicPartitions               *prometheus.GaugeVec
	KafkaTopicPartitionCurrentOffset   *prometheus.GaugeVec
	KafkaTopicPartitionOldestOffset    *prometheus.GaugeVec
	KafkaTopicPartitionLeader          *prometheus.GaugeVec
	KafkaTopicPartitionReplicas        *prometheus.GaugeVec
	KafkaTopicPartitionInSyncReplica   *prometheus.GaugeVec
	KafkaTopicPartitionUnderReplicated *prometheus.GaugeVec
	KafkaConsumerGroupCurrentOffset    *prometheus.GaugeVec
	KafkaConsumerGroupLag              *prometheus.GaugeVec
	KafkaConsumerGroupMembers          *prometheus.GaugeVec
	KafkaConsumerGroupLagSum           *prometheus.GaugeVec
	KafkaConsumerGroupOffsetSum        *prometheus.GaugeVec

	// Дополнительные метрики
	BrokersTotal prometheus.Gauge
	TopicsTotal  prometheus.Gauge
}

func NewClusterMetrics(clusterName string) *ClusterMetrics {
	namespace := "kafka"
	labels := prometheus.Labels{"cluster": clusterName}

	return &ClusterMetrics{
		ClusterName: clusterName,

		KafkaBrokers: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   namespace,
			Name:        "brokers",
			Help:        "Number of Brokers in the Kafka Cluster",
			ConstLabels: labels,
		}),

		KafkaBrokerInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "broker_info",
				Help:        "Information about the Kafka Broker",
				ConstLabels: labels,
			},
			[]string{"address", "id"},
		),

		KafkaTopicPartitions: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "topic_partitions",
				Help:        "Number of partitions for this Topic",
				ConstLabels: labels,
			},
			[]string{"topic"},
		),

		KafkaTopicPartitionCurrentOffset: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "topic_partition_current_offset",
				Help:        "Current Offset of a Broker at Topic/Partition",
				ConstLabels: labels,
			},
			[]string{"topic", "partition"},
		),

		KafkaTopicPartitionOldestOffset: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "topic_partition_oldest_offset",
				Help:        "Oldest Offset of a Broker at Topic/Partition",
				ConstLabels: labels,
			},
			[]string{"topic", "partition"},
		),

		KafkaTopicPartitionLeader: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "topic_partition_leader",
				Help:        "Leader Broker ID of this Topic/Partition",
				ConstLabels: labels,
			},
			[]string{"topic", "partition"},
		),

		KafkaTopicPartitionReplicas: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "topic_partition_replicas",
				Help:        "Number of Replicas for this Topic/Partition",
				ConstLabels: labels,
			},
			[]string{"topic", "partition"},
		),

		KafkaTopicPartitionInSyncReplica: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "topic_partition_in_sync_replica",
				Help:        "Number of In-Sync Replicas for this Topic/Partition",
				ConstLabels: labels,
			},
			[]string{"topic", "partition"},
		),

		KafkaTopicPartitionUnderReplicated: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "topic_partition_under_replicated_partition",
				Help:        "1 if Topic/Partition is under Replicated",
				ConstLabels: labels,
			},
			[]string{"topic", "partition"},
		),

		KafkaConsumerGroupCurrentOffset: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "consumergroup_current_offset",
				Help:        "Current Offset of a ConsumerGroup at Topic/Partition",
				ConstLabels: labels,
			},
			[]string{"consumergroup", "topic", "partition"},
		),

		KafkaConsumerGroupLag: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "consumergroup_lag",
				Help:        "Current Approximate Lag of a ConsumerGroup at Topic/Partition",
				ConstLabels: labels,
			},
			[]string{"consumergroup", "topic", "partition"},
		),

		KafkaConsumerGroupMembers: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "consumergroup_members",
				Help:        "Amount of members in a consumer group",
				ConstLabels: labels,
			},
			[]string{"consumergroup"},
		),

		KafkaConsumerGroupLagSum: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "consumergroup_lag_sum",
				Help:        "Current Approximate Lag of a ConsumerGroup at Topic for all partitions",
				ConstLabels: labels,
			},
			[]string{"consumergroup", "topic"},
		),

		KafkaConsumerGroupOffsetSum: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "consumergroup_current_offset_sum",
				Help:        "Current Offset of a ConsumerGroup at Topic for all partitions",
				ConstLabels: labels,
			},
			[]string{"consumergroup", "topic"},
		),

		// Дополнительные метрики
		BrokersTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   namespace,
			Name:        "brokers_total",
			Help:        "Total number of brokers in the cluster",
			ConstLabels: labels,
		}),

		TopicsTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   namespace,
			Name:        "topics_total",
			Help:        "Total number of topics in the cluster",
			ConstLabels: labels,
		}),
	}
}

func (m *ClusterMetrics) Describe(ch chan<- *prometheus.Desc) {
	m.KafkaBrokers.Describe(ch)
	m.KafkaBrokerInfo.Describe(ch)
	m.KafkaTopicPartitions.Describe(ch)
	m.KafkaTopicPartitionCurrentOffset.Describe(ch)
	m.KafkaTopicPartitionOldestOffset.Describe(ch)
	m.KafkaTopicPartitionLeader.Describe(ch)
	m.KafkaTopicPartitionReplicas.Describe(ch)
	m.KafkaTopicPartitionInSyncReplica.Describe(ch)
	m.KafkaTopicPartitionUnderReplicated.Describe(ch)
	m.KafkaConsumerGroupCurrentOffset.Describe(ch)
	m.KafkaConsumerGroupLag.Describe(ch)
	m.KafkaConsumerGroupMembers.Describe(ch)
	m.KafkaConsumerGroupLagSum.Describe(ch)
	m.KafkaConsumerGroupOffsetSum.Describe(ch)
	m.BrokersTotal.Describe(ch)
	m.TopicsTotal.Describe(ch)
}

func (m *ClusterMetrics) Collect(ch chan<- prometheus.Metric) {
	m.KafkaBrokers.Collect(ch)
	m.KafkaBrokerInfo.Collect(ch)
	m.KafkaTopicPartitions.Collect(ch)
	m.KafkaTopicPartitionCurrentOffset.Collect(ch)
	m.KafkaTopicPartitionOldestOffset.Collect(ch)
	m.KafkaTopicPartitionLeader.Collect(ch)
	m.KafkaTopicPartitionReplicas.Collect(ch)
	m.KafkaTopicPartitionInSyncReplica.Collect(ch)
	m.KafkaTopicPartitionUnderReplicated.Collect(ch)
	m.KafkaConsumerGroupCurrentOffset.Collect(ch)
	m.KafkaConsumerGroupLag.Collect(ch)
	m.KafkaConsumerGroupMembers.Collect(ch)
	m.KafkaConsumerGroupLagSum.Collect(ch)
	m.KafkaConsumerGroupOffsetSum.Collect(ch)
	m.BrokersTotal.Collect(ch)
	m.TopicsTotal.Collect(ch)
}
