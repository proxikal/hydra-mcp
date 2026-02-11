package adaptive

import (
	"sync"
	"time"

	"github.com/proxikal/hydra/internal/metrics"
)

// Suggestion represents an adaptive configuration suggestion.
type Suggestion struct {
	Parameter  string  // "max_restarts", "queue_size", "debounce_ms"
	Current    int     // Current value
	Suggested  int     // Suggested value
	Confidence float64 // 0.0-1.0 confidence score
	Reason     string  // Human-readable explanation
}

// Observation records an event for learning.
type Observation struct {
	Timestamp time.Time
	Event     string // "restart", "queue_full", "high_latency", "crash_loop"
	Context   map[string]interface{}
}

// Learner analyzes observations and suggests configuration improvements.
type Learner struct {
	observations []Observation
	mu           sync.RWMutex
}

// NewLearner creates a new adaptive learner.
func NewLearner() *Learner {
	return &Learner{
		observations: make([]Observation, 0),
	}
}

// Observe records an event for later analysis.
func (l *Learner) Observe(event string, ctx map[string]interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.observations = append(l.observations, Observation{
		Timestamp: time.Now(),
		Event:     event,
		Context:   ctx,
	})
}

// Analyze generates configuration suggestions based on metrics and observations.
func (l *Learner) Analyze(snapshot *metrics.Snapshot, currentConfig map[string]int) []Suggestion {
	l.mu.RLock()
	defer l.mu.RUnlock()

	suggestions := make([]Suggestion, 0)

	// Rule 1: Restart count < 50% max? → Suggest lowering threshold
	suggestions = append(suggestions, l.analyzeRestartThreshold(snapshot, currentConfig)...)

	// Rule 2: Queue frequently near capacity? → Suggest increasing
	suggestions = append(suggestions, l.analyzeQueueCapacity(snapshot, currentConfig)...)

	// Rule 3: Latency spikes correlate with restarts? → Suggest higher debounce
	suggestions = append(suggestions, l.analyzeLatencySpikes(snapshot, currentConfig)...)

	// Rule 4: Crash loops occur? → Suggest higher restart delay
	suggestions = append(suggestions, l.analyzeCrashLoops(snapshot, currentConfig)...)

	return suggestions
}

func (l *Learner) analyzeRestartThreshold(snapshot *metrics.Snapshot, config map[string]int) []Suggestion {
	maxRestarts := config["max_restarts"]
	if maxRestarts == 0 {
		return nil
	}

	// Count restart observations
	restartObs := 0

	for _, obs := range l.observations {
		if obs.Event == "restart" {
			restartObs++
		}
	}

	// Need minimum observations
	if restartObs < 5 {
		return nil
	}

	// If restarts are consistently < 50% of max, suggest lowering
	utilizationPct := float64(snapshot.RestartsTotal) / float64(maxRestarts)
	if utilizationPct < 0.5 && snapshot.RestartsTotal > 0 {
		confidence := float64(restartObs) / 8.0 // More observations = higher confidence
		if confidence > 1.0 {
			confidence = 1.0
		}

		newMax := int(float64(snapshot.RestartsTotal) * 1.5) // 50% buffer above current
		if newMax < 3 {
			newMax = 3 // Minimum sensible value
		}

		if newMax < maxRestarts {
			return []Suggestion{{
				Parameter:  "max_restarts",
				Current:    maxRestarts,
				Suggested:  newMax,
				Confidence: confidence,
				Reason:     "Restart count consistently below threshold",
			}}
		}
	}

	return nil
}

func (l *Learner) analyzeQueueCapacity(snapshot *metrics.Snapshot, config map[string]int) []Suggestion {
	queueSize := config["queue_size"]
	if queueSize == 0 {
		return nil
	}

	// Count queue_full observations
	queueFullCount := 0
	for _, obs := range l.observations {
		if obs.Event == "queue_full" {
			queueFullCount++
		}
	}

	// If queue frequently full (>= 10 observations), suggest increase
	if queueFullCount >= 10 {
		confidence := float64(queueFullCount) / 30.0
		if confidence > 1.0 {
			confidence = 1.0
		}

		newSize := int(float64(queueSize) * 1.5) // 50% increase

		return []Suggestion{{
			Parameter:  "queue_size",
			Current:    queueSize,
			Suggested:  newSize,
			Confidence: confidence,
			Reason:     "Queue frequently near capacity",
		}}
	}

	return nil
}

func (l *Learner) analyzeLatencySpikes(snapshot *metrics.Snapshot, config map[string]int) []Suggestion {
	debounceMS := config["debounce_ms"]
	if debounceMS == 0 {
		return nil
	}

	// Count high_latency observations near restarts
	highLatencyCount := 0
	for _, obs := range l.observations {
		if obs.Event == "high_latency" {
			if nearRestart, ok := obs.Context["near_restart"].(bool); ok && nearRestart {
				highLatencyCount++
			}
		}
	}

	// If latency spikes correlate with restarts (> 5 observations), suggest higher debounce
	if highLatencyCount > 5 {
		confidence := float64(highLatencyCount) / 15.0
		if confidence > 1.0 {
			confidence = 1.0
		}

		newDebounce := int(float64(debounceMS) * 2.0) // Double debounce

		return []Suggestion{{
			Parameter:  "debounce_ms",
			Current:    debounceMS,
			Suggested:  newDebounce,
			Confidence: confidence,
			Reason:     "Latency spikes correlate with restarts",
		}}
	}

	return nil
}

func (l *Learner) analyzeCrashLoops(snapshot *metrics.Snapshot, config map[string]int) []Suggestion {
	// Count crash_loop observations
	crashLoopCount := 0
	for _, obs := range l.observations {
		if obs.Event == "crash_loop" {
			crashLoopCount++
		}
	}

	// If crash loops detected (> 10 observations), suggest intervention
	if crashLoopCount > 10 {
		debounceMS := config["debounce_ms"]
		confidence := float64(crashLoopCount) / 20.0
		if confidence > 1.0 {
			confidence = 1.0
		}

		newDebounce := int(float64(debounceMS) * 3.0) // Triple debounce for crash loops

		return []Suggestion{{
			Parameter:  "debounce_ms",
			Current:    debounceMS,
			Suggested:  newDebounce,
			Confidence: confidence,
			Reason:     "Crash loops detected - increase restart delay",
		}}
	}

	return nil
}
