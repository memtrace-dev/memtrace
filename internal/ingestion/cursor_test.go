package ingestion

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImportCursorRules_NoFile(t *testing.T) {
	results, err := ImportCursorRules(t.TempDir())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("want 0 results, got %d", len(results))
	}
}

func TestImportCursorRules_WithFile(t *testing.T) {
	dir := t.TempDir()
	content := "Use kebab-case for all API routes.\n\nAll database queries must use parameterized statements.\n\nPrefer composition over inheritance."
	os.WriteFile(filepath.Join(dir, ".cursorrules"), []byte(content), 0644)

	results, err := ImportCursorRules(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("want 3 results (one per paragraph), got %d", len(results))
	}
	for _, r := range results {
		if r.Type != "convention" {
			t.Errorf("type: want convention, got %s", r.Type)
		}
		if r.Source != "import" {
			t.Errorf("source: want import, got %s", r.Source)
		}
	}
}
