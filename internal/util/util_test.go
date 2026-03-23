package util

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// --- FindProjectRoot ---

func TestFindProjectRoot_WithMemtraceDir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".memtrace"), 0755); err != nil {
		t.Fatal(err)
	}

	// Searching from the root itself
	got := FindProjectRoot(root)
	if got != root {
		t.Errorf("want %s, got %s", root, got)
	}

	// Searching from a subdirectory should walk up
	sub := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	got = FindProjectRoot(sub)
	if got != root {
		t.Errorf("from subdir: want %s, got %s", root, got)
	}
}

func TestFindProjectRoot_WithGitDir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	sub := filepath.Join(root, "src", "pkg")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}

	got := FindProjectRoot(sub)
	if got != root {
		t.Errorf("want %s, got %s", root, got)
	}
}

func TestFindProjectRoot_MemtracePreferredOverGit(t *testing.T) {
	// .git at root, .memtrace in subdir — should return the subdir level
	outer := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outer, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	inner := filepath.Join(outer, "inner")
	if err := os.MkdirAll(filepath.Join(inner, ".memtrace"), 0755); err != nil {
		t.Fatal(err)
	}

	got := FindProjectRoot(inner)
	if got != inner {
		t.Errorf("want %s (memtrace wins), got %s", inner, got)
	}
}

func TestFindProjectRoot_NotFound(t *testing.T) {
	// Isolated temp dir with nothing in it
	dir := t.TempDir()
	got := FindProjectRoot(dir)
	// May or may not find an ancestor .git — only assert it doesn't panic
	_ = got
}

// --- GetProjectDbPath ---

func TestGetProjectDbPath(t *testing.T) {
	root := "/some/project"
	want := "/some/project/.memtrace/memtrace.db"
	got := GetProjectDbPath(root)
	if got != want {
		t.Errorf("want %s, got %s", want, got)
	}
}

// --- GetConfigDir / GetConfigPath ---

func TestGetConfigDir_UsesHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got := GetConfigDir()
	want := filepath.Join(home, ".config", "memtrace")
	if got != want {
		t.Errorf("want %s, got %s", want, got)
	}
}

func TestGetConfigPath_UnderConfigDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	want := filepath.Join(home, ".config", "memtrace", "config.json")
	if got := GetConfigPath(); got != want {
		t.Errorf("want %s, got %s", want, got)
	}
}

// --- GetProjectConfig ---

func TestGetProjectConfig_FileAbsent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := GetProjectConfig()
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if len(cfg.Projects) != 0 {
		t.Errorf("expected empty projects map, got %d entries", len(cfg.Projects))
	}
}

func TestGetProjectConfig_ValidFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfgDir := filepath.Join(home, ".config", "memtrace")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	data := `{"projects":{"/my/proj":{"id":"abc","name":"myproj","created_at":"2024-01-01T00:00:00Z"}}}`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := GetProjectConfig()
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	entry, ok := cfg.Projects["/my/proj"]
	if !ok {
		t.Fatal("expected project entry")
	}
	if entry.ID != "abc" {
		t.Errorf("want id=abc, got %s", entry.ID)
	}
	if entry.Name != "myproj" {
		t.Errorf("want name=myproj, got %s", entry.Name)
	}
}

func TestGetProjectConfig_MalformedJSON(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfgDir := filepath.Join(home, ".config", "memtrace")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := GetProjectConfig()
	if cfg == nil {
		t.Fatal("expected non-nil fallback config")
	}
	if len(cfg.Projects) != 0 {
		t.Error("expected empty projects map on parse error")
	}
}

// --- SaveProjectConfig ---

func TestSaveProjectConfig_RoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := &ProjectConfig{
		Projects: map[string]ProjectEntry{
			"/proj/a": {ID: "id1", Name: "project-a", CreatedAt: "2024-06-01T00:00:00Z"},
		},
	}

	if err := SaveProjectConfig(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Read the raw file and verify it's valid JSON
	data, err := os.ReadFile(GetConfigPath())
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("invalid JSON written: %v", err)
	}

	// Read via helper and verify round-trip
	got := GetProjectConfig()
	entry, ok := got.Projects["/proj/a"]
	if !ok {
		t.Fatal("project entry missing after round-trip")
	}
	if entry.ID != "id1" {
		t.Errorf("want id=id1, got %s", entry.ID)
	}
}

func TestSaveProjectConfig_CreatesDirectories(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Config dir does not exist yet
	cfg := &ProjectConfig{Projects: make(map[string]ProjectEntry)}
	if err := SaveProjectConfig(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(GetConfigPath()); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

// --- GenerateID ---

func TestGenerateID_NonEmpty(t *testing.T) {
	id := GenerateID()
	if id == "" {
		t.Error("expected non-empty ID")
	}
}

func TestGenerateID_Unique(t *testing.T) {
	ids := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		id := GenerateID()
		if ids[id] {
			t.Fatalf("duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestGenerateID_Length(t *testing.T) {
	// ULIDs are always 26 characters
	id := GenerateID()
	if len(id) != 26 {
		t.Errorf("expected ULID length 26, got %d (%s)", len(id), id)
	}
}
