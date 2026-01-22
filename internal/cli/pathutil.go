package cli

import (
	"os"
	"path/filepath"
)

// expandHome replaces a leading "~" with the current user's home directory.
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
