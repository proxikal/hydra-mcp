package recorder

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/proxikal/hydra/internal/logger"
	"github.com/proxikal/hydra/internal/security"
)

// JSONRPCError represents a JSON-RPC error object.
type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// JSONRPCMessage models a JSON-RPC request or response body.
type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// Options configure the recorder behavior.
type Options struct {
	Enabled               bool
	BufferSize            int
	IncludeRequestBodies  bool
	IncludeResponseBodies bool
	RedactPatterns        []string
}

// Recorder captures JSON-RPC traffic for export.
type Recorder interface {
	RecordRequest(direction string, msg JSONRPCMessage)
	RecordResponse(direction string, msg JSONRPCMessage)
	Export(path string) error
}

type entry struct {
	Direction string
	Type      string
	Message   JSONRPCMessage
	Timestamp time.Time
}

type recorder struct {
	opts     Options
	redactor security.Redactor
	logger   logger.Logger
	buffer   []entry
	maxSize  int
	mu       sync.Mutex
}

// ErrRecorderDisabled indicates recording is turned off.
var ErrRecorderDisabled = errors.New("recorder disabled")

// NewRecorder creates a recorder instance.
func NewRecorder(
	opts Options,
	redactor security.Redactor,
	log logger.Logger,
) Recorder {
	size := opts.BufferSize
	if size <= 0 {
		size = 50
	}

	return &recorder{
		opts:     opts,
		redactor: redactor,
		logger:   log,
		buffer:   make([]entry, 0, size),
		maxSize:  size,
	}
}

func (r *recorder) RecordRequest(direction string, msg JSONRPCMessage) {
	r.record("request", direction, msg)
}

func (r *recorder) RecordResponse(direction string, msg JSONRPCMessage) {
	r.record("response", direction, msg)
}

func (r *recorder) Export(path string) error {
	if !r.opts.Enabled {
		return ErrRecorderDisabled
	}

	entries := r.snapshot()
	exportEntries := make([]entry, len(entries))

	for i, e := range entries {
		exportEntries[i] = entry{
			Direction: e.Direction,
			Type:      e.Type,
			Timestamp: e.Timestamp,
			Message:   r.prepareForExport(e.Type, e.Message),
		}
	}

	payload, err := json.Marshal(exportEntries)
	if err != nil {
		r.logger.Error("recorder marshal failed", err)
		return err
	}

	if err := os.WriteFile(path, payload, 0o644); err != nil {
		r.logger.Error("recorder export failed", err)
		return err
	}

	return nil
}

func (r *recorder) record(kind, direction string, msg JSONRPCMessage) {
	if !r.opts.Enabled {
		return
	}

	clean := r.applyBodyPolicy(kind, msg)

	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.buffer) >= r.maxSize {
		r.buffer = r.buffer[1:]
	}

	r.buffer = append(r.buffer, entry{
		Direction: direction,
		Type:      kind,
		Message:   clean,
		Timestamp: time.Now().UTC(),
	})
}

func (r *recorder) snapshot() []entry {
	r.mu.Lock()
	defer r.mu.Unlock()

	copied := make([]entry, len(r.buffer))
	copy(copied, r.buffer)
	return copied
}

func (r *recorder) applyBodyPolicy(
	kind string,
	msg JSONRPCMessage,
) JSONRPCMessage {
	if kind == "request" && !r.opts.IncludeRequestBodies {
		msg.Params = nil
	}

	if kind == "response" && !r.opts.IncludeResponseBodies {
		msg.Result = nil
		if msg.Error != nil {
			msg.Error.Data = nil
		}
	}

	return msg
}

func (r *recorder) prepareForExport(
	kind string,
	msg JSONRPCMessage,
) JSONRPCMessage {
	msg = r.applyBodyPolicy(kind, msg)

	if r.redactor == nil || len(r.opts.RedactPatterns) == 0 {
		return msg
	}

	msg.Params = r.redactRaw(msg.Params)
	msg.Result = r.redactRaw(msg.Result)

	if msg.Error != nil {
		msg.Error.Message = r.redactor.Redact(
			msg.Error.Message,
			r.opts.RedactPatterns,
		)
		msg.Error.Data = r.redactRaw(msg.Error.Data)
	}

	return msg
}

func (r *recorder) redactRaw(data json.RawMessage) json.RawMessage {
	if data == nil {
		return nil
	}

	redacted := r.redactor.Redact(string(data), r.opts.RedactPatterns)
	return json.RawMessage(redacted)
}
