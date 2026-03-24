package types

import "time"

type MemoryType string

const (
	MemoryTypeDecision   MemoryType = "decision"
	MemoryTypeConvention MemoryType = "convention"
	MemoryTypeFact       MemoryType = "fact"
	MemoryTypeEvent      MemoryType = "event"
)

type MemorySource string

const (
	MemorySourceUser   MemorySource = "user"
	MemorySourceAgent  MemorySource = "agent"
	MemorySourceGit    MemorySource = "git"
	MemorySourceImport MemorySource = "import"
)

type MemoryStatus string

const (
	MemoryStatusActive   MemoryStatus = "active"
	MemoryStatusStale    MemoryStatus = "stale"
	MemoryStatusArchived MemoryStatus = "archived"
)

// Memory is the core domain object.
type Memory struct {
	ID           string       `json:"id"`
	Type         MemoryType   `json:"type"`
	Content      string       `json:"content"`
	Summary      string       `json:"summary,omitempty"`
	Source       MemorySource `json:"source"`
	SourceRef    string       `json:"source_ref,omitempty"`
	Confidence   float64      `json:"confidence"`
	ProjectID    string       `json:"project_id"`
	FilePaths    []string     `json:"file_paths"`
	Tags         []string     `json:"tags"`
	Status       MemoryStatus `json:"status"`
	SupersededBy string       `json:"superseded_by,omitempty"`
	TopicKey     string       `json:"topic_key,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	AccessedAt   *time.Time   `json:"accessed_at,omitempty"`
	AccessCount  int          `json:"access_count"`
}

// MemorySaveInput is what callers provide to save a new memory.
type MemorySaveInput struct {
	Content    string       // Required, max 10000 chars
	Type       MemoryType   // Default: MemoryTypeFact
	Summary    string       // Auto-generated if empty: first 120 chars of Content
	Source     MemorySource // Default: MemorySourceUser
	SourceRef  string
	Confidence float64  // Default: 1.0, range 0.0-1.0
	FilePaths  []string // Relative paths, max 20
	Tags       []string // Max 20 tags, each max 50 chars
	TopicKey   string   // Optional stable key; if set, re-saving upserts instead of creating a duplicate
}

// MemoryRecallInput is what callers provide to search memories.
type MemoryRecallInput struct {
	Query         string       // Required
	Limit         int          // Default: 10, max: 50
	Type          MemoryType   // Empty = no filter
	Status        MemoryStatus // Default: MemoryStatusActive
	Tags          []string     // Memory must have ALL of these
	MinConfidence float64      // Default: 0.0
}

// MemoryUpdateInput is a partial update — nil fields are not updated.
type MemoryUpdateInput struct {
	Content    *string
	Summary    *string
	Type       *MemoryType
	Confidence *float64
	FilePaths  *[]string
	Tags       *[]string
	Status     *MemoryStatus
}

// ScoredMemory is a memory with retrieval relevance score.
type ScoredMemory struct {
	Memory         Memory         `json:"memory"`
	Score          float64        `json:"score"`
	ScoreBreakdown ScoreBreakdown `json:"score_breakdown"`
}

// ScoreBreakdown shows how the final score was computed.
type ScoreBreakdown struct {
	TextRelevance   float64 `json:"text_relevance"`
	Recency         float64 `json:"recency"`
	Confidence      float64 `json:"confidence"`
	AccessFrequency float64 `json:"access_frequency"`
}

// ListOptions controls how memories are listed.
type ListOptions struct {
	Limit  int          // Default: 50
	Offset int          // Default: 0
	Type   MemoryType   // Empty = no filter
	Status MemoryStatus // Default: MemoryStatusActive
	Tags   []string
	Sort   string // "created_at" | "updated_at" | "accessed_at"
	Order  string // "asc" | "desc"
}

// FTSResult holds a SQLite rowid and BM25 rank from a full-text search.
// BM25 rank is negative — more negative means a better match.
type FTSResult struct {
	RowID int64
	Rank  float64
}

// Project represents a registered memtrace project.
type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	RootPath  string    `json:"root_path"`
	CreatedAt time.Time `json:"created_at"`
}
