package processor

import (
	"encoding/json"
	"testing"

	"github.com/rithindattag/realtime-streaming-api/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestProcessData(t *testing.T) {
	p := NewProcessor(logger.NewLogger())

	testCases := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:     "Simple string capitalization",
			input:    map[string]interface{}{"data": "test data"},
			expected: map[string]interface{}{"data": "TEST DATA"},
		},
		{
			name:     "Mixed data types",
			input:    map[string]interface{}{"string": "lowercase", "number": 42},
			expected: map[string]interface{}{"string": "LOWERCASE", "number": 42},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inputJSON, _ := json.Marshal(tc.input)
			result, err := p.ProcessData(inputJSON)
			assert.NoError(t, err)

			var processedData map[string]interface{}
			err = json.Unmarshal(result, &processedData)
			assert.NoError(t, err)

			for key, expectedValue := range tc.expected {
				assert.Equal(t, expectedValue, processedData[key])
			}

			assert.Contains(t, processedData, "processed_at")
		})
	}
}
