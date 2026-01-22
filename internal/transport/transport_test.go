package transport_test

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/mocks"
	"github.com/proxikal/hydra/internal/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockReadCloser helps us simulate reading
type mockReadCloser struct {
	io.Reader
}

func (m *mockReadCloser) Close() error { return nil }

// mockWriteCloser helps us simulate writing
type mockWriteCloser struct {
	io.Writer
}

func (m *mockWriteCloser) Close() error { return nil }

func TestTransport_NDJSON(t *testing.T) {
	input := "{\"jsonrpc\":\"2.0\"}\n"
	in := &mockReadCloser{bytes.NewBufferString(input)}
	var out bytes.Buffer
	outCloser := &mockWriteCloser{&out}
	
mockLogger := mocks.NewLogger(t)
	// We might log detection
	mockLogger.On("Info", mock.Anything, mock.Anything).Maybe()

	tr := transport.NewStdio(in, outCloser, mockLogger)

	// Detection
	p, err := tr.DetectProtocol(time.Second)
	assert.NoError(t, err)
	assert.Equal(t, transport.ProtocolNDJSON, p)

	// Read
	msg, err := tr.Read()
	assert.NoError(t, err)
	assert.Equal(t, "{\"jsonrpc\":\"2.0\"}", string(msg))

	// Write
	err = tr.Write([]byte("response"))
	assert.NoError(t, err)
	assert.Equal(t, "response\n", out.String())
}

func TestTransport_LSP(t *testing.T) {
	input := "Content-Length: 17\r\n\r\n{\"jsonrpc\":\"2.0\"}"
	in := &mockReadCloser{bytes.NewBufferString(input)}
	var out bytes.Buffer
	outCloser := &mockWriteCloser{&out}
	
mockLogger := mocks.NewLogger(t)
	mockLogger.On("Info", mock.Anything, mock.Anything).Maybe()

	tr := transport.NewStdio(in, outCloser, mockLogger)

	// Detection
	p, err := tr.DetectProtocol(time.Second)
	assert.NoError(t, err)
	assert.Equal(t, transport.ProtocolLSP, p)

	// Read
	msg, err := tr.Read()
	assert.NoError(t, err)
	assert.Equal(t, "{\"jsonrpc\":\"2.0\"}", string(msg))

	// Write
	out.Reset()
	err = tr.Write([]byte("12345"))
	assert.NoError(t, err)
	assert.Equal(t, "Content-Length: 5\r\n\r\n12345", out.String())
}