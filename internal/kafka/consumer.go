package kafka

import (
	"context"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/rithindattag/realtime-streaming-api/internal/processor"
	"github.com/rithindattag/realtime-streaming-api/internal/websocket"
	"github.com/rithindattag/realtime-streaming-api/pkg/logger"
)

type Consumer struct {
	consumer  *kafka.Consumer
	logger    *logger.Logger
	processor *processor.Processor
}

func NewConsumer(bootstrapServers, groupID string, logger *logger.Logger) (*Consumer, error) {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  bootstrapServers,
		"group.id":           groupID,
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": false,
	})
	if err != nil {
		return nil, err
	}

	return &Consumer{
		consumer:  c,
		logger:    logger,
		processor: processor.NewProcessor(logger),
	}, nil
}

func (c *Consumer) ConsumeMessages(hub *websocket.Hub) {
	for {
		msg, err := c.consumer.ReadMessage(-1)
		if err == nil {
			c.logger.Info("Received message", "topic", *msg.TopicPartition.Topic, "partition", msg.TopicPartition.Partition, "offset", msg.TopicPartition.Offset)

			processedData, err := c.processor.ProcessData(msg.Value)
			if err != nil {
				c.logger.Error("Failed to process message", "error", err)
				continue
			}

			hub.Broadcast <- websocket.Message{
				StreamID: *msg.TopicPartition.Topic,
				Data:     processedData,
			}

			_, err = c.consumer.CommitMessage(msg)
			if err != nil {
				c.logger.Error("Failed to commit message", "error", err)
			}
		} else {
			c.logger.Error("Error consuming message", "error", err)
		}
	}
}

func (c *Consumer) Close() error {
	return c.consumer.Close()
}
