package proxy

import (
	"io"
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/injectable"
	"github.com/proxikal/hydra/internal/mocks"
	"github.com/proxikal/hydra/internal/supervisor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestProxyStatusReflectsSupervisor(t *testing.T) {
	queue := NewQueue(2, time.Minute)
	_ = queue.Enqueue("req", []byte(`{"id":1}`))

	sup := new(mocks.Supervisor)
	sup.On("State").Return(supervisor.StateRunning)
	sup.On("PID").Return(123)
	sup.On("Uptime").Return(2 * time.Second)
	sup.On("LastError").Return(nil)
	sup.On("ResetRestartCounter").Maybe()

	p := &proxy{
		state:            StateRunning,
		queue:            queue,
		supervisor:       sup,
		logger:           stubLogger{},
		started:          time.Now().Add(-time.Second),
		maxRestartsInWnd: func() int { return 3 },
	}

	status := p.Status()
	assert.Equal(t, StateRunning, status.State)
	assert.Equal(t, 123, status.PID)
	assert.Equal(t, 1, status.QueueSize)
	assert.Equal(t, 3, status.RestartsInWindow)
}

func TestProxyStatusCrashLoopCanRecover(t *testing.T) {
	queue := NewQueue(1, time.Minute)
	sup := new(mocks.Supervisor)
	sup.On("PID").Return(0)
	sup.On("Uptime").Return(time.Second)
	sup.On("LastError").Return(nil)

	p := &proxy{
		state:            StateRunning,
		queue:            queue,
		supervisor:       sup,
		maxRestartsInWnd: func() int { return 5 },
		maxRestarts:      func() int { return 3 },
	}

	status := p.Status()
	assert.True(t, status.CanRecover)
	assert.Contains(t, status.LastError, "crash loop")
}

func TestProxyRunRecoversFromPanic(t *testing.T) {
	p := &proxy{
		logger:   stubLogger{},
		mainLoop: func() error { panic("boom") },
	}

	assert.NotPanics(t, func() {
		_ = p.Run()
	})
}

func TestLoopErrorTransitionsFailed(t *testing.T) {
	child := &stubTransport{err: io.ErrUnexpectedEOF}
	client := &stubTransport{readCh: make(chan []byte)}

	rec := new(mockRecorder)
	rec.On("Export", mock.Anything).Return(nil).Once()

	p := &proxy{
		state:           StateRunning,
		child:           child,
		client:          client,
		recorder:        rec,
		crashExportPath: "/tmp/hydra-recorder-crash.json",
		stopCh:          make(chan struct{}),
	}

	err := p.loop()
	assert.Error(t, err)
	assert.Equal(t, StateFailed, p.state)
	assert.Contains(t, p.lastError, "child read error")
}

func TestProxyNew(t *testing.T) {
	mockSup := new(mocks.Supervisor)
	mockSup.On("State").Return(supervisor.StateStopped)
	mockSup.On("PID").Return(0)
	mockSup.On("Uptime").Return(time.Duration(0))
	mockSup.On("LastError").Return(nil)

	mockTransport := new(mocks.Transport)
	mockSanitizer := new(mocks.Sanitizer)
	mockRedactor := new(mocks.Redactor)
	mockStateStore := new(mocks.StateStore)
	mockLogger := new(mocks.Logger)
	mockRec := new(mockRecorder)

	deps := Dependencies{
		Logger:              mockLogger,
		Sanitizer:           mockSanitizer,
		Supervisor:          mockSup,
		Child:               mockTransport,
		Client:              mockTransport,
		Recorder:            mockRec,
		Redactor:            mockRedactor,
		StateStore:          mockStateStore,
		HydraTools:          []injectable.ToolDefinition{{Name: "hydra_test"}},
		MaxRestartsInWindow: func() int { return 3 },
		MaxRestarts:         func() int { return 5 },
	}

	opts := Options{
		QueueSize:          100,
		QueueTTL:           30 * time.Second,
		CollisionPolicy:    "warn",
		HydraTools:         []injectable.ToolDefinition{{Name: "hydra_test"}},
		CrashExportEnabled: true,
		CrashExportPath:    "/tmp/crash.json",
	}

	p := New(deps, opts)
	assert.NotNil(t, p)

	status := p.Status()
	assert.Equal(t, supervisor.StateStopped, status.State)
}

func TestProxyShutdown(t *testing.T) {
	mockSup := new(mocks.Supervisor)
	mockSup.On("Stop").Return(nil).Once()

	mockTransport := new(mocks.Transport)
	mockTransport.On("Close").Return(nil).Once()

	p := &proxy{
		state:      StateRunning,
		supervisor: mockSup,
		child:      mockTransport,
		client:     nil,
		stopCh:     make(chan struct{}),
	}

	err := p.Shutdown()
	assert.NoError(t, err)
	assert.Equal(t, StateStopped, p.state)
	mockSup.AssertExpectations(t)
	mockTransport.AssertExpectations(t)
}

func TestProxyShutdownWithoutSupervisor(t *testing.T) {
	p := &proxy{
		state:  StateRunning,
		stopCh: make(chan struct{}),
	}

	err := p.Shutdown()
	assert.NoError(t, err)
	assert.Equal(t, StateStopped, p.state)
}
