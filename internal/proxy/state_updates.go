package proxy

import (
	"encoding/json"

	"github.com/proxikal/hydra/internal/recorder"
)

// updateStateStore updates the state store with relevant information from messages.
func (p *proxy) updateStateStore(msg recorder.JSONRPCMessage) {
	if p.stateStore == nil {
		return
	}

	if msg.Method == "initialize" && msg.Params != nil {
		p.stateStore.SetInitialize(msg.Params)
		return
	}

	if msg.Method == "resources/subscribe" && msg.Params != nil {
		var params struct {
			URI string `json:"uri"`
		}
		if err := json.Unmarshal(msg.Params, &params); err == nil && params.URI != "" {
			p.stateStore.AddSubscription(params.URI)
		}
		return
	}

	if msg.Method == "resources/unsubscribe" && msg.Params != nil {
		var params struct {
			URI string `json:"uri"`
		}
		if err := json.Unmarshal(msg.Params, &params); err == nil && params.URI != "" {
			p.stateStore.RemoveSubscription(params.URI)
		}
		return
	}
}
