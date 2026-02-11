package adaptive

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	serverName := "test-server"

	data := &LearningData{
		ServerName: serverName,
		Observations: []Observation{
			{
				Timestamp: time.Now(),
				Event:     "restart",
				Context:   map[string]interface{}{"count": 5},
			},
		},
		LastAnalyzed: time.Now(),
	}

	// Save
	err := Save(data, serverName, tmpDir)
	require.NoError(t, err)

	// Load
	loaded, err := Load(serverName, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, serverName, loaded.ServerName)
	assert.Len(t, loaded.Observations, 1)
	assert.Equal(t, "restart", loaded.Observations[0].Event)
}

func TestLoad_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()

	loaded, err := Load("nonexistent", tmpDir)
	require.NoError(t, err) // Should not error, just return nil
	assert.Nil(t, loaded)
}

func TestSave_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	learningDir := filepath.Join(tmpDir, "learning")

	// Learning dir doesn't exist yet
	_, err := os.Stat(learningDir)
	require.Error(t, err)

	data := &LearningData{
		ServerName:   "test",
		Observations: []Observation{},
		LastAnalyzed: time.Now(),
	}

	// Save should create directory
	err = Save(data, "test", tmpDir)
	require.NoError(t, err)

	// Directory should now exist
	_, err = os.Stat(learningDir)
	require.NoError(t, err)
}

func TestLoad_CorruptedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	learningDir := filepath.Join(tmpDir, "learning")
	require.NoError(t, os.MkdirAll(learningDir, 0755))

	// Write corrupted JSON
	serverName := "corrupted"
	filePath := filepath.Join(learningDir, serverName+".json")
	require.NoError(t, os.WriteFile(filePath, []byte("not json"), 0644))

	// Load should return error for corrupted data
	loaded, err := Load(serverName, tmpDir)
	require.Error(t, err)
	assert.Nil(t, loaded)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestSaveAndLoad_MultipleServers(t *testing.T) {
	tmpDir := t.TempDir()

	// Save data for multiple servers
	servers := []string{"server1", "server2", "server3"}
	for _, name := range servers {
		data := &LearningData{
			ServerName: name,
			Observations: []Observation{
				{
					Timestamp: time.Now(),
					Event:     "test",
					Context:   map[string]interface{}{"server": name},
				},
			},
			LastAnalyzed: time.Now(),
		}

		err := Save(data, name, tmpDir)
		require.NoError(t, err)
	}

	// Load each server's data
	for _, name := range servers {
		loaded, err := Load(name, tmpDir)
		require.NoError(t, err)
		require.NotNil(t, loaded)

		assert.Equal(t, name, loaded.ServerName)
		assert.Equal(t, name, loaded.Observations[0].Context["server"])
	}
}

func TestSave_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	serverName := "overwrite-test"

	// Save initial data
	data1 := &LearningData{
		ServerName:   serverName,
		Observations: []Observation{{Event: "first"}},
		LastAnalyzed: time.Now(),
	}
	err := Save(data1, serverName, tmpDir)
	require.NoError(t, err)

	// Overwrite with new data
	data2 := &LearningData{
		ServerName:   serverName,
		Observations: []Observation{{Event: "second"}, {Event: "third"}},
		LastAnalyzed: time.Now(),
	}
	err = Save(data2, serverName, tmpDir)
	require.NoError(t, err)

	// Load should return latest data
	loaded, err := Load(serverName, tmpDir)
	require.NoError(t, err)

	assert.Len(t, loaded.Observations, 2)
	assert.Equal(t, "second", loaded.Observations[0].Event)
	assert.Equal(t, "third", loaded.Observations[1].Event)
}
