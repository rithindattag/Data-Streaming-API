// Package main is the entry point for the streaming API server
package main

import (
	"os"
	"runtime"
	"time"

	"github.com/joho/godotenv"
	"github.com/rithindattag/realtime-streaming-api/internal/api"
	"github.com/rithindattag/realtime-streaming-api/internal/kafka"
	"github.com/rithindattag/realtime-streaming-api/internal/websocket"
	"github.com/rithindattag/realtime-streaming-api/pkg/logger"
	"github.com/valyala/fasthttp"
	"golang.org/x/sys/unix"
)

// main is the entry point for the API server
func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		logger.NewLogger().Info("No .env file found, using system environment variables")
	}

	// Get environment variables
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	kafkaTopic := os.Getenv("KAFKA_TOPIC")

	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		apiPort = "8000" // Default port if not set
	}

	// Increase the maximum number of open files
	var rLimit unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &rLimit); err != nil {
		logger.NewLogger().Error("Error getting rlimit", "error", err)
		os.Exit(1)
	}
	rLimit.Cur = rLimit.Max
	if err := unix.Setrlimit(unix.RLIMIT_NOFILE, &rLimit); err != nil {
		logger.NewLogger().Error("Error setting rlimit", "error", err)
		os.Exit(1)
	}

	// Set the maximum number of CPUs that can be executing simultaneously
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Initialize logger
	log := logger.NewLogger()

	// Initialize Kafka producer
	producer, err := kafka.NewProducer(kafkaBrokers)
	if err != nil {
		log.Error("Failed to create Kafka producer", "error", err)
		os.Exit(1)
	}
	defer producer.Close()

	log.Info("Kafka configuration", "brokers", kafkaBrokers, "topic", kafkaTopic)

	// Initialize Kafka consumer
	consumer, err := kafka.NewConsumer(kafkaBrokers, kafkaTopic, log)
	if err != nil {
		log.Error("Failed to create Kafka consumer", "error", err)
		os.Exit(1)
	}
	defer consumer.Close()

	// Initialize WebSocket hub
	hub := websocket.NewHub(log)
	go hub.Run()

	// Start consuming messages and broadcasting to WebSocket clients
	go consumer.ConsumeMessages(hub)

	// Initialize and start API server
	handlers := api.NewHandlers(producer, consumer, hub, log)
	router := api.NewRouter(handlers)

	server := &fasthttp.Server{
		Handler:            router,
		Name:               "FastHTTP",
		Concurrency:        1000000,
		MaxConnsPerIP:      1000,
		MaxRequestsPerConn: 10000,
		WriteTimeout:       30 * time.Second,
		ReadTimeout:        30 * time.Second,
		IdleTimeout:        60 * time.Second,
	}

	log.Info("Starting server", "port", apiPort, "kafkaBrokers", kafkaBrokers, "kafkaTopic", kafkaTopic)
	if err := server.ListenAndServe(":" + apiPort); err != nil {
		log.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}
