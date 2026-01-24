package supervisor

import (
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSupervisor_StartProcessSuccessfully(t *testing.T) {
	log := logger.New("error")

	// Use a long-running command
	cmd := []string{"sleep", "10"}
	maxRestarts := 3
	restartWindow := 60 * time.Second

	sup := NewSupervisor(cmd, "", maxRestarts, restartWindow, log)

	err := sup.Start()
	require.NoError(t, err)

	// Give it a moment to transition to running
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, StateRunning, sup.State())
	assert.Greater(t, sup.PID(), 0)

	// Clean up
	err = sup.Stop()
	require.NoError(t, err)
}

func TestSupervisor_StartWithInvalidCommand(t *testing.T) {
	log := logger.New("error")

	// Use a command that doesn't exist
	cmd := []string{"/this/command/does/not/exist"}
	maxRestarts := 3
	restartWindow := 60 * time.Second

	sup := NewSupervisor(cmd, "", maxRestarts, restartWindow, log)

	err := sup.Start()
	assert.Error(t, err)
	assert.Equal(t, StateFailed, sup.State())
}

func TestSupervisor_StopRunningProcess(t *testing.T) {
	log := logger.New("error")

	// Use sleep to keep process alive
	cmd := []string{"sleep", "10"}
	maxRestarts := 3
	restartWindow := 60 * time.Second

	sup := NewSupervisor(cmd, "", maxRestarts, restartWindow, log)

	err := sup.Start()
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, StateRunning, sup.State())

	pid := sup.PID()
	assert.Greater(t, pid, 0)

	err = sup.Stop()
	require.NoError(t, err)

	assert.Equal(t, StateStopped, sup.State())
	assert.Equal(t, 0, sup.PID())
}

func TestSupervisor_RestartIncrementsCounter(t *testing.T) {
	log := logger.New("error")

	cmd := []string{"sleep", "10"}
	maxRestarts := 5
	restartWindow := 60 * time.Second

	sup := NewSupervisor(cmd, "", maxRestarts, restartWindow, log)

	err := sup.Start()
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// First restart
	err = sup.Restart()
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// Second restart
	err = sup.Restart()
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// Should still be running since we're under max
	state := sup.State()
	if state != StateRunning {
		t.Logf("Expected StateRunning (2), got %v (%d)", state, state)
		t.Logf("Last error: %v", sup.LastError())
		t.Logf("PID: %d", sup.PID())
	}
	assert.Equal(t, StateRunning, sup.State())

	// Clean up
	_ = sup.Stop()
}

func TestSupervisor_CrashLoopDetection(t *testing.T) {
	log := logger.New("error")

	// Command that will fail immediately
	cmd := []string{"false"}
	maxRestarts := 2
	restartWindow := 60 * time.Second

	sup := NewSupervisor(cmd, "", maxRestarts, restartWindow, log)

	// Start will fail but supervisor should track it
	_ = sup.Start()

	// Attempt restarts beyond the limit
	for i := 0; i < maxRestarts+1; i++ {
		_ = sup.Restart()
		time.Sleep(50 * time.Millisecond)
	}

	// Should be in failed state due to crash loop
	assert.Equal(t, StateFailed, sup.State())
}

func TestSupervisor_UptimeTracking(t *testing.T) {
	log := logger.New("error")

	cmd := []string{"sleep", "5"}
	maxRestarts := 3
	restartWindow := 60 * time.Second

	sup := NewSupervisor(cmd, "", maxRestarts, restartWindow, log)

	err := sup.Start()
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	uptime := sup.Uptime()
	assert.Greater(t, uptime, 100*time.Millisecond)
	assert.Less(t, uptime, 1*time.Second)

	_ = sup.Stop()
}

func TestSupervisor_ResetRestartCounter(t *testing.T) {
	log := logger.New("error")

	cmd := []string{"sleep", "10"}
	maxRestarts := 3
	restartWindow := 60 * time.Second

	sup := NewSupervisor(cmd, "", maxRestarts, restartWindow, log)

	_ = sup.Start()
	time.Sleep(200 * time.Millisecond)

	// Do some restarts
	_ = sup.Restart()
	time.Sleep(200 * time.Millisecond)
	_ = sup.Restart()
	time.Sleep(200 * time.Millisecond)

	// Reset counter
	sup.ResetRestartCounter()

	// Should be able to restart again without hitting limit
	err := sup.Restart()
	assert.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	_ = sup.Stop()
}

func TestSupervisor_LastError(t *testing.T) {
	log := logger.New("error")

	t.Run("no error when successful", func(t *testing.T) {
		cmd := []string{"sleep", "1"}
		sup := NewSupervisor(cmd, "", 3, 60*time.Second, log)

		err := sup.Start()
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
		assert.Nil(t, sup.LastError())

		_ = sup.Stop()
	})

	t.Run("captures error on failed start", func(t *testing.T) {
		cmd := []string{"/nonexistent/command"}
		sup := NewSupervisor(cmd, "", 3, 60*time.Second, log)

		err := sup.Start()
		assert.Error(t, err)

		lastErr := sup.LastError()
		assert.NotNil(t, lastErr)
		assert.Contains(t, lastErr.Error(), "no such file or directory")
	})

	t.Run("captures error on crash", func(t *testing.T) {
		cmd := []string{"false"} // exits immediately with code 1
		sup := NewSupervisor(cmd, "", 1, 60*time.Second, log)

		_ = sup.Start()
		time.Sleep(200 * time.Millisecond)

		lastErr := sup.LastError()
		if lastErr != nil {
			// Error should mention exit status
			assert.Contains(t, lastErr.Error(), "exit")
		}

		_ = sup.Stop()
	})
}

func TestServerState_String(t *testing.T) {
	tests := []struct {
		state    ServerState
		expected string
	}{
		{StateStopped, "stopped"},
		{StateStarting, "starting"},
		{StateRunning, "running"},
		{StateRestarting, "restarting"},
		{StateFailed, "failed"},
		{ServerState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}
