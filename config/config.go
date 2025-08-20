package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type KafkaConfig struct {
	Name          string   `yaml:"name"`
	Brokers       []string `yaml:"brokers"`
	Version       string   `yaml:"version"`
	TLSCertFile   string   `yaml:"tls_cert_file,omitempty"`
	TLSKeyFile    string   `yaml:"tls_key_file,omitempty"`
	TLSCAFile     string   `yaml:"tls_ca_file,omitempty"`
	SASLMechanism string   `yaml:"sasl_mechanism,omitempty"`
	SASLUsername  string   `yaml:"sasl_username,omitempty"`
	SASLPassword  string   `yaml:"sasl_password,omitempty"`
}

type ExporterConfig struct {
	ListenAddress   string        `yaml:"listen_address"`
	MetricsPath     string        `yaml:"metrics_path"`
	RefreshInterval int           `yaml:"refresh_interval"`
	Clusters        []KafkaConfig `yaml:"clusters"`
}

func LoadConfig(configFile string) (*ExporterConfig, error) {
	// Для теста создаем конфиг по умолчанию, если файл не существует
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return &ExporterConfig{
			ListenAddress:   ":9308",
			MetricsPath:     "/metrics",
			RefreshInterval: 30,
			Clusters: []KafkaConfig{
				{
					Name:    "test-cluster",
					Brokers: []string{"localhost:9092"},
					Version: "2.8.0",
				},
			},
		}, nil
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var config ExporterConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
