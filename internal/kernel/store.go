package kernel

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/memtrace-dev/memtrace/internal/types"
)

// MemoryStore is a thin wrapper over database/sql for raw memory operations.
// It has no business logic — that belongs in the kernel.
type MemoryStore struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *MemoryStore {
	return &MemoryStore{db: db}
}

// Insert writes a new memory record to the database.
func (s *MemoryStore) Insert(m *types.Memory) error {
	filePathsJSON, err := json.Marshal(m.FilePaths)
	if err != nil {
		return fmt.Errorf("marshaling file_paths: %w", err)
	}
	tagsJSON, err := json.Marshal(m.Tags)
	if err != nil {
		return fmt.Errorf("marshaling tags: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO memories (
			id, type, content, summary,
			source, source_ref, confidence,
			project_id, file_paths, tags,
			status, superseded_by,
			created_at, updated_at, accessed_at, access_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, string(m.Type), m.Content, nullableString(m.Summary),
		string(m.Source), nullableString(m.SourceRef), m.Confidence,
		m.ProjectID, string(filePathsJSON), string(tagsJSON),
		string(m.Status), nullableString(m.SupersededBy),
		m.CreatedAt.Format(time.RFC3339Nano), m.UpdatedAt.Format(time.RFC3339Nano),
		nil, 0,
	)
	return err
}

// FindByID retrieves a memory by its ID. Returns nil, nil if not found.
func (s *MemoryStore) FindByID(id string) (*types.Memory, error) {
	row := s.db.QueryRow("SELECT * FROM memories WHERE id = ?", id)
	m, err := scanMemory(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

// FindByRowID retrieves a memory by its SQLite rowid (used after FTS search).
func (s *MemoryStore) FindByRowID(rowid int64) (*types.Memory, error) {
	row := s.db.QueryRow("SELECT * FROM memories WHERE rowid = ?", rowid)
	m, err := scanMemory(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

// Update applies a partial update to a memory. Only non-nil fields are changed.
func (s *MemoryStore) Update(id string, input types.MemoryUpdateInput) error {
	setClauses := []string{"updated_at = ?"}
	args := []interface{}{time.Now().UTC().Format(time.RFC3339Nano)}

	if input.Content != nil {
		setClauses = append(setClauses, "content = ?")
		args = append(args, *input.Content)
	}
	if input.Summary != nil {
		setClauses = append(setClauses, "summary = ?")
		args = append(args, *input.Summary)
	}
	if input.Type != nil {
		setClauses = append(setClauses, "type = ?")
		args = append(args, string(*input.Type))
	}
	if input.Confidence != nil {
		setClauses = append(setClauses, "confidence = ?")
		args = append(args, *input.Confidence)
	}
	if input.FilePaths != nil {
		b, _ := json.Marshal(*input.FilePaths)
		setClauses = append(setClauses, "file_paths = ?")
		args = append(args, string(b))
	}
	if input.Tags != nil {
		b, _ := json.Marshal(*input.Tags)
		setClauses = append(setClauses, "tags = ?")
		args = append(args, string(b))
	}
	if input.Status != nil {
		setClauses = append(setClauses, "status = ?")
		args = append(args, string(*input.Status))
	}

	args = append(args, id)
	query := "UPDATE memories SET " + strings.Join(setClauses, ", ") + " WHERE id = ?"
	_, err := s.db.Exec(query, args...)
	return err
}

// DeleteByID hard-deletes a memory. Returns true if a row was deleted.
func (s *MemoryStore) DeleteByID(id string) (bool, error) {
	res, err := s.db.Exec("DELETE FROM memories WHERE id = ?", id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// List returns memories matching the given options.
func (s *MemoryStore) List(opts types.ListOptions) ([]types.Memory, error) {
	query := "SELECT * FROM memories WHERE 1=1"
	args := []interface{}{}

	if opts.Type != "" {
		query += " AND type = ?"
		args = append(args, string(opts.Type))
	}

	status := opts.Status
	if status == "" {
		status = types.MemoryStatusActive
	}
	query += " AND status = ?"
	args = append(args, string(status))

	sortCol := "created_at"
	if opts.Sort == "updated_at" || opts.Sort == "accessed_at" {
		sortCol = opts.Sort
	}
	order := "DESC"
	if strings.ToLower(opts.Order) == "asc" {
		order = "ASC"
	}
	query += " ORDER BY " + sortCol + " " + order

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	query += " LIMIT ? OFFSET ?"
	args = append(args, limit, opts.Offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []types.Memory
	for rows.Next() {
		m, err := scanMemory(rows)
		if err != nil {
			return nil, err
		}
		// Post-filter tags (SQLite json_each would work but this is simpler for small sets)
		if len(opts.Tags) > 0 && !hasAllTags(m.Tags, opts.Tags) {
			continue
		}
		memories = append(memories, *m)
	}
	return memories, rows.Err()
}

// Count returns the number of memories matching the given filters.
func (s *MemoryStore) Count(memType types.MemoryType, status types.MemoryStatus) (int, error) {
	query := "SELECT COUNT(*) FROM memories WHERE 1=1"
	args := []interface{}{}

	if memType != "" {
		query += " AND type = ?"
		args = append(args, string(memType))
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, string(status))
	}

	var n int
	err := s.db.QueryRow(query, args...).Scan(&n)
	return n, err
}

// SearchFTS runs a full-text search and returns matching row IDs with BM25 ranks.
// Returns up to limit results. Caller should fetch 3x the final desired limit.
func (s *MemoryStore) SearchFTS(query string, projectID string, limit int) ([]types.FTSResult, error) {
	sanitized := sanitizeFTSQuery(query)
	if sanitized == "" {
		return nil, nil
	}

	rows, err := s.db.Query(`
		SELECT m.rowid, fts.rank
		FROM memories_fts fts
		JOIN memories m ON m.rowid = fts.rowid
		WHERE memories_fts MATCH ?
		  AND m.project_id = ?
		  AND m.status = 'active'
		ORDER BY fts.rank
		LIMIT ?
	`, sanitized, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []types.FTSResult
	for rows.Next() {
		var r types.FTSResult
		if err := rows.Scan(&r.RowID, &r.Rank); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// TouchAccess updates accessed_at and increments access_count for a memory.
func (s *MemoryStore) TouchAccess(id string, now time.Time) error {
	_, err := s.db.Exec(`
		UPDATE memories
		SET accessed_at = ?, access_count = access_count + 1
		WHERE id = ?
	`, now.Format(time.RFC3339Nano), id)
	return err
}

// --- Helpers ---

// scanner is the common interface for *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...interface{}) error
}

func scanMemory(s scanner) (*types.Memory, error) {
	var m types.Memory
	var (
		summary, sourceRef, supersededBy sql.NullString
		accessedAt                       sql.NullString
		filePathsJSON, tagsJSON          string
		createdStr, updatedStr           string
		typeStr, sourceStr, statusStr    string
	)

	err := s.Scan(
		&m.ID, &typeStr, &m.Content, &summary,
		&sourceStr, &sourceRef, &m.Confidence,
		&m.ProjectID, &filePathsJSON, &tagsJSON,
		&statusStr, &supersededBy,
		&createdStr, &updatedStr, &accessedAt, &m.AccessCount,
	)
	if err != nil {
		return nil, err
	}

	m.Type = types.MemoryType(typeStr)
	m.Source = types.MemorySource(sourceStr)
	m.Status = types.MemoryStatus(statusStr)
	m.Summary = summary.String
	m.SourceRef = sourceRef.String
	m.SupersededBy = supersededBy.String

	m.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdStr)
	m.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedStr)
	if accessedAt.Valid && accessedAt.String != "" {
		t, _ := time.Parse(time.RFC3339Nano, accessedAt.String)
		m.AccessedAt = &t
	}

	if err := json.Unmarshal([]byte(filePathsJSON), &m.FilePaths); err != nil {
		m.FilePaths = []string{}
	}
	if err := json.Unmarshal([]byte(tagsJSON), &m.Tags); err != nil {
		m.Tags = []string{}
	}
	if m.FilePaths == nil {
		m.FilePaths = []string{}
	}
	if m.Tags == nil {
		m.Tags = []string{}
	}

	return &m, nil
}

// sanitizeFTSQuery removes FTS5 special characters that could cause syntax errors.
func sanitizeFTSQuery(query string) string {
	var b strings.Builder
	for _, r := range query {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.TrimSpace(b.String())
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func hasAllTags(memTags []string, required []string) bool {
	tagSet := make(map[string]bool, len(memTags))
	for _, t := range memTags {
		tagSet[t] = true
	}
	for _, r := range required {
		if !tagSet[r] {
			return false
		}
	}
	return true
}
