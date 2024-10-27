package api

import (
	"encoding/json"
	"net/http"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/rithindattag/realtime-streaming-api/internal/kafka"
	"github.com/rithindattag/realtime-streaming-api/internal/websocket"
	"github.com/rithindattag/realtime-streaming-api/pkg/logger"
	"github.com/rithindattag/realtime-streaming-api/internal/metrics"
)

type Handlers struct {
	Producer      *kafka.Producer
	Consumer      *kafka.Consumer
	Hub           *websocket.Hub
	Logger        *logger.Logger
	ActiveStreams map[string]bool
	StreamsMutex  sync.RWMutex
}

// Add a constructor function
func NewHandlers(producer *kafka.Producer, consumer *kafka.Consumer, hub *websocket.Hub, logger *logger.Logger) *Handlers {
	return &Handlers{
		Producer:      producer,
		Consumer:      consumer,
		Hub:           hub,
		Logger:        logger,
		ActiveStreams: make(map[string]bool),
		StreamsMutex:  sync.RWMutex{},
	}
}

func (h *Handlers) StartStream(w http.ResponseWriter, r *http.Request) {
	// Generate a unique stream ID
	streamID := generateUniqueID()

	h.StreamsMutex.Lock()
	h.ActiveStreams[streamID] = true
	h.StreamsMutex.Unlock()

	// Create a new stream in the WebSocket hub
	h.Hub.CreateStream(streamID)

	h.Logger.Info("New stream created", "stream_id", streamID)

	metrics.StreamsCreated.Inc()

	// Return the stream ID to the client
	json.NewEncoder(w).Encode(map[string]string{"stream_id": streamID})
}

func (h *Handlers) SendData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	streamID := vars["stream_id"]

	if !h.streamExists(streamID) {
		h.Logger.Error("Stream not found", "stream_id", streamID)
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	h.Logger.Info("Received data for stream", "stream_id", streamID)

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		metrics.ProcessingTime.WithLabelValues(streamID).Observe(duration)
	}()

	metrics.MessagesReceived.WithLabelValues(streamID).Inc()

	// Parse the incoming data
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.Logger.Error("Failed to parse JSON data", "error", err, "body", r.Body)
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}

	h.Logger.Info("Parsed data", "data", data)

	// Convert the data to a JSON string
	jsonData, err := json.Marshal(data)
	if err != nil {
		h.Logger.Error("Failed to marshal data to JSON", "error", err)
		http.Error(w, "Failed to process data", http.StatusInternalServerError)
		return
	}

	h.Logger.Info("Attempting to send message to Kafka", "stream_id", streamID)
	if err := h.Producer.SendMessage(streamID, jsonData); err != nil {
		h.Logger.Error("Failed to send message to Kafka", "error", err, "stream_id", streamID)
		http.Error(w, "Failed to process data", http.StatusInternalServerError)
		return
	}
	h.Logger.Info("Message sent to Kafka", "stream_id", streamID)

	metrics.MessagesSent.WithLabelValues(streamID).Inc()

	h.Logger.Info("Data sent to stream", "stream_id", streamID)
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

func (h *Handlers) StreamResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	streamID := vars["stream_id"]

	if !h.streamExists(streamID) {
		h.Logger.Error("Stream not found", "stream_id", streamID)
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := websocket.Upgrade(w, r)
	if err != nil {
		h.Logger.Error("Failed to upgrade to WebSocket", "error", err)
		return
	}

	// Register the client with the WebSocket hub
	client := &websocket.Client{StreamID: streamID, Conn: conn, Send: make(chan []byte, 256)}
	h.Hub.Register <- client

	// Start goroutines for reading and writing messages
	go client.WritePump()
	go client.ReadPump()
}

func generateUniqueID() string {
	// Implement a function to generate a unique ID (e.g., UUID)
	// For simplicity, we'll use a placeholder implementation
	return "stream-123"
}

func (h *Handlers) streamExists(streamID string) bool {
	h.StreamsMutex.RLock()
	defer h.StreamsMutex.RUnlock()
	return h.ActiveStreams[streamID]
}
