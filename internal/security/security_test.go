package security

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Redactor Tests

func TestRedactor_RedactOpenAIAPIKey(t *testing.T) {
	r := NewRedactor()

	content := "API key is sk-1234567890abcdefghijklmnopqrstuvwxyz"
	patterns := []string{`sk-[a-zA-Z0-9]{32,}`}

	result := r.Redact(content, patterns)
	assert.NotContains(t, result, "sk-1234567890")
	assert.Contains(t, result, "[REDACTED]")
}

func TestRedactor_RedactGenericAPIKey(t *testing.T) {
	r := NewRedactor()

	content := "export API_KEY=secret123 and AUTH_TOKEN=abc456"
	patterns := []string{`API_KEY=[^\s]+`, `AUTH_TOKEN=[^\s]+`}

	result := r.Redact(content, patterns)
	assert.NotContains(t, result, "secret123")
	assert.NotContains(t, result, "abc456")
	assert.Contains(t, result, "[REDACTED]")
}

func TestRedactor_RedactPasswordCaseInsensitive(t *testing.T) {
	r := NewRedactor()

	content := "password=secret123 and PASSWORD=secret456"
	patterns := []string{`(?i)password=[^\s]+`}

	result := r.Redact(content, patterns)
	assert.NotContains(t, result, "secret123")
	assert.NotContains(t, result, "secret456")
}

func TestRedactor_NoPatterns(t *testing.T) {
	r := NewRedactor()

	content := "nothing to redact here"
	result := r.Redact(content, []string{})
	assert.Equal(t, content, result)
}

func TestRedactor_InvalidPattern(t *testing.T) {
	r := NewRedactor()

	content := "test content"
	// Invalid regex pattern
	result := r.Redact(content, []string{`[invalid(`})
	// Should return original content on error
	assert.Equal(t, content, result)
}

// RateLimiter Tests

func TestRateLimiter_AllowsBurst(t *testing.T) {
	rl := NewRateLimiter(10, time.Second) // 10 requests per second

	allowed := 0
	for i := 0; i < 10; i++ {
		if rl.Allow() {
			allowed++
		}
	}

	assert.Equal(t, 10, allowed, "should allow burst of 10")
}

func TestRateLimiter_BlocksAfterBurst(t *testing.T) {
	rl := NewRateLimiter(5, time.Second)

	// Consume burst
	for i := 0; i < 5; i++ {
		rl.Allow()
	}

	// Next should be blocked
	assert.False(t, rl.Allow(), "should block after burst consumed")
}

func TestRateLimiter_RefillsOverTime(t *testing.T) {
	rl := NewRateLimiter(10, 100*time.Millisecond)

	// Consume some tokens
	for i := 0; i < 5; i++ {
		rl.Allow()
	}

	// Wait for refill
	time.Sleep(150 * time.Millisecond)

	// Should allow again
	assert.True(t, rl.Allow(), "should refill after time window")
}

func TestRateLimiter_ZeroRate(t *testing.T) {
	rl := NewRateLimiter(0, time.Second)

	// With zero rate, should always block
	assert.False(t, rl.Allow())
}

// SizeLimiter Tests

func TestSizeLimiter_PassesSmallPayload(t *testing.T) {
	sl := NewSizeLimiter()

	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result:  json.RawMessage(`{"small":"payload"}`),
	}

	result := sl.Limit(response, 50) // 50KB limit
	assert.Equal(t, response.Result, result.Result)
}

func TestSizeLimiter_TruncatesLargePayload(t *testing.T) {
	sl := NewSizeLimiter()

	// Create large payload (over 1KB)
	largeData := strings.Repeat("x", 2000)
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result:  json.RawMessage(`{"data":"` + largeData + `"}`),
	}

	result := sl.Limit(response, 1) // 1KB limit

	// Result should be truncated
	assert.Less(t, len(result.Result), len(response.Result))
	assert.Contains(t, string(result.Result), "TRUNCATED")
}

func TestSizeLimiter_HandlesNilResult(t *testing.T) {
	sl := NewSizeLimiter()

	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result:  nil,
	}

	result := sl.Limit(response, 50)
	assert.Equal(t, response, result)
}

func TestSizeLimiter_HandlesError(t *testing.T) {
	sl := NewSizeLimiter()

	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Error:   map[string]interface{}{"code": -32600, "message": "Invalid Request"},
	}

	result := sl.Limit(response, 50)
	assert.Equal(t, response, result)
}

func TestSizeLimiter_ExactLimit(t *testing.T) {
	sl := NewSizeLimiter()

	// Create payload exactly at limit
	data := strings.Repeat("x", 1024) // 1KB
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result:  json.RawMessage(`{"data":"` + data + `"}`),
	}

	result := sl.Limit(response, 2) // 2KB limit (should pass)
	assert.Equal(t, response.Result, result.Result)
}
