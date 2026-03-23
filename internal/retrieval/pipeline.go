package retrieval

import (
	"sort"
	"time"

	"github.com/memtrace-dev/memtrace/internal/types"
)

// StoreReader defines the subset of MemoryStore methods needed by the pipeline.
// Defined here (not in kernel/) to avoid circular imports.
type StoreReader interface {
	SearchFTS(query string, projectID string, limit int) ([]types.FTSResult, error)
	FindByRowID(rowid int64) (*types.Memory, error)
}

// Pipeline executes the retrieval flow: FTS → filter → score → sort → limit.
type Pipeline struct {
	store     StoreReader
	projectID string
}

// New creates a retrieval pipeline. store must implement StoreReader.
func New(store StoreReader, projectID string) *Pipeline {
	return &Pipeline{store: store, projectID: projectID}
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

	// Score, sort, and limit.
	scored := scoreCandidates(candidates, time.Now())
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})
	if len(scored) > input.Limit {
		scored = scored[:input.Limit]
	}

	return scored, nil
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
