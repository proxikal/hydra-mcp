package proxy

import (
	"encoding/json"
	"testing"

	"github.com/proxikal/hydra/internal/injectable"
	"github.com/proxikal/hydra/internal/recorder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolsListMergedAndForwarded(t *testing.T) {
	child := &stubTransport{}
	client := &stubTransport{}

	p := &proxy{
		state:  StateRunning,
		child:  child,
		client: client,
		hydraTools: []injectable.ToolDefinition{
			{Name: "hydra_restart"},
			{Name: "hydra_status"},
		},
		collisionPolicy: "warn",
	}

	resp := []byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":["child_tool","hydra_restart"]}}`)
	p.handleChildMessage(resp)

	require.Len(t, client.writes, 2)

	var forwarded recorder.JSONRPCMessage
	err := json.Unmarshal(client.writes[1], &forwarded)
	require.NoError(t, err)

	var result struct {
		Tools []map[string]interface{} `json:"tools"`
	}
	require.NoError(t, json.Unmarshal(forwarded.Result, &result))

	// hydra_restart collision removed; hydra_status added
	names := []string{}
	for _, t := range result.Tools {
		names = append(names, t["name"].(string))
	}
	assert.Contains(t, names, "hydra_status")
	assert.NotContains(t, names, "hydra_restart")
	assert.Contains(t, names, "child_tool")

	// tools/list_changed notification sent
	// first write is tools/list_changed notification
	require.Len(t, client.writes, 2)
	var notif map[string]interface{}
	require.NoError(t, json.Unmarshal(client.writes[0], &notif))
	assert.Equal(t, "tools/list_changed", notif["method"])
}

func TestToolsMergeCollisionErrorSendsErrorResponse(t *testing.T) {
	child := &stubTransport{}
	client := &stubTransport{}

	p := &proxy{
		state:  StateRunning,
		child:  child,
		client: client,
		hydraTools: []injectable.ToolDefinition{
			{Name: "hydra_restart"},
		},
		collisionPolicy: "error",
	}

	resp := []byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":["hydra_restart"]}}`)
	p.handleChildMessage(resp)

	require.Len(t, client.writes, 2)
	var errResp map[string]interface{}
	require.NoError(t, json.Unmarshal(client.writes[0], &errResp))
	errObj := errResp["error"].(map[string]interface{})
	assert.Contains(t, errObj["message"], "namespace collision")
}
