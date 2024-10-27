// Package main provides a benchmark tool for the streaming API
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"sort"

	"github.com/joho/godotenv"
	"github.com/valyala/fasthttp"
)

// Constants for benchmark configuration
const (
	baseURL         = "http://localhost:8000"
	concurrentUsers = 1000
	messagesPerUser = 10
	maxRetries      = 3
)

// Global variables for benchmark metrics
var (
	totalRequests int64
	totalErrors   int64
	latencies     []float64
	latenciesMu   sync.Mutex
	client        *fasthttp.Client
)

// init initializes the HTTP client
func init() {
	client = &fasthttp.Client{
		MaxConnsPerHost: 10000,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
	}
}

// main is the entry point for the benchmark tool
func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

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

// runUserWorkload simulates a single user's workload
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

// createStream creates a new stream and returns the stream ID
func createStream(apiKey string) string {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(baseURL + "/stream/start")
	req.Header.SetMethod("POST")
	req.Header.Set("X-API-Key", apiKey)

	err := client.Do(req, resp)
	if err != nil {
		fmt.Printf("Error creating stream: %v\n", err)
		return ""
	}

	var result map[string]string
	json.Unmarshal(resp.Body(), &result)
	return result["stream_id"]
}

// sendData sends data to a specific stream
func sendData(streamID, message, apiKey string) {
	data := map[string]string{"data": message}
	jsonData, _ := json.Marshal(data)

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(fmt.Sprintf("%s/stream/%s/send", baseURL, streamID))
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.Header.Set("X-API-Key", apiKey)
	req.SetBody(jsonData)

	start := time.Now() // Define start here

	var err error
	var respBody []byte
	for i := 0; i < maxRetries; i++ {
		err = client.Do(req, resp)
		respBody = resp.Body()
		if err == nil && resp.StatusCode() == fasthttp.StatusAccepted {
			break
		}
		time.Sleep(time.Duration(1<<uint(i)) * 100 * time.Millisecond)
	}

	atomic.AddInt64(&totalRequests, 1)
	if err != nil || resp.StatusCode() != fasthttp.StatusAccepted {
		atomic.AddInt64(&totalErrors, 1)
		fmt.Printf("Error sending data: %v, Status: %d, Body: %s\n", err, resp.StatusCode(), string(respBody))
	}

	latency := float64(time.Since(start).Milliseconds())
	latenciesMu.Lock()
	latencies = append(latencies, latency)
	latenciesMu.Unlock()
}
