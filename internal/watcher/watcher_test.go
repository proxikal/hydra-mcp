package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWatcher_SingleFileChange(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	log := logger.New("error")
	debounceTime := 100 * time.Millisecond
	batchWindow := 200 * time.Millisecond

	w, err := New([]string{tmpDir}, []string{}, debounceTime, batchWindow, log)
	require.NoError(t, err)

	err = w.Start()
	require.NoError(t, err)
	defer func() { _ = w.Stop() }()

	// Create file
	err = os.WriteFile(testFile, []byte("hello"), 0644)
	require.NoError(t, err)

	// Should get one event
	select {
	case event := <-w.Events():
		assert.NotEmpty(t, event.Path)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestWatcher_RapidChangesBatched(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	log := logger.New("error")
	debounceTime := 200 * time.Millisecond
	batchWindow := 300 * time.Millisecond

	w, err := New([]string{tmpDir}, []string{}, debounceTime, batchWindow, log)
	require.NoError(t, err)

	err = w.Start()
	require.NoError(t, err)
	defer func() { _ = w.Stop() }()

	// Make rapid changes
	for i := 0; i < 5; i++ {
		err = os.WriteFile(testFile, []byte("change"), 0644)
		require.NoError(t, err)
		time.Sleep(20 * time.Millisecond)
	}

	// Should get a small number of batched events (fsnotify can fire multiple events per write)
	eventCount := 0
	timeout := time.After(1 * time.Second)

	for {
		select {
		case <-w.Events():
			eventCount++
		case <-timeout:
			// Should have received 1-2 events (batched, fsnotify may fire CREATE+WRITE)
			assert.LessOrEqual(t, eventCount, 3, "rapid changes should be batched into few events")
			assert.Greater(t, eventCount, 0, "should have received at least one event")
			return
		}
	}
}

func TestWatcher_StopCleanly(t *testing.T) {
	tmpDir := t.TempDir()
	log := logger.New("error")

	w, err := New([]string{tmpDir}, []string{}, 100*time.Millisecond, 200*time.Millisecond, log)
	require.NoError(t, err)

	err = w.Start()
	require.NoError(t, err)

	err = w.Stop()
	assert.NoError(t, err)

	// Events channel should be closed
	_, ok := <-w.Events()
	assert.False(t, ok, "events channel should be closed after stop")
}

func TestWatcher_GitignorePatterns(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	ignoredFile := filepath.Join(tmpDir, "node_modules", "package.json")

	// Create node_modules dir
	err := os.MkdirAll(filepath.Join(tmpDir, "node_modules"), 0755)
	require.NoError(t, err)

	log := logger.New("error")

	// Ignore node_modules
	w, err := New([]string{tmpDir}, []string{"node_modules"}, 100*time.Millisecond, 200*time.Millisecond, log)
	require.NoError(t, err)

	err = w.Start()
	require.NoError(t, err)
	defer func() { _ = w.Stop() }()

	// Change ignored file
	err = os.WriteFile(ignoredFile, []byte("ignored"), 0644)
	require.NoError(t, err)

	// Change non-ignored file
	err = os.WriteFile(testFile, []byte("not ignored"), 0644)
	require.NoError(t, err)

	// Should only get event for non-ignored file
	select {
	case event := <-w.Events():
		assert.Contains(t, event.Path, "test.txt")
		assert.NotContains(t, event.Path, "node_modules")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestWatcher_InvalidPath(t *testing.T) {
	log := logger.New("error")

	w, err := New([]string{"/this/path/does/not/exist"}, []string{}, 100*time.Millisecond, 200*time.Millisecond, log)
	require.NoError(t, err) // New() should succeed

	// Start() should fail when trying to watch non-existent path
	err = w.Start()
	assert.Error(t, err)
}

func TestWatcher_BatchWindowExpiration(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	log := logger.New("error")
	debounceTime := 50 * time.Millisecond
	batchWindow := 150 * time.Millisecond // Force batch after this time

	w, err := New([]string{tmpDir}, []string{}, debounceTime, batchWindow, log)
	require.NoError(t, err)

	err = w.Start()
	require.NoError(t, err)
	defer func() { _ = w.Stop() }()

	// Create file
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Wait for batch window to force event
	select {
	case event := <-w.Events():
		assert.NotEmpty(t, event.Path)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for batched event")
	}
}

func TestWatcher_RecursiveDirectories(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	subDir1 := filepath.Join(tmpDir, "level1")
	subDir2 := filepath.Join(subDir1, "level2")
	require.NoError(t, os.MkdirAll(subDir2, 0755))

	log := logger.New("error")
	w, err := New([]string{tmpDir}, []string{}, 50*time.Millisecond, 100*time.Millisecond, log)
	require.NoError(t, err)

	err = w.Start()
	require.NoError(t, err)
	defer func() { _ = w.Stop() }()

	// Create file in deep subdirectory
	testFile := filepath.Join(subDir2, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Wait for event
	select {
	case event := <-w.Events():
		assert.Contains(t, event.Path, "level2")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for file change event in subdirectory")
	}
}

func TestWatcher_DynamicDirectoryCreation(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	log := logger.New("error")
	w, err := New([]string{tmpDir}, []string{}, 50*time.Millisecond, 100*time.Millisecond, log)
	require.NoError(t, err)

	err = w.Start()
	require.NoError(t, err)
	defer func() { _ = w.Stop() }()

	// Create new subdirectory after watcher started
	newDir := filepath.Join(tmpDir, "newdir")
	require.NoError(t, os.Mkdir(newDir, 0755))

	// Wait for directory creation event
	select {
	case <-w.Events():
		// Got directory creation event
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for directory creation event")
	}

	// Create file in newly created directory
	testFile := filepath.Join(newDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

	// Should receive event from file in dynamically created directory
	select {
	case event := <-w.Events():
		assert.Contains(t, event.Path, "newdir")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for file event in dynamically created directory")
	}
}
