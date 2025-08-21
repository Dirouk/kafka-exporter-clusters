package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"kafka-exporter-clusters/collector"
	"kafka-exporter-clusters/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Создаем registry
	registry := prometheus.NewRegistry()

	// Регистрируем стандартные Go метрики
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	// Создаем и регистрируем наш кастомный коллектор
	kafkaCollector, err := collector.NewClusterCollector(cfg)
	if err != nil {
		log.Fatalf("Failed to create collector: %v", err)
	}
	defer kafkaCollector.Close()
	registry.MustRegister(kafkaCollector)

	// Создаем handler с нашим registry
	metricsHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	// Настраиваем HTTP сервер
	http.Handle(cfg.MetricsPath, metricsHandler)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
			<html>
			<head><title>Kafka Exporter Clusters</title></head>
			<body>
				<h1>Kafka Exporter Clusters</h1>
				<p><a href="` + cfg.MetricsPath + `">Metrics</a></p>
				<p><a href="/health">Health</a></p>
			</body>
			</html>
		`))
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Запускаем сервер
	log.Printf("Starting Kafka exporter on %s", cfg.ListenAddress)

	// Канал для graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := http.ListenAndServe(cfg.ListenAddress, nil); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Ожидаем сигнал завершения
	<-stopChan
	log.Println("Shutting down Kafka exporter...")
}
