package kernel

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/memtrace-dev/memtrace/internal/embedding"
	"github.com/memtrace-dev/memtrace/internal/retrieval"
	"github.com/memtrace-dev/memtrace/internal/types"
	"github.com/memtrace-dev/memtrace/internal/util"

	_ "modernc.org/sqlite" // register the "sqlite" driver
)

// MemoryKernel is the single facade for all memory operations.
// All other modules (CLI, MCP, ingestion) must go through the kernel.
type MemoryKernel struct {
	dbPath    string
	projectID string
	db        *sql.DB
	store     *MemoryStore
	pipeline  *retrieval.Pipeline
	embedder  embedding.Embedder // nil when embeddings are not configured
}

// New creates a new MemoryKernel. Call Open() before any other method.
func New(dbPath string, projectID string) *MemoryKernel {
	return &MemoryKernel{
		dbPath:    dbPath,
		projectID: projectID,
	}
}

// Open opens the database connection, applies the schema, and sets PRAGMAs.
func (k *MemoryKernel) Open() error {
	db, err := sql.Open("sqlite", k.dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	if err := ApplySchema(db); err != nil {
		db.Close()
		return fmt.Errorf("applying schema: %w", err)
	}
	k.db = db
	k.store = NewStore(db)
	k.pipeline = retrieval.New(k.store, k.projectID) // MemoryStore satisfies retrieval.StoreReader

	// Wire up optional embedder. Env vars take precedence; config file is the fallback.
	cfg := util.GetProjectConfig()
	if e := embedding.NewClient(
		firstNonEmpty(os.Getenv("MEMTRACE_EMBED_KEY"), os.Getenv("OPENAI_API_KEY"), cfg.Embed.Key),
		"",
		firstNonEmpty(os.Getenv("MEMTRACE_EMBED_URL"), cfg.Embed.URL),
		firstNonEmpty(os.Getenv("MEMTRACE_EMBED_MODEL"), cfg.Embed.Model),
	); e != nil {
		k.embedder = e
		k.pipeline.WithEmbedder(e)
	}
	return nil
}

// Close closes the underlying database connection.
func (k *MemoryKernel) Close() error {
	if k.db != nil {
		return k.db.Close()
	}
	return nil
}

// Save validates input, generates ID and timestamps, and writes to the store.
func (k *MemoryKernel) Save(input types.MemorySaveInput) (*types.Memory, error) {
	// Apply defaults
	if input.Type == "" {
		input.Type = types.MemoryTypeFact
	}
	if input.Source == "" {
		input.Source = types.MemorySourceUser
	}
	if input.Confidence == 0 {
		input.Confidence = 1.0
	}
	if input.FilePaths == nil {
		input.FilePaths = []string{}
	}
	if input.Tags == nil {
		input.Tags = []string{}
	}
	if input.Summary == "" && input.Content != "" {
		input.Summary = truncate(input.Content, 120)
	}

	now := time.Now().UTC()
	mem := &types.Memory{
		ID:         util.GenerateID(),
		Type:       input.Type,
		Content:    input.Content,
		Summary:    input.Summary,
		Source:     input.Source,
		SourceRef:  input.SourceRef,
		Confidence: input.Confidence,
		ProjectID:  k.projectID,
		FilePaths:  input.FilePaths,
		Tags:       input.Tags,
		Status:     types.MemoryStatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := k.store.Insert(mem); err != nil {
		return nil, fmt.Errorf("saving memory: %w", err)
	}

	// Compute and persist embedding asynchronously so Save() stays fast.
	if k.embedder != nil {
		go func(id, text string) {
			vec, err := k.embedder.Embed(text)
			if err == nil {
				_ = k.store.StoreEmbedding(id, vec)
			}
		}(mem.ID, mem.Content)
	}

	return mem, nil
}

// Get retrieves a single memory by ID. Returns nil, nil if not found.
func (k *MemoryKernel) Get(id string) (*types.Memory, error) {
	return k.store.FindByID(id)
}

// Update partially updates a memory. Only non-nil fields in input are changed.
func (k *MemoryKernel) Update(id string, input types.MemoryUpdateInput) (*types.Memory, error) {
	if err := k.store.Update(id, input); err != nil {
		return nil, fmt.Errorf("updating memory: %w", err)
	}
	return k.store.FindByID(id)
}

// Delete hard-deletes a memory by ID. Returns false if not found.
func (k *MemoryKernel) Delete(id string) (bool, error) {
	return k.store.DeleteByID(id)
}

// List returns memories matching the given options.
func (k *MemoryKernel) List(opts types.ListOptions) ([]types.Memory, error) {
	return k.store.List(opts)
}

// Count returns the number of memories matching the given filters.
func (k *MemoryKernel) Count(memType types.MemoryType, status types.MemoryStatus) (int, error) {
	return k.store.Count(memType, status)
}

// Recall searches memories using the retrieval pipeline, then updates access tracking.
func (k *MemoryKernel) Recall(input types.MemoryRecallInput) ([]types.ScoredMemory, error) {
	if input.Limit <= 0 {
		input.Limit = 10
	}
	if input.Limit > 50 {
		input.Limit = 50
	}
	if input.Status == "" {
		input.Status = types.MemoryStatusActive
	}

	results, err := k.pipeline.Recall(input)
	if err != nil {
		return nil, fmt.Errorf("recall: %w", err)
	}

	// Update access tracking for returned memories
	now := time.Now().UTC()
	for _, r := range results {
		_ = k.store.TouchAccess(r.Memory.ID, now)
	}

	return results, nil
}

// HasEmbedder reports whether an embedding client is configured.
func (k *MemoryKernel) HasEmbedder() bool {
	return k.embedder != nil
}

// ReindexResult holds the outcome of a Reindex run.
type ReindexResult struct {
	Total     int   // memories with no stored embedding
	Succeeded int   // successfully embedded and stored
	FirstErr  error // first embed/store error encountered, if any
}

// Reindex computes and persists embeddings for all active memories that have
// none stored yet. If no embedder is configured it returns a zero result.
func (k *MemoryKernel) Reindex(progress func(done, total int)) (ReindexResult, error) {
	if k.embedder == nil {
		return ReindexResult{}, nil
	}

	rows, err := k.store.FindUnembedded(k.projectID)
	if err != nil {
		return ReindexResult{}, fmt.Errorf("listing unembedded memories: %w", err)
	}

	res := ReindexResult{Total: len(rows)}
	for _, row := range rows {
		vec, err := k.embedder.Embed(row.Content)
		if err != nil {
			if res.FirstErr == nil {
				res.FirstErr = fmt.Errorf("embed %s: %w", row.ID[:8], err)
			}
			continue
		}
		if storeErr := k.store.StoreEmbedding(row.ID, vec); storeErr != nil {
			if res.FirstErr == nil {
				res.FirstErr = fmt.Errorf("store %s: %w", row.ID[:8], storeErr)
			}
			continue
		}
		res.Succeeded++
		if progress != nil {
			progress(res.Succeeded, res.Total)
		}
	}
	return res, nil
}

// Store returns the underlying store (used by the retrieval pipeline).
func (k *MemoryKernel) Store() *MemoryStore {
	return k.store
}

// firstNonEmpty returns the first non-empty string from the arguments.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}
