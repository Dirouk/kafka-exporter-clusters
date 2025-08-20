package collector

import (
	"github.com/prometheus/client_golang/prometheus"
)

type ClusterMetrics struct {
	ClusterName string

	// Базовые метрики кластера
	BrokersTotal         prometheus.Gauge
	TopicsTotal          prometheus.Gauge
	ConsumerGroupsTotal  prometheus.Gauge
	ControllerBroker     prometheus.Gauge
	OfflinePartitions    prometheus.Gauge
	UnderReplicatedParts prometheus.Gauge

	// Метрики по топикам
	TopicPartitions  *prometheus.GaugeVec
	TopicReplication *prometheus.GaugeVec
	TopicSize        *prometheus.GaugeVec
	TopicMessages    *prometheus.CounterVec

	// Метрики по consumer groups
	GroupLag     *prometheus.GaugeVec
	GroupMembers *prometheus.GaugeVec
	GroupOffset  *prometheus.GaugeVec

	// Метрики по брокерам
	BrokerDiskUsage   *prometheus.GaugeVec
	BrokerNetworkRate *prometheus.GaugeVec
	BrokerCPUUsage    *prometheus.GaugeVec
}

func NewClusterMetrics(clusterName string) *ClusterMetrics {
	namespace := "kafka"
	labels := prometheus.Labels{"cluster": clusterName}

	return &ClusterMetrics{
		ClusterName: clusterName,

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

		TopicPartitions: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "topic_partitions",
				Help:        "Number of partitions per topic",
				ConstLabels: labels,
			},
			[]string{"topic"},
		),

		TopicReplication: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "topic_replication_factor",
				Help:        "Replication factor per topic",
				ConstLabels: labels,
			},
			[]string{"topic"},
		),

		GroupLag: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "consumer_group_lag",
				Help:        "Consumer group lag per topic and partition",
				ConstLabels: labels,
			},
			[]string{"group", "topic", "partition"},
		),

		// Добавьте другие метрики по аналогии
	}
}

func (m *ClusterMetrics) Describe(ch chan<- *prometheus.Desc) {
	m.BrokersTotal.Describe(ch)
	m.TopicsTotal.Describe(ch)
	m.TopicPartitions.Describe(ch)
	m.TopicReplication.Describe(ch)
	m.GroupLag.Describe(ch)
}

func (m *ClusterMetrics) Collect(ch chan<- prometheus.Metric) {
	m.BrokersTotal.Collect(ch)
	m.TopicsTotal.Collect(ch)
	m.TopicPartitions.Collect(ch)
	m.TopicReplication.Collect(ch)
	m.GroupLag.Collect(ch)
}
