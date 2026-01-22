package proxy

import (
	"io"
	"testing"

	"github.com/proxikal/hydra/internal/mocks"
	"github.com/stretchr/testify/mock"
)

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
