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

	MessagesReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "messages_received_total",
		Help: "The total number of messages received",
	}, []string{"stream_id"})

	MessagesSent = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "messages_sent_total",
		Help: "The total number of messages sent to Kafka",
	}, []string{"stream_id"})

	ProcessingTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "message_processing_time_seconds",
		Help:    "Time taken to process messages",
		Buckets: prometheus.DefBuckets,
	}, []string{"stream_id"})
)
