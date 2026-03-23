package retrieval

import (
	"sort"
	"time"

	"github.com/memtrace-dev/memtrace/internal/embedding"
	"github.com/memtrace-dev/memtrace/internal/types"
)

// StoreReader defines the subset of MemoryStore methods needed by the pipeline.
// Defined here (not in kernel/) to avoid circular imports.
type StoreReader interface {
	SearchFTS(query string, projectID string, limit int) ([]types.FTSResult, error)
	FindByRowID(rowid int64) (*types.Memory, error)
	FindEmbeddings(projectID string) ([]EmbeddingRow, error)
}

// EmbeddingRow mirrors kernel.EmbeddingRow to avoid circular imports.
type EmbeddingRow struct {
	ID        string
	Embedding []float64
}

// Pipeline executes the retrieval flow: FTS → filter → score → sort → limit.
type Pipeline struct {
	store     StoreReader
	projectID string
	embedder  embedding.Embedder // nil = embeddings disabled
}

// New creates a retrieval pipeline. store must implement StoreReader.
func New(store StoreReader, projectID string) *Pipeline {
	return &Pipeline{store: store, projectID: projectID}
}

// WithEmbedder attaches an embedder to the pipeline, enabling hybrid BM25+semantic search.
func (p *Pipeline) WithEmbedder(e embedding.Embedder) {
	p.embedder = e
}

// Recall runs the full retrieval pipeline for the given input.
func (p *Pipeline) Recall(input types.MemoryRecallInput) ([]types.ScoredMemory, error) {
	// Fetch 3× the desired limit for reranking headroom.
	candidateLimit := input.Limit * 3
	if candidateLimit < 30 {
		candidateLimit = 30
	}

	ftsResults, err := p.store.SearchFTS(input.Query, p.projectID, candidateLimit)
	if err != nil {
		return nil, err
	}
	if len(ftsResults) == 0 {
		return nil, nil
	}

	// Resolve each FTS row to a full memory and apply hard filters.
	candidates := make([]candidate, 0, len(ftsResults))
	for _, r := range ftsResults {
		r := r // capture loop variable
		mem, err := p.store.FindByRowID(r.RowID)
		if err != nil || mem == nil {
			continue
		}
		if input.Type != "" && mem.Type != input.Type {
			continue
		}
		if input.MinConfidence > 0 && mem.Confidence < input.MinConfidence {
			continue
		}
		if len(input.Tags) > 0 && !hasAllTags(mem.Tags, input.Tags) {
			continue
		}
		candidates = append(candidates, candidate{memory: *mem, bm25Rank: r.Rank})
	}

	// Optionally rerank using embedding similarity.
	var semanticScores map[string]float64
	if p.embedder != nil && len(candidates) > 0 {
		semanticScores = p.computeSemanticScores(input.Query, candidates)
	}

	// Score, sort, and limit.
	scored := scoreCandidates(candidates, time.Now(), semanticScores)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})
	if len(scored) > input.Limit {
		scored = scored[:input.Limit]
	}

	return scored, nil
}

// computeSemanticScores embeds the query and computes cosine similarity against
// stored embeddings for the given candidates. Returns a map of memory ID → similarity.
// If the query embedding fails or no stored embeddings exist, returns nil.
func (p *Pipeline) computeSemanticScores(query string, candidates []candidate) map[string]float64 {
	queryVec, err := p.embedder.Embed(query)
	if err != nil || len(queryVec) == 0 {
		return nil
	}

	storedEmbeddings, err := p.store.FindEmbeddings(p.projectID)
	if err != nil || len(storedEmbeddings) == 0 {
		return nil
	}

	// Build a set of candidate IDs for fast lookup
	candidateIDs := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		candidateIDs[c.memory.ID] = true
	}

	scores := make(map[string]float64, len(candidates))
	for _, row := range storedEmbeddings {
		if !candidateIDs[row.ID] {
			continue
		}
		scores[row.ID] = embedding.CosineSimilarity(queryVec, row.Embedding)
	}
	return scores
}

func hasAllTags(memTags, required []string) bool {
	set := make(map[string]bool, len(memTags))
	for _, t := range memTags {
		set[t] = true
	}
	for _, r := range required {
		if !set[r] {
			return false
		}
	}
	return true
}
