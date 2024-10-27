package api

import (
	"encoding/json"
	"sync"
	"time"
	"golang.org/x/time/rate"
	"github.com/google/uuid"

	"github.com/valyala/fasthttp"
	"github.com/gomodule/redigo/redis"
	"github.com/rithindattag/realtime-streaming-api/internal/kafka"
	"github.com/rithindattag/realtime-streaming-api/internal/websocket"
	"github.com/rithindattag/realtime-streaming-api/pkg/logger"
	"github.com/rithindattag/realtime-streaming-api/internal/metrics"
)

var (
	globalLimiter = rate.NewLimiter(rate.Limit(5000), 10000) // 5000 requests per second with burst of 10000
	redisPool     *redis.Pool
)

func init() {
	redisPool = &redis.Pool{
		MaxIdle:     100,
		MaxActive:   12000,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", "localhost:6379")
		},
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

func NewHandlers(producer *kafka.Producer, consumer *kafka.Consumer, hub *websocket.Hub, logger *logger.Logger) *Handlers {
	h := &Handlers{
		Producer:      producer,
		Consumer:      consumer,
		Hub:           hub,
		Logger:        logger,
			ActiveStreams: make(map[string]bool),
		StreamsMutex:  sync.RWMutex{},
	}
	go h.messageWorker()
	return h
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
	if !globalLimiter.Allow() {
		ctx.Error("Too many requests", fasthttp.StatusTooManyRequests)
		return
	}

	streamID := ctx.UserValue("stream_id").(string)

	if !h.streamExists(streamID) {
		h.Logger.Error("Stream not found", "stream_id", streamID)
		ctx.Error("Stream not found", fasthttp.StatusNotFound)
		return
	}

	h.Logger.Info("Received data for stream", "stream_id", streamID)

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		metrics.ProcessingTime.WithLabelValues(streamID).Observe(duration)
	}()

	metrics.MessagesReceived.WithLabelValues(streamID).Inc()

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

	conn := redisPool.Get()
	defer conn.Close()

	_, err = conn.Do("LPUSH", "message_queue", string(jsonData))
	if err != nil {
		h.Logger.Error("Failed to push message to Redis", "error", err)
		ctx.Error("Failed to process data", fasthttp.StatusInternalServerError)
		return
	}

	metrics.MessagesSent.WithLabelValues(streamID).Inc()

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

func (h *Handlers) messageWorker() {
	for {
		conn := redisPool.Get()
		reply, err := redis.Bytes(conn.Do("RPOP", "message_queue"))
		conn.Close()

		if err == redis.ErrNil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if err != nil {
			h.Logger.Error("Failed to pop message from Redis", "error", err)
			continue
		}

		h.Producer.SendMessage("topic", reply)
	}
}
