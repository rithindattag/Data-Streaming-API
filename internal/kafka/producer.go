package kafka

import (
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/rithindattag/realtime-streaming-api/pkg/logger"
	"fmt"
	"time"
)

type Producer struct {
	producer *kafka.Producer
	logger   *logger.Logger
}

func NewProducer(bootstrapServers string) (*Producer, error) {
	logger := logger.NewLogger()
	logger.Info("Creating new Kafka producer", "bootstrapServers", bootstrapServers)
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": bootstrapServers,
		"message.timeout.ms": 30000,
		"debug": "all",
		"api.version.request": true,
		"broker.version.fallback": "0.10.0.0",
		"api.version.fallback.ms": 0,
		"socket.keepalive.enable": true,
		"metadata.max.age.ms": 30000,
	})
	if err != nil {
		logger.Error("Failed to create Kafka producer", "error", err, "bootstrapServers", bootstrapServers)
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}
	logger.Info("Kafka producer created successfully")
	return &Producer{
		producer: p,
		logger:   logger,
	}, nil
}

func (p *Producer) SendMessage(topic string, message []byte) error {
	p.logger.Info("Preparing to send message", "topic", topic, "message_length", len(message))
	deliveryChan := make(chan kafka.Event)
	err := p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          message,
	}, deliveryChan)

	if err != nil {
		p.logger.Error("Failed to produce message", "error", err)
		return err
	}

	p.logger.Info("Message produced, waiting for delivery report")

	select {
	case e := <-deliveryChan:
		m := e.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			p.logger.Error("Message delivery failed", "error", m.TopicPartition.Error, "topic", topic, "partition", m.TopicPartition.Partition)
			return m.TopicPartition.Error
		}
		p.logger.Info("Message sent to Kafka", "topic", topic, "partition", m.TopicPartition.Partition, "offset", m.TopicPartition.Offset)
		return nil
	case <-time.After(30 * time.Second):
		p.logger.Error("Timeout waiting for Kafka response", "topic", topic)
		return fmt.Errorf("timeout waiting for Kafka response for topic %s", topic)
	}
}

func (p *Producer) Close() {
	p.producer.Close()
}
