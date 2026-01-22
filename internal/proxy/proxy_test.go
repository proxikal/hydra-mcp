package proxy

import (
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/injectable"
	"github.com/proxikal/hydra/internal/mocks"
	"github.com/proxikal/hydra/internal/recorder"
	"github.com/proxikal/hydra/internal/sanitizer"
	"github.com/proxikal/hydra/internal/supervisor"
	"github.com/proxikal/hydra/internal/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type stubTransport struct {
	writes [][]byte
	readCh chan []byte
	err    error
}

type mockRecorder struct {
	mock.Mock
}

func (m *mockRecorder) RecordRequest(direction string, msg recorder.JSONRPCMessage) {
	m.Called(direction, msg)
}

func (m *mockRecorder) RecordResponse(direction string, msg recorder.JSONRPCMessage) {
	m.Called(direction, msg)
}

func (m *mockRecorder) Export(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (s *stubTransport) Read() ([]byte, error) {
	if s.err != nil {
		err := s.err
		s.err = nil
		return nil, err
	}
	if s.readCh == nil {
		return nil, io.EOF
	}
	msg, ok := <-s.readCh
	if !ok {
		return nil, io.EOF
	}
	return msg, nil
}
func (s *stubTransport) Write(p []byte) error {
	s.writes = append(s.writes, append([]byte{}, p...))
	return nil
}
func (s *stubTransport) Close() error { return nil }
func (s *stubTransport) DetectProtocol(time.Duration) (transport.Protocol, error) {
	return transport.ProtocolNDJSON, nil
}

type stubLogger struct{}

func (stubLogger) Debug(string, ...map[string]interface{}) {}
func (stubLogger) Info(string, ...map[string]interface{})  {}
func (stubLogger) Warn(string, ...map[string]interface{})  {}
func (stubLogger) Error(string, error, ...map[string]interface{}) {
}

func TestProxyQueuesDuringStartingAndDrainsAfterRunning(t *testing.T) {
	child := &stubTransport{}
	client := &stubTransport{}
	queue := NewQueue(2, time.Minute)

	p := &proxy{
		state:   StateStarting,
		queue:   queue,
		child:   child,
		client:  client,
		logger:  stubLogger{},
		started: time.Now(),
	}

	p.handleClientMessage([]byte(`{"id":1}`))
	p.handleClientMessage([]byte(`{"id":2}`))

	assert.Equal(t, 2, queue.Size())

	p.state = StateRunning
	p.drainQueue()

	assert.Len(t, child.writes, 2)

	var msg1 map[string]interface{}
	_ = json.Unmarshal(child.writes[0], &msg1)
	assert.Equal(t, float64(1), msg1["id"])
}

func TestProxyQueueOverflowRespondsWithError(t *testing.T) {
	child := &stubTransport{}
	client := &stubTransport{}
	queue := NewQueue(1, time.Minute)

	p := &proxy{
		state:   StateRestarting,
		queue:   queue,
		child:   child,
		client:  client,
		logger:  stubLogger{},
		started: time.Now(),
	}

	p.handleClientMessage([]byte(`{"id":1}`))
	p.handleClientMessage([]byte(`{"id":2}`))

	assert.Equal(t, 1, queue.Size())
	assert.Len(t, client.writes, 1)
	assert.Contains(t, string(client.writes[0]), "queue full")
}

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

func TestHandleInitializeResponseCallsReplayBeforeDrain(t *testing.T) {
	calls := []string{}
	queue := NewQueue(2, time.Minute)
	_ = queue.Enqueue("req-1", []byte(`{"id":1}`))

	p := &proxy{
		state: StateRestarting,
		queue: queue,
		replay: func() {
			calls = append(calls, "replay")
		},
		child: &stubTransport{},
	}

	p.handleInitializeResponse()

	assert.Equal(t, []string{"replay"}, calls)
	assert.Equal(t, 0, queue.Size())
	assert.Equal(t, StateRunning, p.state)
}

func TestChildInitializeDrainsQueueAndForwards(t *testing.T) {
	child := &stubTransport{}
	client := &stubTransport{}
	queue := NewQueue(2, time.Minute)
	_ = queue.Enqueue("req-1", []byte(`{"id":1}`))
	_ = queue.Enqueue("req-2", []byte(`{"id":2}`))

	p := &proxy{
		state:  StateStarting,
		queue:  queue,
		child:  child,
		client: client,
		hydraTools: []injectable.ToolDefinition{
			{Name: "hydra_restart"},
		},
		collisionPolicy: "error",
	}

	initResp := []byte(`{"jsonrpc":"2.0","id":0,"result":{"ok":true}}`)
	p.handleChildMessage(initResp)

	assert.Equal(t, StateRunning, p.state)
	assert.Len(t, child.writes, 2)
	assert.Equal(t, 0, queue.Size())
	assert.Len(t, client.writes, 1)
}

func TestSanitizerDropsInvalidChunks(t *testing.T) {
	readCh := make(chan []byte, 1)
	child := &stubTransport{readCh: readCh}
	client := &stubTransport{}
	san := new(mocks.Sanitizer)
	san.On("Classify", mock.Anything).Return(sanitizer.ChunkPollution)

	p := &proxy{
		state:     StateRunning,
		child:     child,
		client:    client,
		sanitizer: san,
		stopCh:    make(chan struct{}),
	}

	readCh <- []byte("garbage")
	close(readCh)

	_ = p.loop()
	assert.Empty(t, client.writes)
}

func TestToolsListMergedAndForwarded(t *testing.T) {
	child := &stubTransport{}
	client := &stubTransport{}

	p := &proxy{
		state:  StateRunning,
		child:  child,
		client: client,
		hydraTools: []injectable.ToolDefinition{
			{Name: "hydra_restart"},
			{Name: "hydra_status"},
		},
		collisionPolicy: "warn",
	}

	resp := []byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":["child_tool","hydra_restart"]}}`)
	p.handleChildMessage(resp)

	require.Len(t, client.writes, 2)

	var forwarded recorder.JSONRPCMessage
	err := json.Unmarshal(client.writes[1], &forwarded)
	require.NoError(t, err)

	var result struct {
		Tools []map[string]interface{} `json:"tools"`
	}
	require.NoError(t, json.Unmarshal(forwarded.Result, &result))

	// hydra_restart collision removed; hydra_status added
	names := []string{}
	for _, t := range result.Tools {
		names = append(names, t["name"].(string))
	}
	assert.Contains(t, names, "hydra_status")
	assert.NotContains(t, names, "hydra_restart")
	assert.Contains(t, names, "child_tool")

	// tools/list_changed notification sent
	// first write is tools/list_changed notification
	require.Len(t, client.writes, 2)
	var notif map[string]interface{}
	require.NoError(t, json.Unmarshal(client.writes[0], &notif))
	assert.Equal(t, "tools/list_changed", notif["method"])
}

func TestToolsMergeCollisionErrorSendsErrorResponse(t *testing.T) {
	child := &stubTransport{}
	client := &stubTransport{}

	p := &proxy{
		state:  StateRunning,
		child:  child,
		client: client,
		hydraTools: []injectable.ToolDefinition{
			{Name: "hydra_restart"},
		},
		collisionPolicy: "error",
	}

	resp := []byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":["hydra_restart"]}}`)
	p.handleChildMessage(resp)

	require.Len(t, client.writes, 2)
	var errResp map[string]interface{}
	require.NoError(t, json.Unmarshal(client.writes[0], &errResp))
	errObj := errResp["error"].(map[string]interface{})
	assert.Contains(t, errObj["message"], "namespace collision")
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

func TestProxyUpdateStateStore(t *testing.T) {
	t.Run("stores initialize params", func(t *testing.T) {
		mockStateStore := new(mocks.StateStore)
		mockStateStore.On("SetInitialize", mock.Anything).Once()

		p := &proxy{
			stateStore: mockStateStore,
		}

		msg := recorder.JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "initialize",
			Params:  json.RawMessage(`{"version":"1.0"}`),
		}

		p.updateStateStore(msg)
		mockStateStore.AssertExpectations(t)
	})

	t.Run("adds subscription on resources/subscribe", func(t *testing.T) {
		mockStateStore := new(mocks.StateStore)
		mockStateStore.On("AddSubscription", "file:///test").Once()

		p := &proxy{
			stateStore: mockStateStore,
		}

		msg := recorder.JSONRPCMessage{
			JSONRPC: "2.0",
			Method:  "resources/subscribe",
			Params:  json.RawMessage(`{"uri":"file:///test"}`),
		}

		p.updateStateStore(msg)
		mockStateStore.AssertExpectations(t)
	})

	t.Run("removes subscription on resources/unsubscribe", func(t *testing.T) {
		mockStateStore := new(mocks.StateStore)
		mockStateStore.On("RemoveSubscription", "file:///test").Once()

		p := &proxy{
			stateStore: mockStateStore,
		}

		msg := recorder.JSONRPCMessage{
			JSONRPC: "2.0",
			Method:  "resources/unsubscribe",
			Params:  json.RawMessage(`{"uri":"file:///test"}`),
		}

		p.updateStateStore(msg)
		mockStateStore.AssertExpectations(t)
	})
}

func TestProxyRecord(t *testing.T) {
	mockRec := new(mockRecorder)

	t.Run("records request when recorder exists", func(t *testing.T) {
		mockRec.On("RecordRequest", "client->child", mock.Anything).Once()

		p := &proxy{
			recorder: mockRec,
		}

		payload := []byte(`{"jsonrpc":"2.0","id":1,"method":"test"}`)
		p.record("client->child", payload, true)
		mockRec.AssertExpectations(t)
	})

	t.Run("records response when recorder exists", func(t *testing.T) {
		mockRec.On("RecordResponse", "child->client", mock.Anything).Once()

		p := &proxy{
			recorder: mockRec,
		}

		payload := []byte(`{"jsonrpc":"2.0","id":1,"result":"ok"}`)
		p.record("child->client", payload, false)
		mockRec.AssertExpectations(t)
	})

	t.Run("handles nil recorder", func(t *testing.T) {
		p := &proxy{
			recorder: nil,
		}

		payload := []byte(`{"jsonrpc":"2.0","id":1,"method":"test"}`)
		// Should not panic
		p.record("client->child", payload, true)
	})

	t.Run("handles invalid JSON", func(t *testing.T) {
		p := &proxy{
			recorder: mockRec,
		}

		payload := []byte(`invalid json`)
		// Should not panic
		p.record("client->child", payload, true)
	})
}

func TestProxyForwardToChild(t *testing.T) {
	t.Run("forwards when child exists", func(t *testing.T) {
		mockTransport := new(mocks.Transport)
		mockTransport.On("Write", mock.Anything).Return(nil).Once()

		p := &proxy{
			child: mockTransport,
		}

		payload := []byte(`{"jsonrpc":"2.0","id":1,"method":"test"}`)
		p.forwardToChild(payload)
		mockTransport.AssertExpectations(t)
	})

	t.Run("handles nil child", func(t *testing.T) {
		mockLogger := new(mocks.Logger)
		mockLogger.On("Error", mock.Anything, mock.Anything).Maybe()

		p := &proxy{
			child:  nil,
			logger: mockLogger,
		}

		payload := []byte(`{"jsonrpc":"2.0","id":1,"method":"test"}`)
		// Should not panic
		p.forwardToChild(payload)
	})

	t.Run("handles write error", func(t *testing.T) {
		mockTransport := new(mocks.Transport)
		mockTransport.On("Write", mock.Anything).Return(io.ErrClosedPipe).Once()

		mockLogger := new(mocks.Logger)
		mockLogger.On("Error", mock.Anything, mock.Anything).Once()

		p := &proxy{
			child:  mockTransport,
			logger: mockLogger,
		}

		payload := []byte(`{"jsonrpc":"2.0","id":1,"method":"test"}`)
		p.forwardToChild(payload)
		mockTransport.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})
}

func TestProxyForwardToClient(t *testing.T) {
	t.Run("forwards when client exists", func(t *testing.T) {
		mockTransport := new(mocks.Transport)
		mockTransport.On("Write", mock.Anything).Return(nil).Once()

		p := &proxy{
			client: mockTransport,
		}

		payload := []byte(`{"jsonrpc":"2.0","id":1,"result":"ok"}`)
		p.forwardToClient(payload)
		mockTransport.AssertExpectations(t)
	})

	t.Run("handles nil client", func(t *testing.T) {
		mockLogger := new(mocks.Logger)
		mockLogger.On("Error", mock.Anything, mock.Anything).Maybe()

		p := &proxy{
			client: nil,
			logger: mockLogger,
		}

		payload := []byte(`{"jsonrpc":"2.0","id":1,"result":"ok"}`)
		// Should not panic
		p.forwardToClient(payload)
	})

	t.Run("handles write error", func(t *testing.T) {
		mockTransport := new(mocks.Transport)
		mockTransport.On("Write", mock.Anything).Return(io.ErrClosedPipe).Once()

		mockLogger := new(mocks.Logger)
		mockLogger.On("Error", mock.Anything, mock.Anything).Once()

		p := &proxy{
			client: mockTransport,
			logger: mockLogger,
		}

		payload := []byte(`{"jsonrpc":"2.0","id":1,"result":"ok"}`)
		p.forwardToClient(payload)
		mockTransport.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})
}
