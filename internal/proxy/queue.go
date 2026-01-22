package proxy

import (
	"errors"
	"time"
)

// ErrQueueFull is returned when the queue is at capacity.
var ErrQueueFull = errors.New("restart queue full")

// queuedRequest represents a buffered client request.
type queuedRequest struct {
	ID       interface{}
	Payload  []byte
	Enqueued time.Time
}

// Queue buffers requests during restarts.
type Queue interface {
	Enqueue(id interface{}, payload []byte) error
	Drain() []queuedRequest
	Size() int
}

type restartQueue struct {
	max int
	ttl time.Duration
	buf []queuedRequest
}

// NewQueue creates a restart queue with capacity and TTL.
func NewQueue(max int, ttl time.Duration) Queue {
	if max <= 0 {
		max = 1
	}
	return &restartQueue{
		max: max,
		ttl: ttl,
		buf: make([]queuedRequest, 0, max),
	}
}

func (q *restartQueue) Enqueue(id interface{}, payload []byte) error {
	if len(q.buf) >= q.max {
		return ErrQueueFull
	}

	q.buf = append(q.buf, queuedRequest{
		ID:       id,
		Payload:  payload,
		Enqueued: time.Now(),
	})
	return nil
}

func (q *restartQueue) Drain() []queuedRequest {
	now := time.Now()
	valid := make([]queuedRequest, 0, len(q.buf))
	for _, item := range q.buf {
		if q.ttl > 0 && now.Sub(item.Enqueued) > q.ttl {
			continue
		}
		valid = append(valid, item)
	}
	q.buf = q.buf[:0]
	return valid
}

func (q *restartQueue) Size() int {
	return len(q.buf)
}
