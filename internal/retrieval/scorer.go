package retrieval

import (
	"math"
	"time"

	"github.com/memtrace-dev/memtrace/internal/types"
)

// Scoring weights — must sum to 1.0.
const (
	weightText       = 0.50
	weightRecency    = 0.25
	weightConfidence = 0.15
	weightAccess     = 0.10
)

// recencyHalfLifeMs is the half-life for recency decay: 30 days.
const recencyHalfLifeMs = float64(30 * 24 * 60 * 60 * 1000)

// candidate holds a memory and its raw BM25 rank for scoring.
type candidate struct {
	memory   types.Memory
	bm25Rank float64 // negative — more negative = better match
}

// scoreCandidates computes a combined relevance score for each candidate.
func scoreCandidates(candidates []candidate, now time.Time) []types.ScoredMemory {
	if len(candidates) == 0 {
		return nil
	}

	// Find the largest BM25 magnitude for normalization.
	maxRankMag := 0.0
	for _, c := range candidates {
		mag := math.Abs(c.bm25Rank)
		if mag > maxRankMag {
			maxRankMag = mag
		}
	}

	results := make([]types.ScoredMemory, 0, len(candidates))
	for _, c := range candidates {
		// Text relevance: normalize BM25 to 0–1
		textRelevance := 0.0
		if maxRankMag > 0 {
			textRelevance = math.Abs(c.bm25Rank) / maxRankMag
		}

		// Recency: exponential decay from creation time
		ageMs := float64(now.Sub(c.memory.CreatedAt).Milliseconds())
		recency := math.Pow(0.5, ageMs/recencyHalfLifeMs)

		// Confidence: direct pass-through (already 0–1)
		confidence := c.memory.Confidence

		// Access frequency: logarithmic scaling, capped at 1.0
		accessFreq := math.Min(1.0, math.Log2(float64(c.memory.AccessCount)+1)/10.0)

		score := weightText*textRelevance +
			weightRecency*recency +
			weightConfidence*confidence +
			weightAccess*accessFreq

		results = append(results, types.ScoredMemory{
			Memory: c.memory,
			Score:  score,
			ScoreBreakdown: types.ScoreBreakdown{
				TextRelevance:   textRelevance,
				Recency:         recency,
				Confidence:      confidence,
				AccessFrequency: accessFreq,
			},
		})
	}
	return results
}
