package proxy

import (
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/metrics"
	"github.com/stretchr/testify/assert"
)

func TestRecordRequest_Success(t *testing.T) {
	collector := metrics.NewCollector()
	p := &proxy{
		metricsCollector: collector,
	}

	start := time.Now()
	time.Sleep(10 * time.Millisecond)
	p.recordRequestComplete(start, true)

	snapshot := collector.Snapshot()
	assert.Equal(t, int64(1), snapshot.RequestsTotal)
	assert.Equal(t, int64(1), snapshot.RequestsSuccess)
	assert.Equal(t, int64(0), snapshot.RequestsFailed)
	assert.GreaterOrEqual(t, snapshot.AvgLatencyMs, 10.0)
}

func TestRecordRequest_Failure(t *testing.T) {
	collector := metrics.NewCollector()
	p := &proxy{
		metricsCollector: collector,
	}

	start := time.Now()
	time.Sleep(5 * time.Millisecond)
	p.recordRequestComplete(start, false)

	snapshot := collector.Snapshot()
	assert.Equal(t, int64(1), snapshot.RequestsTotal)
	assert.Equal(t, int64(0), snapshot.RequestsSuccess)
	assert.Equal(t, int64(1), snapshot.RequestsFailed)
}

func TestRecordQueueWait(t *testing.T) {
	collector := metrics.NewCollector()
	p := &proxy{
		metricsCollector: collector,
	}

	start := time.Now()
	time.Sleep(15 * time.Millisecond)
	p.recordQueueWait(start)

	snapshot := collector.Snapshot()
	assert.Equal(t, int64(1), snapshot.QueueWaitsTotal)
	assert.GreaterOrEqual(t, snapshot.AvgQueueWaitMs, 15.0)
}

func TestRecordRestart(t *testing.T) {
	collector := metrics.NewCollector()
	p := &proxy{
		metricsCollector: collector,
	}

	p.recordRestart("file_change")
	p.recordRestart("crash")
	p.recordRestart("crash")

	snapshot := collector.Snapshot()
	assert.Equal(t, int64(3), snapshot.RestartsTotal)
	assert.Equal(t, int64(1), snapshot.RestartReasons["file_change"])
	assert.Equal(t, int64(2), snapshot.RestartReasons["crash"])
}

func TestMetrics_NilCollector(t *testing.T) {
	p := &proxy{
		metricsCollector: nil,
	}

	// Should not panic when collector is nil
	assert.NotPanics(t, func() {
		p.recordRequestComplete(time.Now(), true)
		p.recordQueueWait(time.Now())
		p.recordRestart("test")
	})
}

func TestMetrics_MultipleRequests(t *testing.T) {
	collector := metrics.NewCollector()
	p := &proxy{
		metricsCollector: collector,
	}

	// Record multiple requests
	for i := 0; i < 10; i++ {
		start := time.Now()
		time.Sleep(time.Duration(i) * time.Millisecond)
		p.recordRequestComplete(start, i%3 != 0) // i%3==0 fails (i=0,3,6,9 = 4 failures)
	}

	snapshot := collector.Snapshot()
	assert.Equal(t, int64(10), snapshot.RequestsTotal)
	assert.Equal(t, int64(6), snapshot.RequestsSuccess) // 10 - 4 = 6
	assert.Equal(t, int64(4), snapshot.RequestsFailed)  // 4 failures
}
