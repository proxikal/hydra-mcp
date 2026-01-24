package injectable

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateToolNamesAcceptsValidNames(t *testing.T) {
	tools := []string{
		"hydra_restart",
		"hydra_status",
		"hydra_logs",
		"hydra_force_restart",
	}

	err := ValidateToolNames(tools)
	assert.NoError(t, err)
}

func TestValidateToolNamesRejectsInvalidPrefix(t *testing.T) {
	tools := []string{
		"hydra_restart",
		"invalid_tool",
		"hydra_status",
	}

	err := ValidateToolNames(tools)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToolName)
	assert.Contains(t, err.Error(), "invalid_tool")
}

func TestValidateToolNamesRejectsTooShortName(t *testing.T) {
	tools := []string{
		"hydra_restart",
		"short",
	}

	err := ValidateToolNames(tools)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToolName)
}

func TestValidateToolNamesRejectsEmptyPrefix(t *testing.T) {
	tools := []string{
		"hydra_restart",
		"",
	}

	err := ValidateToolNames(tools)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToolName)
}

func TestDetectCollisionFindsNoCollisions(t *testing.T) {
	hydraTools := []ToolDefinition{
		{Name: "hydra_restart"},
		{Name: "hydra_status"},
	}
	childTools := []string{"child_tool", "another_tool"}

	collisions, err := DetectCollision(hydraTools, childTools)
	assert.NoError(t, err)
	assert.Nil(t, collisions)
}

func TestDetectCollisionFindsCollision(t *testing.T) {
	hydraTools := []ToolDefinition{
		{Name: "hydra_restart"},
		{Name: "hydra_status"},
	}
	childTools := []string{"child_tool", "hydra_restart"}

	collisions, err := DetectCollision(hydraTools, childTools)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrNamespaceCollision)
	assert.Equal(t, []string{"hydra_restart"}, collisions)
}

func TestDetectCollisionFindsMultipleCollisions(t *testing.T) {
	hydraTools := []ToolDefinition{
		{Name: "hydra_restart"},
		{Name: "hydra_status"},
		{Name: "hydra_logs"},
	}
	childTools := []string{
		"child_tool",
		"hydra_restart",
		"hydra_logs",
	}

	collisions, err := DetectCollision(hydraTools, childTools)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrNamespaceCollision)
	assert.Len(t, collisions, 2)
	assert.Contains(t, collisions, "hydra_restart")
	assert.Contains(t, collisions, "hydra_logs")
}

func TestDetectCollisionHandlesEmptyChildTools(t *testing.T) {
	hydraTools := []ToolDefinition{
		{Name: "hydra_restart"},
		{Name: "hydra_status"},
	}
	childTools := []string{}

	collisions, err := DetectCollision(hydraTools, childTools)
	assert.NoError(t, err)
	assert.Nil(t, collisions)
}

func TestDetectCollisionHandlesEmptyHydraTools(t *testing.T) {
	hydraTools := []ToolDefinition{}
	childTools := []string{"child_tool", "another_tool"}

	collisions, err := DetectCollision(hydraTools, childTools)
	assert.NoError(t, err)
	assert.Nil(t, collisions)
}
