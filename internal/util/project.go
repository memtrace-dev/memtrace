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

// EmbedConfig holds optional embedding API settings persisted in the global config.
// Environment variables always take precedence over these values.
type EmbedConfig struct {
	Key   string `json:"key,omitempty"`
	URL   string `json:"url,omitempty"`
	Model string `json:"model,omitempty"`
}

// ProjectConfig represents the contents of the global config.json.
type ProjectConfig struct {
	Projects map[string]ProjectEntry `json:"projects"`
	Embed    EmbedConfig             `json:"embed,omitempty"`
}

// GetProjectConfig reads and returns the global config.
// Returns an empty config (not an error) if the file doesn't exist yet.
// If a legacy config exists at ~/.config/memtrace/config.json and contains
// projects not present in the primary config, they are merged in and the
// merged result is saved to the primary location.
func GetProjectConfig() *ProjectConfig {
	primary := GetConfigPath()
	cfg := readConfig(primary)

	// Check for a legacy config at ~/.config/memtrace/config.json (used before
	// os.UserConfigDir() was adopted). Merge any projects that are missing from
	// the primary config, then persist so future reads are self-contained.
	legacy := legacyConfigPath()
	if legacy != "" && legacy != primary {
		if lcfg := readConfig(legacy); len(lcfg.Projects) > 0 {
			merged := false
			for k, v := range lcfg.Projects {
				if _, exists := cfg.Projects[k]; !exists {
					cfg.Projects[k] = v
					merged = true
				}
			}
			if merged {
				_ = os.MkdirAll(GetConfigDir(), 0755)
				_ = SaveProjectConfig(cfg)
			}
		}
	}

	return cfg
}

// readConfig reads and parses a config file. Returns an empty config on any error.
func readConfig(path string) *ProjectConfig {
	data, err := os.ReadFile(path)
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

// legacyConfigPath returns the old ~/.config/memtrace/config.json path that was
// used before os.UserConfigDir() was adopted. Returns "" if it cannot be determined.
func legacyConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "memtrace", "config.json")
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
