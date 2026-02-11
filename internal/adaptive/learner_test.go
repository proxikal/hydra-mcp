package adaptive

import (
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLearner(t *testing.T) {
	l := NewLearner()
	require.NotNil(t, l)
}

func TestObserve(t *testing.T) {
	l := NewLearner()

	l.Observe("restart", map[string]interface{}{
		"restarts": 5,
		"max":      10,
	})

	// Verify observation was recorded
	observations := l.observations
	assert.Len(t, observations, 1)
	assert.Equal(t, "restart", observations[0].Event)
	assert.Equal(t, 5, observations[0].Context["restarts"])
}

func TestAnalyze_LowRestartCount(t *testing.T) {
	l := NewLearner()

	// Simulate low restart usage (2 out of 10 max)
	for i := 0; i < 5; i++ {
		l.Observe("restart", map[string]interface{}{
			"restarts":     2,
			"max_restarts": 10,
		})
	}

	snapshot := &metrics.Snapshot{
		RestartsTotal: 2,
	}

	currentConfig := map[string]int{
		"max_restarts": 10,
		"queue_size":   100,
		"debounce_ms":  500,
	}

	suggestions := l.Analyze(snapshot, currentConfig)

	// Should suggest lowering max_restarts
	var foundRestart bool
	for _, s := range suggestions {
		if s.Parameter == "max_restarts" {
			foundRestart = true
			assert.Less(t, s.Suggested, s.Current, "Should suggest lower max_restarts")
			assert.Greater(t, s.Confidence, 0.5, "Should have reasonable confidence")
		}
	}

	assert.True(t, foundRestart, "Should suggest max_restarts adjustment")
}

func TestAnalyze_HighQueueUtilization(t *testing.T) {
	l := NewLearner()

	// Simulate frequent queue_full events
	for i := 0; i < 10; i++ {
		l.Observe("queue_full", map[string]interface{}{
			"queue_size": 95,
			"max":        100,
		})
	}

	snapshot := &metrics.Snapshot{
		MaxQueueWaitMs: 95,
	}

	currentConfig := map[string]int{
		"max_restarts": 10,
		"queue_size":   100,
		"debounce_ms":  500,
	}

	suggestions := l.Analyze(snapshot, currentConfig)

	// Should suggest increasing queue_size
	var foundQueue bool
	for _, s := range suggestions {
		if s.Parameter == "queue_size" {
			foundQueue = true
			assert.Greater(t, s.Suggested, s.Current, "Should suggest larger queue")
			assert.Equal(t, "Queue frequently near capacity", s.Reason)
		}
	}

	assert.True(t, foundQueue, "Should suggest queue_size adjustment")
}

func TestAnalyze_HighLatencyWithRestarts(t *testing.T) {
	l := NewLearner()

	// Simulate latency spikes correlating with restarts
	for i := 0; i < 8; i++ {
		l.Observe("high_latency", map[string]interface{}{
			"latency_ms":  2000,
			"near_restart": true,
		})
	}

	snapshot := &metrics.Snapshot{
		MaxLatencyMs: 2000,
		AvgLatencyMs: 1500,
	}

	currentConfig := map[string]int{
		"max_restarts": 10,
		"queue_size":   100,
		"debounce_ms":  500,
	}

	suggestions := l.Analyze(snapshot, currentConfig)

	// Should suggest increasing debounce
	var foundDebounce bool
	for _, s := range suggestions {
		if s.Parameter == "debounce_ms" {
			foundDebounce = true
			assert.Greater(t, s.Suggested, s.Current, "Should suggest higher debounce")
		}
	}

	assert.True(t, foundDebounce, "Should suggest debounce adjustment")
}

func TestAnalyze_CrashLoops(t *testing.T) {
	l := NewLearner()

	// Simulate crash loops
	for i := 0; i < 15; i++ {
		l.Observe("crash_loop", map[string]interface{}{
			"restarts": 25,
			"window":   60,
		})
	}

	snapshot := &metrics.Snapshot{
		RestartsTotal:    25,
			}

	currentConfig := map[string]int{
		"max_restarts": 10,
		"queue_size":   100,
		"debounce_ms":  500,
	}

	suggestions := l.Analyze(snapshot, currentConfig)

	// Should suggest increasing restart delay or debounce
	assert.NotEmpty(t, suggestions, "Should have suggestions for crash loops")
}

func TestAnalyze_ConfidenceScoring(t *testing.T) {
	l := NewLearner()

	// Few observations = low confidence
	l.Observe("restart", map[string]interface{}{"restarts": 2, "max_restarts": 10})

	snapshot := &metrics.Snapshot{RestartsTotal: 2}
	currentConfig := map[string]int{"max_restarts": 10, "queue_size": 100}

	suggestions := l.Analyze(snapshot, currentConfig)

	for _, s := range suggestions {
		// With only 1 observation, confidence should be low
		assert.LessOrEqual(t, s.Confidence, 0.6, "Low observation count should mean low confidence")
	}
}

func TestAnalyze_NoSuggestionsWhenHealthy(t *testing.T) {
	l := NewLearner()

	// Healthy metrics
	snapshot := &metrics.Snapshot{
		RestartsTotal: 5,
		MaxLatencyMs:  100,
	}

	currentConfig := map[string]int{
		"max_restarts": 10,
		"queue_size":   100,
		"debounce_ms":  500,
	}

	suggestions := l.Analyze(snapshot, currentConfig)

	// Should have no high-confidence suggestions
	for _, s := range suggestions {
		assert.LessOrEqual(t, s.Confidence, 0.6, "Healthy system should not have high-confidence suggestions")
	}
}

func TestObservation_Timestamp(t *testing.T) {
	l := NewLearner()

	before := time.Now()
	l.Observe("test", map[string]interface{}{"foo": "bar"})
	after := time.Now()

	require.Len(t, l.observations, 1)
	assert.True(t, l.observations[0].Timestamp.After(before) || l.observations[0].Timestamp.Equal(before))
	assert.True(t, l.observations[0].Timestamp.Before(after) || l.observations[0].Timestamp.Equal(after))
}
