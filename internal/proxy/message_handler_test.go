package proxy

import (
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/injectable"
	"github.com/proxikal/hydra/internal/mocks"
	"github.com/proxikal/hydra/internal/sanitizer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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
