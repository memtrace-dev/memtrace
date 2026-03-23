package mcp

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/memtrace-dev/memtrace/internal/kernel"
	"github.com/memtrace-dev/memtrace/internal/types"
)

// --- Helpers ---

func setupServer(t *testing.T) (*server.MCPServer, *kernel.MemoryKernel) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	k := kernel.New(dbPath, "test-project")
	if err := k.Open(); err != nil {
		t.Fatalf("open kernel: %v", err)
	}
	t.Cleanup(func() { k.Close() })

	s := server.NewMCPServer("memtrace", "0.0.0", server.WithToolCapabilities(true))
	registerTools(s, k)
	return s, k
}

func callTool(t *testing.T, s *server.MCPServer, name string, args map[string]interface{}) *mcpgo.CallToolResult {
	t.Helper()
	tool := s.GetTool(name)
	if tool == nil {
		t.Fatalf("tool %q not registered", name)
	}
	req := mcpgo.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("tool %q returned error: %v", name, err)
	}
	return result
}

func resultText(t *testing.T, r *mcpgo.CallToolResult) string {
	t.Helper()
	if r == nil || len(r.Content) == 0 {
		t.Fatal("empty result")
	}
	tc, ok := r.Content[0].(mcpgo.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", r.Content[0])
	}
	return tc.Text
}

// --- extractStringSlice ---

func TestExtractStringSlice_Normal(t *testing.T) {
	args := map[string]interface{}{
		"tags": []interface{}{"auth", "api", "security"},
	}
	got := extractStringSlice(args, "tags")
	if len(got) != 3 || got[0] != "auth" || got[1] != "api" || got[2] != "security" {
		t.Errorf("unexpected result: %v", got)
	}
}

func TestExtractStringSlice_Missing(t *testing.T) {
	got := extractStringSlice(map[string]interface{}{}, "tags")
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

func TestExtractStringSlice_WrongType(t *testing.T) {
	args := map[string]interface{}{"tags": "not-a-slice"}
	got := extractStringSlice(args, "tags")
	if len(got) != 0 {
		t.Errorf("expected empty slice for wrong type, got %v", got)
	}
}

func TestExtractStringSlice_SkipsNonStrings(t *testing.T) {
	args := map[string]interface{}{
		"tags": []interface{}{"a", 42, "b", nil},
	}
	got := extractStringSlice(args, "tags")
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("expected [a b], got %v", got)
	}
}

// --- formatAge ---

func TestFormatAge_Minutes(t *testing.T) {
	got := formatAge(time.Now().Add(-30 * time.Minute))
	if !strings.HasSuffix(got, "m ago") {
		t.Errorf("unexpected format: %s", got)
	}
}

func TestFormatAge_Hours(t *testing.T) {
	got := formatAge(time.Now().Add(-5 * time.Hour))
	if !strings.HasSuffix(got, "h ago") {
		t.Errorf("unexpected format: %s", got)
	}
}

func TestFormatAge_Days(t *testing.T) {
	got := formatAge(time.Now().Add(-3 * 24 * time.Hour))
	if !strings.HasSuffix(got, "d ago") {
		t.Errorf("unexpected format: %s", got)
	}
}

func TestFormatAge_Months(t *testing.T) {
	got := formatAge(time.Now().Add(-60 * 24 * time.Hour))
	if !strings.HasSuffix(got, "mo ago") {
		t.Errorf("unexpected format: %s", got)
	}
}

// --- truncateStr ---

func TestTruncateStr_Short(t *testing.T) {
	s := "hello"
	got := truncateStr(s, 20)
	if got != s {
		t.Errorf("short string should not be truncated: got %q", got)
	}
}

func TestTruncateStr_Exact(t *testing.T) {
	s := "hello"
	got := truncateStr(s, 5)
	if got != s {
		t.Errorf("exact length should not be truncated: got %q", got)
	}
}

func TestTruncateStr_Long(t *testing.T) {
	s := "hello world"
	got := truncateStr(s, 8)
	if got != "hello..." {
		t.Errorf("want 'hello...', got %q", got)
	}
}

func TestTruncateStr_Unicode(t *testing.T) {
	s := "héllo wörld"
	got := truncateStr(s, 7)
	// Should truncate on rune boundary
	if len([]rune(got)) > 7 {
		t.Errorf("truncated too long: %q", got)
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("expected ellipsis suffix: %q", got)
	}
}

// --- memory_save tool ---

func TestMemorySaveTool_Basic(t *testing.T) {
	s, _ := setupServer(t)

	result := callTool(t, s, "memory_save", map[string]interface{}{
		"content": "We use PostgreSQL for persistence",
	})
	text := resultText(t, result)
	if !strings.Contains(text, "Saved memory") {
		t.Errorf("unexpected output: %s", text)
	}
	if result.IsError {
		t.Errorf("expected success, got error: %s", text)
	}
}

func TestMemorySaveTool_WithTypeAndTags(t *testing.T) {
	s, k := setupServer(t)

	callTool(t, s, "memory_save", map[string]interface{}{
		"content": "Auth uses JWT with RS256",
		"type":    "decision",
		"tags":    []interface{}{"auth", "security"},
	})

	results, err := k.Recall(types.MemoryRecallInput{Query: "auth JWT", Limit: 5})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected saved memory to be recalled")
	}
	m := results[0].Memory
	if m.Type != types.MemoryTypeDecision {
		t.Errorf("type: want decision, got %s", m.Type)
	}
	if m.Source != types.MemorySourceAgent {
		t.Errorf("source: want agent, got %s", m.Source)
	}
}

func TestMemorySaveTool_EmptyContent(t *testing.T) {
	s, _ := setupServer(t)

	// Empty content is saved without error (kernel does not validate)
	result := callTool(t, s, "memory_save", map[string]interface{}{
		"content": "",
	})
	if result.IsError {
		t.Errorf("unexpected error for empty content: %s", resultText(t, result))
	}
}

// --- memory_recall tool ---

func TestMemoryRecallTool_ReturnsResults(t *testing.T) {
	s, k := setupServer(t)

	k.Save(types.MemorySaveInput{
		Content: "We use Redis for caching session data",
		Type:    types.MemoryTypeDecision,
		Tags:    []string{"cache", "redis"},
	})

	result := callTool(t, s, "memory_recall", map[string]interface{}{
		"query": "caching Redis",
	})
	text := resultText(t, result)
	if !strings.Contains(text, "Redis") {
		t.Errorf("expected Redis in results, got: %s", text)
	}
}

func TestMemoryRecallTool_NoResults(t *testing.T) {
	s, _ := setupServer(t)

	result := callTool(t, s, "memory_recall", map[string]interface{}{
		"query": "obscure topic xyz",
	})
	text := resultText(t, result)
	if !strings.Contains(text, "No relevant memories") {
		t.Errorf("expected no-results message, got: %s", text)
	}
}

func TestMemoryRecallTool_LimitRespected(t *testing.T) {
	s, k := setupServer(t)

	for i := 0; i < 10; i++ {
		k.Save(types.MemorySaveInput{Content: "database connection pooling tip"})
	}

	result := callTool(t, s, "memory_recall", map[string]interface{}{
		"query": "database",
		"limit": float64(3),
	})
	text := resultText(t, result)
	// The output lists memories as [1], [2], ... — check [4] is absent
	if strings.Contains(text, "[4]") {
		t.Errorf("limit=3 should not return a 4th result; got: %s", text)
	}
}

func TestMemoryRecallTool_TypeFilter(t *testing.T) {
	s, k := setupServer(t)

	k.Save(types.MemorySaveInput{Content: "deploy to Kubernetes", Type: types.MemoryTypeEvent})
	k.Save(types.MemorySaveInput{Content: "Kubernetes naming conventions", Type: types.MemoryTypeConvention})

	result := callTool(t, s, "memory_recall", map[string]interface{}{
		"query": "Kubernetes",
		"type":  "convention",
	})
	text := resultText(t, result)
	if strings.Contains(text, "deploy to Kubernetes") {
		t.Errorf("type filter should exclude event memory; got: %s", text)
	}
}

// --- memory_forget tool ---

func TestMemoryForgetTool_ByID(t *testing.T) {
	s, k := setupServer(t)

	mem, _ := k.Save(types.MemorySaveInput{Content: "temporary memory to delete"})

	result := callTool(t, s, "memory_forget", map[string]interface{}{
		"id": mem.ID,
	})
	text := resultText(t, result)
	if !strings.Contains(text, "Deleted") {
		t.Errorf("expected deletion confirmation, got: %s", text)
	}

	got, _ := k.Get(mem.ID)
	if got != nil {
		t.Error("memory should be deleted after memory_forget")
	}
}

func TestMemoryForgetTool_ByQuery(t *testing.T) {
	s, k := setupServer(t)

	k.Save(types.MemorySaveInput{Content: "old auth approach using session cookies"})

	result := callTool(t, s, "memory_forget", map[string]interface{}{
		"query": "old auth session cookies",
	})
	text := resultText(t, result)
	if !strings.Contains(text, "Deleted") {
		t.Errorf("expected deletion confirmation, got: %s", text)
	}
}

func TestMemoryForgetTool_ByIDNotFound(t *testing.T) {
	s, _ := setupServer(t)

	result := callTool(t, s, "memory_forget", map[string]interface{}{
		"id": "01NONEXISTENTID00000000000",
	})
	text := resultText(t, result)
	if !strings.Contains(text, "not found") {
		t.Errorf("expected not-found message, got: %s", text)
	}
}

// --- memory_update tool ---

func TestMemoryUpdateTool_Content(t *testing.T) {
	s, k := setupServer(t)

	mem, _ := k.Save(types.MemorySaveInput{Content: "original content"})

	result := callTool(t, s, "memory_update", map[string]interface{}{
		"id":      mem.ID,
		"content": "updated content",
	})
	text := resultText(t, result)
	if !strings.Contains(text, "Updated memory") {
		t.Errorf("expected update confirmation, got: %s", text)
	}

	got, _ := k.Get(mem.ID)
	if got.Content != "updated content" {
		t.Errorf("want 'updated content', got %q", got.Content)
	}
}

func TestMemoryUpdateTool_TypeAndTags(t *testing.T) {
	s, k := setupServer(t)

	mem, _ := k.Save(types.MemorySaveInput{Content: "some fact", Type: types.MemoryTypeFact})

	callTool(t, s, "memory_update", map[string]interface{}{
		"id":   mem.ID,
		"type": "decision",
		"tags": []interface{}{"auth", "security"},
	})

	got, _ := k.Get(mem.ID)
	if got.Type != types.MemoryTypeDecision {
		t.Errorf("want decision, got %s", got.Type)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "auth" {
		t.Errorf("unexpected tags: %v", got.Tags)
	}
}

func TestMemoryUpdateTool_NotFound(t *testing.T) {
	s, _ := setupServer(t)

	result := callTool(t, s, "memory_update", map[string]interface{}{
		"id":      "01NONEXISTENTID0000000000X",
		"content": "new content",
	})
	text := resultText(t, result)
	if !strings.Contains(text, "not found") {
		t.Errorf("expected not-found message, got: %s", text)
	}
}

func TestMemoryUpdateTool_MissingID(t *testing.T) {
	s, _ := setupServer(t)

	result := callTool(t, s, "memory_update", map[string]interface{}{
		"content": "something",
	})
	if !result.IsError {
		t.Error("expected error when id is missing")
	}
}

func TestMemoryForgetTool_NoArgs(t *testing.T) {
	s, _ := setupServer(t)

	result := callTool(t, s, "memory_forget", map[string]interface{}{})
	text := resultText(t, result)
	if !strings.Contains(text, "Provide either") {
		t.Errorf("expected usage hint, got: %s", text)
	}
}
