package proxy

import (
	"encoding/json"

	"github.com/proxikal/hydra/internal/recorder"
)

// routeClientMessage inspects client requests for queueing and statestore updates.
func (p *proxy) routeClientMessage(payload []byte) {
	var msg recorder.JSONRPCMessage
	if err := json.Unmarshal(payload, &msg); err == nil {
		p.updateStateStore(msg)
	}
	p.handleClientMessage(payload)
}
