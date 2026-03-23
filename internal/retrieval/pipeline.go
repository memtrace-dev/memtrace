package retrieval

import (
	"math"
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
	FindByID(id string) (*types.Memory, error)
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

	// When FTS finds nothing but an embedder is configured, fall back to pure
	// semantic search so conceptually related queries still find results.
	if len(ftsResults) == 0 {
		if p.embedder == nil {
			return nil, nil
		}
		return p.semanticOnlySearch(input)
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

// semanticOnlySearch is used when FTS returns no candidates. It embeds the query,
// scores all stored embeddings by cosine similarity, applies hard filters, and
// returns the top results. Used as a fallback for conceptual/paraphrased queries.
func (p *Pipeline) semanticOnlySearch(input types.MemoryRecallInput) ([]types.ScoredMemory, error) {
	queryVec, err := p.embedder.Embed(input.Query)
	if err != nil || len(queryVec) == 0 {
		return nil, nil
	}

	rows, err := p.store.FindEmbeddings(p.projectID)
	if err != nil || len(rows) == 0 {
		return nil, nil
	}

	type scoredRow struct {
		id  string
		sim float64
	}
	ranked := make([]scoredRow, 0, len(rows))
	for _, row := range rows {
		sim := embedding.CosineSimilarity(queryVec, row.Embedding)
		if sim > 0 {
			ranked = append(ranked, scoredRow{row.ID, sim})
		}
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].sim > ranked[j].sim })

	results := make([]types.ScoredMemory, 0, input.Limit)
	for _, r := range ranked {
		if len(results) >= input.Limit {
			break
		}
		mem, err := p.store.FindByID(r.id)
		if err != nil || mem == nil {
			continue
		}
		if mem.Status != input.Status {
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
		ageMs := float64(time.Now().Sub(mem.CreatedAt).Milliseconds())
		recency := math.Pow(0.5, ageMs/recencyHalfLifeMs)
		accessFreq := math.Min(1.0, math.Log2(float64(mem.AccessCount)+1)/10.0)
		score := weightSemantic*r.sim + weightRecency*recency +
			weightConfidence*mem.Confidence + weightAccess*accessFreq
		results = append(results, types.ScoredMemory{
			Memory: *mem,
			Score:  score,
			ScoreBreakdown: types.ScoreBreakdown{
				TextRelevance:   r.sim,
				Recency:         recency,
				Confidence:      mem.Confidence,
				AccessFrequency: accessFreq,
			},
		})
	}
	return results, nil
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
