// Package kafka provides Kafka producer and consumer implementations
package kafka

import (
	"encoding/json"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/rithindattag/realtime-streaming-api/internal/websocket"
	"github.com/rithindattag/realtime-streaming-api/pkg/logger"
)

// Consumer represents a Kafka consumer.
type Consumer struct {
	consumer *kafka.Consumer
	topic    string
	logger   *logger.Logger
}

// NewConsumer creates and returns a new Kafka consumer.
// It initializes the consumer with the provided bootstrap servers and topic.
func NewConsumer(bootstrapServers, topic string, logger *logger.Logger) (*Consumer, error) {
	logger.Info("Creating new Kafka consumer", "bootstrapServers", bootstrapServers, "topic", topic)

	// Initialize Kafka consumer with configuration
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  bootstrapServers,
		"group.id":           "my-group",
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": false,
	})
	if err != nil {
		logger.Error("Failed to create Kafka consumer", "error", err)
		return nil, err
	}

	// Subscribe to the specified topic
	logger.Info("Subscribing to topic", "topic", topic)
	err = c.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		logger.Error("Failed to subscribe to topic", "error", err)
		return nil, err
	}

	return &Consumer{
		consumer: c,
		topic:    topic,
		logger:   logger,
	}, nil
}

// ConsumeMessages starts consuming messages from the subscribed Kafka topic.
// It continuously reads messages and broadcasts them to connected WebSocket clients.
func (c *Consumer) ConsumeMessages(hub *websocket.Hub) {
	c.logger.Info("Starting to consume messages", "topic", c.topic)
	for {
		// Read message from Kafka
		msg, err := c.consumer.ReadMessage(-1)
		if err != nil {
			c.logger.Error("Error reading message", "error", err)
			continue
		}

		c.logger.Info("Received message", "topic", *msg.TopicPartition.Topic, "partition", msg.TopicPartition.Partition, "offset", msg.TopicPartition.Offset)

		// Unmarshal the message value
		var data map[string]interface{}
		if err := json.Unmarshal(msg.Value, &data); err != nil {
			c.logger.Error("Error unmarshalling message", "error", err)
			continue
		}

		// Broadcast the message to WebSocket clients
		hub.BroadcastMessage(websocket.Message{
			StreamID: *msg.TopicPartition.Topic,
			Data:     msg.Value,
		})
	}
}

// Close closes the Kafka consumer connection.
func (c *Consumer) Close() error {
	return c.consumer.Close()
}
