package integration

import (
	"io"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/logger"
	"github.com/proxikal/hydra/internal/mocks"
	"github.com/proxikal/hydra/internal/proxy"
	"github.com/proxikal/hydra/internal/sanitizer"
	"github.com/proxikal/hydra/internal/transport"
	"github.com/stretchr/testify/assert"
)

// Smoke test to ensure crash_server fixture exits with non-zero immediately.
func TestCrashServerExitsNonZero(t *testing.T) {
	dir := filepath.Join("..", "fixtures")
	cmd := exec.Command(filepath.Join(dir, "crash_server.py"))
	cmd.Dir = dir

	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected non-zero exit")
	}
}

// Integration-ish test: simulate crash loop by surfacing restart error to proxy.
func TestProxyCrashLoopTransitionsFailed(t *testing.T) {
	log := logger.New("error")
	child := &chanTransport{err: io.ErrUnexpectedEOF}
	client := &chanTransport{readCh: make(chan []byte)}

	sup := new(mocks.Supervisor)
	sup.On("Start").Return(nil)
	sup.On("Restart").Return(assert.AnError)
	sup.On("PID").Return(0)
	sup.On("Uptime").Return(time.Second)
	sup.On("LastError").Return(assert.AnError)

	p := proxy.New(
		proxy.Dependencies{
			Logger:     log,
			Sanitizer:  sanitizer.New(),
			Supervisor: sup,
			Child:      child,
			Client:     client,
		},
		proxy.Options{},
	)

	err := p.Run()
	assert.Error(t, err)
	status := p.Status()
	assert.Equal(t, proxy.StateFailed, status.State)
}

// Process-level crash: start crash_server, expect proxy to fail.
func TestProxyCrashServerProcessFails(t *testing.T) {
	log := logger.New("error")

	dir := filepath.Join("..", "fixtures")
	cmd := exec.Command(filepath.Join(dir, "crash_server.py"))
	cmd.Dir = dir

	childIn, err := cmd.StdinPipe()
	assert.NoError(t, err)
	childOut, err := cmd.StdoutPipe()
	assert.NoError(t, err)

	err = cmd.Start()
	assert.NoError(t, err)
	defer cmd.Process.Kill()

	child := transport.NewStdio(childOut, childIn, log)
	client := newChanTransport()

	sup := new(mocks.Supervisor)
	sup.On("Start").Return(nil)
	sup.On("PID").Return(0)
	sup.On("Uptime").Return(time.Second)
	sup.On("LastError").Return(assert.AnError)

	p := proxy.New(
		proxy.Dependencies{
			Logger:     log,
			Sanitizer:  sanitizer.New(),
			Supervisor: sup,
			Child:      child,
			Client:     client,
		},
		proxy.Options{},
	)

	err = p.Run()
	assert.Error(t, err)
	assert.Equal(t, proxy.StateFailed, p.Status().State)
}
