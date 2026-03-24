package retrieval

import (
	"math"
	"time"

	"github.com/memtrace-dev/memtrace/internal/types"
)

// Scoring weights — each set must sum to 1.0.
const (
	// BM25-only mode
	weightText       = 0.50
	weightRecency    = 0.25
	weightConfidence = 0.15
	weightAccess     = 0.10

	// Hybrid mode: BM25 + semantic (text weight split equally between the two)
	weightBM25     = 0.25
	weightSemantic = 0.25
	// weightRecency, weightConfidence, weightAccess reused — total = 0.25+0.25+0.25+0.15+0.10 = 1.0

	// Semantic-only mode (mirrors weightText)
	weightSemanticOnly = 0.50
	// weightRecency, weightConfidence, weightAccess reused — total = 0.50+0.25+0.15+0.10 = 1.0
)

// recencyHalfLifeMs is the half-life for recency decay: 30 days.
const recencyHalfLifeMs = float64(30 * 24 * 60 * 60 * 1000)

// confDecayHalfLifeMs is the half-life for confidence decay: 90 days.
// A memory not accessed or updated for 90 days loses half its confidence weight.
const confDecayHalfLifeMs = float64(90 * 24 * 60 * 60 * 1000)

// confDecayFloor is the minimum effective confidence — memories never become
// completely irrelevant just from age.
const confDecayFloor = 0.1

// effectiveConfidence returns the time-decayed confidence for scoring.
// It does NOT modify the stored confidence value.
//
// The last-signal time is max(updated_at, accessed_at): either the user
// confirmed/edited the memory, or an agent recently recalled it.
// Confidence halves every 90 days without such a signal, down to confDecayFloor.
func effectiveConfidence(m *types.Memory, now time.Time) float64 {
	signal := m.UpdatedAt
	if m.AccessedAt != nil && m.AccessedAt.After(signal) {
		signal = *m.AccessedAt
	}
	ageMs := float64(now.Sub(signal).Milliseconds())
	if ageMs < 0 {
		ageMs = 0
	}
	decayed := m.Confidence * math.Pow(0.5, ageMs/confDecayHalfLifeMs)
	if decayed < confDecayFloor {
		decayed = confDecayFloor
	}
	return decayed
}

// candidate holds a memory and its raw BM25 rank for scoring.
type candidate struct {
	memory   types.Memory
	bm25Rank float64 // negative — more negative = better match
}

// scoreCandidates computes a combined relevance score for each candidate.
// semanticScores is an optional map of memory ID → cosine similarity (0–1).
// When nil, BM25-only weights are used.
func scoreCandidates(candidates []candidate, now time.Time, semanticScores map[string]float64) []types.ScoredMemory {
	if len(candidates) == 0 {
		return nil
	}

	hybrid := len(semanticScores) > 0

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
		bm25Norm := 0.0
		if maxRankMag > 0 {
			bm25Norm = math.Abs(c.bm25Rank) / maxRankMag
		}

		// Recency: exponential decay from creation time
		ageMs := float64(now.Sub(c.memory.CreatedAt).Milliseconds())
		recency := math.Pow(0.5, ageMs/recencyHalfLifeMs)

		// Confidence: decayed by time since last access or update
		confidence := effectiveConfidence(&c.memory, now)

		// Access frequency: logarithmic scaling, capped at 1.0
		accessFreq := math.Min(1.0, math.Log2(float64(c.memory.AccessCount)+1)/10.0)

		var score, textRelevance float64
		if hybrid {
			semScore := semanticScores[c.memory.ID] // 0 if not present
			// Clamp cosine to 0–1 (can be negative for dissimilar vectors)
			if semScore < 0 {
				semScore = 0
			}
			textRelevance = (bm25Norm + semScore) / 2.0
			score = weightBM25*bm25Norm + weightSemantic*semScore +
				weightRecency*recency + weightConfidence*confidence + weightAccess*accessFreq
		} else {
			textRelevance = bm25Norm
			score = weightText*bm25Norm +
				weightRecency*recency + weightConfidence*confidence + weightAccess*accessFreq
		}

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
