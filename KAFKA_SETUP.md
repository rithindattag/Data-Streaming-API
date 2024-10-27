# Real-Time Data Streaming API Setup and Usage Guide

This document provides instructions for setting up Redpanda (a Kafka-compatible event streaming platform) and running our Real-Time Data Streaming API service locally.

## Prerequisites

- Go 1.16 or later
- Docker
- Docker Compose

## Kafka (Redpanda) Setup

1. Create a `docker-compose.yml` file in your project root with the following content:

## Running the API Service Locally

1. Clone the repository:

2. Install dependencies:

3. Set up environment variables (create a `.env` file in the project root):

4. Run the API service:

## API Endpoints

- `POST /stream`: Send data to the stream
- `GET /stream`: Establish a WebSocket connection to receive real-time updates

## Testing

Run unit tests:

Run integration tests:

## Performance Benchmarking

To run performance benchmarks:

## Stopping the Service

To stop the API service, press `Ctrl+C` in the terminal where it's running.

To stop the Redpanda cluster and remove the containers:

Note: Be cautious when using the `-v` flag, as it will delete all data stored in the Redpanda volumes.
