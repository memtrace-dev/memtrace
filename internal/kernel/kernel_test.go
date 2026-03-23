package kernel

import (
	"path/filepath"
	"testing"

	"github.com/memtrace-dev/memtrace/internal/types"
)

func setupTestKernel(t *testing.T) *MemoryKernel {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	k := New(dbPath, "test-project")
	if err := k.Open(); err != nil {
		t.Fatalf("open kernel: %v", err)
	}
	t.Cleanup(func() { k.Close() })
	return k
}

func TestKernel_Save_Defaults(t *testing.T) {
	k := setupTestKernel(t)

	mem, err := k.Save(types.MemorySaveInput{
		Content: "We use PostgreSQL as the main database",
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	if mem.ID == "" {
		t.Error("expected non-empty ID")
	}
	if mem.Type != types.MemoryTypeFact {
		t.Errorf("type: want fact, got %s", mem.Type)
	}
	if mem.Source != types.MemorySourceUser {
		t.Errorf("source: want user, got %s", mem.Source)
	}
	if mem.Confidence != 1.0 {
		t.Errorf("confidence: want 1.0, got %f", mem.Confidence)
	}
	if mem.Summary == "" {
		t.Error("expected auto-generated summary")
	}
	if mem.ProjectID != "test-project" {
		t.Errorf("project_id: want test-project, got %s", mem.ProjectID)
	}
	if mem.Status != types.MemoryStatusActive {
		t.Errorf("status: want active, got %s", mem.Status)
	}
}

func TestKernel_Save_ExplicitType(t *testing.T) {
	k := setupTestKernel(t)

	mem, err := k.Save(types.MemorySaveInput{
		Content: "We chose JWT over sessions",
		Type:    types.MemoryTypeDecision,
		Tags:    []string{"auth"},
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if mem.Type != types.MemoryTypeDecision {
		t.Errorf("type: want decision, got %s", mem.Type)
	}
	if len(mem.Tags) != 1 || mem.Tags[0] != "auth" {
		t.Errorf("tags: want [auth], got %v", mem.Tags)
	}
}

func TestKernel_Get(t *testing.T) {
	k := setupTestKernel(t)

	saved, _ := k.Save(types.MemorySaveInput{Content: "test memory"})

	got, err := k.Get(saved.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("expected memory, got nil")
	}
	if got.ID != saved.ID {
		t.Errorf("ID mismatch: want %s, got %s", saved.ID, got.ID)
	}
}

func TestKernel_Get_NotFound(t *testing.T) {
	k := setupTestKernel(t)
	got, err := k.Get("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for missing ID")
	}
}

func TestKernel_Delete(t *testing.T) {
	k := setupTestKernel(t)
	saved, _ := k.Save(types.MemorySaveInput{Content: "to be deleted"})

	deleted, err := k.Delete(saved.ID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !deleted {
		t.Error("expected deleted=true")
	}

	got, _ := k.Get(saved.ID)
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestKernel_Update_Status(t *testing.T) {
	k := setupTestKernel(t)
	saved, _ := k.Save(types.MemorySaveInput{Content: "active memory"})

	archived := types.MemoryStatusArchived
	updated, err := k.Update(saved.ID, types.MemoryUpdateInput{Status: &archived})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Status != types.MemoryStatusArchived {
		t.Errorf("status: want archived, got %s", updated.Status)
	}
}

func TestKernel_List(t *testing.T) {
	k := setupTestKernel(t)
	k.Save(types.MemorySaveInput{Content: "memory 1", Type: types.MemoryTypeDecision})
	k.Save(types.MemorySaveInput{Content: "memory 2", Type: types.MemoryTypeFact})
	k.Save(types.MemorySaveInput{Content: "memory 3", Type: types.MemoryTypeDecision})

	all, err := k.List(types.ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("want 3, got %d", len(all))
	}

	decisions, err := k.List(types.ListOptions{Limit: 10, Type: types.MemoryTypeDecision})
	if err != nil {
		t.Fatalf("list decisions: %v", err)
	}
	if len(decisions) != 2 {
		t.Errorf("want 2 decisions, got %d", len(decisions))
	}
}

func TestKernel_Recall(t *testing.T) {
	k := setupTestKernel(t)

	k.Save(types.MemorySaveInput{
		Content: "We use PostgreSQL as the primary database",
		Type:    types.MemoryTypeDecision,
		Tags:    []string{"database"},
	})
	k.Save(types.MemorySaveInput{
		Content: "All API routes use kebab-case naming convention",
		Type:    types.MemoryTypeConvention,
		Tags:    []string{"api"},
	})

	results, err := k.Recall(types.MemoryRecallInput{
		Query: "database",
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}
	if results[0].Memory.Content != "We use PostgreSQL as the primary database" {
		t.Errorf("top result mismatch: %s", results[0].Memory.Content)
	}
}

func TestKernel_Recall_Empty(t *testing.T) {
	k := setupTestKernel(t)

	results, err := k.Recall(types.MemoryRecallInput{Query: "anything"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("want 0 results on empty store, got %d", len(results))
	}
}

func TestKernel_Recall_LimitEnforced(t *testing.T) {
	k := setupTestKernel(t)
	for i := 0; i < 20; i++ {
		k.Save(types.MemorySaveInput{Content: "database memory repeated"})
	}

	results, err := k.Recall(types.MemoryRecallInput{Query: "database", Limit: 5})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(results) > 5 {
		t.Errorf("want ≤5 results, got %d", len(results))
	}
}

func TestKernel_Recall_UpdatesAccessCount(t *testing.T) {
	k := setupTestKernel(t)
	saved, _ := k.Save(types.MemorySaveInput{Content: "access tracking test memory"})

	k.Recall(types.MemoryRecallInput{Query: "access tracking", Limit: 5})

	got, _ := k.Get(saved.ID)
	if got.AccessCount == 0 {
		t.Error("expected access_count > 0 after recall")
	}
}

func TestKernel_Count(t *testing.T) {
	k := setupTestKernel(t)
	k.Save(types.MemorySaveInput{Content: "d1", Type: types.MemoryTypeDecision})
	k.Save(types.MemorySaveInput{Content: "d2", Type: types.MemoryTypeDecision})
	k.Save(types.MemorySaveInput{Content: "f1", Type: types.MemoryTypeFact})

	n, err := k.Count(types.MemoryTypeDecision, "")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 2 {
		t.Errorf("want 2, got %d", n)
	}
}
