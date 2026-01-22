package injectable

import (
	"errors"
	"time"

	"github.com/proxikal/hydra/internal/logger"
	"github.com/proxikal/hydra/internal/security"
	"github.com/proxikal/hydra/internal/supervisor"
)

// InjectableTools exposes Hydra meta-tools.
type InjectableTools interface {
	GetDefinitions() []ToolDefinition
	Handle(toolName string, params map[string]interface{}) (interface{}, error)
}

// ToolDefinition describes an injectable tool.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// Behavior holds restart configuration for status reporting.
type Behavior struct {
	MaxRestarts          int
	RestartWindowSeconds int
}

// StatusSnapshot represents current proxy/supervisor state.
type StatusSnapshot struct {
	State             supervisor.ServerState
	PID               int
	Uptime            time.Duration
	RestartsInWindow  int
	QueueSize         int
	LastRestartReason string
	LastError         string
	CanRecover        bool
}

// Options configure injectable tools.
type Options struct {
	Enabled        bool
	Tools          []string
	RedactPatterns []string
	MaxLogLines    int
	Server         string
	Behavior       Behavior
}

type toolset struct {
	opts       Options
	supervisor supervisor.Supervisor
	status     func() StatusSnapshot
	redactor   security.Redactor
	logger     logger.Logger
	buffer     LogBuffer
}

// Errors returned by injectable tools.
var (
	ErrDisabled    = errors.New("injectable tools disabled")
	ErrUnknownTool = errors.New("unknown tool")
)

// New creates a new injectable tool set.
func New(
	opts Options,
	sup supervisor.Supervisor,
	status func() StatusSnapshot,
	redactor security.Redactor,
	log logger.Logger,
	buffer LogBuffer,
) InjectableTools {
	if len(opts.Tools) == 0 {
		opts.Tools = []string{
			"hydra_restart",
			"hydra_status",
			"hydra_logs",
			"hydra_force_restart",
		}
	}

	if opts.MaxLogLines <= 0 {
		opts.MaxLogLines = 500
	}

	if buffer == nil {
		buffer = NewLogBuffer(opts.MaxLogLines)
	}

	return &toolset{
		opts:       opts,
		supervisor: sup,
		status:     status,
		redactor:   redactor,
		logger:     log,
		buffer:     buffer,
	}
}

func (t *toolset) GetDefinitions() []ToolDefinition {
	definitions := make([]ToolDefinition, 0, len(t.opts.Tools))
	for _, name := range t.opts.Tools {
		if def, ok := t.definitionFor(name); ok {
			definitions = append(definitions, def)
		}
	}
	return definitions
}

func (t *toolset) Handle(
	toolName string,
	params map[string]interface{},
) (interface{}, error) {
	if !t.opts.Enabled {
		return nil, ErrDisabled
	}

	switch toolName {
	case "hydra_restart":
		return t.handleRestart(params)
	case "hydra_status":
		return t.handleStatus()
	case "hydra_logs":
		return t.handleLogs(params)
	case "hydra_force_restart":
		return t.handleForceRestart(params)
	default:
		return nil, ErrUnknownTool
	}
}

func (t *toolset) definitionFor(
	name string,
) (ToolDefinition, bool) {
	switch name {
	case "hydra_restart":
		return ToolDefinition{
			Name:        "hydra_restart",
			Description: "Manually restart the supervised MCP server.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"reason": map[string]interface{}{
						"type":        "string",
						"description": "Optional reason for restart",
					},
				},
			},
		}, true
	case "hydra_status":
		return ToolDefinition{
			Name:        "hydra_status",
			Description: "Get current Hydra supervisor status.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		}, true
	case "hydra_logs":
		return ToolDefinition{
			Name:        "hydra_logs",
			Description: "Retrieve recent child stderr logs.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"lines": map[string]interface{}{
						"type":        "number",
						"description": "Number of recent log lines to retrieve",
						"minimum":     1,
						"maximum":     500,
					},
				},
			},
		}, true
	case "hydra_force_restart":
		return ToolDefinition{
			Name:        "hydra_force_restart",
			Description: "Force restart even if in FAILED state.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"confirm": map[string]interface{}{
						"type":        "boolean",
						"description": "Must be true to confirm force restart",
					},
				},
				"required": []string{"confirm"},
			},
		}, true
	default:
		return ToolDefinition{}, false
	}
}
