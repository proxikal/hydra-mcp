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

func TestPsCommand_NoInstances(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := state.NewManager(tmpDir)

	var buf bytes.Buffer
	err := runPs(mgr, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "No running Hydra instances")
}

func TestPsCommand_SingleInstance(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := state.NewManager(tmpDir)

	// Create a state for current process
	collector := metrics.NewCollector()
	collector.RecordRequest(50*time.Millisecond, true)

	err := mgr.SaveState("test-server", os.Getpid(), collector)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = runPs(mgr, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "test-server")
	assert.Contains(t, output, "SERVER")
	assert.Contains(t, output, "PID")
	assert.Contains(t, output, "UPTIME")
}

func TestPsCommand_MultipleInstances(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := state.NewManager(tmpDir)

	// Create multiple states
	for i := 1; i <= 3; i++ {
		collector := metrics.NewCollector()
		err := mgr.SaveState("server-"+string(rune('0'+i)), os.Getpid(), collector)
		require.NoError(t, err)
	}

	var buf bytes.Buffer
	err := runPs(mgr, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "server-1")
	assert.Contains(t, output, "server-2")
	assert.Contains(t, output, "server-3")
}

func TestPsCommand_WithMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := state.NewManager(tmpDir)

	collector := metrics.NewCollector()
	collector.RecordRequest(50*time.Millisecond, true)
	collector.RecordRequest(100*time.Millisecond, false)
	collector.RecordRestart("crash")

	err := mgr.SaveState("metrics-server", os.Getpid(), collector)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = runPs(mgr, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "metrics-server")
	// Should show request count
	assert.Contains(t, output, "2")
}

func TestPsCommand_StaleProcesses(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := state.NewManager(tmpDir)

	// Create state for dead process
	collector := metrics.NewCollector()
	err := mgr.SaveState("dead-server", 999999, collector)
	require.NoError(t, err)

	// Create state for alive process
	err = mgr.SaveState("alive-server", os.Getpid(), collector)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = runPs(mgr, &buf)
	require.NoError(t, err)

	output := buf.String()
	// Should only show alive process after cleanup
	assert.Contains(t, output, "alive-server")
}

func TestPsCommand_Formatting(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := state.NewManager(tmpDir)

	collector := metrics.NewCollector()
	for i := 0; i < 10; i++ {
		collector.RecordRequest(50*time.Millisecond, true)
	}

	err := mgr.SaveState("format-test", os.Getpid(), collector)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = runPs(mgr, &buf)
	require.NoError(t, err)

	// Check table structure
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	assert.GreaterOrEqual(t, len(lines), 3, "Should have header + data + footer")
}
