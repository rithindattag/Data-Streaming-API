package processor

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/rithindattag/realtime-streaming-api/pkg/logger"
)

// Processor handles message processing
type Processor struct {
	logger *logger.Logger
}

// NewProcessor creates a new Processor instance
func NewProcessor(logger *logger.Logger) *Processor {
	return &Processor{
		logger: logger,
	}
}

// ProcessMessage processes a single message
func (p *Processor) ProcessMessage(message []byte) []byte {
	// TODO: Implement message processing logic
	p.logger.Info("Processing message", "message", string(message))
	return message // For now, just return the original message
}

func (p *Processor) ProcessData(data []byte) ([]byte, error) {
	var inputData map[string]interface{}
	err := json.Unmarshal(data, &inputData)
	if err != nil {
		p.logger.Error("Failed to unmarshal input data", "error", err)
		return nil, err
	}

	// Simple processing: Add a timestamp and capitalize string values
	processedData := make(map[string]interface{})
	for key, value := range inputData {
		if strValue, ok := value.(string); ok {
			processedData[key] = strings.ToUpper(strValue)
		} else {
			processedData[key] = value
		}
	}
	processedData["processed_at"] = time.Now().UTC().Format(time.RFC3339)

	result, err := json.Marshal(processedData)
	if err != nil {
		p.logger.Error("Failed to marshal processed data", "error", err)
		return nil, err
	}

	return result, nil
}
