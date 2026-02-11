package injectable

import (
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHydraMetrics_Definition(t *testing.T) {
	collector := metrics.NewCollector()

	tools := New(
		Options{Enabled: true, Tools: []string{"hydra_metrics"}},
		nil,
		nil,
		collector,
		nil,
		nil,
		nil,
	)

	defs := tools.GetDefinitions()
	require.Len(t, defs, 1)
	assert.Equal(t, "hydra_metrics", defs[0].Name)
	assert.Contains(t, defs[0].Description, "metrics")
}

func TestHydraMetrics_Handler(t *testing.T) {
	collector := metrics.NewCollector()

	// Record some metrics
	collector.RecordRequest(50*time.Millisecond, true)
	collector.RecordRequest(100*time.Millisecond, true)
	collector.RecordRequest(75*time.Millisecond, false)
	collector.RecordQueueWait(10 * time.Millisecond)
	collector.RecordRestart("crash")

	tools := New(
		Options{Enabled: true, Tools: []string{"hydra_metrics"}},
		nil,
		nil,
		collector,
		nil,
		nil,
		nil,
	)

	result, err := tools.Handle("hydra_metrics", map[string]interface{}{})
	require.NoError(t, err)

	response, ok := result.(map[string]interface{})
	require.True(t, ok, "Response should be a map")

	// Verify basic metrics
	assert.Equal(t, int64(3), response["requests_total"])
	assert.Equal(t, int64(2), response["requests_success"])
	assert.Equal(t, int64(1), response["requests_failed"])
	assert.Equal(t, int64(1), response["restarts_total"])
	assert.Equal(t, int64(1), response["queue_waits_total"])

	// Verify health score is present
	healthScore, ok := response["health_score"].(int)
	require.True(t, ok, "health_score should be an int")
	assert.GreaterOrEqual(t, healthScore, 0)
	assert.LessOrEqual(t, healthScore, 100)

	// Verify health breakdown
	breakdown, ok := response["health_breakdown"].(map[string]int)
	require.True(t, ok, "health_breakdown should be a map")
	assert.Contains(t, breakdown, "uptime_stability")
	assert.Contains(t, breakdown, "error_rate")
	assert.Contains(t, breakdown, "response_latency")
	assert.Contains(t, breakdown, "queue_depth")
	assert.Contains(t, breakdown, "restart_frequency")
}

func TestHydraMetrics_NilCollector(t *testing.T) {
	tools := New(
		Options{Enabled: true, Tools: []string{"hydra_metrics"}},
		nil,
		nil,
		nil, // nil metrics func
		nil,
		nil,
		nil,
	)

	result, err := tools.Handle("hydra_metrics", map[string]interface{}{})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "metrics not available")
}

func TestHydraMetrics_Disabled(t *testing.T) {
	collector := metrics.NewCollector()

	tools := New(
		Options{Enabled: false}, // disabled
		nil,
		nil,
		collector,
		nil,
		nil,
		nil,
	)

	_, err := tools.Handle("hydra_metrics", map[string]interface{}{})
	assert.ErrorIs(t, err, ErrDisabled)
}
