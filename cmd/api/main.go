package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"

	"github.com/rithindattag/realtime-streaming-api/internal/api"
	"github.com/rithindattag/realtime-streaming-api/internal/kafka"
	"github.com/rithindattag/realtime-streaming-api/internal/websocket"
	"github.com/rithindattag/realtime-streaming-api/pkg/auth"
	"github.com/rithindattag/realtime-streaming-api/pkg/logger"
	"github.com/rithindattag/realtime-streaming-api/pkg/ratelimit"
)

func testKafkaConnection(bootstrapServers string) error {
	producer, err := kafka.NewProducer(bootstrapServers)
	if err != nil {
		return fmt.Errorf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	log.Printf("Producer created successfully for %s", bootstrapServers)
	return nil
}

func main() {
	// Initialize logger
	logger := logger.NewLogger()

	// Initialize Kafka producer and consumer
	producer, err := kafka.NewProducer("localhost:9092")
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer producer.Close()

	consumer, err := kafka.NewConsumer("localhost:9092", "test-group")
	if err != nil {
		log.Fatalf("Failed to create Kafka consumer: %v", err)
	}
	defer consumer.Close()

	err = testKafkaConnection("localhost:9092")
	if err != nil {
		log.Fatalf("Failed to connect to Kafka: %v", err)
	}

	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	handlers := api.NewHandlers(producer, consumer, hub, logger)
	rateLimiter := ratelimit.NewRateLimiter(rate.Limit(100), 5)

	// Initialize API routes
	router := mux.NewRouter()
	router.HandleFunc("/stream/start", auth.APIKeyMiddleware(rateLimiter.Middleware(handlers.StartStream))).Methods("POST")
	router.HandleFunc("/stream/{stream_id}/send", auth.APIKeyMiddleware(rateLimiter.Middleware(handlers.SendData))).Methods("POST")

	// Remove the API key middleware for the WebSocket endpoint
	router.HandleFunc("/stream/{stream_id}/results", handlers.StreamResults).Methods("GET")

	// Prometheus metrics endpoint
	router.Handle("/metrics", promhttp.Handler())

	// Start the server
	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Your API server setup and logic here...
}
