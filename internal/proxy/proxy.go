package proxy

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"
)

type HydraProxy struct {
	command []string
	mu      sync.Mutex
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	restarting bool
}

func New(command []string) *HydraProxy {
	return &HydraProxy{
		command: command,
	}
}

// Start launches the initial process
func (h *HydraProxy) Start() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.startProcess()
}

// Restart kills the current process and starts a new one
func (h *HydraProxy) Restart() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.restarting {
		return nil
	}
	h.restarting = true
	defer func() { h.restarting = false }()

	log.Println("🔄 Hydra: Restarting server...")

	// Kill existing
	if h.cmd != nil && h.cmd.Process != nil {
		// Try graceful sigterm first
		h.cmd.Process.Signal(os.Interrupt)
		time.Sleep(100 * time.Millisecond)
		h.cmd.Process.Kill()
	}

	// Start new
	return h.startProcess()
}

func (h *HydraProxy) startProcess() error {
	cmd := exec.Command(h.command[0], h.command[1:]...)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil { return err }

	stdout, err := cmd.StdoutPipe()
	if err != nil { return err }

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	h.cmd = cmd
	h.stdin = stdin
	h.stdout = stdout

	go h.pipeOutput(stdout)

	log.Println("✅ Hydra: Server started/reloaded")
	return nil
}

func (h *HydraProxy) pipeOutput(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		// Direct pass-through to stdout (Agent sees this)
		fmt.Println(scanner.Text())
	}
}

// Write sends data to the child process stdin
func (h *HydraProxy) Write(p []byte) (n int, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.stdin == nil {
		return 0, fmt.Errorf("process not running")
	}
	return h.stdin.Write(p)
}
