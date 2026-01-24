package injectable

import (
	"errors"
	"fmt"
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
	ErrDisabled            = errors.New("injectable tools disabled")
	ErrUnknownTool         = errors.New("unknown tool")
	ErrNamespaceCollision  = errors.New("namespace collision detected")
	ErrInvalidToolName     = errors.New("tool name must start with hydra_")
)

// ValidateToolNames checks that all tool names follow the hydra_ convention.
func ValidateToolNames(tools []string) error {
	for _, name := range tools {
		if len(name) < 6 || name[:6] != "hydra_" {
			return fmt.Errorf("%w: %s", ErrInvalidToolName, name)
		}
	}
	return nil
}

// DetectCollision checks if any child tools collide with Hydra tools.
func DetectCollision(
	hydraTools []ToolDefinition,
	childTools []string,
) ([]string, error) {
	hydraNames := make(map[string]bool)
	for _, tool := range hydraTools {
		hydraNames[tool.Name] = true
	}

	var collisions []string
	for _, childTool := range childTools {
		if hydraNames[childTool] {
			collisions = append(collisions, childTool)
		}
	}

	if len(collisions) > 0 {
		return collisions, fmt.Errorf(
			"%w: %v",
			ErrNamespaceCollision,
			collisions,
		)
	}

	return nil, nil
}

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
		opts.MaxLogLines = 50
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
