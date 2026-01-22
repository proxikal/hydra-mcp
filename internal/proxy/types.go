package proxy

import (
	"time"

	"github.com/proxikal/hydra/internal/injectable"
	"github.com/proxikal/hydra/internal/logger"
	"github.com/proxikal/hydra/internal/recorder"
	"github.com/proxikal/hydra/internal/sanitizer"
	"github.com/proxikal/hydra/internal/security"
	"github.com/proxikal/hydra/internal/statestore"
	"github.com/proxikal/hydra/internal/supervisor"
	"github.com/proxikal/hydra/internal/transport"
)

// Proxy orchestrates routing between client and child MCP server.
type Proxy interface {
	Run() error
	Shutdown() error
	Status() ProxyStatus
}

// ProxyStatus exposes current proxy metrics.
type ProxyStatus struct {
	State             supervisor.ServerState
	PID               int
	Uptime            time.Duration
	RestartsInWindow  int
	QueueSize         int
	CanRecover        bool
	LastRestartReason string
	LastError         string
}

// ProxyState mirrors supervisor.ServerState for readability.
type ProxyState = supervisor.ServerState

const (
	StateStopped    ProxyState = supervisor.StateStopped
	StateStarting   ProxyState = supervisor.StateStarting
	StateRunning    ProxyState = supervisor.StateRunning
	StateRestarting ProxyState = supervisor.StateRestarting
	StateFailed     ProxyState = supervisor.StateFailed
)

// Options configure proxy behavior.
type Options struct {
	QueueSize          int
	QueueTTL           time.Duration
	CollisionPolicy    string
	HydraTools         []injectable.ToolDefinition
	CrashExportPath    string
	CrashExportEnabled bool
}

// Dependencies for constructing a proxy.
type Dependencies struct {
	Logger              logger.Logger
	Sanitizer           sanitizer.Sanitizer
	Supervisor          supervisor.Supervisor
	Child               transport.Transport
	Client              transport.Transport
	Recorder            recorder.Recorder
	Redactor            security.Redactor
	StateStore          statestore.StateStore
	HydraTools          []injectable.ToolDefinition
	MaxRestartsInWindow func() int
	MaxRestarts         func() int
}

type proxy struct {
	state              ProxyState
	queue              Queue
	logger             logger.Logger
	sanitizer          sanitizer.Sanitizer
	supervisor         supervisor.Supervisor
	child              transport.Transport
	client             transport.Transport
	recorder           recorder.Recorder
	redactor           security.Redactor
	stateStore         statestore.StateStore
	hydraTools         []injectable.ToolDefinition
	collisionPolicy    string
	crashExportPath    string
	crashExportEnabled bool
	maxRestartsInWnd   func() int
	maxRestarts        func() int
	lastRestartReason  string
	lastError          string
	started            time.Time
	replay             func()
	stopCh             chan struct{}
	mainLoop           func() error
}
