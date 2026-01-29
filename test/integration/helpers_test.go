package integration

import (
	"context"
	"io"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/logger"
	"github.com/proxikal/hydra/internal/mocks"
	"github.com/proxikal/hydra/internal/proxy"
	"github.com/proxikal/hydra/internal/sanitizer"
	"github.com/proxikal/hydra/internal/transport"
	"github.com/stretchr/testify/require"
)

// testProxy wraps a proxy instance with helpers for testing.
type testProxy struct {
	proxy          proxy.Proxy
	clientTransport *channelTransport
	serverCmd      *exec.Cmd
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	t              *testing.T
}

// newTestProxy creates a proxy with a test server.
// Returns proxy and cleanup function.
func newTestProxy(t *testing.T, serverScript string) (*testProxy, func()) {
	return newTestProxyWithTimeout(t, serverScript, 30*time.Second)
}

// newTestProxyWithTimeout creates a proxy with custom timeout.
func newTestProxyWithTimeout(t *testing.T, serverScript string, timeout time.Duration) (*testProxy, func()) {
	log := logger.New("error")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	// Start server process
	dir := filepath.Join("..", "fixtures")
	cmd := exec.Command(filepath.Join(dir, serverScript))
	cmd.Dir = dir

	childIn, err := cmd.StdinPipe()
	require.NoError(t, err)
	childOut, err := cmd.StdoutPipe()
	require.NoError(t, err)

	err = cmd.Start()
	require.NoError(t, err)

	childTransport := transport.NewStdio(childOut, childIn, log)
	clientTransport := newChannelTransport()

	// Create mock supervisor
	sup := new(mocks.Supervisor)
	sup.On("Start").Return(nil)
	sup.On("PID").Return(cmd.Process.Pid)
	sup.On("Uptime").Return(time.Second)
	sup.On("LastError").Return(nil)

	// Create proxy
	p := proxy.New(
		proxy.Dependencies{
			Logger:     log,
			Sanitizer:  sanitizer.New(),
			Supervisor: sup,
			Child:      childTransport,
			Client:     clientTransport,
		},
		proxy.Options{},
	)

	tp := &testProxy{
		proxy:           p,
		clientTransport: clientTransport,
		serverCmd:       cmd,
		ctx:             ctx,
		cancel:          cancel,
		t:               t,
	}

	// Start proxy loop
	tp.wg.Add(1)
	go func() {
		defer tp.wg.Done()
		_ = p.Run()
	}()

	// Wait for initialization
	time.Sleep(100 * time.Millisecond)

	cleanup := func() {
		tp.Stop()
	}

	return tp, cleanup
}

// SendRequest sends a JSON-RPC request to the proxy.
func (tp *testProxy) SendRequest(msg string) {
	select {
	case tp.clientTransport.in <- []byte(msg):
	case <-time.After(1 * time.Second):
		tp.t.Fatal("Timeout sending request")
	}
}

// WaitForResponse waits for a response from the proxy.
func (tp *testProxy) WaitForResponse() []byte {
	select {
	case write := <-tp.clientTransport.out:
		return write
	case <-time.After(5 * time.Second):
		tp.t.Fatal("Timeout waiting for response")
		return nil
	}
}

// GetWrites returns all writes to the client transport.
func (tp *testProxy) GetWrites() [][]byte {
	tp.clientTransport.mu.Lock()
	defer tp.clientTransport.mu.Unlock()
	return append([][]byte{}, tp.clientTransport.writes...)
}

// Stop stops the proxy and cleans up resources.
func (tp *testProxy) Stop() {
	tp.cancel()
	_ = tp.clientTransport.Close()
	tp.wg.Wait()
	if tp.serverCmd != nil && tp.serverCmd.Process != nil {
		_ = tp.serverCmd.Process.Kill()
		_ = tp.serverCmd.Wait()
	}
}
