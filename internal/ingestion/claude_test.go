package ingestion

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFrontmatter_WithFrontmatter(t *testing.T) {
	content := "---\nname: test\ndescription: a test memory\ntype: decision\n---\nThis is the body."
	fm, body := parseFrontmatter(content)

	if fm["name"] != "test" {
		t.Errorf("name: want 'test', got %q", fm["name"])
	}
	if fm["description"] != "a test memory" {
		t.Errorf("description: want 'a test memory', got %q", fm["description"])
	}
	if fm["type"] != "decision" {
		t.Errorf("type: want 'decision', got %q", fm["type"])
	}
	if body != "This is the body." {
		t.Errorf("body: want 'This is the body.', got %q", body)
	}
}

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	content := "Just plain content with no frontmatter."
	fm, body := parseFrontmatter(content)

	if len(fm) != 0 {
		t.Errorf("expected empty frontmatter, got %v", fm)
	}
	if body != content {
		t.Errorf("body mismatch")
	}
}

func TestImportClaudeMemories_NoDir(t *testing.T) {
	// Should return empty slice, not an error
	results, err := ImportClaudeMemories("/nonexistent/project/path")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("want 0 results, got %d", len(results))
	}
}

func TestImportClaudeMemories_WithFiles(t *testing.T) {
	// Build a fake Claude memory directory structure
	home := t.TempDir()
	projectRoot := "/fake/project"
	projectKey := "-fake-project"
	memDir := filepath.Join(home, ".claude", "projects", projectKey, "memory")
	if err := os.MkdirAll(memDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write a fake memory file
	content := "---\nname: auth decision\ndescription: JWT decision\ntype: decision\n---\nWe use JWT with RS256."
	os.WriteFile(filepath.Join(memDir, "auth.md"), []byte(content), 0644)

	// Temporarily override HOME
	t.Setenv("HOME", home)

	results, err := ImportClaudeMemories(projectRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	if results[0].Content != "We use JWT with RS256." {
		t.Errorf("content: %q", results[0].Content)
	}
	if results[0].Summary != "JWT decision" {
		t.Errorf("summary: want 'JWT decision', got %q", results[0].Summary)
	}
}
