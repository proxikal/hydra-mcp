package integration

import (
	"bytes"
	"encoding/json"
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
	"github.com/stretchr/testify/assert"
)

// NOTE: This is a lightweight smoke test to ensure fixtures run.
// Full integration wiring of proxy/supervisor is pending.
func TestEchoServerFixtureRuns(t *testing.T) {
	dir := filepath.Join("..", "fixtures")
	cmd := exec.Command(filepath.Join(dir, "echo_server.py"))
	cmd.Dir = dir

	// Start server
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill() }()

	// Send initialize
	initReq := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize"}` + "\n")
	if _, err := stdin.Write(initReq); err != nil {
		t.Fatal(err)
	}

	// Read response
	done := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 1024)
		n, _ := stdout.Read(buf)
		done <- buf[:n]
	}()

	select {
	case resp := <-done:
		if !bytes.Contains(resp, []byte(`"protocolVersion"`)) {
			t.Fatalf("unexpected response: %s", string(resp))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for initialize response")
	}

	// Send echo call
	callReq := []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"hello":"world"}}` + "\n")
	if _, err := stdin.Write(callReq); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	n, err := stdout.Read(buf)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	resp := buf[:n]
	var obj map[string]interface{}
	if err := json.Unmarshal(resp, &obj); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	result := obj["result"].(map[string]interface{})
	if result["hello"] != "world" {
		t.Fatalf("unexpected result: %v", result)
	}
}

type chanTransport struct {
	in     chan []byte
	closed chan struct{}
	readCh chan []byte
	writes [][]byte
	err    error
}

func newChanTransport() *chanTransport {
	return &chanTransport{
		in:     make(chan []byte, 8),
		closed: make(chan struct{}),
		writes: make([][]byte, 0),
	}
}

func (t *chanTransport) Read() ([]byte, error) {
	if t.err != nil {
		err := t.err
		t.err = nil
		return nil, err
	}
	if t.readCh != nil {
		select {
		case msg, ok := <-t.readCh:
			if !ok {
				return nil, io.EOF
			}
			return msg, nil
		default:
		}
	}
	select {
	case msg := <-t.in:
		return msg, nil
	case <-t.closed:
		return nil, io.EOF
	}
}

func (t *chanTransport) Write(p []byte) error {
	t.writes = append(t.writes, append([]byte{}, p...))
	return nil
}

func (t *chanTransport) Close() error {
	select {
	case <-t.closed:
	default:
		close(t.closed)
	}
	return nil
}

func (t *chanTransport) DetectProtocol(time.Duration) (transport.Protocol, error) {
	return transport.ProtocolNDJSON, nil
}

// Integration-ish check: queue requests during STARTING, drain after initialize result.
func TestProxyQueueAndDrainWithChannelTransports(t *testing.T) {
	log := logger.New("error")
	client := newChanTransport()
	child := newChanTransport()

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
			Child:      child,
			Client:     client,
		},
		proxy.Options{QueueSize: 2},
	)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = p.Run()
	}()

	// Queue while starting
	client.in <- []byte(`{"jsonrpc":"2.0","id":1,"method":"call1"}`)
	client.in <- []byte(`{"jsonrpc":"2.0","id":2,"method":"call2"}`)

	// Child sends initialize response
	child.in <- []byte(`{"jsonrpc":"2.0","id":0,"result":{"ok":true}}`)

	time.Sleep(200 * time.Millisecond)
	_ = client.Close()
	_ = child.Close()
	wg.Wait()

	assert.Len(t, child.writes, 2)
}
