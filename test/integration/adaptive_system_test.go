package integration

import (
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/adaptive"
	"github.com/proxikal/hydra/internal/metrics"
	"github.com/stretchr/testify/assert"
)

// TestAdaptiveSystem_EndToEnd validates the complete adaptive learning pipeline:
// metrics collection → health scoring → observation learning → suggestions
func TestAdaptiveSystem_EndToEnd(t *testing.T) {
	// Phase 1: Metrics Collection & Health Scoring
	collector := metrics.NewCollector()

	// Simulate successful operation
	for i := 0; i < 100; i++ {
		collector.RecordRequest(50*time.Millisecond, true)
	}

	// Wait for uptime
	time.Sleep(200 * time.Millisecond)

	// Verify health score is high for healthy system
	healthScore := collector.HealthScore()
	assert.GreaterOrEqual(t, healthScore, 90, "Healthy system should have high score")

	breakdown := collector.HealthBreakdown()
	assert.Equal(t, 100, breakdown.ErrorRate, "No errors → perfect error rate score")
	assert.GreaterOrEqual(t, breakdown.ResponseLatency, 85, "Low latency → high latency score")

	// Phase 2: Degraded Performance
	collector2 := metrics.NewCollector()

	// Simulate high error rate (50% failures)
	for i := 0; i < 50; i++ {
		collector2.RecordRequest(50*time.Millisecond, true)
		collector2.RecordRequest(50*time.Millisecond, false)
	}

	healthScore2 := collector2.HealthScore()
	assert.Less(t, healthScore2, 80, "High error rate should lower health score")

	breakdown2 := collector2.HealthBreakdown()
	assert.Equal(t, 0, breakdown2.ErrorRate, "50% error rate → 0 error score")

	// Phase 3: Adaptive Learning - Restart Threshold
	learner := adaptive.NewLearner()
	collector3 := metrics.NewCollector()

	// Simulate low restart usage (2 out of 10 max)
	// Record 2 actual restarts
	collector3.RecordRestart("crash")
	collector3.RecordRestart("crash")

	// Record 5+ observations to build confidence
	for i := 0; i < 5; i++ {
		learner.Observe("restart", map[string]interface{}{
			"restarts":     2,
			"max_restarts": 10,
		})
	}

	snapshot := collector3.Snapshot()
	currentConfig := map[string]int{
		"max_restarts": 10,
		"queue_size":   100,
		"debounce_ms":  500,
	}

	suggestions := learner.Analyze(snapshot, currentConfig)

	// Should suggest lowering max_restarts
	var found bool
	for _, s := range suggestions {
		if s.Parameter == "max_restarts" {
			found = true
			assert.Less(t, s.Suggested, s.Current, "Should suggest lower threshold")
			assert.GreaterOrEqual(t, s.Suggested, 3, "Should maintain minimum value")
			t.Logf("Restart suggestion: %d → %d (confidence: %.0f%%)",
				s.Current, s.Suggested, s.Confidence*100)
		}
	}
	assert.True(t, found, "Should suggest max_restarts adjustment")

	// Phase 4: Queue Capacity Learning
	learner2 := adaptive.NewLearner()

	// Simulate queue pressure
	for i := 0; i < 12; i++ {
		learner2.Observe("queue_full", map[string]interface{}{
			"queue_size": 95,
			"max":        100,
		})
	}

	suggestions2 := learner2.Analyze(snapshot, currentConfig)

	// Should suggest increasing queue_size
	var foundQueue bool
	for _, s := range suggestions2 {
		if s.Parameter == "queue_size" {
			foundQueue = true
			assert.Greater(t, s.Suggested, s.Current, "Should suggest larger queue")
			assert.Equal(t, "Queue frequently near capacity", s.Reason)
			t.Logf("Queue suggestion: %d → %d (confidence: %.0f%%)",
				s.Current, s.Suggested, s.Confidence*100)
		}
	}
	assert.True(t, foundQueue, "Should suggest queue_size adjustment")

	// Phase 5: Latency Spike Detection
	learner3 := adaptive.NewLearner()
	collector4 := metrics.NewCollector()

	// Simulate high latency correlated with restarts
	for i := 0; i < 8; i++ {
		learner3.Observe("high_latency", map[string]interface{}{
			"latency_ms":   2000,
			"near_restart": true,
		})
		collector4.RecordRequest(2*time.Second, true)
	}

	snapshot2 := collector4.Snapshot()
	suggestions3 := learner3.Analyze(snapshot2, currentConfig)

	// Should suggest increasing debounce
	var foundDebounce bool
	for _, s := range suggestions3 {
		if s.Parameter == "debounce_ms" {
			foundDebounce = true
			assert.Greater(t, s.Suggested, s.Current, "Should suggest higher debounce")
			t.Logf("Debounce suggestion: %d → %d (confidence: %.0f%%)",
				s.Current, s.Suggested, s.Confidence*100)
		}
	}
	assert.True(t, foundDebounce, "Should suggest debounce adjustment")

	t.Log("✓ Adaptive learning system validated end-to-end")
}

// TestAdaptiveSystem_ConfidenceProgression validates that confidence
// increases with more observations
func TestAdaptiveSystem_ConfidenceProgression(t *testing.T) {
	collector := metrics.NewCollector()
	collector.RecordRestart("crash")
	collector.RecordRestart("crash")

	currentConfig := map[string]int{
		"max_restarts": 10,
		"queue_size":   100,
		"debounce_ms":  500,
	}

	snapshot := collector.Snapshot()

	// Test with increasing observation counts
	observationCounts := []int{5, 8, 12}
	var prevConfidence float64

	for _, count := range observationCounts {
		learner := adaptive.NewLearner()

		// Add observations
		for i := 0; i < count; i++ {
			learner.Observe("restart", map[string]interface{}{
				"restarts":     2,
				"max_restarts": 10,
			})
		}

		suggestions := learner.Analyze(snapshot, currentConfig)

		// Find restart suggestion
		for _, s := range suggestions {
			if s.Parameter == "max_restarts" {
				t.Logf("Observations: %d → Confidence: %.0f%%",
					count, s.Confidence*100)

				// Confidence should increase with more observations
				if prevConfidence > 0 {
					assert.GreaterOrEqual(t, s.Confidence, prevConfidence,
						"More observations should increase confidence")
				}
				prevConfidence = s.Confidence
			}
		}
	}

	assert.Greater(t, prevConfidence, 0.6,
		"High observation count should yield high confidence")
}

// TestAdaptiveSystem_NoSuggestionsWhenHealthy validates that healthy
// systems don't generate high-confidence suggestions
func TestAdaptiveSystem_NoSuggestionsWhenHealthy(t *testing.T) {
	learner := adaptive.NewLearner()
	collector := metrics.NewCollector()

	// Simulate healthy system
	for i := 0; i < 100; i++ {
		collector.RecordRequest(50*time.Millisecond, true)
	}
	time.Sleep(200 * time.Millisecond)

	// Record a few restarts (normal usage)
	for i := 0; i < 5; i++ {
		collector.RecordRestart("file_change")
	}

	snapshot := collector.Snapshot()
	currentConfig := map[string]int{
		"max_restarts": 10,
		"queue_size":   100,
		"debounce_ms":  500,
	}

	suggestions := learner.Analyze(snapshot, currentConfig)

	// Should have no high-confidence suggestions for healthy system
	for _, s := range suggestions {
		assert.LessOrEqual(t, s.Confidence, 0.6,
			"Healthy system should not have high-confidence suggestions")
	}

	healthScore := collector.HealthScore()
	assert.GreaterOrEqual(t, healthScore, 85, "System should be healthy")

	t.Logf("✓ Healthy system (score: %d) generates no high-confidence suggestions", healthScore)
}
