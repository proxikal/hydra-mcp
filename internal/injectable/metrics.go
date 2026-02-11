package injectable

import "errors"

// handleMetrics returns comprehensive proxy metrics.
func (t *toolset) handleMetrics() (interface{}, error) {
	if t.metrics == nil {
		return nil, errors.New("metrics not available")
	}

	snapshot := t.metrics.Snapshot()
	breakdown := t.metrics.HealthBreakdown()

	// Convert health breakdown to map
	healthComponents := map[string]int{
		"uptime_stability":  breakdown.UptimeStability,
		"error_rate":        breakdown.ErrorRate,
		"response_latency":  breakdown.ResponseLatency,
		"queue_depth":       breakdown.QueueDepth,
		"restart_frequency": breakdown.RestartFrequency,
	}

	return map[string]interface{}{
		"requests_total":    snapshot.RequestsTotal,
		"requests_success":  snapshot.RequestsSuccess,
		"requests_failed":   snapshot.RequestsFailed,
		"restarts_total":    snapshot.RestartsTotal,
		"queue_waits_total": snapshot.QueueWaitsTotal,
		"avg_latency_ms":    snapshot.AvgLatencyMs,
		"p50_latency_ms":    snapshot.P50LatencyMs,
		"p95_latency_ms":    snapshot.P95LatencyMs,
		"p99_latency_ms":    snapshot.P99LatencyMs,
		"max_latency_ms":    snapshot.MaxLatencyMs,
		"avg_queue_wait_ms": snapshot.AvgQueueWaitMs,
		"max_queue_wait_ms": snapshot.MaxQueueWaitMs,
		"uptime_seconds":    snapshot.Uptime.Seconds(),
		"health_score":      t.metrics.HealthScore(),
		"health_breakdown":  healthComponents,
		"restart_reasons":   snapshot.RestartReasons,
	}, nil
}
