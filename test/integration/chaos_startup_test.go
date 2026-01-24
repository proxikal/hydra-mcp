package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestSlowStartupWithQueuedRequests tests 30s startup with queued requests.
// Verifies that Hydra queues requests during slow server initialization
// and replays them once the server enters RUNNING state.
func TestSlowStartupWithQueuedRequests(t *testing.T) {
	// Note: slow_server.py has 30s startup delay
	tp, cleanup := newTestProxyWithTimeout(t, "slow_server.py", 45*time.Second)
	defer cleanup()

	// Send requests immediately while server is still starting
	// These should be queued
	for i := 1; i <= 5; i++ {
		tp.SendRequest(`{"jsonrpc":"2.0","method":"test","id":` + string(rune('0'+i)) + `}`)
	}

	// Wait for server to finish 30s startup
	time.Sleep(32 * time.Second)

	// Verify we got responses (requests were queued and replayed)
	writes := tp.GetWrites()
	assert.GreaterOrEqual(t, len(writes), 1, "Should have received responses after slow startup")
}
