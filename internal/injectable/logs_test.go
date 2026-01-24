package injectable

import (
	"testing"

	"github.com/proxikal/hydra/internal/mocks"
	"github.com/proxikal/hydra/internal/supervisor"
	"github.com/stretchr/testify/assert"
)

func TestHandleLogsRedactsAndLimits(t *testing.T) {
	sup := new(mocks.Supervisor)
	status := func() StatusSnapshot {
		return StatusSnapshot{State: supervisor.StateRunning}
	}

	buffer := NewLogBuffer(3)
	buffer.Add("first secret")
	buffer.Add("second line")
	buffer.Add("third secret")

	tools := New(
		Options{
			Enabled:        true,
			Tools:          []string{"hydra_logs"},
			RedactPatterns: []string{"secret"},
		},
		sup,
		status,
		stubRedactor{},
		stubLogger{},
		buffer,
	)

	resp, err := tools.Handle("hydra_logs", map[string]interface{}{
		"lines": float64(2),
	})

	assert.NoError(t, err)
	out := resp.(map[string]interface{})
	logs := out["logs"].([]string)
	assert.Len(t, logs, 2)
	assert.Equal(t, "[REDACTED]", logs[0])
}

func TestNewLogBufferDefaultsToFiveHundredWhenMaxIsZero(t *testing.T) {
	buffer := NewLogBuffer(0)
	assert.NotNil(t, buffer)

	for i := 0; i < 600; i++ {
		buffer.Add("line")
	}

	logs := buffer.GetRecent(600)
	assert.Len(t, logs, 500)
}

func TestNewLogBufferDefaultsToFiveHundredWhenMaxIsNegative(t *testing.T) {
	buffer := NewLogBuffer(-10)
	assert.NotNil(t, buffer)

	for i := 0; i < 600; i++ {
		buffer.Add("line")
	}

	logs := buffer.GetRecent(600)
	assert.Len(t, logs, 500)
}

func TestHandleLogsFailsWhenBufferNil(t *testing.T) {
	sup := new(mocks.Supervisor)
	status := func() StatusSnapshot {
		return StatusSnapshot{State: supervisor.StateRunning}
	}

	tools := &toolset{
		opts: Options{
			Enabled: true,
			Tools:   []string{"hydra_logs"},
		},
		supervisor: sup,
		status:     status,
		redactor:   stubRedactor{},
		logger:     stubLogger{},
		buffer:     nil,
	}

	_, err := tools.Handle("hydra_logs", map[string]interface{}{
		"lines": float64(10),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "log buffer not configured")
}

func TestGetRecentReturnsEmptyWhenNIsZero(t *testing.T) {
	buffer := NewLogBuffer(100)
	buffer.Add("line1")
	buffer.Add("line2")

	logs := buffer.GetRecent(0)
	assert.Empty(t, logs)
}

func TestGetRecentReturnsEmptyWhenNIsNegative(t *testing.T) {
	buffer := NewLogBuffer(100)
	buffer.Add("line1")
	buffer.Add("line2")

	logs := buffer.GetRecent(-5)
	assert.Empty(t, logs)
}
