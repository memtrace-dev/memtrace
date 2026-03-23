package ingestion

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	run("init", "-b", "main")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")

	// Commit 1: boring commit (should be skipped)
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0644)
	run("add", ".")
	run("commit", "-m", "add file.txt")

	// Commit 2: decision commit (should be included)
	os.WriteFile(filepath.Join(dir, "auth.go"), []byte("package main"), 0644)
	run("add", ".")
	run("commit", "-m", "auth: decided to use JWT over session tokens\n\nAfter evaluating security requirements, JWT with RS256 is the better choice for our stateless API architecture.")

	// Commit 3: migration commit (should be included)
	os.WriteFile(filepath.Join(dir, "db.go"), []byte("package main"), 0644)
	run("add", ".")
	run("commit", "-m", "migrated from SQLite to PostgreSQL for production")

	return dir
}

func TestImportGitHistory_NotARepo(t *testing.T) {
	results, err := ImportGitHistory(t.TempDir(), 100)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("want 0 results for non-repo, got %d", len(results))
	}
}

func TestImportGitHistory_FiltersDecisions(t *testing.T) {
	dir := initGitRepo(t)

	results, err := ImportGitHistory(dir, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 2 qualifying commits (JWT and migration), not the boring one
	if len(results) < 2 {
		t.Errorf("want ≥2 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Source != "git" {
			t.Errorf("source: want git, got %s", r.Source)
		}
		if r.Confidence != 0.7 {
			t.Errorf("confidence: want 0.7, got %f", r.Confidence)
		}
	}
}
