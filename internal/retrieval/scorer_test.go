package retrieval

import (
	"testing"
	"time"

	"github.com/memtrace-dev/memtrace/internal/types"
)

func TestScoreCandidates_Empty(t *testing.T) {
	results := scoreCandidates(nil, time.Now())
	if results != nil {
		t.Error("expected nil for empty candidates")
	}
}

func TestScoreCandidates_ScoreRange(t *testing.T) {
	now := time.Now()
	candidates := []candidate{
		{
			memory: types.Memory{
				ID:          "01",
				Confidence:  1.0,
				AccessCount: 0,
				CreatedAt:   now.Add(-24 * time.Hour),
			},
			bm25Rank: -5.0,
		},
		{
			memory: types.Memory{
				ID:          "02",
				Confidence:  0.5,
				AccessCount: 10,
				CreatedAt:   now.Add(-7 * 24 * time.Hour),
			},
			bm25Rank: -2.0,
		},
	}

	results := scoreCandidates(candidates, now)
	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Score < 0 || r.Score > 1 {
			t.Errorf("score out of range [0,1]: %f", r.Score)
		}
	}
}

func TestScoreCandidates_BetterBM25Wins(t *testing.T) {
	now := time.Now()
	// Both candidates identical except BM25 rank
	candidates := []candidate{
		{
			memory: types.Memory{
				ID: "high", Confidence: 1.0, CreatedAt: now,
			},
			bm25Rank: -10.0, // better match
		},
		{
			memory: types.Memory{
				ID: "low", Confidence: 1.0, CreatedAt: now,
			},
			bm25Rank: -1.0, // worse match
		},
	}

	results := scoreCandidates(candidates, now)
	if results[0].Memory.ID != "high" {
		t.Errorf("expected high-rank match first, got %s", results[0].Memory.ID)
	}
}

func TestScoreCandidates_RecencyDecay(t *testing.T) {
	now := time.Now()
	recent := candidate{
		memory: types.Memory{
			ID: "recent", Confidence: 1.0, CreatedAt: now,
		},
		bm25Rank: -5.0,
	}
	old := candidate{
		memory: types.Memory{
			ID: "old", Confidence: 1.0, CreatedAt: now.Add(-365 * 24 * time.Hour),
		},
		bm25Rank: -5.0,
	}

	results := scoreCandidates([]candidate{recent, old}, now)

	var recentScore, oldScore float64
	for _, r := range results {
		if r.Memory.ID == "recent" {
			recentScore = r.ScoreBreakdown.Recency
		} else {
			oldScore = r.ScoreBreakdown.Recency
		}
	}
	if recentScore <= oldScore {
		t.Errorf("recent should have higher recency score: recent=%f, old=%f", recentScore, oldScore)
	}
}

func TestScoreCandidates_AccessFrequency(t *testing.T) {
	now := time.Now()
	results := scoreCandidates([]candidate{
		{
			memory:   types.Memory{ID: "accessed", Confidence: 1.0, CreatedAt: now, AccessCount: 100},
			bm25Rank: -5.0,
		},
	}, now)

	if results[0].ScoreBreakdown.AccessFrequency == 0 {
		t.Error("expected non-zero access frequency for count=100")
	}
	if results[0].ScoreBreakdown.AccessFrequency > 1.0 {
		t.Error("access frequency must not exceed 1.0")
	}
}

func TestScoreBreakdown_WeightsSumToOne(t *testing.T) {
	sum := weightText + weightRecency + weightConfidence + weightAccess
	if sum < 0.999 || sum > 1.001 {
		t.Errorf("weights must sum to 1.0, got %f", sum)
	}
}
