package metrics

import (
	"math"
	"sort"
	"sync"
	"time"
)

// Collector implements MetricsCollector with thread-safe metric collection.
type Collector struct {
	mu sync.RWMutex

	startTime time.Time

	// Request metrics
	requestsTotal   int64
	requestsSuccess int64
	requestsFailed  int64

	// Latency tracking (milliseconds)
	latencies    []float64
	maxLatency   float64
	totalLatency float64

	// Queue metrics
	queueWaitsTotal int64
	queueWaits      []float64
	maxQueueWait    float64
	totalQueueWait  float64

	// Restart tracking
	restartsTotal  int64
	restartReasons map[string]int64
	restartTimes   []time.Time
}

// NewCollector creates a new metrics collector.
func NewCollector() *Collector {
	return &Collector{
		startTime:      time.Now(),
		latencies:      make([]float64, 0, 1000),
		queueWaits:     make([]float64, 0, 1000),
		restartReasons: make(map[string]int64),
		restartTimes:   make([]time.Time, 0, 100),
	}
}

// RecordRequest records a completed request.
func (c *Collector) RecordRequest(duration time.Duration, success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.requestsTotal++
	if success {
		c.requestsSuccess++
	} else {
		c.requestsFailed++
	}

	latencyMs := float64(duration.Milliseconds())
	c.latencies = append(c.latencies, latencyMs)
	c.totalLatency += latencyMs

	if latencyMs > c.maxLatency {
		c.maxLatency = latencyMs
	}
}

// RecordQueueWait records time spent waiting in queue.
func (c *Collector) RecordQueueWait(duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.queueWaitsTotal++
	waitMs := float64(duration.Milliseconds())
	c.queueWaits = append(c.queueWaits, waitMs)
	c.totalQueueWait += waitMs

	if waitMs > c.maxQueueWait {
		c.maxQueueWait = waitMs
	}
}

// RecordRestart records a server restart.
func (c *Collector) RecordRestart(reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.restartsTotal++
	c.restartReasons[reason]++
	c.restartTimes = append(c.restartTimes, time.Now())
}

// Snapshot returns current metrics snapshot.
func (c *Collector) Snapshot() *Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snapshot := &Snapshot{
		RequestsTotal:   c.requestsTotal,
		RequestsSuccess: c.requestsSuccess,
		RequestsFailed:  c.requestsFailed,
		RestartsTotal:   c.restartsTotal,
		QueueWaitsTotal: c.queueWaitsTotal,
		StartTime:       c.startTime,
		Uptime:          time.Since(c.startTime),
		RestartReasons:  make(map[string]int64),
		MaxLatencyMs:    c.maxLatency,
		MaxQueueWaitMs:  c.maxQueueWait,
	}

	// Copy restart reasons
	for reason, count := range c.restartReasons {
		snapshot.RestartReasons[reason] = count
	}

	// Calculate average latency
	if c.requestsTotal > 0 {
		snapshot.AvgLatencyMs = c.totalLatency / float64(c.requestsTotal)
	}

	// Calculate latency percentiles
	if len(c.latencies) > 0 {
		sorted := make([]float64, len(c.latencies))
		copy(sorted, c.latencies)
		sort.Float64s(sorted)

		snapshot.P50LatencyMs = percentile(sorted, 50)
		snapshot.P95LatencyMs = percentile(sorted, 95)
		snapshot.P99LatencyMs = percentile(sorted, 99)
	}

	// Calculate average queue wait
	if c.queueWaitsTotal > 0 {
		snapshot.AvgQueueWaitMs = c.totalQueueWait / float64(c.queueWaitsTotal)
	}

	return snapshot
}

// HealthScore returns overall health score (0-100).
func (c *Collector) HealthScore() int {
	breakdown := c.HealthBreakdown()
	return breakdown.WeightedScore()
}

// HealthBreakdown returns individual health component scores.
func (c *Collector) HealthBreakdown() HealthComponents {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return HealthComponents{
		UptimeStability:  c.calculateUptimeScore(),
		ErrorRate:        c.calculateErrorScore(),
		ResponseLatency:  c.calculateLatencyScore(),
		QueueDepth:       c.calculateQueueScore(),
		RestartFrequency: c.calculateRestartScore(),
	}
}

// calculateUptimeScore returns uptime stability score (0-100).
// Score decreases with restart frequency relative to uptime.
func (c *Collector) calculateUptimeScore() int {
	uptime := time.Since(c.startTime)
	if uptime < 10*time.Second {
		return 95 // Short uptime but stable
	}

	// Calculate restarts per hour
	hours := uptime.Hours()
	if hours < 0.001 {
		hours = 0.001 // Prevent division by zero for very short uptimes
	}
	restartsPerHour := float64(c.restartsTotal) / hours

	// Perfect score if < 0.1 restarts/hour
	// 0 score if >= 5 restarts/hour
	score := 100 - int(restartsPerHour*20)
	return clamp(score, 0, 100)
}

// calculateErrorScore returns error rate score (0-100).
func (c *Collector) calculateErrorScore() int {
	if c.requestsTotal == 0 {
		return 100 // No requests yet
	}

	errorRate := float64(c.requestsFailed) / float64(c.requestsTotal)

	// Perfect score if error rate < 1%
	// 0 score if error rate >= 50%
	score := 100 - int(errorRate*200)
	return clamp(score, 0, 100)
}

// calculateLatencyScore returns latency score (0-100).
func (c *Collector) calculateLatencyScore() int {
	if len(c.latencies) == 0 {
		return 100 // No data yet
	}

	sorted := make([]float64, len(c.latencies))
	copy(sorted, c.latencies)
	sort.Float64s(sorted)

	p95 := percentile(sorted, 95)

	// Perfect score if P95 < 100ms
	// 50 score if P95 = 1000ms
	// 0 score if P95 >= 5000ms
	score := 100 - int((p95-100)/49)
	return clamp(score, 0, 100)
}

// calculateQueueScore returns queue depth score (0-100).
func (c *Collector) calculateQueueScore() int {
	if c.queueWaitsTotal == 0 {
		return 100 // No queue waits
	}

	avgWait := c.totalQueueWait / float64(c.queueWaitsTotal)

	// Perfect score if avg wait < 10ms
	// 0 score if avg wait >= 1000ms
	score := 100 - int((avgWait-10)/10)
	return clamp(score, 0, 100)
}

// calculateRestartScore returns restart frequency score (0-100).
func (c *Collector) calculateRestartScore() int {
	if c.restartsTotal == 0 {
		return 100 // No restarts
	}

	uptime := time.Since(c.startTime)
	if uptime < 10*time.Second {
		// For short uptimes, penalize based on absolute count
		score := 100 - (int(c.restartsTotal) * 10)
		return clamp(score, 0, 100)
	}

	// Count recent restarts (last hour, or entire uptime if less)
	recentRestarts := 0
	lookback := time.Hour
	if uptime < time.Hour {
		lookback = uptime
	}
	cutoff := time.Now().Add(-lookback)
	for _, t := range c.restartTimes {
		if t.After(cutoff) {
			recentRestarts++
		}
	}

	// Perfect score if 0 restarts in last hour
	// 0 score if >= 10 restarts in last hour
	score := 100 - (recentRestarts * 10)
	return clamp(score, 0, 100)
}

// percentile calculates the nth percentile of a sorted slice.
func percentile(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}

	rank := float64(p) / 100.0 * float64(len(sorted)-1)
	lower := int(math.Floor(rank))
	upper := int(math.Ceil(rank))

	if lower == upper {
		return sorted[lower]
	}

	// Linear interpolation
	weight := rank - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

// clamp restricts value to [min, max] range.
func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
