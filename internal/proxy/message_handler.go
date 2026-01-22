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

// handleClientMessage processes incoming messages from the client.
func (p *proxy) handleClientMessage(payload []byte) {
	p.record("client", payload, true)
	id := extractID(payload)

	switch p.state {
	case StateRunning:
		p.forwardToChild(payload)
	case StateStarting, StateRestarting:
		if err := p.queue.Enqueue(id, payload); err != nil {
			p.sendErrorResponse(id, "Server restarting, queue full")
			p.logger.Warn("restart queue full", map[string]interface{}{"id": id})
		}
	default:
		p.sendErrorResponse(id, "Server not running")
	}
}

// handleChildMessage processes incoming messages from the child server.
func (p *proxy) handleChildMessage(payload []byte) {
	p.record("child", payload, false)

	var msg recorder.JSONRPCMessage
	if err := json.Unmarshal(payload, &msg); err == nil {
		if p.state == StateStarting || p.state == StateRestarting {
			// Treat first result as initialize completion.
			if msg.Result != nil {
				p.handleInitializeResponse()
			}
		}

		if merged, ok := p.tryMergeTools(msg); ok {
			payload = merged
		}

		p.updateStateStore(msg)
	}

	p.forwardToClient(payload)
}
