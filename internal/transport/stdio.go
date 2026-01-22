package transport

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/proxikal/hydra/internal/logger"
)

type stdioTransport struct {
	in       io.ReadCloser
	out      io.WriteCloser
	reader   *bufio.Reader
	protocol Protocol
	logger   logger.Logger
}

// NewStdio creates a transport that wraps standard input/output.
func NewStdio(in io.ReadCloser, out io.WriteCloser, l logger.Logger) Transport {
	return &stdioTransport{
		in:       in,
		out:      out,
		reader:   bufio.NewReader(in),
		protocol: ProtocolUnknown, // Default until detected
		logger:   l,
	}
}

func (t *stdioTransport) DetectProtocol(timeout time.Duration) (Protocol, error) {
	// Simple peek to check for Content-Length header
	// We need to implement a mechanism to peek with timeout, 
	// but for now, we'll assume the caller handles the timeout via context if needed,
	// or we implement a simple peek logic.
	
	// WARNING: bufio.Peek blocks. To support true timeout, we'd need a non-blocking underlying reader
	// or a goroutine. For simplicity in Phase 1, we will implement a blocking peek 
	// and assume the child sends *something* reasonably fast on startup.
	
	bytes, err := t.reader.Peek(14) // "Content-Length" is 14 chars
	if err != nil {
		return ProtocolUnknown, err
	}

	if string(bytes) == "Content-Length" {
		t.protocol = ProtocolLSP
		t.logger.Info("Detected Protocol: LSP")
	} else {
		t.protocol = ProtocolNDJSON
		t.logger.Info("Detected Protocol: NDJSON")
	}

	return t.protocol, nil
}

func (t *stdioTransport) Read() ([]byte, error) {
	if t.protocol == ProtocolLSP {
		return t.readLSP()
	}
	// Default to NDJSON if unknown or explicitly set
	return t.readNDJSON()
}

func (t *stdioTransport) readNDJSON() ([]byte, error) {
	line, err := t.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	return bytes.TrimSpace(line), nil
}

func (t *stdioTransport) readLSP() ([]byte, error) {
	// 1. Read Headers until \r\n\r\n
	var contentLength int
	for {
		line, err := t.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		// Empty line \r\n signifies end of headers
		if line == "\r\n" {
			break
		}
		
		// Parse Content-Length
		if len(line) > 16 && line[:16] == "Content-Length: " {
			val := line[16 : len(line)-2] // trim \r\n
			cl, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid content-length: %w", err)
			}
			contentLength = cl
		}
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("missing content-length header")
	}

	// 2. Read Body
	body := make([]byte, contentLength)
	_, err := io.ReadFull(t.reader, body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (t *stdioTransport) Write(payload []byte) error {
	if t.protocol == ProtocolLSP {
		header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(payload))
		if _, err := t.out.Write([]byte(header)); err != nil {
			return err
		}
	} else {
		// Ensure ends with newline for NDJSON
		if len(payload) == 0 || payload[len(payload)-1] != '\n' {
			payload = append(payload, '\n')
		}
	}
	
	_, err := t.out.Write(payload)
	return err
}

func (t *stdioTransport) Close() error {
	inErr := t.in.Close()
	outErr := t.out.Close()
	if inErr != nil {
		return inErr
	}
	return outErr
}
