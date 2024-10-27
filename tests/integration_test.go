package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"context"
	"time"
	"io/ioutil"
	"os"

	"github.com/gorilla/mux"
	gorillaWS "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/rithindattag/realtime-streaming-api/internal/api"
	"github.com/rithindattag/realtime-streaming-api/internal/kafka"
	"github.com/rithindattag/realtime-streaming-api/internal/websocket"
	"github.com/rithindattag/realtime-streaming-api/pkg/logger"
)

func setupTestServer() *httptest.Server {
	producer, _ := kafka.NewProducer("localhost:9092")
	consumer, _ := kafka.NewConsumer("localhost:9092", "test-group")
	hub := websocket.NewHub()
	go hub.Run()

	// Use the constructor here
	handlers := api.NewHandlers(producer, consumer, hub, logger.NewLogger())

	r := mux.NewRouter()
	r.HandleFunc("/stream/start", handlers.StartStream).Methods("POST")
	r.HandleFunc("/stream/{stream_id}/send", handlers.SendData).Methods("POST")
	r.HandleFunc("/stream/{stream_id}/results", handlers.StreamResults).Methods("GET")

	return httptest.NewServer(r)
}

func TestIntegration(t *testing.T) {
	// Get API key from environment variable
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		t.Fatal("API_KEY environment variable is not set")
	}

	// Use apiKey in your tests
	// ...
}

func TestStreamCreation(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	req, _ := http.NewRequest("POST", server.URL+"/stream/start", nil)
	req.Header.Set("X-API-Key", os.Getenv("API_KEY"))
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]string
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Contains(t, result, "stream_id")
}

func TestDataSending(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	producer, err := kafka.NewProducer("localhost:9092")
	if err != nil {
		t.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer producer.Close()

	// Add a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// First, create a stream
	req, _ := http.NewRequestWithContext(ctx, "POST", server.URL+"/stream/start", nil)
	req.Header.Set("X-API-Key", os.Getenv("API_KEY"))
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	var result map[string]string
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	streamID := result["stream_id"]

	// Now, send data to the stream
	data := map[string]string{"data": "test data"}
	jsonData, err := json.Marshal(data)
	assert.NoError(t, err)
	
	// Send the JSON data directly without base64 encoding
	req, _ = http.NewRequestWithContext(ctx, "POST", server.URL+"/stream/"+streamID+"/send", bytes.NewBuffer(jsonData))
	req.Header.Set("X-API-Key", os.Getenv("API_KEY"))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// Add this block for more detailed error information
	if err != nil {
		t.Logf("Error sending data: %v", err)
	}
	if resp != nil && resp.StatusCode != http.StatusAccepted {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Logf("Unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// Add this line to print the response body for debugging
	body, _ := ioutil.ReadAll(resp.Body)
	t.Logf("Response body: %s", string(body))

	// Add this line to indicate the test has completed
	t.Log("TestDataSending completed")
}

func TestResultStreaming(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	// Create a stream
	req, _ := http.NewRequest("POST", server.URL+"/stream/start", nil)
	req.Header.Set("X-API-Key", os.Getenv("API_KEY"))
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	var result map[string]string
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	streamID := result["stream_id"]

	// Set up a channel to receive WebSocket messages
	messageChan := make(chan []byte)

	// Start a goroutine to handle WebSocket connection
	go func() {
		wsURL := fmt.Sprintf("ws%s/stream/%s/results", server.URL[4:], streamID)
		ws, _, err := gorillaWS.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Errorf("Failed to connect to WebSocket: %v", err)
			return
		}
		defer ws.Close()

		_, message, err := ws.ReadMessage()
		if err != nil {
			t.Errorf("Failed to read message from WebSocket: %v", err)
			return
		}
		messageChan <- message
	}()

	// Send data to the stream
	data := map[string]string{"data": "test data"}
	jsonData, err := json.Marshal(data)
	assert.NoError(t, err)
	
	req, _ = http.NewRequest("POST", server.URL+"/stream/"+streamID+"/send", bytes.NewBuffer(jsonData))
	req.Header.Set("X-API-Key", os.Getenv("API_KEY"))
	req.Header.Set("Content-Type", "application/json")
	_, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)

	// Wait for the message with a timeout
	select {
	case message := <-messageChan:
		assert.NotEmpty(t, message)
		t.Logf("Received message: %s", string(message))
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for WebSocket message")
	}
}
