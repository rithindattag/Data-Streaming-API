// Package metrics provides Prometheus metrics for the streaming API
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// StreamsCreated tracks the number of streams created
	StreamsCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "streams_created_total",
		Help: "The total number of streams created",
	})

	// MessagesReceived tracks the number of messages received per stream
	MessagesReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "messages_received_total",
		Help: "The total number of messages received per stream",
	}, []string{"stream_id"})

	// MessagesSent tracks the number of messages sent per stream
	MessagesSent = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "messages_sent_total",
		Help: "The total number of messages sent per stream",
	}, []string{"stream_id"})

	// ProcessingTime tracks the processing time for messages
	ProcessingTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "message_processing_time_seconds",
		Help:    "The time taken to process a message",
		Buckets: prometheus.DefBuckets,
	}, []string{"stream_id"})
)
