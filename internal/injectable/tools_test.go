package injectable

import (
	"testing"

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
		nil, // metrics collector
		stubRedactor{},
		stubLogger{},
		NewLogBuffer(100),
	)

	defs := tools.GetDefinitions()
	assert.Len(t, defs, 2)
	assert.Equal(t, "hydra_restart", defs[0].Name)
	assert.Equal(t, "hydra_status", defs[1].Name)
}

func TestHandleUnknownToolReturnsError(t *testing.T) {
	sup := new(mocks.Supervisor)
	status := func() StatusSnapshot { return StatusSnapshot{} }

	tools := New(
		Options{Enabled: true},
		sup,
		status,
		nil, // metrics collector
		stubRedactor{},
		logger.New("info"),
		NewLogBuffer(10),
	)

	_, err := tools.Handle("unknown_tool", nil)
	assert.Error(t, err)
}

func TestHandleReturnsErrDisabledWhenToolsDisabled(t *testing.T) {
	sup := new(mocks.Supervisor)
	status := func() StatusSnapshot { return StatusSnapshot{} }

	tools := New(
		Options{
			Enabled: false,
			Tools:   []string{"hydra_restart"},
		},
		sup,
		status,
		nil, // metrics collector
		stubRedactor{},
		logger.New("info"),
		NewLogBuffer(10),
	)

	_, err := tools.Handle("hydra_restart", nil)
	assert.Error(t, err)
	assert.Equal(t, ErrDisabled, err)
}

func TestGetDefinitionsFiltersUnknownTools(t *testing.T) {
	sup := new(mocks.Supervisor)
	status := func() StatusSnapshot {
		return StatusSnapshot{State: supervisor.StateRunning}
	}

	tools := New(
		Options{
			Enabled: true,
			Tools: []string{
				"hydra_restart",
				"unknown_tool",
				"hydra_status",
			},
		},
		sup,
		status,
		nil, // metrics collector
		stubRedactor{},
		stubLogger{},
		NewLogBuffer(100),
	)

	defs := tools.GetDefinitions()
	assert.Len(t, defs, 2)
	assert.Equal(t, "hydra_restart", defs[0].Name)
	assert.Equal(t, "hydra_status", defs[1].Name)
}
