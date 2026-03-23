package kernel

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/memtrace-dev/memtrace/internal/types"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := ApplySchema(db); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func makeMemory(id, projectID string, memType types.MemoryType) *types.Memory {
	now := time.Now().UTC()
	return &types.Memory{
		ID:          id,
		Type:        memType,
		Content:     "test content for " + id,
		Summary:     "summary " + id,
		Source:      types.MemorySourceUser,
		Confidence:  1.0,
		ProjectID:   projectID,
		FilePaths:   []string{},
		Tags:        []string{"test"},
		Status:      types.MemoryStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
		AccessCount: 0,
	}
}

func TestStore_Insert_FindByID(t *testing.T) {
	store := NewStore(setupTestDB(t))
	m := makeMemory("01TEST001", "proj1", types.MemoryTypeDecision)

	if err := store.Insert(m); err != nil {
		t.Fatalf("insert: %v", err)
	}

	got, err := store.FindByID(m.ID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got == nil {
		t.Fatal("expected memory, got nil")
	}
	if got.ID != m.ID {
		t.Errorf("ID: want %s, got %s", m.ID, got.ID)
	}
	if got.Type != m.Type {
		t.Errorf("Type: want %s, got %s", m.Type, got.Type)
	}
	if got.Content != m.Content {
		t.Errorf("Content mismatch")
	}
	if len(got.Tags) != 1 || got.Tags[0] != "test" {
		t.Errorf("Tags: want [test], got %v", got.Tags)
	}
}

func TestStore_FindByID_NotFound(t *testing.T) {
	store := NewStore(setupTestDB(t))
	got, err := store.FindByID("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for missing ID")
	}
}

func TestStore_DeleteByID(t *testing.T) {
	store := NewStore(setupTestDB(t))
	m := makeMemory("01TESTDEL", "proj1", types.MemoryTypeFact)
	store.Insert(m)

	deleted, err := store.DeleteByID(m.ID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !deleted {
		t.Error("expected deleted=true")
	}

	got, _ := store.FindByID(m.ID)
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestStore_DeleteByID_NotFound(t *testing.T) {
	store := NewStore(setupTestDB(t))
	deleted, err := store.DeleteByID("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted {
		t.Error("expected deleted=false for missing ID")
	}
}

func TestStore_Update(t *testing.T) {
	store := NewStore(setupTestDB(t))
	m := makeMemory("01TESTUPD", "proj1", types.MemoryTypeFact)
	store.Insert(m)

	newStatus := types.MemoryStatusArchived
	newContent := "updated content"
	err := store.Update(m.ID, types.MemoryUpdateInput{
		Status:  &newStatus,
		Content: &newContent,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	got, _ := store.FindByID(m.ID)
	if got.Status != types.MemoryStatusArchived {
		t.Errorf("status: want archived, got %s", got.Status)
	}
	if got.Content != newContent {
		t.Errorf("content: want %q, got %q", newContent, got.Content)
	}
}

func TestStore_List(t *testing.T) {
	store := NewStore(setupTestDB(t))
	store.Insert(makeMemory("01LIST001", "proj1", types.MemoryTypeDecision))
	store.Insert(makeMemory("01LIST002", "proj1", types.MemoryTypeConvention))
	store.Insert(makeMemory("01LIST003", "proj2", types.MemoryTypeFact)) // different project

	memories, err := store.List(types.ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	// Should return all active memories (no project filter in List)
	if len(memories) != 3 {
		t.Errorf("want 3 memories, got %d", len(memories))
	}
}

func TestStore_List_FilterByType(t *testing.T) {
	store := NewStore(setupTestDB(t))
	store.Insert(makeMemory("01LTYPE001", "proj1", types.MemoryTypeDecision))
	store.Insert(makeMemory("01LTYPE002", "proj1", types.MemoryTypeConvention))
	store.Insert(makeMemory("01LTYPE003", "proj1", types.MemoryTypeDecision))

	memories, err := store.List(types.ListOptions{
		Limit: 10,
		Type:  types.MemoryTypeDecision,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(memories) != 2 {
		t.Errorf("want 2 decisions, got %d", len(memories))
	}
}

func TestStore_Count(t *testing.T) {
	store := NewStore(setupTestDB(t))
	store.Insert(makeMemory("01CNT001", "proj1", types.MemoryTypeDecision))
	store.Insert(makeMemory("01CNT002", "proj1", types.MemoryTypeDecision))
	store.Insert(makeMemory("01CNT003", "proj1", types.MemoryTypeFact))

	n, err := store.Count(types.MemoryTypeDecision, "")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 2 {
		t.Errorf("want 2, got %d", n)
	}
}

func TestStore_SearchFTS(t *testing.T) {
	store := NewStore(setupTestDB(t))

	m1 := makeMemory("01FTS001", "proj1", types.MemoryTypeDecision)
	m1.Content = "We use PostgreSQL for the main database"
	m1.Summary = "PostgreSQL decision"
	store.Insert(m1)

	m2 := makeMemory("01FTS002", "proj1", types.MemoryTypeConvention)
	m2.Content = "All API routes use kebab-case naming"
	m2.Summary = "API naming convention"
	store.Insert(m2)

	results, err := store.SearchFTS("PostgreSQL", "proj1", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}

	got, _ := store.FindByRowID(results[0].RowID)
	if got == nil || got.ID != m1.ID {
		t.Errorf("expected m1, got %v", got)
	}
}

func TestStore_SearchFTS_NoResults(t *testing.T) {
	store := NewStore(setupTestDB(t))
	results, err := store.SearchFTS("nonexistentterm12345", "proj1", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("want 0 results, got %d", len(results))
	}
}

func TestStore_TouchAccess(t *testing.T) {
	store := NewStore(setupTestDB(t))
	m := makeMemory("01TOUCH01", "proj1", types.MemoryTypeFact)
	store.Insert(m)

	now := time.Now().UTC()
	if err := store.TouchAccess(m.ID, now); err != nil {
		t.Fatalf("touch: %v", err)
	}

	got, _ := store.FindByID(m.ID)
	if got.AccessCount != 1 {
		t.Errorf("access_count: want 1, got %d", got.AccessCount)
	}
	if got.AccessedAt == nil {
		t.Error("expected accessed_at to be set")
	}
}
