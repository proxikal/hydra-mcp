package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCrashLoopSetsFailedAndLastError(t *testing.T) {
	p := &proxy{}
	p.crashLoop(assert.AnError)

	assert.Equal(t, StateFailed, p.state)
	assert.Equal(t, assert.AnError.Error(), p.lastError)
	assert.Equal(t, "crash_loop", p.lastRestartReason)
}
