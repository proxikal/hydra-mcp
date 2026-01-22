package proxy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestQueueFIFOAndCapacity(t *testing.T) {
	queue := NewQueue(2, time.Minute)

	err := queue.Enqueue("req-1", []byte(`{"id":1}`))
	assert.NoError(t, err)
	err = queue.Enqueue("req-2", []byte(`{"id":2}`))
	assert.NoError(t, err)

	err = queue.Enqueue("req-3", []byte(`{"id":3}`))
	assert.ErrorIs(t, err, ErrQueueFull)

	items := queue.Drain()
	assert.Len(t, items, 2)
	assert.Equal(t, "req-1", items[0].ID)
	assert.Equal(t, "req-2", items[1].ID)
}

func TestQueueDropsExpired(t *testing.T) {
	queue := NewQueue(5, 10*time.Millisecond)

	_ = queue.Enqueue("req-1", []byte(`{"id":1}`))
	time.Sleep(15 * time.Millisecond)

	items := queue.Drain()
	assert.Empty(t, items)
}
