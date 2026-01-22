package proxy

import (
	"encoding/json"
	"fmt"
)

// handleFileChange transitions proxy to restarting and triggers supervisor restart.
func (p *proxy) handleFileChange() {
	if p.state != StateRunning {
		return
	}

	p.state = StateRestarting
	p.lastRestartReason = "file_change"

	if p.supervisor == nil {
		return
	}

	if err := p.supervisor.Restart(); err != nil {
		p.lastRestartReason = "crash_loop"
		p.crashLoop(err)
	}
}

// handleInitializeResponse marks the proxy as running and drains any queued requests.
func (p *proxy) handleInitializeResponse() {
	if p.state != StateStarting && p.state != StateRestarting {
		return
	}

	p.replayInitialize()
	if p.replay != nil {
		p.replay()
	}

	p.replaySubscriptions()
	p.drainQueue()
	p.state = StateRunning
}

func (p *proxy) replayInitialize() {
	if p.stateStore == nil || p.child == nil {
		return
	}

	params := p.stateStore.GetInitialize()
	if params == nil {
		return
	}

	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      "reinit",
		"method":  "initialize",
		"params":  json.RawMessage(params),
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return
	}
	_ = p.child.Write(payload)
}

func (p *proxy) replaySubscriptions() {
	if p.stateStore == nil || p.child == nil {
		return
	}

	subs := p.stateStore.GetSubscriptions()
	for _, uri := range subs {
		req := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      fmt.Sprintf("resub-%s", uri),
			"method":  "resources/subscribe",
			"params": map[string]interface{}{
				"uri": uri,
			},
		}
		payload, err := json.Marshal(req)
		if err != nil {
			continue
		}
		_ = p.child.Write(payload)
	}
}
