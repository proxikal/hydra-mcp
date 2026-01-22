package proxy

import "encoding/json"

// drainQueue replays all queued messages to the child.
func (p *proxy) drainQueue() {
	if p.queue == nil {
		return
	}
	for _, item := range p.queue.Drain() {
		p.forwardToChild(item.Payload)
	}
}

// forwardToChild sends a message to the child server.
func (p *proxy) forwardToChild(payload []byte) {
	if p.child == nil {
		return
	}
	if err := p.child.Write(payload); err != nil && p.logger != nil {
		p.logger.Error("failed to forward to child", err)
		p.lastError = err.Error()
	}
}

// forwardToClient sends a message to the client.
func (p *proxy) forwardToClient(payload []byte) {
	if p.client == nil {
		return
	}

	if err := p.client.Write(payload); err != nil && p.logger != nil {
		p.logger.Error("failed to forward to client", err)
	}
}

// sendErrorResponse sends a JSON-RPC error response to the client.
func (p *proxy) sendErrorResponse(id interface{}, message string) {
	if p.client == nil {
		return
	}

	resp := map[string]interface{}{
		"jsonrpc": "2.0",
		"error": map[string]interface{}{
			"code":    -32000,
			"message": message,
		},
	}
	if id != nil {
		resp["id"] = id
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_ = p.client.Write(data)
}
