package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
)

// Request represents a JSON-RPC request
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC response
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result"`
}

var requestCount int

func handleRequest(req Request) {
	requestCount++

	// Chaos behaviors
	chaosRoll := rand.Float64()

	// 1% chance: Immediate crash
	if chaosRoll < 0.01 {
		fmt.Fprintf(os.Stderr, "[CHAOS] Request %d: CRASH\n", requestCount)
		os.Exit(1)
	}

	// 3% chance: Hang for 30 seconds
	if chaosRoll < 0.04 {
		fmt.Fprintf(os.Stderr, "[CHAOS] Request %d: HANG (30s)\n", requestCount)
		time.Sleep(30 * time.Second)
	}

	// 5% chance: Slow response (2-5 seconds)
	if chaosRoll < 0.09 {
		delay := time.Duration(2000+rand.Intn(3000)) * time.Millisecond
		fmt.Fprintf(os.Stderr, "[CHAOS] Request %d: SLOW (%.1fs)\n", requestCount, delay.Seconds())
		time.Sleep(delay)
	}

	// Normal response
	sendResponse(req)
}

func sendResponse(req Request) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"status":        "ok",
			"request_count": requestCount,
			"server":        "go-chaos",
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to marshal response: %v\n", err)
		return
	}

	fmt.Println(string(data))
}

func main() {
	// Seed random
	rand.Seed(time.Now().UnixNano())

	// Simulate slow initialization (2 seconds)
	fmt.Fprintln(os.Stderr, "[CHAOS] Initializing (2s)...")
	time.Sleep(2 * time.Second)
	fmt.Fprintln(os.Stderr, "[CHAOS] Ready")

	// Process requests
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Invalid JSON: %v\n", err)
			continue
		}

		handleRequest(req)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Scanner error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "[CHAOS] Shutting down")
}
