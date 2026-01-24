package supervisor

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/proxikal/hydra/internal/logger"
)

type supervisor struct {
	cmd           []string
	cwd           string
	maxRestarts   int
	restartWindow time.Duration
	logger        logger.Logger
	mu            sync.RWMutex
	state         ServerState
	process       *exec.Cmd
	stdin         io.WriteCloser
	stdout        io.ReadCloser
	pid           int
	startTime     time.Time
	lastError     error
	restartCount  int
	restartTimes  []time.Time
	monitorCancel context.CancelFunc
	monitorWait   sync.WaitGroup
}

// NewSupervisor creates a new supervisor for managing a child process
func NewSupervisor(cmd []string, cwd string, maxRestarts int, restartWindow time.Duration, log logger.Logger) Supervisor {
	return &supervisor{
		cmd:           cmd,
		cwd:           cwd,
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
		// Already started, return success (idempotent)
		return nil
	}

	s.state = StateStarting

	// Create command
	if len(s.cmd) == 0 {
		s.state = StateFailed
		s.lastError = fmt.Errorf("empty command")
		return s.lastError
	}

	cmd := exec.Command(s.cmd[0], s.cmd[1:]...)

	// Set working directory if specified
	if s.cwd != "" {
		cmd.Dir = s.cwd
	}

	// Set process group for tree kill support
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Create pipes for stdin/stdout
	stdin, err := cmd.StdinPipe()
	if err != nil {
		s.state = StateFailed
		s.lastError = err
		s.logger.Error("failed to create stdin pipe", err, nil)
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.state = StateFailed
		s.lastError = err
		s.logger.Error("failed to create stdout pipe", err, nil)
		return err
	}

	// Start process
	if err := cmd.Start(); err != nil {
		s.state = StateFailed
		s.lastError = err
		s.logger.Error("failed to start process", err, nil)
		return err
	}

	// Store pipes
	s.stdin = stdin
	s.stdout = stdout

	s.process = cmd
	s.pid = cmd.Process.Pid
	s.startTime = time.Now()
	s.state = StateRunning

	// Create cancelable context for monitor
	ctx, cancel := context.WithCancel(context.Background())
	s.monitorCancel = cancel

	// Monitor process in background
	s.monitorWait.Add(1)
	go s.monitor(ctx)

	return nil
}

func (s *supervisor) Stop() error {
	s.mu.Lock()
	if s.state != StateRunning && s.state != StateRestarting {
		s.mu.Unlock()
		return nil
	}

	// Get process reference while holding lock
	proc := s.process
	if proc == nil || proc.Process == nil {
		s.state = StateStopped
		s.pid = 0
		s.mu.Unlock()
		return nil
	}

	// Signal process first
	_ = proc.Process.Signal(syscall.SIGTERM)

	// Release lock BEFORE waiting for monitor (to avoid deadlock)
	s.mu.Unlock()

	// Wait for monitor to finish (with timeout)
	done := make(chan struct{})
	go func() {
		s.monitorWait.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Monitor finished
	case <-time.After(5 * time.Second):
		// Timeout - force kill
		_ = proc.Process.Kill()
		<-done
	}

	// Re-acquire lock to update state
	s.mu.Lock()
	if s.monitorCancel != nil {
		s.monitorCancel()
		s.monitorCancel = nil
	}
	s.state = StateStopped
	s.pid = 0
	s.process = nil
	s.mu.Unlock()

	return nil
}

// stopProcessLocked stops the process. MUST be called with lock held.
// Returns the process reference for waiting, and releases the lock.
// Caller must re-acquire lock after waiting and call cleanupAfterStop().
func (s *supervisor) stopProcessLocked() *exec.Cmd {
	proc := s.process
	if proc == nil || proc.Process == nil {
		s.state = StateStopped
		s.pid = 0
		return nil
	}

	// Signal process first
	_ = proc.Process.Signal(syscall.SIGTERM)

	return proc
}

// waitForMonitorUnlocked waits for monitor to finish. Must NOT hold lock.
func (s *supervisor) waitForMonitorUnlocked(proc *exec.Cmd) {
	done := make(chan struct{})
	go func() {
		s.monitorWait.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Monitor finished
	case <-time.After(5 * time.Second):
		// Timeout - force kill
		if proc != nil && proc.Process != nil {
			_ = proc.Process.Kill()
		}
		<-done
	}
}

// cleanupAfterStop finalizes state after process stopped. MUST be called with lock held.
func (s *supervisor) cleanupAfterStop() {
	if s.monitorCancel != nil {
		s.monitorCancel()
		s.monitorCancel = nil
	}
	s.state = StateStopped
	s.pid = 0
	s.process = nil
}

func (s *supervisor) Restart() error {
	s.mu.Lock()

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
		s.mu.Unlock()
		return s.lastError
	}

	s.state = StateRestarting

	// Stop current process - get proc reference and release lock
	proc := s.stopProcessLocked()
	s.mu.Unlock()

	// Wait for monitor without holding lock
	if proc != nil {
		s.waitForMonitorUnlocked(proc)
	}

	// Cleanup and start new process
	s.mu.Lock()
	s.cleanupAfterStop()
	s.mu.Unlock()

	// Start new process
	return s.Start()
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

func (s *supervisor) Stdin() io.WriteCloser {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stdin
}

func (s *supervisor) Stdout() io.ReadCloser {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stdout
}

func (s *supervisor) monitor(ctx context.Context) {
	defer s.monitorWait.Done()

	// Save process reference to avoid race
	s.mu.RLock()
	proc := s.process
	s.mu.RUnlock()

	if proc == nil {
		return
	}

	// Wait for process to exit
	// This is the ONLY place that calls Wait() on the process
	err := proc.Wait()

	// Check if we were cancelled
	select {
	case <-ctx.Done():
		// Cancelled - don't update state
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
