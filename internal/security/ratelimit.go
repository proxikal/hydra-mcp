package security

import (
	"sync"
	"time"
)

type rateLimiter struct {
	mu         sync.Mutex
	rate       int
	interval   time.Duration
	tokens     int
	lastRefill time.Time
}

// NewRateLimiter creates a token bucket rate limiter
func NewRateLimiter(rate int, interval time.Duration) RateLimiter {
	return &rateLimiter{
		rate:       rate,
		interval:   interval,
		tokens:     rate,
		lastRefill: time.Now(),
	}
}

func (rl *rateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)

	if elapsed >= rl.interval {
		rl.tokens = rl.rate
		rl.lastRefill = now
	}

	// Check if we have tokens
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}
