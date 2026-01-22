package proxy

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleFileChangeTriggersRestart(t *testing.T) {
	sup := new(mocks.Supervisor)
	sup.On("Restart").Return(nil).Once()

	p := &proxy{
		state:      StateRunning,
		supervisor: sup,
		queue:      NewQueue(2, time.Second),
	}

	p.handleFileChange()

	assert.Equal(t, StateRestarting, p.state)
	sup.AssertExpectations(t)
}

func TestHandleFileChangeFailsTransitionToFailed(t *testing.T) {
	sup := new(mocks.Supervisor)
	sup.On("Restart").Return(assert.AnError).Once()

	p := &proxy{
		state:      StateRunning,
		supervisor: sup,
		queue:      NewQueue(2, time.Second),
	}

	p.handleFileChange()

	assert.Equal(t, StateFailed, p.state)
	assert.Equal(t, assert.AnError.Error(), p.lastError)
	assert.Equal(t, "crash_loop", p.lastRestartReason)
}

func TestHandleInitializeResponseDrainsQueueAndSetsRunning(t *testing.T) {
	queue := NewQueue(2, time.Minute)
	_ = queue.Enqueue("req-1", []byte(`{"id":1}`))

	replayCalled := false

	p := &proxy{
		state:  StateRestarting,
		queue:  queue,
		replay: func() { replayCalled = true },
		child:  &stubTransport{},
	}

	p.handleInitializeResponse()

	assert.True(t, replayCalled)
	assert.Equal(t, StateRunning, p.state)
	assert.Equal(t, 0, queue.Size())
}

func TestReplayInitializeAndSubscriptions(t *testing.T) {
	store := new(mocks.StateStore)
	store.On("GetInitialize").Return(json.RawMessage(`{"foo":"bar"}`)).Once()
	store.On("GetSubscriptions").Return([]string{"file://a", "file://b"}).Once()

	child := &stubTransport{}

	p := &proxy{
		state:      StateRestarting,
		stateStore: store,
		child:      child,
		queue:      NewQueue(1, time.Minute),
	}

	p.handleInitializeResponse()

	require.GreaterOrEqual(t, len(child.writes), 2)
	first := string(child.writes[0])
	assert.Contains(t, first, `"method":"initialize"`)
	assert.Contains(t, first, `"foo":"bar"`)

	containsSub := false
	for _, w := range child.writes {
		if bytes.Contains(w, []byte(`"resources/subscribe"`)) {
			containsSub = true
		}
	}
	assert.True(t, containsSub, "expected subscription replay")
}
