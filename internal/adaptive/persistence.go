package adaptive

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LearningData represents persistent learning state for a server.
type LearningData struct {
	ServerName   string        `json:"server_name"`
	Observations []Observation `json:"observations"`
	LastAnalyzed time.Time     `json:"last_analyzed"`
}

// Save persists learning data to disk.
// Storage location: <baseDir>/learning/{server_name}.json
// Default baseDir: ~/.hydra
func Save(data *LearningData, serverName string, baseDir string) error {
	learningDir := filepath.Join(baseDir, "learning")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(learningDir, 0755); err != nil {
		return fmt.Errorf("failed to create learning directory: %w", err)
	}

	// Marshal data
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal learning data: %w", err)
	}

	// Write file
	filePath := filepath.Join(learningDir, serverName+".json")
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write learning file: %w", err)
	}

	return nil
}

// Load retrieves learning data from disk.
// Returns nil if file doesn't exist (not an error).
func Load(serverName string, baseDir string) (*LearningData, error) {
	filePath := filepath.Join(baseDir, "learning", serverName+".json")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil // Not an error, just no data yet
	}

	// Read file
	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read learning file: %w", err)
	}

	// Unmarshal data
	var data LearningData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to parse learning data: %w", err)
	}

	return &data, nil
}
