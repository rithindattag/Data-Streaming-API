# Real-Time Data Streaming API

This project provides a Real-Time Data Streaming API service using Redpanda (a Kafka-compatible event streaming platform) to enable high-performance, real-time data streaming.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Kafka (Redpanda) Setup](#kafka-redpanda-setup)
3. [Running the API Service Locally](#running-the-api-service-locally)
4. [API Endpoints](#api-endpoints)
5. [Testing](#testing)
6. [Performance Benchmarking](#performance-benchmarking)
7. [Stopping the Service](#stopping-the-service)
8. [License](#license)

---

## Prerequisites

- [Go](https://golang.org/) version 1.16 or later
- [Docker](https://www.docker.com/)
- [Docker Compose](https://docs.docker.com/compose/)

---

## Kafka (Redpanda) Setup

1. In your project root, create a `docker-compose.yml` file with the following configuration:

   ```yaml
   version: "3.7"
   services:
     redpanda:
       image: vectorized/redpanda:v21.11.15
       container_name: redpanda
       ports:
         - "9092:9092"
         - "9644:9644"
       command:
         - redpanda start
         - --smp 1
         - --overprovisioned
         - --node-id 0
         - --kafka-addr PLAINTEXT://0.0.0.0:29092,OUTSIDE://0.0.0.0:9092
         - --advertise-kafka-addr PLAINTEXT://redpanda:29092,OUTSIDE://localhost:9092
       environment:
         - REDPANDA_BROKERS=redpanda:9093
     console:
       image: vectorized/console:v21.11.15
       container_name: redpanda-console
       ports:
         - "8080:8080"
       environment:
         - KAFKA_BROKERS=redpanda:29092
       depends_on:
         - redpanda
   ```

2. Start the Redpanda cluster and console:

   ```
   docker-compose up -d
   ```

3. Verify that Redpanda is running:

   ```
   docker-compose ps
   ```

4. Create a topic (replace `my-topic` with your desired topic name):

   ```
   docker-compose exec redpanda rpk topic create my-topic
   ```

---

## Running the API Service Locally

1. Clone the repository:

   ```
   git clone https://github.com/your-repo/real-time-data-streaming-api.git
   cd real-time-data-streaming-api
   ```

2. Install dependencies:

   ```
   go mod tidy
   ```

3. Set up environment variables:

   Create a `.env` file in the project root (do not commit this file to version control):

   ```
   KAFKA_BROKERS=localhost:9093
   KAFKA_TOPIC=my-topic
   API_PORT=8000
   # Add any other sensitive configuration here
   ```

   Load these environment variables in your application code.

4. Run the API service:

   ```
   go run cmd/api/main.go
   ```

The API should now be running on `http://localhost:8000`.

---

## API Endpoints

- `POST /stream`: Send data to the stream

  - Request body: JSON object with the data to be streamed
  - Response: 200 OK if successful, error message otherwise

- `GET /stream`: Establish a WebSocket connection to receive real-time updates
  - Connects to the WebSocket endpoint for real-time data streaming

---

## Testing

Run unit tests:

```
go test ./...
```

Run integration tests:

```
go test ./tests -tags=integration
```

---

## Performance Benchmarking

To run performance benchmarks:

```
go run cmd/benchmark/main.go
```

This will generate a report on the system's performance under load, including throughput and latency metrics.

---

## Stopping the Service

To stop the API service, press `Ctrl+C` in the terminal where it's running.

To stop the Redpanda cluster and remove the containers:

```
docker-compose down
```

If you want to completely remove everything, including the volumes, use:

```
docker-compose down -v
```

Note: Be cautious when using the `-v` flag, as it will delete all data stored in the Redpanda volumes.

---

## API Key Setup

This project requires an API key for authentication. To set up your API key:

1. Create a `.env` file in the project root if it doesn't exist.
2. Add the following line to your `.env` file:
   ```
   API_KEY=your_actual_api_key_here
   ```
3. Replace `your_actual_api_key_here` with your actual API key.
4. Ensure that `.env` is listed in your `.gitignore` file to prevent accidentally committing your API key.

## Performance Report

For detailed information about the system's performance under high load, please refer to the [Performance Benchmarking Report](PERFORMANCE_REPORT.md).
