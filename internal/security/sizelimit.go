package security

import "encoding/json"

type sizeLimiter struct{}

// NewSizeLimiter creates a new payload size limiter
func NewSizeLimiter() SizeLimiter {
	return &sizeLimiter{}
}

func (sl *sizeLimiter) Limit(response JSONRPCResponse, maxKB int) JSONRPCResponse {
	// Skip if no result
	if response.Result == nil {
		return response
	}

	maxBytes := maxKB * 1024
	currentSize := len(response.Result)

	if currentSize <= maxBytes {
		return response
	}

	// Truncate and add indicator
	truncatedMsg := map[string]interface{}{
		"_truncated":     true,
		"_original_size": currentSize,
		"_message":       "[TRUNCATED: Response exceeded size limit]",
	}

	truncatedJSON, _ := json.Marshal(truncatedMsg)
	response.Result = truncatedJSON

	return response
}
