package proxy

import (
	"fmt"
	"sort"

	"github.com/proxikal/hydra/internal/injectable"
)

// mergeTools merges child tool names with Hydra tool definitions and handles collisions.
func mergeTools(
	childTools []string,
	hydraTools []injectable.ToolDefinition,
	onCollision string,
) ([]injectable.ToolDefinition, error) {
	switch onCollision {
	case "", "error", "warn", "disable_hydra_tool":
	default:
		return nil, fmt.Errorf("invalid collision policy: %s", onCollision)
	}

	childSet := make(map[string]struct{}, len(childTools))
	for _, name := range childTools {
		childSet[name] = struct{}{}
	}

	merged := make([]injectable.ToolDefinition, 0, len(childTools)+len(hydraTools))

	for _, tool := range hydraTools {
		_, exists := childSet[tool.Name]
		if exists {
			switch onCollision {
			case "error":
				return nil, fmt.Errorf("%w: %s", ErrCollision, tool.Name)
			case "warn":
				// Drop both colliding tools
				delete(childSet, tool.Name)
				continue
			case "disable_hydra_tool":
				// Keep child tool, drop Hydra tool
				continue
			}
		}
		merged = append(merged, tool)
	}

	for _, name := range childTools {
		if _, exists := childSet[name]; !exists {
			continue
		}
		merged = append(merged, injectable.ToolDefinition{Name: name})
	}

	sort.SliceStable(merged, func(i, j int) bool {
		return merged[i].Name < merged[j].Name
	})

	return merged, nil
}
