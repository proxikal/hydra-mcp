package injectable

import (
	"testing"

	"github.com/proxikal/hydra/internal/mocks"
	"github.com/proxikal/hydra/internal/supervisor"
	"github.com/stretchr/testify/assert"
)

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
		nil, // metrics collector
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
		nil, // metrics collector
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
