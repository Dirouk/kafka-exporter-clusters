package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"your-project/collector"
	"your-project/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Создаем коллектор
	collector, err := collector.NewClusterCollector(cfg)
	if err != nil {
		log.Fatalf("Failed to create collector: %v", err)
	}
	defer collector.Close()

	// Регистрируем коллектор
	prometheus.MustRegister(collector)

	// Настраиваем HTTP сервер
	http.Handle(cfg.MetricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
			<html>
			<head><title>Kafka Exporter</title></head>
			<body>
				<h1>Kafka Exporter</h1>
				<p><a href="` + cfg.MetricsPath + `">Metrics</a></p>
			</body>
			</html>
		`))
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Запускаем сервер
	go func() {
		log.Printf("Starting Kafka exporter on %s", cfg.ListenAddress)
		if err := http.ListenAndServe(cfg.ListenAddress, nil); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Ожидаем сигналов для graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	log.Println("Shutting down Kafka exporter...")
}
