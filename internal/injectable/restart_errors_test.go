package injectable

import (
	"testing"

	"github.com/proxikal/hydra/internal/mocks"
	"github.com/proxikal/hydra/internal/supervisor"
	"github.com/stretchr/testify/assert"
)

func TestHandleRestartFailsWhenSupervisorNil(t *testing.T) {
	status := func() StatusSnapshot {
		return StatusSnapshot{State: supervisor.StateRunning}
	}

	tools := New(
		Options{
			Enabled: true,
			Tools:   []string{"hydra_restart"},
		},
		nil,
		status,
		stubRedactor{},
		stubLogger{},
		NewLogBuffer(100),
	)

	_, err := tools.Handle("hydra_restart", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "supervisor not configured")
}

func TestHandleRestartFailsWhenInFailedState(t *testing.T) {
	sup := new(mocks.Supervisor)
	sup.On("State").Return(supervisor.StateFailed)

	status := func() StatusSnapshot {
		return StatusSnapshot{State: supervisor.StateFailed}
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

	_, err := tools.Handle("hydra_restart", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "FAILED state")
	sup.AssertExpectations(t)
}

func TestHandleRestartFailsWhenSupervisorReturnsError(t *testing.T) {
	sup := new(mocks.Supervisor)
	sup.On("State").Return(supervisor.StateRunning)
	sup.On("Restart").Return(assert.AnError).Once()

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

	_, err := tools.Handle("hydra_restart", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "restart failed")
	sup.AssertExpectations(t)
}

func TestHandleForceRestartFailsWhenSupervisorNil(t *testing.T) {
	status := func() StatusSnapshot {
		return StatusSnapshot{State: supervisor.StateRunning}
	}

	tools := New(
		Options{
			Enabled: true,
			Tools:   []string{"hydra_force_restart"},
		},
		nil,
		status,
		stubRedactor{},
		stubLogger{},
		NewLogBuffer(100),
	)

	_, err := tools.Handle("hydra_force_restart", map[string]interface{}{
		"confirm": true,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "supervisor not configured")
}

func TestHandleForceRestartFailsWhenRestartFails(t *testing.T) {
	sup := new(mocks.Supervisor)
	sup.On("ResetRestartCounter").Return().Once()
	sup.On("Restart").Return(assert.AnError).Once()

	status := func() StatusSnapshot {
		return StatusSnapshot{State: supervisor.StateRunning}
	}

	tools := New(
		Options{
			Enabled: true,
			Tools:   []string{"hydra_force_restart"},
		},
		sup,
		status,
		stubRedactor{},
		stubLogger{},
		NewLogBuffer(100),
	)

	_, err := tools.Handle("hydra_force_restart", map[string]interface{}{
		"confirm": true,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "force restart failed")
	sup.AssertExpectations(t)
}
