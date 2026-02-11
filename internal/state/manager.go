package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/proxikal/hydra/internal/metrics"
)

// State represents the persisted state of a running server instance.
type State struct {
	ServerName  string             `json:"server_name"`
	PID         int                `json:"pid"`
	StartTime   time.Time          `json:"start_time"`
	LastUpdated time.Time          `json:"last_updated"`
	Metrics     *metrics.Snapshot  `json:"metrics"`
}

// Manager handles state file persistence.
type Manager struct {
	stateDir string
	mu       sync.RWMutex
}

// NewManager creates a new state manager.
func NewManager(stateDir string) *Manager {
	return &Manager{
		stateDir: stateDir,
	}
}

// SaveState persists the current state of a server instance.
func (m *Manager) SaveState(serverName string, pid int, collector metrics.MetricsCollector) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure state directory exists
	if err := os.MkdirAll(m.stateDir, 0755); err != nil {
		return err
	}

	snapshot := collector.Snapshot()

	state := State{
		ServerName:  serverName,
		PID:         pid,
		StartTime:   snapshot.StartTime,
		LastUpdated: time.Now(),
		Metrics:     snapshot,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	statePath := filepath.Join(m.stateDir, serverName+".json")
	return os.WriteFile(statePath, data, 0644)
}

// LoadState loads the persisted state for a server.
func (m *Manager) LoadState(serverName string) (*State, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statePath := filepath.Join(m.stateDir, serverName+".json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// DeleteState removes the persisted state for a server.
func (m *Manager) DeleteState(serverName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	statePath := filepath.Join(m.stateDir, serverName+".json")
	err := os.Remove(statePath)
	if err != nil && os.IsNotExist(err) {
		// Not an error if file doesn't exist
		return nil
	}
	return err
}

// ListAll returns all persisted server states.
func (m *Manager) ListAll() ([]State, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Ensure directory exists
	if err := os.MkdirAll(m.stateDir, 0755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(m.stateDir)
	if err != nil {
		return nil, err
	}

	var states []State
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		serverName := entry.Name()[:len(entry.Name())-5] // Remove .json extension
		state, err := m.loadStateNoLock(serverName)
		if err != nil {
			// Skip invalid state files
			continue
		}

		states = append(states, *state)
	}

	return states, nil
}

// loadStateNoLock loads state without acquiring lock (internal use).
func (m *Manager) loadStateNoLock(serverName string) (*State, error) {
	statePath := filepath.Join(m.stateDir, serverName+".json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}
