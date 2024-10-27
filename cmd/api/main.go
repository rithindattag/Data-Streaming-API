package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/rithindattag/realtime-streaming-api/internal/api"
	"github.com/rithindattag/realtime-streaming-api/internal/kafka"
	"github.com/rithindattag/realtime-streaming-api/internal/websocket"
	"github.com/rithindattag/realtime-streaming-api/pkg/logger"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Get environment variables
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		log.Fatal("KAFKA_BROKERS environment variable is not set")
	}

	kafkaTopic := os.Getenv("KAFKA_TOPIC")
	if kafkaTopic == "" {
		log.Fatal("KAFKA_TOPIC environment variable is not set")
	}

	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		apiPort = "8000" // Default port if not set
	}

	// Initialize logger
	logger := logger.NewLogger()

	// Initialize Kafka producer
	producer, err := kafka.NewProducer(kafkaBrokers)
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer producer.Close()

	// Initialize Kafka consumer
	consumer, err := kafka.NewConsumer(kafkaBrokers, kafkaTopic, logger)
	if err != nil {
		log.Fatalf("Failed to create Kafka consumer: %v", err)
	}
	defer consumer.Close()

	// Initialize WebSocket hub
	hub := websocket.NewHub(logger)
	go hub.Run()

	// Start consuming messages and broadcasting to WebSocket clients
	go consumer.ConsumeMessages(hub)

	// Initialize and start API server
	handlers := api.NewHandlers(producer, consumer, hub, logger)
	router := api.NewRouter(handlers)
	
	log.Printf("Starting server on :%s", apiPort)
	if err := http.ListenAndServe(":"+apiPort, router); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
