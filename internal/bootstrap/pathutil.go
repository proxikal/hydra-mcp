package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func expandHome(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	return filepath.Join(home, path[1:])
}

// detectClientConfigPath returns the configuration file path for a given client.
func detectClientConfigPath(client string) (string, error) {
	var path string

	switch client {
	case "claude":
		path = detectClaudeDesktopPath()
	case "cline":
		path = detectClinePath()
	case "cursor":
		path = detectCursorPath()
	case "claude-cli":
		path = detectClaudeCLIPath()
	case "continue":
		path = detectContinuePath()
	case "cursor-composer":
		path = detectCursorComposerPath()
	default:
		return "", fmt.Errorf("unknown client: %s", client)
	}

	return path, nil
}

// detectClaudeDesktopPath returns the Claude Desktop config path.
func detectClaudeDesktopPath() string {
	home, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library/Application Support/Claude/claude_desktop_config.json")
	case "windows":
		return filepath.Join(home, "AppData/Roaming/Claude/claude_desktop_config.json")
	default: // linux
		return filepath.Join(home, ".config/Claude/claude_desktop_config.json")
	}
}

// detectClinePath returns the Cline (VSCode extension) config path.
func detectClinePath() string {
	home, _ := os.UserHomeDir()

	// Try VSCode first, then VSCodium
	vscodeDir := filepath.Join(home, ".vscode/extensions")
	if _, err := os.Stat(vscodeDir); err == nil {
		return filepath.Join(home, ".vscode/mcp_settings.json")
	}

	return filepath.Join(home, ".vscodium/mcp_settings.json")
}

// detectCursorPath returns the Cursor config path.
func detectCursorPath() string {
	home, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library/Application Support/Cursor/User/globalStorage/mcp_settings.json")
	case "windows":
		return filepath.Join(home, "AppData/Roaming/Cursor/User/globalStorage/mcp_settings.json")
	default: // linux
		return filepath.Join(home, ".config/Cursor/User/globalStorage/mcp_settings.json")
	}
}

// detectClaudeCLIPath returns the claude-cli config path.
func detectClaudeCLIPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config/claude-cli/mcp_config.json")
}

// detectContinuePath returns the Continue (VSCode extension) config path.
func detectContinuePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".continue/config.json")
}

// detectCursorComposerPath returns the Cursor Composer config path.
func detectCursorComposerPath() string {
	home, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library/Application Support/Cursor/mcp_config.json")
	case "windows":
		return filepath.Join(home, "AppData/Roaming/Cursor/mcp_config.json")
	default: // linux
		return filepath.Join(home, ".config/Cursor/mcp_config.json")
	}
}

// configFileExists checks if a config file exists.
func configFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
