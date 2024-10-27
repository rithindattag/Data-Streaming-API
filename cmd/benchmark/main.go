package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/joho/godotenv"
	"sort"
)

const (
	baseURL         = "http://localhost:8000"
	concurrentUsers = 1000
	messagesPerUser = 10
)

var (
	totalRequests int64
	totalErrors   int64
	latencies     []float64
	latenciesMu   sync.Mutex
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
			runUserWorkload(userID, apiKey)
		}(i)
	}

	wg.Wait()

	elapsed := time.Since(start)
	successfulRequests := totalRequests - totalErrors

	fmt.Printf("Benchmark completed in %s\n", elapsed)
	fmt.Printf("Total requests: %d\n", totalRequests)
	fmt.Printf("Successful requests: %d\n", successfulRequests)
	fmt.Printf("Failed requests: %d\n", totalErrors)
	fmt.Printf("Requests per second: %.2f\n", float64(successfulRequests)/elapsed.Seconds())

	// Calculate latency percentiles
	sort.Float64s(latencies)
	len := len(latencies)
	if len > 0 {
		fmt.Printf("Latency (ms):\n")
		fmt.Printf("  50%%: %.2f\n", latencies[len*50/100])
		fmt.Printf("  90%%: %.2f\n", latencies[len*90/100])
		fmt.Printf("  95%%: %.2f\n", latencies[len*95/100])
		fmt.Printf("  99%%: %.2f\n", latencies[len*99/100])
	}
}

func runUserWorkload(userID int, apiKey string) {
	streamID := createStream(apiKey)
	if streamID == "" {
		fmt.Printf("Failed to create stream for user %d\n", userID)
		return
	}

	for i := 0; i < messagesPerUser; i++ {
		sendData(streamID, fmt.Sprintf("Message %d from user %d", i, userID), apiKey)
	}
}

func createStream(apiKey string) string {
	req, _ := http.NewRequest("POST", baseURL+"/stream/start", nil)
	req.Header.Set("X-API-Key", apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error creating stream: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	return result["stream_id"]
}

func sendData(streamID, message, apiKey string) {
	start := time.Now()
	data := map[string]string{"data": message}
	jsonData, _ := json.Marshal(data)

	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/stream/%s/send", baseURL, streamID), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error sending data: %v\n", err)
		return
	}
	defer resp.Body.Close()

	atomic.AddInt64(&totalRequests, 1)
	if resp.StatusCode != http.StatusAccepted {
		atomic.AddInt64(&totalErrors, 1)
	}

	latency := float64(time.Since(start).Milliseconds())
	latenciesMu.Lock()
	latencies = append(latencies, latency)
	latenciesMu.Unlock()
}
