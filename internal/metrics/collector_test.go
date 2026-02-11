package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCollector(t *testing.T) {
	c := NewCollector()
	require.NotNil(t, c)

	snapshot := c.Snapshot()
	assert.Equal(t, int64(0), snapshot.RequestsTotal)
	assert.Equal(t, int64(0), snapshot.RequestsSuccess)
	assert.Equal(t, int64(0), snapshot.RequestsFailed)
	assert.Equal(t, int64(0), snapshot.RestartsTotal)
	assert.NotZero(t, snapshot.StartTime)
}

func TestRecordRequest_Success(t *testing.T) {
	c := NewCollector()

	c.RecordRequest(50*time.Millisecond, true)
	c.RecordRequest(100*time.Millisecond, true)
	c.RecordRequest(150*time.Millisecond, true)

	snapshot := c.Snapshot()
	assert.Equal(t, int64(3), snapshot.RequestsTotal)
	assert.Equal(t, int64(3), snapshot.RequestsSuccess)
	assert.Equal(t, int64(0), snapshot.RequestsFailed)
	assert.Greater(t, snapshot.AvgLatencyMs, 0.0)
	assert.Equal(t, 150.0, snapshot.MaxLatencyMs)
}

func TestRecordRequest_Failure(t *testing.T) {
	c := NewCollector()

	c.RecordRequest(50*time.Millisecond, false)
	c.RecordRequest(100*time.Millisecond, false)

	snapshot := c.Snapshot()
	assert.Equal(t, int64(2), snapshot.RequestsTotal)
	assert.Equal(t, int64(0), snapshot.RequestsSuccess)
	assert.Equal(t, int64(2), snapshot.RequestsFailed)
}

func TestRecordRequest_Mixed(t *testing.T) {
	c := NewCollector()

	c.RecordRequest(50*time.Millisecond, true)
	c.RecordRequest(75*time.Millisecond, false)
	c.RecordRequest(100*time.Millisecond, true)

	snapshot := c.Snapshot()
	assert.Equal(t, int64(3), snapshot.RequestsTotal)
	assert.Equal(t, int64(2), snapshot.RequestsSuccess)
	assert.Equal(t, int64(1), snapshot.RequestsFailed)
}

func TestRecordRequest_Percentiles(t *testing.T) {
	c := NewCollector()

	// Record 100 requests with varying latencies
	for i := 1; i <= 100; i++ {
		c.RecordRequest(time.Duration(i)*time.Millisecond, true)
	}

	snapshot := c.Snapshot()
	assert.Equal(t, int64(100), snapshot.RequestsTotal)

	// P50 should be around 50ms
	assert.InDelta(t, 50.0, snapshot.P50LatencyMs, 5.0)

	// P95 should be around 95ms
	assert.InDelta(t, 95.0, snapshot.P95LatencyMs, 5.0)

	// P99 should be around 99ms
	assert.InDelta(t, 99.0, snapshot.P99LatencyMs, 5.0)

	// Max should be 100ms
	assert.Equal(t, 100.0, snapshot.MaxLatencyMs)
}

func TestRecordQueueWait(t *testing.T) {
	c := NewCollector()

	c.RecordQueueWait(10 * time.Millisecond)
	c.RecordQueueWait(20 * time.Millisecond)
	c.RecordQueueWait(30 * time.Millisecond)

	snapshot := c.Snapshot()
	assert.Equal(t, int64(3), snapshot.QueueWaitsTotal)
	assert.Greater(t, snapshot.AvgQueueWaitMs, 0.0)
	assert.Equal(t, 30.0, snapshot.MaxQueueWaitMs)
}

func TestRecordRestart(t *testing.T) {
	c := NewCollector()

	c.RecordRestart("crash")
	c.RecordRestart("file_change")
	c.RecordRestart("crash")
	c.RecordRestart("manual")

	snapshot := c.Snapshot()
	assert.Equal(t, int64(4), snapshot.RestartsTotal)
	assert.Equal(t, int64(2), snapshot.RestartReasons["crash"])
	assert.Equal(t, int64(1), snapshot.RestartReasons["file_change"])
	assert.Equal(t, int64(1), snapshot.RestartReasons["manual"])
}

func TestHealthScore_Perfect(t *testing.T) {
	c := NewCollector()

	// Record successful requests with low latency
	for i := 0; i < 100; i++ {
		c.RecordRequest(10*time.Millisecond, true)
	}

	// Wait a bit to build uptime
	time.Sleep(100 * time.Millisecond)

	score := c.HealthScore()
	assert.Greater(t, score, 90, "Perfect conditions should yield high health score")
}

func TestHealthScore_HighErrorRate(t *testing.T) {
	c := NewCollector()

	// Record mostly failed requests
	for i := 0; i < 100; i++ {
		c.RecordRequest(10*time.Millisecond, false)
	}

	score := c.HealthScore()
	// Error rate is 25% of total, so 100% errors gives ~75 score (other components perfect)
	assert.Less(t, score, 80, "High error rate should lower health score")
	assert.Greater(t, score, 60, "Other components keep score from being too low")

	breakdown := c.HealthBreakdown()
	assert.Equal(t, 0, breakdown.ErrorRate, "Error rate component should be 0")
}

func TestHealthScore_HighLatency(t *testing.T) {
	c := NewCollector()

	// Record slow requests
	for i := 0; i < 100; i++ {
		c.RecordRequest(5*time.Second, true)
	}

	score := c.HealthScore()
	// Latency is 20% of total, so high latency gives ~80 score (other components perfect)
	assert.Less(t, score, 85, "High latency should lower health score")
	assert.Greater(t, score, 70, "Other components keep score from being too low")

	breakdown := c.HealthBreakdown()
	assert.Less(t, breakdown.ResponseLatency, 50, "Latency component should be low")
}

func TestHealthScore_FrequentRestarts(t *testing.T) {
	c := NewCollector()

	// Record many restarts
	for i := 0; i < 20; i++ {
		c.RecordRestart("crash")
	}

	// Add some successful requests
	for i := 0; i < 100; i++ {
		c.RecordRequest(10*time.Millisecond, true)
	}

	score := c.HealthScore()
	// 20 restarts significantly impacts both restart (10%) and uptime (30%) components
	assert.Less(t, score, 90, "Frequent restarts should lower health score")

	breakdown := c.HealthBreakdown()
	assert.Equal(t, 0, breakdown.RestartFrequency, "Restart component should be 0 with 20 restarts")
}

func TestHealthBreakdown(t *testing.T) {
	c := NewCollector()

	// Record mixed activity
	for i := 0; i < 50; i++ {
		c.RecordRequest(50*time.Millisecond, true)
	}
	for i := 0; i < 10; i++ {
		c.RecordRequest(50*time.Millisecond, false)
	}
	c.RecordRestart("crash")

	breakdown := c.HealthBreakdown()

	// All components should be 0-100
	assert.GreaterOrEqual(t, breakdown.UptimeStability, 0)
	assert.LessOrEqual(t, breakdown.UptimeStability, 100)
	assert.GreaterOrEqual(t, breakdown.ErrorRate, 0)
	assert.LessOrEqual(t, breakdown.ErrorRate, 100)
	assert.GreaterOrEqual(t, breakdown.ResponseLatency, 0)
	assert.LessOrEqual(t, breakdown.ResponseLatency, 100)
	assert.GreaterOrEqual(t, breakdown.QueueDepth, 0)
	assert.LessOrEqual(t, breakdown.QueueDepth, 100)
	assert.GreaterOrEqual(t, breakdown.RestartFrequency, 0)
	assert.LessOrEqual(t, breakdown.RestartFrequency, 100)

	// Weighted score should match HealthScore
	assert.Equal(t, c.HealthScore(), breakdown.WeightedScore())
}

func TestSnapshot_Uptime(t *testing.T) {
	c := NewCollector()

	time.Sleep(100 * time.Millisecond)

	snapshot := c.Snapshot()
	assert.GreaterOrEqual(t, snapshot.Uptime, 100*time.Millisecond)
}

func TestConcurrentRecording(t *testing.T) {
	c := NewCollector()

	// Record from multiple goroutines
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				c.RecordRequest(10*time.Millisecond, true)
				c.RecordQueueWait(5 * time.Millisecond)
				c.RecordRestart("test")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	snapshot := c.Snapshot()
	assert.Equal(t, int64(1000), snapshot.RequestsTotal)
	assert.Equal(t, int64(1000), snapshot.QueueWaitsTotal)
	assert.Equal(t, int64(1000), snapshot.RestartsTotal)
}
