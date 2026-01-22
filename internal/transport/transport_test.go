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

// errorReader simulates read errors
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}
func (e *errorReader) Close() error { return nil }

// errorWriter simulates write errors
type errorWriter struct{}

func (e *errorWriter) Write(p []byte) (n int, err error) {
	return 0, io.ErrShortWrite
}
func (e *errorWriter) Close() error { return io.EOF }

func TestTransport_Close(t *testing.T) {
	mockLogger := mocks.NewLogger(t)

	t.Run("Close success", func(t *testing.T) {
		in := &mockReadCloser{bytes.NewBufferString("")}
		out := &mockWriteCloser{&bytes.Buffer{}}

		tr := transport.NewStdio(in, out, mockLogger)
		err := tr.Close()
		assert.NoError(t, err)
	})

	t.Run("Close with errors", func(t *testing.T) {
		in := &errorReader{}
		out := &errorWriter{}

		tr := transport.NewStdio(in, out, mockLogger)
		err := tr.Close()
		assert.Error(t, err) // Should return in.Close() error
	})
}

func TestTransport_ReadNDJSON_Errors(t *testing.T) {
	mockLogger := mocks.NewLogger(t)

	t.Run("Read error", func(t *testing.T) {
		in := &errorReader{}
		out := &mockWriteCloser{&bytes.Buffer{}}

		tr := transport.NewStdio(in, out, mockLogger)
		mockLogger.On("Info", mock.Anything, mock.Anything).Maybe()

		_, err := tr.Read()
		assert.Error(t, err)
	})

	t.Run("EOF error", func(t *testing.T) {
		in := &mockReadCloser{bytes.NewBufferString("")} // Empty buffer causes EOF
		out := &mockWriteCloser{&bytes.Buffer{}}

		tr := transport.NewStdio(in, out, mockLogger)
		mockLogger.On("Info", mock.Anything, mock.Anything).Maybe()

		_, err := tr.Read()
		assert.Error(t, err)
		assert.Equal(t, io.EOF, err)
	})
}

func TestTransport_ReadLSP_Errors(t *testing.T) {
	mockLogger := mocks.NewLogger(t)

	t.Run("Missing Content-Length header", func(t *testing.T) {
		input := "Content-Length: \r\n\r\n{}"
		in := &mockReadCloser{bytes.NewBufferString(input)}
		out := &mockWriteCloser{&bytes.Buffer{}}

		tr := transport.NewStdio(in, out, mockLogger)
		mockLogger.On("Info", mock.Anything, mock.Anything).Maybe()

		// Detect LSP
		_, _ = tr.DetectProtocol(time.Second)

		// Read should fail with missing/empty content-length
		_, err := tr.Read()
		assert.Error(t, err)
	})

	t.Run("Invalid Content-Length value", func(t *testing.T) {
		input := "Content-Length: invalid\r\n\r\n{}"
		in := &mockReadCloser{bytes.NewBufferString(input)}
		out := &mockWriteCloser{&bytes.Buffer{}}

		tr := transport.NewStdio(in, out, mockLogger)
		mockLogger.On("Info", mock.Anything, mock.Anything).Maybe()

		_, _ = tr.DetectProtocol(time.Second)

		_, err := tr.Read()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid content-length")
	})

	t.Run("Body read error", func(t *testing.T) {
		input := "Content-Length: 100\r\n\r\nshort"
		in := &mockReadCloser{bytes.NewBufferString(input)}
		out := &mockWriteCloser{&bytes.Buffer{}}

		tr := transport.NewStdio(in, out, mockLogger)
		mockLogger.On("Info", mock.Anything, mock.Anything).Maybe()

		_, _ = tr.DetectProtocol(time.Second)

		// Should fail trying to read 100 bytes when only "short" is available
		_, err := tr.Read()
		assert.Error(t, err)
	})
}

func TestTransport_Write_Errors(t *testing.T) {
	mockLogger := mocks.NewLogger(t)

	t.Run("Write error NDJSON", func(t *testing.T) {
		in := &mockReadCloser{bytes.NewBufferString("")}
		out := &errorWriter{}

		tr := transport.NewStdio(in, out, mockLogger)
		mockLogger.On("Info", mock.Anything, mock.Anything).Maybe()

		err := tr.Write([]byte("test"))
		assert.Error(t, err)
	})

	t.Run("Write error LSP", func(t *testing.T) {
		input := "Content-Length: 5\r\n\r\nhello"
		in := &mockReadCloser{bytes.NewBufferString(input)}
		out := &errorWriter{}

		tr := transport.NewStdio(in, out, mockLogger)
		mockLogger.On("Info", mock.Anything, mock.Anything).Maybe()

		// Detect LSP
		_, _ = tr.DetectProtocol(time.Second)

		// Write should fail
		err := tr.Write([]byte("test"))
		assert.Error(t, err)
	})

	t.Run("Write NDJSON with newline already", func(t *testing.T) {
		in := &mockReadCloser{bytes.NewBufferString("{}\n")}
		var out bytes.Buffer
		outCloser := &mockWriteCloser{&out}

		tr := transport.NewStdio(in, outCloser, mockLogger)
		mockLogger.On("Info", mock.Anything, mock.Anything).Maybe()

		// Detect NDJSON
		_, _ = tr.DetectProtocol(time.Second)

		// Write payload that already has newline
		err := tr.Write([]byte("test\n"))
		assert.NoError(t, err)
		assert.Equal(t, "test\n", out.String())
	})
}

func TestTransport_DetectProtocol_Error(t *testing.T) {
	mockLogger := mocks.NewLogger(t)

	t.Run("Peek error", func(t *testing.T) {
		in := &errorReader{}
		out := &mockWriteCloser{&bytes.Buffer{}}

		tr := transport.NewStdio(in, out, mockLogger)

		_, err := tr.DetectProtocol(time.Second)
		assert.Error(t, err)
	})
}
