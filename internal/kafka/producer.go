package kafka

import (
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/rithindattag/realtime-streaming-api/pkg/logger"
)

// Producer represents a Kafka producer
type Producer struct {
	producer *kafka.Producer
	logger   *logger.Logger
}

// NewProducer creates and returns a new Kafka producer
func NewProducer(bootstrapServers string) (*Producer, error) {
	logger := logger.NewLogger()
	logger.Info("Creating new Kafka producer", "bootstrapServers", bootstrapServers)

	// Initialize Kafka producer with configuration
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": bootstrapServers,
	})
	if err != nil {
		logger.Error("Failed to create Kafka producer", "error", err)
		return nil, err
	}

	logger.Info("Kafka producer created successfully")
	return &Producer{
		producer: p,
		logger:   logger,
	}, nil
}

// SendMessage sends a message to a specified Kafka topic
func (p *Producer) SendMessage(topic string, message []byte) error {
	// Produce message to Kafka topic
	err := p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          message,
	}, nil)

	if err != nil {
		p.logger.Error("Failed to produce message", "error", err)
		return err
	}

	return nil
}

// Close closes the Kafka producer
func (p *Producer) Close() {
	p.producer.Close()
}
