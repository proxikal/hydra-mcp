package transport

import (
	"time"
)

// Protocol defines the message framing type detected on the connection.
type Protocol int

const (
	ProtocolUnknown Protocol = iota
	ProtocolNDJSON           // Newline Delimited JSON
	ProtocolLSP              // Content-Length: <len>\r\n\r\n
)

// Transport manages the low-level read/write operations with the child process.
// It handles protocol detection (LSP vs NDJSON) and safe reading.
type Transport interface {
	// Read returns the next complete message payload.
	// It blocks until a message is available or the connection closes.
	Read() ([]byte, error)

	// Write sends a payload to the child process.
	// It automatically adds framing based on the detected protocol.
	Write(payload []byte) error

	// Close shuts down the transport and releases resources.
	Close() error

	// DetectProtocol waits for the first message to sniff the protocol.
	// If timeout is reached, it returns an error.
	DetectProtocol(timeout time.Duration) (Protocol, error)
}

