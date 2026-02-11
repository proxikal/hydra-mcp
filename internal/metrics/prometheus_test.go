package metrics

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPrometheusExporter(t *testing.T) {
	c := NewCollector()

	// Record some metrics
	c.RecordRequest(50*time.Millisecond, true)
	c.RecordRequest(100*time.Millisecond, true)
	c.RecordRequest(75*time.Millisecond, false)
	c.RecordQueueWait(10 * time.Millisecond)
	c.RecordQueueWait(20 * time.Millisecond)
	c.RecordRestart("crash")
	c.RecordRestart("file_change")

	exporter := NewPrometheusExporter(c)
	output := exporter.Export("test_server")

	// Verify format
	assert.Contains(t, output, "# HELP", "Should contain help text")
	assert.Contains(t, output, "# TYPE", "Should contain type declarations")

	// Verify metrics exist
	assert.Contains(t, output, "hydra_requests_total", "Should export total requests")
	assert.Contains(t, output, "hydra_requests_success", "Should export successful requests")
	assert.Contains(t, output, "hydra_requests_failed", "Should export failed requests")
	assert.Contains(t, output, "hydra_restarts_total", "Should export total restarts")
	assert.Contains(t, output, "hydra_latency_ms", "Should export latency metrics")
	assert.Contains(t, output, "hydra_health_score", "Should export health score")
	assert.Contains(t, output, "hydra_uptime_seconds", "Should export uptime")

	// Verify labels
	assert.Contains(t, output, `server="test_server"`, "Should include server label")
	assert.Contains(t, output, `reason="crash"`, "Should include restart reason labels")
	assert.Contains(t, output, `reason="file_change"`, "Should include restart reason labels")

	// Verify values
	assert.Contains(t, output, "hydra_requests_total{server=\"test_server\"} 3")
	assert.Contains(t, output, "hydra_requests_success{server=\"test_server\"} 2")
	assert.Contains(t, output, "hydra_requests_failed{server=\"test_server\"} 1")
}

func TestPrometheusExporter_EmptyMetrics(t *testing.T) {
	c := NewCollector()
	exporter := NewPrometheusExporter(c)
	output := exporter.Export("empty_server")

	// Should still produce valid output
	assert.Contains(t, output, "# HELP")
	assert.Contains(t, output, "# TYPE")
	assert.Contains(t, output, "hydra_requests_total{server=\"empty_server\"} 0")
}

func TestPrometheusExporter_Percentiles(t *testing.T) {
	c := NewCollector()

	// Record requests with known latencies
	for i := 1; i <= 100; i++ {
		c.RecordRequest(time.Duration(i)*time.Millisecond, true)
	}

	exporter := NewPrometheusExporter(c)
	output := exporter.Export("percentile_test")

	// Verify percentile metrics
	assert.Contains(t, output, `hydra_latency_ms{server="percentile_test",quantile="0.5"}`)
	assert.Contains(t, output, `hydra_latency_ms{server="percentile_test",quantile="0.95"}`)
	assert.Contains(t, output, `hydra_latency_ms{server="percentile_test",quantile="0.99"}`)
	assert.Contains(t, output, `hydra_latency_ms{server="percentile_test",quantile="1.0"}`) // max
}

func TestPrometheusExporter_HealthBreakdown(t *testing.T) {
	c := NewCollector()

	// Record mixed activity
	for i := 0; i < 50; i++ {
		c.RecordRequest(50*time.Millisecond, true)
	}
	for i := 0; i < 10; i++ {
		c.RecordRequest(50*time.Millisecond, false)
	}

	exporter := NewPrometheusExporter(c)
	output := exporter.Export("health_test")

	// Verify health component metrics
	assert.Contains(t, output, `hydra_health_component{server="health_test",component="uptime_stability"}`)
	assert.Contains(t, output, `hydra_health_component{server="health_test",component="error_rate"}`)
	assert.Contains(t, output, `hydra_health_component{server="health_test",component="response_latency"}`)
	assert.Contains(t, output, `hydra_health_component{server="health_test",component="queue_depth"}`)
	assert.Contains(t, output, `hydra_health_component{server="health_test",component="restart_frequency"}`)
}

func TestPrometheusExporter_RestartReasons(t *testing.T) {
	c := NewCollector()

	c.RecordRestart("crash")
	c.RecordRestart("crash")
	c.RecordRestart("file_change")
	c.RecordRestart("manual")

	exporter := NewPrometheusExporter(c)
	output := exporter.Export("restart_test")

	// Verify restart reason breakdown
	assert.Contains(t, output, `hydra_restarts_by_reason{server="restart_test",reason="crash"} 2`)
	assert.Contains(t, output, `hydra_restarts_by_reason{server="restart_test",reason="file_change"} 1`)
	assert.Contains(t, output, `hydra_restarts_by_reason{server="restart_test",reason="manual"} 1`)
}

func TestPrometheusExporter_QueueMetrics(t *testing.T) {
	c := NewCollector()

	c.RecordQueueWait(10 * time.Millisecond)
	c.RecordQueueWait(20 * time.Millisecond)
	c.RecordQueueWait(30 * time.Millisecond)

	exporter := NewPrometheusExporter(c)
	output := exporter.Export("queue_test")

	assert.Contains(t, output, "hydra_queue_waits_total")
	assert.Contains(t, output, "hydra_queue_wait_ms{server=\"queue_test\",stat=\"avg\"}")
	assert.Contains(t, output, "hydra_queue_wait_ms{server=\"queue_test\",stat=\"max\"}")
}

func TestPrometheusFormat(t *testing.T) {
	c := NewCollector()
	c.RecordRequest(50*time.Millisecond, true)

	exporter := NewPrometheusExporter(c)
	output := exporter.Export("format_test")

	// Verify Prometheus format compliance
	lines := strings.Split(output, "\n")

	hasHelp := false
	hasType := false
	hasMetric := false

	for _, line := range lines {
		if strings.HasPrefix(line, "# HELP") {
			hasHelp = true
		}
		if strings.HasPrefix(line, "# TYPE") {
			hasType = true
		}
		if strings.Contains(line, "hydra_requests_total") && !strings.HasPrefix(line, "#") {
			hasMetric = true
			// Verify metric format: name{labels} value
			assert.Regexp(t, `^[a-z_]+\{.*\} \d+`, line, "Metric should match Prometheus format")
		}
	}

	assert.True(t, hasHelp, "Should have HELP comments")
	assert.True(t, hasType, "Should have TYPE comments")
	assert.True(t, hasMetric, "Should have actual metrics")
}

func TestPrometheusExporter_MultipleServers(t *testing.T) {
	c1 := NewCollector()
	c2 := NewCollector()

	c1.RecordRequest(50*time.Millisecond, true)
	c2.RecordRequest(100*time.Millisecond, true)

	exporter1 := NewPrometheusExporter(c1)
	exporter2 := NewPrometheusExporter(c2)

	output1 := exporter1.Export("server1")
	output2 := exporter2.Export("server2")

	// Each should have different server labels
	assert.Contains(t, output1, `server="server1"`)
	assert.NotContains(t, output1, `server="server2"`)
	assert.Contains(t, output2, `server="server2"`)
	assert.NotContains(t, output2, `server="server1"`)
}

func TestPrometheusExporter_NoNaN(t *testing.T) {
	c := NewCollector()
	exporter := NewPrometheusExporter(c)
	output := exporter.Export("nan_test")

	// Ensure no NaN or Inf values
	assert.NotContains(t, output, "NaN")
	assert.NotContains(t, output, "Inf")
	assert.NotContains(t, output, "-Inf")
}
