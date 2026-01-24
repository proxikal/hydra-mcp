package benchmarks

import (
	"context"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
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

// benchProxy wraps a proxy for benchmarking.
type benchProxy struct {
	proxy           proxy.Proxy
	clientTransport *benchTransport
	serverCmd       *exec.Cmd
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

// newBenchProxy creates a proxy for benchmarking.
func newBenchProxy(b *testing.B, serverScript string) *benchProxy {
	log := logger.New("error")
	ctx, cancel := context.WithCancel(context.Background())

	// Start server
	dir := filepath.Join("../../test/fixtures")
	cmd := exec.Command(filepath.Join(dir, serverScript))
	cmd.Dir = dir

	childIn, err := cmd.StdinPipe()
	require.NoError(b, err)
	childOut, err := cmd.StdoutPipe()
	require.NoError(b, err)

	err = cmd.Start()
	require.NoError(b, err)

	childTransport := transport.NewStdio(childOut, childIn, log)
	clientTransport := newBenchTransport()

	sup := new(mocks.Supervisor)
	sup.On("Start").Return(nil)
	sup.On("PID").Return(cmd.Process.Pid)
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

	bp := &benchProxy{
		proxy:           p,
		clientTransport: clientTransport,
		serverCmd:       cmd,
		ctx:             ctx,
		cancel:          cancel,
	}

	bp.wg.Add(1)
	go func() {
		defer bp.wg.Done()
		_ = p.Run()
	}()

	time.Sleep(100 * time.Millisecond)
	return bp
}

// Stop stops the benchmark proxy.
func (bp *benchProxy) Stop() {
	bp.cancel()
	_ = bp.clientTransport.Close()
	bp.wg.Wait()
	if bp.serverCmd != nil && bp.serverCmd.Process != nil {
		_ = bp.serverCmd.Process.Kill()
		_ = bp.serverCmd.Wait()
	}
}

// benchTransport is a minimal transport for benchmarking.
type benchTransport struct {
	in     chan []byte
	out    chan []byte
	closed chan struct{}
}

func newBenchTransport() *benchTransport {
	return &benchTransport{
		in:     make(chan []byte, 10),
		out:    make(chan []byte, 10),
		closed: make(chan struct{}),
	}
}

func (t *benchTransport) Read() ([]byte, error) {
	select {
	case msg := <-t.in:
		return msg, nil
	case <-t.closed:
		return nil, io.EOF
	}
}

func (t *benchTransport) Write(p []byte) error {
	select {
	case t.out <- append([]byte{}, p...):
		return nil
	case <-t.closed:
		return io.ErrClosedPipe
	}
}

func (t *benchTransport) Close() error {
	close(t.closed)
	return nil
}

func (t *benchTransport) DetectProtocol(_ time.Duration) (transport.Protocol, error) {
	return transport.ProtocolNDJSON, nil
}

// memorySnapshot captures memory statistics at a point in time.
type memorySnapshot struct {
	AllocBytes   uint64
	TotalAlloc   uint64
	Goroutines   int
	HeapObjects  uint64
}

// captureMemory captures current memory and goroutine stats.
func captureMemory() memorySnapshot {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return memorySnapshot{
		AllocBytes:  m.Alloc,
		TotalAlloc:  m.TotalAlloc,
		Goroutines:  runtime.NumGoroutine(),
		HeapObjects: m.HeapObjects,
	}
}

// memoryIncrease calculates memory increase between snapshots.
func memoryIncrease(before, after memorySnapshot) float64 {
	increase := int64(after.AllocBytes - before.AllocBytes)
	return float64(increase) / (1024 * 1024)
}

// goroutineIncrease calculates goroutine count increase.
func goroutineIncrease(before, after memorySnapshot) int {
	return after.Goroutines - before.Goroutines
}
