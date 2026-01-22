package proxy

import (
	"testing"

	"github.com/proxikal/hydra/internal/injectable"
	"github.com/stretchr/testify/assert"
)

func TestMergeToolsDetectsCollisionAndErrors(t *testing.T) {
	hydraTools := []injectable.ToolDefinition{
		{Name: "hydra_restart"},
		{Name: "hydra_status"},
	}

	_, err := mergeTools(
		[]string{"hydra_restart", "child_tool"},
		hydraTools,
		"error",
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hydra_restart")
}

func TestMergeToolsWarnAndDisableOnCollision(t *testing.T) {
	hydraTools := []injectable.ToolDefinition{
		{Name: "hydra_restart"},
		{Name: "hydra_status"},
	}

	merged, err := mergeTools(
		[]string{"hydra_restart", "child_tool"},
		hydraTools,
		"warn",
	)

	assert.NoError(t, err)
	assert.Len(t, merged, 2)
	assert.Equal(t, "child_tool", merged[0].Name)
	assert.Equal(t, "hydra_status", merged[1].Name)
}

func TestMergeToolsNoCollisionPassThrough(t *testing.T) {
	hydraTools := []injectable.ToolDefinition{
		{Name: "hydra_restart"},
		{Name: "hydra_status"},
	}

	merged, err := mergeTools(
		[]string{"child_tool"},
		hydraTools,
		"error",
	)

	assert.NoError(t, err)
	assert.Len(t, merged, 3)
	assert.Equal(t, "child_tool", merged[0].Name)
}
