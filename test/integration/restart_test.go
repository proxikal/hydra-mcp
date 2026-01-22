package integration

import (
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"io"

	"github.com/proxikal/hydra/internal/logger"
	"github.com/proxikal/hydra/internal/mocks"
	"github.com/proxikal/hydra/internal/proxy"
	"github.com/proxikal/hydra/internal/sanitizer"
	"github.com/proxikal/hydra/internal/transport"
	"github.com/stretchr/testify/assert"
)

type channelTransport struct {
	in     chan []byte
	out    chan []byte
	closed chan struct{}
	mu     sync.Mutex
	writes [][]byte
}

func newChannelTransport() *channelTransport {
	return &channelTransport{
		in:     make(chan []byte, 4),
		out:    make(chan []byte, 4),
		closed: make(chan struct{}),
	}
}

func (t *channelTransport) Read() ([]byte, error) {
	select {
	case msg := <-t.in:
		return msg, nil
	case <-t.closed:
		return nil, io.EOF
	}
}

func (t *channelTransport) Write(p []byte) error {
	t.mu.Lock()
	t.writes = append(t.writes, append([]byte{}, p...))
	t.mu.Unlock()
	return nil
}

func (t *channelTransport) Close() error {
	close(t.closed)
	return nil
}

func (t *channelTransport) DetectProtocol(_ time.Duration) (transport.Protocol, error) {
	return transport.ProtocolNDJSON, nil
}

// End-to-end smoke test: proxy between client channel and echo server process.
func TestProxyEchoIntegration(t *testing.T) {
	log := logger.New("error")

	// Start echo server process
	dir := filepath.Join("..", "fixtures")
	cmd := exec.Command(filepath.Join(dir, "echo_server.py"))
	cmd.Dir = dir

	childIn, err := cmd.StdinPipe()
	assert.NoError(t, err)
	childOut, err := cmd.StdoutPipe()
	assert.NoError(t, err)

	err = cmd.Start()
	assert.NoError(t, err)
	defer cmd.Process.Kill()

	childTransport := transport.NewStdio(childOut, childIn, log)
	clientTransport := newChannelTransport()

	sup := new(mocks.Supervisor)
	sup.On("Start").Return(nil)
	sup.On("PID").Return(0)
	sup.On("Uptime").Return(time.Second)
	sup.On("LastError").Return(nil)

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

	// Run proxy loop
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = p.Run()
	}()

	// Send initialize
	clientTransport.in <- []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	time.Sleep(200 * time.Millisecond)

	// Send echo call
	clientTransport.in <- []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"foo":"bar"}}`)
	time.Sleep(200 * time.Millisecond)

	clientTransport.Close()
	wg.Wait()

	assert.GreaterOrEqual(t, len(clientTransport.writes), 2)
}

// Process-level queue drain: start echo server process, queue while starting, assert forwarded after init.
func TestProxyProcessQueueDrain(t *testing.T) {
	log := logger.New("error")

	dir := filepath.Join("..", "fixtures")
	cmd := exec.Command(filepath.Join(dir, "echo_server.py"))
	cmd.Dir = dir

	childIn, err := cmd.StdinPipe()
	assert.NoError(t, err)
	childOut, err := cmd.StdoutPipe()
	assert.NoError(t, err)

	err = cmd.Start()
	assert.NoError(t, err)
	defer cmd.Process.Kill()

	childTransport := transport.NewStdio(childOut, childIn, log)
	clientTransport := newChannelTransport()

	sup := new(mocks.Supervisor)
	sup.On("Start").Return(nil)
	sup.On("PID").Return(0)
	sup.On("Uptime").Return(time.Second)
	sup.On("LastError").Return(nil)

	p := proxy.New(
		proxy.Dependencies{
			Logger:     log,
			Sanitizer:  sanitizer.New(),
			Supervisor: sup,
			Child:      childTransport,
			Client:     clientTransport,
		},
		proxy.Options{QueueSize: 2},
	)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = p.Run()
	}()

	// Queue messages while starting
	clientTransport.in <- []byte(`{"jsonrpc":"2.0","id":10,"method":"call1"}`)
	clientTransport.in <- []byte(`{"jsonrpc":"2.0","id":11,"method":"call2"}`)

	time.Sleep(300 * time.Millisecond)

	clientTransport.Close()
	wg.Wait()

	assert.GreaterOrEqual(t, len(clientTransport.writes), 2)
}
