package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

const (
	baseURL         = "http://localhost:8080"
	concurrentUsers = 1000
	messagesPerUser = 10
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Get API key from environment variable
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Fatal("API_KEY environment variable is not set")
	}

	start := time.Now()

	var wg sync.WaitGroup
	wg.Add(concurrentUsers)

	for i := 0; i < concurrentUsers; i++ {
		go func(userID int) {
			defer wg.Done()
			runUserWorkload(userID)
		}(i)
	}

	wg.Wait()

	elapsed := time.Since(start)
	fmt.Printf("Benchmark completed in %s\n", elapsed)
	fmt.Printf("Total messages sent: %d\n", concurrentUsers*messagesPerUser)
	fmt.Printf("Messages per second: %.2f\n", float64(concurrentUsers*messagesPerUser)/elapsed.Seconds())
}

func runUserWorkload(userID int) {
	streamID := createStream()
	if streamID == "" {
		fmt.Printf("Failed to create stream for user %d\n", userID)
		return
	}

	for i := 0; i < messagesPerUser; i++ {
		sendData(streamID, fmt.Sprintf("Message %d from user %d", i, userID))
	}
}

func createStream() string {
	resp, err := http.Post(baseURL+"/stream/start", "application/json", nil)
	if err != nil {
		fmt.Printf("Error creating stream: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	return result["stream_id"]
}

func sendData(streamID, message string) {
	data := map[string]string{"data": message}
	jsonData, _ := json.Marshal(data)

	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/stream/%s/send", baseURL, streamID), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending data: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		fmt.Printf("Unexpected status code: %d\n", resp.StatusCode)
	}
}
