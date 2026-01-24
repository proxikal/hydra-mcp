package injectable

func (t *toolset) definitionFor(
	name string,
) (ToolDefinition, bool) {
	switch name {
	case "hydra_restart":
		return ToolDefinition{
			Name:        "hydra_restart",
			Description: "Manually restart the supervised MCP server.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"reason": map[string]interface{}{
						"type":        "string",
						"description": "Optional reason for restart",
					},
				},
			},
		}, true
	case "hydra_status":
		return ToolDefinition{
			Name:        "hydra_status",
			Description: "Get current Hydra supervisor status.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		}, true
	case "hydra_logs":
		return ToolDefinition{
			Name:        "hydra_logs",
			Description: "Retrieve recent child stderr logs.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"lines": map[string]interface{}{
						"type":        "number",
						"description": "Number of recent log lines to retrieve",
						"minimum":     1,
						"maximum":     500,
					},
				},
			},
		}, true
	case "hydra_force_restart":
		return ToolDefinition{
			Name:        "hydra_force_restart",
			Description: "Force restart even if in FAILED state.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"confirm": map[string]interface{}{
						"type":        "boolean",
						"description": "Must be true to confirm force restart",
					},
				},
				"required": []string{"confirm"},
			},
		}, true
	default:
		return ToolDefinition{}, false
	}
}
