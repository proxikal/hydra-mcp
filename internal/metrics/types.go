package metrics

import "time"

// MetricsCollector defines the interface for collecting and reporting proxy metrics.
type MetricsCollector interface {
	// RecordRequest records a completed request with its duration and success status.
	RecordRequest(duration time.Duration, success bool)

	// RecordQueueWait records time spent waiting in queue.
	RecordQueueWait(duration time.Duration)

	// RecordRestart records a server restart with the triggering reason.
	RecordRestart(reason string)

	// Snapshot returns the current metrics snapshot.
	Snapshot() *Snapshot

	// HealthScore returns a 0-100 health score based on multiple dimensions.
	HealthScore() int

	// HealthBreakdown returns individual health component scores.
	HealthBreakdown() HealthComponents
}

// Snapshot represents a point-in-time view of all metrics.
type Snapshot struct {
	// Basic counters
	RequestsTotal   int64         `json:"requests_total"`
	RequestsSuccess int64         `json:"requests_success"`
	RequestsFailed  int64         `json:"requests_failed"`
	RestartsTotal   int64         `json:"restarts_total"`
	QueueWaitsTotal int64         `json:"queue_waits_total"`

	// Latency metrics (milliseconds)
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	P50LatencyMs float64 `json:"p50_latency_ms"`
	P95LatencyMs float64 `json:"p95_latency_ms"`
	P99LatencyMs float64 `json:"p99_latency_ms"`
	MaxLatencyMs float64 `json:"max_latency_ms"`

	// Queue metrics
	AvgQueueWaitMs float64 `json:"avg_queue_wait_ms"`
	MaxQueueWaitMs float64 `json:"max_queue_wait_ms"`

	// Timing
	StartTime time.Time     `json:"start_time"`
	Uptime    time.Duration `json:"uptime"`

	// Restart reasons (reason -> count)
	RestartReasons map[string]int64 `json:"restart_reasons"`
}

// HealthComponents contains individual health dimension scores (0-100).
type HealthComponents struct {
	UptimeStability int `json:"uptime_stability"`     // 30% weight
	ErrorRate       int `json:"error_rate"`           // 25% weight
	ResponseLatency int `json:"response_latency"`     // 20% weight
	QueueDepth      int `json:"queue_depth"`          // 15% weight
	RestartFrequency int `json:"restart_frequency"`   // 10% weight
}

// WeightedScore calculates the overall health score from components.
func (h HealthComponents) WeightedScore() int {
	score := float64(h.UptimeStability)*0.30 +
		float64(h.ErrorRate)*0.25 +
		float64(h.ResponseLatency)*0.20 +
		float64(h.QueueDepth)*0.15 +
		float64(h.RestartFrequency)*0.10
	return int(score)
}
