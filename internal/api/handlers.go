package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
	"golang.org/x/time/rate"
	"github.com/google/uuid"

	"github.com/gorilla/mux"
	"github.com/rithindattag/realtime-streaming-api/internal/kafka"
	"github.com/rithindattag/realtime-streaming-api/internal/websocket"
	"github.com/rithindattag/realtime-streaming-api/pkg/logger"
	"github.com/rithindattag/realtime-streaming-api/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	limiter = rate.NewLimiter(rate.Limit(100), 200) // 100 requests per second with burst of 200
	clients = make(map[string]*rate.Limiter)
	mu      sync.Mutex
)

func getClientLimiter(clientIP string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	limiter, exists := clients[clientIP]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(10), 20) // 10 requests per second with burst of 20
		clients[clientIP] = limiter
	}

	return limiter
}

func RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := r.RemoteAddr

		if !getClientLimiter(clientIP).Allow() {
			respondWithError(w, http.StatusTooManyRequests, "Rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	}
}

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
		respondWithError(w, http.StatusNotFound, "Stream not found")
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
		respondWithError(w, http.StatusInternalServerError, "Failed to process data")
		return
	}
	h.Logger.Info("Message sent to Kafka", "stream_id", streamID)

	metrics.MessagesSent.WithLabelValues(streamID).Inc()

	h.Logger.Info("Data sent to stream", "stream_id", streamID)
	respondWithJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
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
	return uuid.New().String()
}

func (h *Handlers) streamExists(streamID string) bool {
	h.StreamsMutex.RLock()
	defer h.StreamsMutex.RUnlock()
	return h.ActiveStreams[streamID]
}

var (
	requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint"},
	)

	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)
)

func MetricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)

		requestsTotal.WithLabelValues(r.Method, r.URL.Path).Inc()
		requestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration.Seconds())
	}
}

// Add this function at the beginning of the file
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
