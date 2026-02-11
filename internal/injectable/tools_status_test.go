package injectable

import (
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/mocks"
	"github.com/proxikal/hydra/internal/supervisor"
	"github.com/stretchr/testify/assert"
)

func TestHandleStatusUsesSnapshot(t *testing.T) {
	sup := new(mocks.Supervisor)

	status := func() StatusSnapshot {
		return StatusSnapshot{
			State:             supervisor.StateRunning,
			PID:               123,
			Uptime:            time.Second,
			RestartsInWindow:  2,
			QueueSize:         1,
			LastRestartReason: "file_change",
			LastError:         "none",
			CanRecover:        false,
		}
	}

	tools := New(
		Options{
			Enabled: true,
			Tools:   []string{"hydra_status"},
			Server:  "demo",
			Behavior: Behavior{
				MaxRestarts:          10,
				RestartWindowSeconds: 60,
			},
		},
		sup,
		status,
		nil, // metrics collector
		stubRedactor{},
		stubLogger{},
		NewLogBuffer(100),
	)

	resp, err := tools.Handle("hydra_status", map[string]interface{}{})
	assert.NoError(t, err)

	out := resp.(map[string]interface{})
	assert.Equal(t, "running", out["state"])
	assert.Equal(t, 123, out["pid"])
	assert.Equal(t, 2, out["restarts_in_window"])
	assert.Equal(t, 10, out["max_restarts"])
	assert.Equal(t, 60, out["restart_window_seconds"])
	assert.Equal(t, 1, out["queue_size"])
}

func TestHandleStatusWithZeroValues(t *testing.T) {
	sup := new(mocks.Supervisor)

	status := func() StatusSnapshot {
		return StatusSnapshot{
			State: supervisor.StateStopped,
		}
	}

	tools := New(
		Options{
			Enabled: true,
			Tools:   []string{"hydra_status"},
		},
		sup,
		status,
		nil, // metrics collector
		stubRedactor{},
		stubLogger{},
		NewLogBuffer(100),
	)

	resp, err := tools.Handle("hydra_status", map[string]interface{}{})
	assert.NoError(t, err)

	out := resp.(map[string]interface{})
	assert.Equal(t, "stopped", out["state"])
	assert.Equal(t, 0, out["pid"])
}

func TestHandleStatusFailsWhenStatusProviderNil(t *testing.T) {
	sup := new(mocks.Supervisor)

	tools := &toolset{
		opts: Options{
			Enabled: true,
			Tools:   []string{"hydra_status"},
		},
		supervisor: sup,
		status:     nil,
		redactor:   stubRedactor{},
		logger:     stubLogger{},
		buffer:     NewLogBuffer(100),
	}

	_, err := tools.Handle("hydra_status", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status provider not configured")
}
