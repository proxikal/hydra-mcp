package supervisor

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/proxikal/hydra/internal/logger"
)

type supervisor struct {
	cmd           []string
	maxRestarts   int
	restartWindow time.Duration
	logger        logger.Logger
	mu            sync.RWMutex
	state         ServerState
	process       *exec.Cmd
	pid           int
	startTime     time.Time
	lastError     error
	restartCount  int
	restartTimes  []time.Time
	monitorCancel context.CancelFunc
}

// NewSupervisor creates a new supervisor for managing a child process
func NewSupervisor(cmd []string, maxRestarts int, restartWindow time.Duration, log logger.Logger) Supervisor {
	return &supervisor{
		cmd:           cmd,
		maxRestarts:   maxRestarts,
		restartWindow: restartWindow,
		logger:        log,
		state:         StateStopped,
		restartTimes:  make([]time.Time, 0),
	}
}

func (s *supervisor) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == StateRunning || s.state == StateStarting {
		return fmt.Errorf("process already running")
	}

	s.state = StateStarting

	// Create command
	if len(s.cmd) == 0 {
		s.state = StateFailed
		s.lastError = fmt.Errorf("empty command")
		return s.lastError
	}

	cmd := exec.Command(s.cmd[0], s.cmd[1:]...)

	// Set process group for tree kill support
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Start process
	if err := cmd.Start(); err != nil {
		s.state = StateFailed
		s.lastError = err
		s.logger.Error("failed to start process", err, nil)
		return err
	}

	s.process = cmd
	s.pid = cmd.Process.Pid
	s.startTime = time.Now()
	s.state = StateRunning

	// Create cancelable context for monitor
	ctx, cancel := context.WithCancel(context.Background())
	s.monitorCancel = cancel

	// Monitor process in background
	go s.monitor(ctx)

	return nil
}

func (s *supervisor) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != StateRunning && s.state != StateRestarting {
		return nil
	}

	return s.stopProcess()
}

func (s *supervisor) stopProcess() error {
	// Cancel old monitor first to prevent race condition
	if s.monitorCancel != nil {
		s.monitorCancel()
		s.monitorCancel = nil
	}

	if s.process == nil || s.process.Process == nil {
		s.state = StateStopped
		s.pid = 0
		return nil
	}

	// Send SIGTERM
	if err := s.process.Process.Signal(syscall.SIGTERM); err != nil {
		// Process may have already exited
		s.state = StateStopped
		s.pid = 0
		return nil
	}

	// Wait for graceful shutdown with timeout
	done := make(chan error, 1)
	go func() {
		done <- s.process.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		// Force kill if graceful shutdown failed
		_ = s.process.Process.Kill()
		<-done
	case <-done:
		// Process exited gracefully
	}

	s.state = StateStopped
	s.pid = 0
	s.process = nil

	return nil
}

func (s *supervisor) Restart() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check crash loop
	now := time.Now()
	s.restartTimes = append(s.restartTimes, now)

	// Filter restarts within window
	cutoff := now.Add(-s.restartWindow)
	validRestarts := make([]time.Time, 0)
	for _, t := range s.restartTimes {
		if t.After(cutoff) {
			validRestarts = append(validRestarts, t)
		}
	}
	s.restartTimes = validRestarts

	if len(s.restartTimes) > s.maxRestarts {
		s.state = StateFailed
		s.lastError = fmt.Errorf("crash loop detected: %d restarts in %v", len(s.restartTimes), s.restartWindow)
		s.logger.Error("crash loop detected", s.lastError, nil)
		return s.lastError
	}

	s.state = StateRestarting

	// Stop current process
	_ = s.stopProcess()

	// Start new process
	s.mu.Unlock()
	err := s.Start()
	s.mu.Lock()

	return err
}

func (s *supervisor) State() ServerState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *supervisor) PID() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pid
}

func (s *supervisor) Uptime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.state != StateRunning {
		return 0
	}

	return time.Since(s.startTime)
}

func (s *supervisor) LastError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastError
}

func (s *supervisor) ResetRestartCounter() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.restartTimes = make([]time.Time, 0)
	s.restartCount = 0
}

func (s *supervisor) monitor(ctx context.Context) {
	if s.process == nil {
		return
	}

	// Wait for process to exit
	done := make(chan error, 1)
	go func() {
		done <- s.process.Wait()
	}()

	var err error
	select {
	case <-ctx.Done():
		// Monitor cancelled (intentional stop/restart)
		return
	case err = <-done:
		// Process exited
	}

	// Check context again before changing state
	select {
	case <-ctx.Done():
		// Cancelled while waiting, don't update state
		return
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		s.lastError = err
	}

	// If we're still in running state, the process died unexpectedly
	if s.state == StateRunning {
		s.state = StateStopped
		s.pid = 0
	}
}
