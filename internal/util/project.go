package util

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// FindProjectRoot walks up from startDir looking for .memtrace/ then .git/.
// Returns the absolute path of the project root, or "" if not found.
func FindProjectRoot(startDir string) string {
	// First pass: look for an already-initialized .memtrace/ directory
	dir := startDir
	for {
		if dirExists(filepath.Join(dir, ".memtrace")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// Second pass: look for .git/ to infer project root
	dir = startDir
	for {
		if dirExists(filepath.Join(dir, ".git")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// GetProjectDbPath returns the path to the SQLite database for a project root.
func GetProjectDbPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".memtrace", "memtrace.db")
}

// ProjectEntry represents a single project in the config file.
type ProjectEntry struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// ProjectConfig represents the contents of ~/.config/memtrace/config.json.
type ProjectConfig struct {
	Projects map[string]ProjectEntry `json:"projects"`
}

// GetProjectConfig reads and returns the global config.
// Returns an empty config (not an error) if the file doesn't exist yet.
func GetProjectConfig() *ProjectConfig {
	data, err := os.ReadFile(GetConfigPath())
	if err != nil {
		return &ProjectConfig{Projects: make(map[string]ProjectEntry)}
	}
	var cfg ProjectConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &ProjectConfig{Projects: make(map[string]ProjectEntry)}
	}
	if cfg.Projects == nil {
		cfg.Projects = make(map[string]ProjectEntry)
	}
	return &cfg
}

// SaveProjectConfig writes the config to disk, creating the directory if needed.
func SaveProjectConfig(cfg *ProjectConfig) error {
	if err := os.MkdirAll(GetConfigDir(), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(GetConfigPath(), data, 0644)
}

// dirExists returns true if the given path is an existing directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
