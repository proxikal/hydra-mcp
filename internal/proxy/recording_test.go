package proxy

import (
	"testing"

	"github.com/stretchr/testify/mock"
)

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
