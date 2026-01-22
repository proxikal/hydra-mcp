package injectable

import (
	"fmt"
)

func (t *toolset) handleStatus() (interface{}, error) {
	if t.status == nil {
		return nil, fmt.Errorf("status provider not configured")
	}

	snapshot := t.status()

	return map[string]interface{}{
		"state":                  snapshot.State.String(),
		"server_name":            t.opts.Server,
		"pid":                    snapshot.PID,
		"uptime_seconds":         int(snapshot.Uptime.Seconds()),
		"restarts_in_window":     snapshot.RestartsInWindow,
		"max_restarts":           t.opts.Behavior.MaxRestarts,
		"restart_window_seconds": t.opts.Behavior.RestartWindowSeconds,
		"last_restart_reason":    snapshot.LastRestartReason,
		"last_error":             snapshot.LastError,
		"queue_size":             snapshot.QueueSize,
		"can_recover":            snapshot.CanRecover,
	}, nil
}
