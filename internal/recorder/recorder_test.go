package recorder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type stubLogger struct{}

func (stubLogger) Debug(string, ...map[string]interface{}) {}
func (stubLogger) Info(string, ...map[string]interface{})  {}
func (stubLogger) Warn(string, ...map[string]interface{})  {}
func (stubLogger) Error(string, error, ...map[string]interface{}) {
}

type stubRedactor struct{}

func (stubRedactor) Redact(content string, _ []string) string {
	return strings.ReplaceAll(content, "secret", "[REDACTED]")
}

func TestRecorderStoresRequestsAndResponses(t *testing.T) {
	rec := NewRecorder(
		Options{
			Enabled:               true,
			BufferSize:            2,
			IncludeRequestBodies:  true,
			IncludeResponseBodies: true,
		},
		stubRedactor{},
		stubLogger{},
	)

	req := JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  "initialize",
		ID:      1,
		Params:  json.RawMessage(`{"a":1}`),
	}
	resp := JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Result:  json.RawMessage(`{"ok":true}`),
	}

	rec.RecordRequest("client", req)
	rec.RecordResponse("server", resp)

	r := rec.(*recorder)
	assert.Len(t, r.buffer, 2)
	assert.Equal(t, "client", r.buffer[0].Direction)
	assert.Equal(t, "request", r.buffer[0].Type)
	assert.Equal(t, req.Method, r.buffer[0].Message.Method)
	assert.Equal(t, "response", r.buffer[1].Type)
}

func TestRecorderKeepsLastEntriesWhenBufferFull(t *testing.T) {
	rec := NewRecorder(
		Options{
			Enabled:    true,
			BufferSize: 2,
		},
		stubRedactor{},
		stubLogger{},
	)

	for i := 0; i < 3; i++ {
		payload := fmt.Sprintf(`{"seq":%d}`, i)
		msg := JSONRPCMessage{
			JSONRPC: "2.0",
			Method:  "call",
			ID:      i,
			Params:  json.RawMessage(payload),
		}
		rec.RecordRequest("client", msg)
	}

	r := rec.(*recorder)
	assert.Len(t, r.buffer, 2)
	assert.Equal(t, 1, r.buffer[0].Message.ID)
	assert.Equal(t, 2, r.buffer[1].Message.ID)
}

func TestRecorderExportRedactsAndHonorsBodyFlags(t *testing.T) {
	dir := t.TempDir()
	exportPath := filepath.Join(dir, "recorder.json")

	rec := NewRecorder(
		Options{
			Enabled:               true,
			BufferSize:            5,
			IncludeRequestBodies:  false,
			IncludeResponseBodies: true,
			RedactPatterns:        []string{"secret"},
		},
		stubRedactor{},
		stubLogger{},
	)

	rec.RecordRequest("client", JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  "tools/call",
		ID:      "req-1",
		Params:  json.RawMessage(`{"token":"secret"}`),
	})

	rec.RecordResponse("server", JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      "req-1",
		Result:  json.RawMessage(`{"result":"secret"}`),
	})

	err := rec.Export(exportPath)
	assert.NoError(t, err)

	data, err := os.ReadFile(exportPath)
	assert.NoError(t, err)

	type exportedEntry struct {
		Direction string         `json:"direction"`
		Type      string         `json:"type"`
		Message   JSONRPCMessage `json:"message"`
	}

	var entries []exportedEntry
	assert.NoError(t, json.Unmarshal(data, &entries))
	assert.Len(t, entries, 2)

	// Request body omitted
	assert.Nil(t, entries[0].Message.Params)

	// Response body redacted
	assert.Contains(t, string(entries[1].Message.Result), "[REDACTED]")
}
