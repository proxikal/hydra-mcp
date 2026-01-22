package security

import "encoding/json"

// Redactor redacts sensitive information from strings
type Redactor interface {
	Redact(content string, patterns []string) string
}

// RateLimiter controls the rate of operations
type RateLimiter interface {
	Allow() bool
}

// JSONRPCResponse represents a JSON-RPC response structure
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   interface{}     `json:"error,omitempty"`
}

// SizeLimiter limits payload sizes
type SizeLimiter interface {
	Limit(response JSONRPCResponse, maxKB int) JSONRPCResponse
}
