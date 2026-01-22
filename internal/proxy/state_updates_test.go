package proxy

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/mocks"
	"github.com/proxikal/hydra/internal/recorder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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
