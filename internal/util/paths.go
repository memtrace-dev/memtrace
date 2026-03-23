package util

import (
	"os"
	"path/filepath"
)

// GetConfigDir returns ~/.config/memtrace regardless of platform.
func GetConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "memtrace")
}

// GetConfigPath returns the full path to config.json.
func GetConfigPath() string {
	return filepath.Join(GetConfigDir(), "config.json")
}
