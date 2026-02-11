package metrics

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// PrometheusExporter exports metrics in Prometheus text format.
type PrometheusExporter struct {
	collector MetricsCollector
}

// NewPrometheusExporter creates a new Prometheus exporter.
func NewPrometheusExporter(collector MetricsCollector) *PrometheusExporter {
	return &PrometheusExporter{collector: collector}
}

// Export generates Prometheus-formatted metrics.
func (e *PrometheusExporter) Export(serverName string) string {
	snapshot := e.collector.Snapshot()
	breakdown := e.collector.HealthBreakdown()

	var b strings.Builder

	// Request metrics
	b.WriteString("# HELP hydra_requests_total Total number of requests processed\n")
	b.WriteString("# TYPE hydra_requests_total counter\n")
	b.WriteString(fmt.Sprintf("hydra_requests_total{server=%q} %d\n", serverName, snapshot.RequestsTotal))

	b.WriteString("# HELP hydra_requests_success Number of successful requests\n")
	b.WriteString("# TYPE hydra_requests_success counter\n")
	b.WriteString(fmt.Sprintf("hydra_requests_success{server=%q} %d\n", serverName, snapshot.RequestsSuccess))

	b.WriteString("# HELP hydra_requests_failed Number of failed requests\n")
	b.WriteString("# TYPE hydra_requests_failed counter\n")
	b.WriteString(fmt.Sprintf("hydra_requests_failed{server=%q} %d\n", serverName, snapshot.RequestsFailed))

	// Latency metrics
	b.WriteString("# HELP hydra_latency_ms Request latency in milliseconds\n")
	b.WriteString("# TYPE hydra_latency_ms gauge\n")
	b.WriteString(fmt.Sprintf("hydra_latency_ms{server=%q,stat=\"avg\"} %.2f\n", serverName, sanitizeFloat(snapshot.AvgLatencyMs)))
	b.WriteString(fmt.Sprintf("hydra_latency_ms{server=%q,stat=\"max\"} %.2f\n", serverName, sanitizeFloat(snapshot.MaxLatencyMs)))
	b.WriteString(fmt.Sprintf("hydra_latency_ms{server=%q,quantile=\"0.5\"} %.2f\n", serverName, sanitizeFloat(snapshot.P50LatencyMs)))
	b.WriteString(fmt.Sprintf("hydra_latency_ms{server=%q,quantile=\"0.95\"} %.2f\n", serverName, sanitizeFloat(snapshot.P95LatencyMs)))
	b.WriteString(fmt.Sprintf("hydra_latency_ms{server=%q,quantile=\"0.99\"} %.2f\n", serverName, sanitizeFloat(snapshot.P99LatencyMs)))
	b.WriteString(fmt.Sprintf("hydra_latency_ms{server=%q,quantile=\"1.0\"} %.2f\n", serverName, sanitizeFloat(snapshot.MaxLatencyMs)))

	// Queue metrics
	b.WriteString("# HELP hydra_queue_waits_total Total number of queue waits\n")
	b.WriteString("# TYPE hydra_queue_waits_total counter\n")
	b.WriteString(fmt.Sprintf("hydra_queue_waits_total{server=%q} %d\n", serverName, snapshot.QueueWaitsTotal))

	b.WriteString("# HELP hydra_queue_wait_ms Queue wait time in milliseconds\n")
	b.WriteString("# TYPE hydra_queue_wait_ms gauge\n")
	b.WriteString(fmt.Sprintf("hydra_queue_wait_ms{server=%q,stat=\"avg\"} %.2f\n", serverName, sanitizeFloat(snapshot.AvgQueueWaitMs)))
	b.WriteString(fmt.Sprintf("hydra_queue_wait_ms{server=%q,stat=\"max\"} %.2f\n", serverName, sanitizeFloat(snapshot.MaxQueueWaitMs)))

	// Restart metrics
	b.WriteString("# HELP hydra_restarts_total Total number of server restarts\n")
	b.WriteString("# TYPE hydra_restarts_total counter\n")
	b.WriteString(fmt.Sprintf("hydra_restarts_total{server=%q} %d\n", serverName, snapshot.RestartsTotal))

	b.WriteString("# HELP hydra_restarts_by_reason Number of restarts by reason\n")
	b.WriteString("# TYPE hydra_restarts_by_reason counter\n")
	// Sort reasons for deterministic output
	reasons := make([]string, 0, len(snapshot.RestartReasons))
	for reason := range snapshot.RestartReasons {
		reasons = append(reasons, reason)
	}
	sort.Strings(reasons)
	for _, reason := range reasons {
		count := snapshot.RestartReasons[reason]
		b.WriteString(fmt.Sprintf("hydra_restarts_by_reason{server=%q,reason=%q} %d\n", serverName, reason, count))
	}

	// Health metrics
	b.WriteString("# HELP hydra_health_score Overall health score (0-100)\n")
	b.WriteString("# TYPE hydra_health_score gauge\n")
	b.WriteString(fmt.Sprintf("hydra_health_score{server=%q} %d\n", serverName, e.collector.HealthScore()))

	b.WriteString("# HELP hydra_health_component Health component score (0-100)\n")
	b.WriteString("# TYPE hydra_health_component gauge\n")
	b.WriteString(fmt.Sprintf("hydra_health_component{server=%q,component=\"uptime_stability\"} %d\n", serverName, breakdown.UptimeStability))
	b.WriteString(fmt.Sprintf("hydra_health_component{server=%q,component=\"error_rate\"} %d\n", serverName, breakdown.ErrorRate))
	b.WriteString(fmt.Sprintf("hydra_health_component{server=%q,component=\"response_latency\"} %d\n", serverName, breakdown.ResponseLatency))
	b.WriteString(fmt.Sprintf("hydra_health_component{server=%q,component=\"queue_depth\"} %d\n", serverName, breakdown.QueueDepth))
	b.WriteString(fmt.Sprintf("hydra_health_component{server=%q,component=\"restart_frequency\"} %d\n", serverName, breakdown.RestartFrequency))

	// Uptime
	b.WriteString("# HELP hydra_uptime_seconds Server uptime in seconds\n")
	b.WriteString("# TYPE hydra_uptime_seconds gauge\n")
	b.WriteString(fmt.Sprintf("hydra_uptime_seconds{server=%q} %.0f\n", serverName, snapshot.Uptime.Seconds()))

	return b.String()
}

// sanitizeFloat ensures no NaN or Inf values in output.
func sanitizeFloat(f float64) float64 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	return f
}
