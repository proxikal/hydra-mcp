package proxy

import (
	"encoding/json"
	"errors"

	"github.com/proxikal/hydra/internal/injectable"
	"github.com/proxikal/hydra/internal/recorder"
)

// tryMergeTools attempts to merge Hydra tools into the child's tool list.
func (p *proxy) tryMergeTools(msg recorder.JSONRPCMessage) ([]byte, bool) {
	if msg.Result == nil || len(p.hydraTools) == 0 {
		return nil, false
	}

	var result map[string]interface{}
	if err := json.Unmarshal(msg.Result, &result); err != nil {
		return nil, false
	}

	rawTools, ok := result["tools"]
	if !ok {
		return nil, false
	}

	toolList, ok := rawTools.([]interface{})
	if !ok {
		return nil, false
	}

	childNames := make([]string, 0, len(toolList))
	for _, t := range toolList {
		switch v := t.(type) {
		case string:
			childNames = append(childNames, v)
		case map[string]interface{}:
			if name, ok := v["name"].(string); ok {
				childNames = append(childNames, name)
			}
		}
	}

	merged, err := mergeTools(childNames, p.hydraTools, p.collisionPolicy)
	if err != nil {
		if p.logger != nil {
			p.logger.Error("tool merge failed", err)
		}
		p.lastError = err.Error()
		p.sendErrorResponse(msg.ID, err.Error())
		if errors.Is(err, ErrCollision) {
			p.state = StateFailed
		}
		return nil, false
	}

	result["tools"] = merged
	p.notifyToolsChanged(merged)
	msg.Result, _ = json.Marshal(result)

	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, false
	}

	return payload, true
}

// notifyToolsChanged sends a tools/list_changed notification to the client.
func (p *proxy) notifyToolsChanged(tools []injectable.ToolDefinition) {
	if p.client == nil {
		return
	}

	names := make([]string, 0, len(tools))
	for _, t := range tools {
		names = append(names, t.Name)
	}

	notification := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "tools/list_changed",
		"params": map[string]interface{}{
			"tools": names,
		},
	}

	payload, err := json.Marshal(notification)
	if err != nil {
		return
	}
	_ = p.client.Write(payload)
}
