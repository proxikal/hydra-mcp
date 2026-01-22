package injectable

import (
	"fmt"
	"time"

	"github.com/proxikal/hydra/internal/supervisor"
)

func (t *toolset) handleRestart(
	params map[string]interface{},
) (interface{}, error) {
	if t.supervisor == nil {
		return nil, fmt.Errorf("supervisor not configured")
	}

	if state := t.supervisor.State(); state == supervisor.StateFailed {
		return nil, fmt.Errorf(
			"cannot restart: server in FAILED state (use hydra_force_restart)",
		)
	}

	reason, _ := params["reason"].(string)
	if t.logger != nil && reason != "" {
		t.logger.Info("manual restart requested", map[string]interface{}{
			"reason": reason,
		})
	}

	if err := t.supervisor.Restart(); err != nil {
		if t.logger != nil {
			t.logger.Error("restart failed", err)
		}
		return nil, fmt.Errorf("restart failed: %w", err)
	}

	if err := t.waitForState(supervisor.StateRunning); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": "Server restarted successfully",
		"state":   "RUNNING",
	}, nil
}

func (t *toolset) handleForceRestart(
	params map[string]interface{},
) (interface{}, error) {
	if t.supervisor == nil {
		return nil, fmt.Errorf("supervisor not configured")
	}

	confirm, ok := params["confirm"].(bool)
	if !ok || !confirm {
		return nil, fmt.Errorf("confirm must be true")
	}

	t.supervisor.ResetRestartCounter()

	if err := t.supervisor.Restart(); err != nil {
		if t.logger != nil {
			t.logger.Error("force restart failed", err)
		}
		return nil, fmt.Errorf("force restart failed: %w", err)
	}

	if err := t.waitForState(supervisor.StateRunning); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": "Server force-restarted successfully",
		"state":   "RUNNING",
	}, nil
}

func (t *toolset) waitForState(
	target supervisor.ServerState,
) error {
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			state := t.supervisor.State()
			if state == target {
				return nil
			}

			if state == supervisor.StateFailed {
				return fmt.Errorf("restart failed: still in FAILED state")
			}
		case <-timeout:
			return fmt.Errorf("restart timeout")
		}
	}
}
