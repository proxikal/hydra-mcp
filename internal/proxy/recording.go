package proxy

import (
	"encoding/json"

	"github.com/proxikal/hydra/internal/recorder"
)

// record logs a JSON-RPC message to the recorder.
func (p *proxy) record(direction string, payload []byte, isRequest bool) {
	if p.recorder == nil {
		return
	}

	var msg recorder.JSONRPCMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}

	if isRequest {
		p.recorder.RecordRequest(direction, msg)
		return
	}
	p.recorder.RecordResponse(direction, msg)
}

// extractID extracts the ID field from a JSON-RPC message payload.
func extractID(payload []byte) interface{} {
	var obj map[string]interface{}
	if err := json.Unmarshal(payload, &obj); err != nil {
		return nil
	}
	if id, ok := obj["id"]; ok {
		return id
	}
	return nil
}
