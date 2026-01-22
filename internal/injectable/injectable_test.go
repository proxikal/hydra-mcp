package injectable

import (
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/logger"
	"github.com/proxikal/hydra/internal/mocks"
	"github.com/proxikal/hydra/internal/supervisor"
	"github.com/stretchr/testify/assert"
)

type stubRedactor struct{}

func (stubRedactor) Redact(content string, _ []string) string {
	return "[REDACTED]"
}

type stubLogger struct{}

func (stubLogger) Debug(string, ...map[string]interface{}) {}
func (stubLogger) Info(string, ...map[string]interface{})  {}
func (stubLogger) Warn(string, ...map[string]interface{})  {}
func (stubLogger) Error(string, error, ...map[string]interface{}) {
}

func TestDefinitionsIncludeHydraTools(t *testing.T) {
	sup := new(mocks.Supervisor)
	status := func() StatusSnapshot {
		return StatusSnapshot{State: supervisor.StateRunning}
	}

	tools := New(
		Options{
			Enabled: true,
			Tools: []string{
				"hydra_restart",
				"hydra_status",
			},
		},
		sup,
		status,
		stubRedactor{},
		stubLogger{},
		NewLogBuffer(100),
	)

	defs := tools.GetDefinitions()
	assert.Len(t, defs, 2)
	assert.Equal(t, "hydra_restart", defs[0].Name)
	assert.Equal(t, "hydra_status", defs[1].Name)
}

func TestHandleRestartTriggersSupervisorRestart(t *testing.T) {
	sup := new(mocks.Supervisor)
	sup.On("Restart").Return(nil).Once()
	sup.On("State").Return(supervisor.StateRunning)

	status := func() StatusSnapshot {
		return StatusSnapshot{State: supervisor.StateRunning}
	}

	tools := New(
		Options{
			Enabled: true,
			Tools:   []string{"hydra_restart"},
		},
		sup,
		status,
		stubRedactor{},
		stubLogger{},
		NewLogBuffer(100),
	)

	resp, err := tools.Handle("hydra_restart", map[string]interface{}{
		"reason": "test restart",
	})

	assert.NoError(t, err)
	out, ok := resp.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, true, out["success"])
	sup.AssertExpectations(t)
}

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

func TestHandleForceRestartRequiresConfirmation(t *testing.T) {
	sup := new(mocks.Supervisor)
	sup.On("ResetRestartCounter").Return().Once()
	sup.On("Restart").Return(nil).Once()
	sup.On("State").Return(supervisor.StateRunning)

	status := func() StatusSnapshot {
		return StatusSnapshot{State: supervisor.StateRunning}
	}

	tools := New(
		Options{
			Enabled: true,
			Tools: []string{
				"hydra_force_restart",
			},
		},
		sup,
		status,
		stubRedactor{},
		stubLogger{},
		NewLogBuffer(100),
	)

	_, err := tools.Handle("hydra_force_restart", map[string]interface{}{
		"confirm": false,
	})
	assert.Error(t, err)

	resp, err := tools.Handle("hydra_force_restart", map[string]interface{}{
		"confirm": true,
	})

	assert.NoError(t, err)
	out := resp.(map[string]interface{})
	assert.Equal(t, true, out["success"])
	sup.AssertExpectations(t)
}

func TestHandleUnknownToolReturnsError(t *testing.T) {
	sup := new(mocks.Supervisor)
	status := func() StatusSnapshot { return StatusSnapshot{} }

	tools := New(
		Options{Enabled: true},
		sup,
		status,
		stubRedactor{},
		logger.New("info"),
		NewLogBuffer(10),
	)

	_, err := tools.Handle("unknown_tool", nil)
	assert.Error(t, err)
}
