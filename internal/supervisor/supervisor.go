package supervisor

import (
	"io"
	"time"
)

// ServerState represents the current state of the supervised process
type ServerState int

const (
	StateStopped ServerState = iota
	StateStarting
	StateRunning
	StateRestarting
	StateFailed
)

// String returns the string representation of ServerState
func (s ServerState) String() string {
	switch s {
	case StateStopped:
		return "stopped"
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StateRestarting:
		return "restarting"
	case StateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// Supervisor manages the lifecycle of a child process
type Supervisor interface {
	Start() error
	Stop() error
	Restart() error
	State() ServerState
	PID() int
	Uptime() time.Duration
	LastError() error
	ResetRestartCounter()
	Stdin() io.WriteCloser
	Stdout() io.ReadCloser
}
