package util

import (
	"os"
	"path/filepath"
)

// GetConfigDir returns the OS-appropriate config directory for memtrace.
//   - macOS:   $HOME/Library/Application Support/memtrace
//   - Linux:   $XDG_CONFIG_HOME/memtrace  (fallback: $HOME/.config/memtrace)
//   - Windows: %AppData%\memtrace
func GetConfigDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		// Last-resort fallback if home dir can't be resolved.
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "memtrace")
}

// GetConfigPath returns the full path to config.json.
func GetConfigPath() string {
	return filepath.Join(GetConfigDir(), "config.json")
}
