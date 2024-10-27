package api

import (
	"encoding/json"
	"sync"
	"golang.org/x/time/rate"
	"github.com/google/uuid"
	"strings"
	"fmt"

	"github.com/valyala/fasthttp"
	"github.com/rithindattag/realtime-streaming-api/internal/kafka"
	"github.com/rithindattag/realtime-streaming-api/internal/websocket"
	"github.com/rithindattag/realtime-streaming-api/pkg/logger"
	"github.com/rithindattag/realtime-streaming-api/internal/metrics"
)

var (
	globalLimiter = rate.NewLimiter(rate.Limit(50000), 100000) // 50000 requests per second with burst of 100000
)

type Handlers struct {
	Producer      *kafka.Producer
	Consumer      *kafka.Consumer
	Hub           *websocket.Hub
	Logger        *logger.Logger
	ActiveStreams map[string]bool
	StreamsMutex  sync.RWMutex
}

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

func (h *Handlers) HandleFastHTTP(ctx *fasthttp.RequestCtx) {
	switch string(ctx.Path()) {
	case "/stream/start":
		h.StartStream(ctx)
	case "/stream/{stream_id}/send":
		h.SendData(ctx)
	case "/stream/{stream_id}/results":
		h.StreamResults(ctx)
	default:
		ctx.Error("Not found", fasthttp.StatusNotFound)
	}
}

func (h *Handlers) StartStream(ctx *fasthttp.RequestCtx) {
	streamID := uuid.New().String()

	h.StreamsMutex.Lock()
	h.ActiveStreams[streamID] = true
	h.StreamsMutex.Unlock()

	h.Hub.CreateStream(streamID)

	h.Logger.Info("New stream created", "stream_id", streamID)

	metrics.StreamsCreated.Inc()

	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(map[string]string{"stream_id": streamID})
}

func (h *Handlers) SendData(ctx *fasthttp.RequestCtx) {
	h.Logger.Info("SendData function called")

	// Log the request headers and body
	h.Logger.Info("Request headers", "headers", ctx.Request.Header.String())
	h.Logger.Info("Request body", "body", string(ctx.PostBody()))

	if !globalLimiter.Allow() {
		h.Logger.Error("Rate limit exceeded")
		ctx.Error("Too many requests", fasthttp.StatusTooManyRequests)
		return
	}

	path := string(ctx.Path())
	h.Logger.Info("Request path", "path", path)

	parts := strings.Split(path, "/")
	if len(parts) != 4 || parts[1] != "stream" || parts[3] != "send" {
		h.Logger.Error("Invalid path", "path", path)
		ctx.Error("Invalid path", fasthttp.StatusBadRequest)
		return
	}
	streamID := parts[2]
	h.Logger.Info("Stream ID extracted", "streamID", streamID)

	if !h.streamExists(streamID) {
		h.Logger.Error("Stream not found", "stream_id", streamID)
		ctx.Error("Stream not found", fasthttp.StatusNotFound)
		return
	}

	h.Logger.Info("Received data for stream", "stream_id", streamID)

	var data map[string]interface{}
	if err := json.Unmarshal(ctx.PostBody(), &data); err != nil {
		h.Logger.Error("Failed to parse JSON data", "error", err)
		ctx.Error("Invalid JSON data", fasthttp.StatusBadRequest)
		return
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		h.Logger.Error("Failed to marshal data to JSON", "error", err)
		ctx.Error("Failed to process data", fasthttp.StatusInternalServerError)
		return
	}

	// Log before sending to Kafka
	h.Logger.Info("Attempting to send message to Kafka", "stream_id", streamID)
	if err := h.Producer.SendMessage(streamID, jsonData); err != nil {
		h.Logger.Error("Failed to send message to Kafka", "error", err, "stream_id", streamID)
		ctx.Error(fmt.Sprintf("Failed to process data: %v", err), fasthttp.StatusInternalServerError)
		return
	}
	h.Logger.Info("Successfully sent message to Kafka", "stream_id", streamID)

	h.Logger.Info("Data sent to stream", "stream_id", streamID)
	ctx.SetStatusCode(fasthttp.StatusAccepted)
	json.NewEncoder(ctx).Encode(map[string]string{"status": "accepted"})
}

func (h *Handlers) StreamResults(ctx *fasthttp.RequestCtx) {
	// Implementation remains the same, but adapt it to use fasthttp.RequestCtx instead of http.ResponseWriter and *http.Request
}

func (h *Handlers) streamExists(streamID string) bool {
	h.StreamsMutex.RLock()
	defer h.StreamsMutex.RUnlock()
	return h.ActiveStreams[streamID]
}
