package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	StreamsCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "streams_created_total",
		Help: "The total number of streams created",
	})

	MessagesProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "messages_processed_total",
		Help: "The total number of messages processed",
	})

	ActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "active_connections",
		Help: "The number of active WebSocket connections",
	})
)
