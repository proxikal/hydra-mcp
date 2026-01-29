// +build windows

package supervisor

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

func (s *Supervisor) startProcess() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == StateRunning {
		return fmt.Errorf("server already running")
	}

	s.logger.Debug("starting server process", map[string]interface{}{
		"command": s.command,
		"args":    s.args,
	})

	// Build command
	cmd := exec.Command(s.command[0], s.command[1:]...)
	if s.cwd != "" {
		cmd.Dir = s.cwd
	}

	// Windows-specific: CREATE_NEW_PROCESS_GROUP
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
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

	cmd.Stderr = os.Stderr

	// Start process
	if err := cmd.Start(); err != nil {
		s.state = StateFailed
		s.lastError = err
		s.logger.Error("failed to start process", err, nil)
		return err
	}

	s.process = &Process{
		Process:   cmd.Process,
		Stdin:     stdin,
		Stdout:    stdout,
		StartTime: time.Now(),
	}

	s.state = StateRunning
	s.startTime = time.Now()
	s.lastError = nil

	// Monitor process
	go s.monitorProcess(cmd)

	s.logger.Info("server process started", map[string]interface{}{
		"pid": cmd.Process.Pid,
	})

	return nil
}
