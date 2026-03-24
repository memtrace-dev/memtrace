# Import & Export

`memtrace export` dumps memories to a file. `memtrace import` loads them back. Both support JSON and Markdown.

---

## Export

```bash
# JSON (default)
memtrace export --output memories.json

# Markdown — human-readable, editable by hand
memtrace export --format markdown --output memories.md

# Filtered export
memtrace export --type decision --output decisions.json
memtrace export --status stale --output stale.json
```

---

## Import

```bash
# Auto-detected by file extension
memtrace import memories.md
memtrace import memories.json

# Preview without saving
memtrace import memories.md --dry-run

# Import only decisions
memtrace import memories.json --type decision

# Force format
memtrace import backup.txt --format json
```

---

## Markdown format

The Markdown export format uses `## [type] first line` headings with a metadata list block, separated by `---`. It is readable as-is and editable before reimporting.

```markdown
## [decision] We use JWT with RS256 — stateless API, no session storage

- Tags: auth, security
- Confidence: 1.00
- Created: 2026-03-22T10:00:00Z
- Files: src/middleware/auth.go

We use JWT with RS256 for authentication. The API is completely stateless — no session
storage anywhere in the system. Access tokens expire after 1 hour, refresh tokens after 30 days.

---

## [convention] Error handling: always wrap with fmt.Errorf

- Tags: go, errors
- Confidence: 0.95
- Created: 2026-03-20T08:00:00Z

All errors must be wrapped with fmt.Errorf("context: %w", err) so they are inspectable
with errors.Is / errors.As at the call site.
```

---

## Importing from a URL

Both commands accept an HTTP/HTTPS URL in place of a file path:

```bash
memtrace import https://example.com/memories.json
```

---

## Round-trip example

```bash
# Export from project A
cd project-a
memtrace export --format markdown --output ../shared-conventions.md

# Import into project B
cd ../project-b
memtrace import ../shared-conventions.md --dry-run   # preview first
memtrace import ../shared-conventions.md
```
