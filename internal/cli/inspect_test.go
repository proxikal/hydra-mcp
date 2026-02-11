package cli

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/metrics"
	"github.com/proxikal/hydra/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInspectCommand_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := state.NewManager(tmpDir)

	var buf bytes.Buffer
	err := runInspect(mgr, "nonexistent", &buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestInspectCommand_BasicInfo(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := state.NewManager(tmpDir)

	collector := metrics.NewCollector()
	collector.RecordRequest(50*time.Millisecond, true)

	err := mgr.SaveState("test-server", os.Getpid(), collector)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = runInspect(mgr, "test-server", &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "test-server")
	assert.Contains(t, output, "PID")
	assert.Contains(t, output, "Uptime")
}

func TestInspectCommand_HealthBreakdown(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := state.NewManager(tmpDir)

	collector := metrics.NewCollector()
	for i := 0; i < 50; i++ {
		collector.RecordRequest(50*time.Millisecond, true)
	}
	for i := 0; i < 10; i++ {
		collector.RecordRequest(50*time.Millisecond, false)
	}

	err := mgr.SaveState("health-server", os.Getpid(), collector)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = runInspect(mgr, "health-server", &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "HEALTH SCORE")
	assert.Contains(t, output, "Uptime Stability")
	assert.Contains(t, output, "Error Rate")
	assert.Contains(t, output, "Response Latency")
	assert.Contains(t, output, "Queue Depth")
	assert.Contains(t, output, "Restart Frequency")
}

func TestInspectCommand_Metrics(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := state.NewManager(tmpDir)

	collector := metrics.NewCollector()
	collector.RecordRequest(50*time.Millisecond, true)
	collector.RecordRequest(100*time.Millisecond, false)
	collector.RecordQueueWait(10 * time.Millisecond)
	collector.RecordRestart("crash")

	err := mgr.SaveState("metrics-server", os.Getpid(), collector)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = runInspect(mgr, "metrics-server", &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Requests Total")
	assert.Contains(t, output, "Requests Success")
	assert.Contains(t, output, "Requests Failed")
	assert.Contains(t, output, "Restarts Total")
}

func TestInspectCommand_Latency(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := state.NewManager(tmpDir)

	collector := metrics.NewCollector()
	for i := 1; i <= 100; i++ {
		collector.RecordRequest(time.Duration(i)*time.Millisecond, true)
	}

	err := mgr.SaveState("latency-server", os.Getpid(), collector)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = runInspect(mgr, "latency-server", &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Average")
	assert.Contains(t, output, "P50")
	assert.Contains(t, output, "P95")
	assert.Contains(t, output, "P99")
	assert.Contains(t, output, "Max")
}

func TestInspectCommand_RestartReasons(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := state.NewManager(tmpDir)

	collector := metrics.NewCollector()
	collector.RecordRestart("crash")
	collector.RecordRestart("crash")
	collector.RecordRestart("file_change")
	collector.RecordRestart("manual")

	err := mgr.SaveState("restart-server", os.Getpid(), collector)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = runInspect(mgr, "restart-server", &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "crash")
	assert.Contains(t, output, "file_change")
	assert.Contains(t, output, "manual")
}

func TestInspectCommand_QueueMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := state.NewManager(tmpDir)

	collector := metrics.NewCollector()
	collector.RecordQueueWait(10 * time.Millisecond)
	collector.RecordQueueWait(20 * time.Millisecond)
	collector.RecordQueueWait(30 * time.Millisecond)

	err := mgr.SaveState("queue-server", os.Getpid(), collector)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = runInspect(mgr, "queue-server", &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Queue")
}
