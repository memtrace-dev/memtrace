# memtrace

Persistent, searchable memory for AI coding agents — local, structured, zero-config.

Memtrace gives Claude Code, Cursor, and any MCP-compatible agent a memory that survives every new session. Architectural decisions, project conventions, and codebase knowledge — all there the next time you open a chat.

---

## The problem

Every AI coding session starts from zero. You've explained your conventions. You've told it why you chose PostgreSQL, how your auth works, what patterns to follow. Tomorrow it's gone.

Copy-pasting context into every chat doesn't scale. `CLAUDE.md` helps, but it's static. Memtrace gives your agent a living, searchable memory it builds over time.

---

## How it works

Memtrace runs as a local MCP server backed by SQLite inside your project. Your agent calls tools to save and search memories:

```
Session 1
  agent → memory_save("We use JWT with RS256. Stateless API, no session storage.")
  agent → memory_save("Auth middleware is in src/middleware/auth.go")

Session 2 (new chat, blank context)
  agent → memory_recall("auth")
  ← "We use JWT with RS256. Stateless API, no session storage."
  ← "Auth middleware is in src/middleware/auth.go"
```

On `memtrace init`, it auto-imports your existing Claude Code memories, Cursor rules, and relevant git history — so memory starts populated, not empty.

---

## Requirements

- Go 1.22+
- Git (optional, used for history import on init)

---

## Install

**Homebrew (macOS / Linux):**

```bash
brew install memtrace-dev/tap/memtrace
```

**Go install:**

```bash
go install github.com/memtrace-dev/memtrace/cmd/memtrace@latest
```

**Prebuilt binaries** — download from the [latest release](https://github.com/memtrace-dev/memtrace/releases/latest) for macOS, Linux, or Windows.

**Build from source:**

```bash
git clone https://github.com/memtrace-dev/memtrace
cd memtrace
make install
```

---

## Quickstart

```bash
# 1. Initialize in your project
cd your-project
memtrace init

# 2. Wire up the MCP server
memtrace setup

# 3. Start a new Claude Code session — memory is live
```

That's it. Your agent now has `memory_save`, `memory_recall`, `memory_get`, `memory_forget`, `memory_update`, `memory_context`, and `memory_prompt` available in every session.

---

## MCP Configuration

`memtrace setup` writes the MCP config for you — no manual JSON editing needed.

```bash
memtrace setup              # auto-detect agents from .claude/, .cursor/, .vscode/
memtrace setup claude-code  # Claude Code, project scope (.claude/mcp.json)
memtrace setup cursor       # Cursor (.cursor/mcp.json)
memtrace setup vscode       # VS Code (.vscode/mcp.json)
memtrace setup --global     # Claude Code, user scope (~/.claude/mcp.json)
```

The command is idempotent and merges into existing configs without overwriting other entries.

`memtrace init` automatically adds instructions to `CLAUDE.md` so Claude routes memory operations to memtrace instead of its built-in memory tools.

---

## MCP Tools

### `memory_save`

Save something worth remembering across sessions.

```
memory_save(
  content:    "We use JWT with RS256 — stateless API, no session storage",
  type:       "decision",           // decision | convention | fact | event
  tags:       ["auth", "security"],
  file_paths: ["src/middleware/auth.go"],
  topic_key:  "decision/auth"       // optional — re-saving with the same key updates instead of duplicating
)
```

### `memory_recall`

Search memories by natural language query. Returns summaries — call `memory_get` with an ID to read the full content of any result.

```
memory_recall(
  query: "authentication approach",
  limit: 10,
  type:  "decision"   // optional filter
)
```

### `memory_get`

Retrieve the full content of a memory by ID.

```
memory_get(id: "01KMDX71NT...")
```

### `memory_forget`

Delete a memory by ID or by query.

```
memory_forget(id: "01KMDX71NT...")    // delete by ID
memory_forget(query: "old approach")  // delete top match
```

### `memory_update`

Update an existing memory by ID. Only provided fields are changed.

```
memory_update(
  id:         "01KMDX71NT...",
  content:    "Updated content",
  type:       "decision",
  tags:       ["auth"],
  confidence: 0.8
)
```

### `memory_prompt`

Capture the user's original request at the start of a session. Call this before any other memory operations so future sessions can understand what was attempted and why.

```
memory_prompt(
  content:    "Refactor the auth middleware to support multi-tenant JWT validation",
  file_paths: ["src/auth/middleware.go"]
)
```

Saves as an `event` memory tagged `prompt`. Shows up in session history alongside the auto-generated session summary.

### `memory_context`

Get all memories relevant to a set of files you are about to read or edit. Combines direct file-path matching with inferred keyword recall — call this at the start of any task touching specific files.

```
memory_context(
  file_paths: ["src/auth/middleware.go", "src/auth/handler.go"],
  limit:      10
)
```

Returns file-matched memories first (labeled `[file match]`), followed by semantically related memories (`[related]`).

---

## Private content

Wrap any part of memory content in `<private>...</private>` tags to prevent it from being stored. The tags and their contents are stripped before the memory reaches the database — only the surrounding text is saved.

```
memory_save(
  content: "Auth uses JWT RS256. <private>Signing key: sk-prod-abc123</private> Tokens expire after 1h.",
  type: "convention"
)
// stored as: "Auth uses JWT RS256.  Tokens expire after 1h."
```

Tags are case-insensitive and support multiline blocks. This lets agents include full context in a save call without sensitive details ever being persisted.

---

## Memory types

| Type | Use for |
|------|---------|
| `decision` | Architecture choices, tooling selections, approach rationale |
| `convention` | Naming rules, code style, structural standards |
| `fact` | Durable truths about the codebase |
| `event` | Migrations, incidents, major refactors |

---

## What gets imported on `init`

`memtrace init` auto-imports from three sources:

- **Claude Code memories** — `~/.claude/projects/<project>/memory/*.md`
- **Cursor rules** — `.cursorrules` in your project root, split into per-convention memories
- **Git history** — recent commits containing decisions, migrations, or refactor keywords

Skip with `--no-import` if you want a clean start.

---

## CLI Reference

```
memtrace init    [--name <name>] [--no-import]
memtrace setup   [claude-code|cursor|vscode] [--global]
memtrace save    <content> [--type decision|convention|fact|event] [--tags auth,api] [--files src/auth.go] [--confidence 0.9]
memtrace update  <id|prefix> [--content "..."] [--type ...] [--tags ...] [--files ...] [--confidence 0.9]
memtrace edit    <id|prefix>
memtrace search  <query> [--limit 10] [--type decision] [--json]
memtrace list    [--limit 20] [--type convention] [--status active] [--json]
memtrace rm      <id|prefix>
memtrace export  [--output memories.json] [--format json|markdown] [--type decision] [--status active]
memtrace import  <file|url> [--format json|markdown] [--type decision] [--dry-run]
memtrace browse
memtrace serve   [--dir <path>]
memtrace status  [--json]
memtrace reindex
memtrace scan
memtrace link    <file> [file...] [--dry-run] [--type fact]
memtrace doctor
memtrace config  get
memtrace config  set <key> <value>
memtrace config  unset <key>
memtrace stats   [--days 7] [--json]
```

---

## Semantic search

`memory_recall` and `memtrace search` use hybrid BM25 + semantic scoring when an embedder is available. The pipeline runs both FTS keyword search and vector similarity search independently, then merges the candidate pools — so memories that match on meaning but not exact keywords still surface. Without an embedder, memtrace falls back to BM25-only search.

### Zero-config with Ollama

If [Ollama](https://ollama.com) is running on your machine, memtrace auto-detects it and uses it — no configuration needed:

```bash
ollama pull nomic-embed-text
# that's it — memtrace picks it up automatically
```

Verify with `memtrace status`:

```
Embeddings: ollama (nomic-embed-text)
```

### OpenAI (or any compatible API)

Persist the settings in memtrace's config so CLI commands and the MCP server both pick them up:

```bash
memtrace config set embed.key sk-...
memtrace config set embed.model text-embedding-3-small   # optional, this is the default
```

Settings are stored in the OS config directory (`~/Library/Application Support/memtrace/config.json` on macOS, `~/.config/memtrace/config.json` on Linux, `%AppData%\memtrace\config.json` on Windows). Environment variables always take precedence.

#### Passing via MCP client (Claude Code)

```bash
claude mcp add memtrace \
  --env MEMTRACE_EMBED_KEY=sk-... \
  memtrace serve
```

Or in `.claude/mcp.json`:

```json
{
  "mcpServers": {
    "memtrace": {
      "command": "memtrace",
      "args": ["serve"],
      "env": {
        "MEMTRACE_EMBED_KEY": "sk-..."
      }
    }
  }
}
```

#### Passing via MCP client (Cursor)

```json
{
  "mcpServers": {
    "memtrace": {
      "command": "memtrace",
      "args": ["serve"],
      "env": {
        "MEMTRACE_EMBED_KEY": "sk-..."
      }
    }
  }
}
```

### Custom local server

Any OpenAI-compatible endpoint works without an API key. Set the URL and model; memtrace sends a placeholder auth header that local servers ignore:

```bash
memtrace config set embed.url http://localhost:8080/v1
memtrace config set embed.model my-model
```

### Backfilling existing memories

Memories saved before an embedder was configured have no stored vector. Run `reindex` once to backfill them:

```bash
memtrace reindex
```

### Disabling semantic search

```bash
memtrace config set embed.provider disabled
# or: MEMTRACE_EMBED_PROVIDER=disabled
```

### Environment variables

Environment variables override config file values.

| Variable | Default | Description |
|---|---|---|
| `MEMTRACE_EMBED_KEY` | — | API key. Falls back to `OPENAI_API_KEY`. |
| `MEMTRACE_EMBED_URL` | `https://api.openai.com/v1` | Base URL of the embeddings API. |
| `MEMTRACE_EMBED_MODEL` | `text-embedding-3-small` | Model name. |
| `MEMTRACE_EMBED_PROVIDER` | `auto` | Set to `disabled` to turn off embeddings entirely. |

---

## Interactive browser

`memtrace browse` opens a full-screen terminal UI for browsing and managing memories:

```
  [decision  ] We use JWT with RS256 — stateless API, no session storage
               3d ago  ·  tags: auth, security

  [convention] Error handling: always wrap with fmt.Errorf("...: %w", err)
               1h ago  ·  tags: go, errors

  [fact      ] Database schema is on v3 — see db/migrations/003.sql
               5d ago  ·  files: db/migrations/003.sql
```

Key bindings: `/` filter · `enter` view full memory · `e` edit in `$EDITOR` · `d` delete (with confirmation) · `esc` back · `q` quit.

---

## Memory-to-code linking

`memtrace link` parses source files and creates one memory per top-level symbol — functions, types, classes, interfaces, structs, enums, and traits. Supports Go (via `go/ast`), TypeScript, JavaScript, Python, and Rust.

```bash
memtrace link src/auth/middleware.go

# Linking src/auth/middleware.go (3 symbols):
#   saved  01KMFOO...  function `ValidateJWT`
#   saved  01KMBAR...  struct `AuthConfig`
#   saved  01KMBAZ...  interface `Validator`
# 3 symbols linked.
```

Preview without saving:

```bash
memtrace link --dry-run src/auth/*.go
```

Linked memories are tagged with `symbol`, the kind (`function`, `struct`, ...), and the language (`go`, `typescript`, ...). They are also linked to the source file path, so `memory_context` and `memtrace scan` pick them up automatically.

---

## Export and import

`memtrace export` dumps all memories to JSON (default) or Markdown. `memtrace import` loads them back from either format.

```bash
# Export to Markdown — human-readable, easy to edit
memtrace export --format markdown --output memories.md

# Export filtered subset to JSON
memtrace export --format json --type decision --output decisions.json

# Import from Markdown (auto-detected by .md extension)
memtrace import memories.md

# Import from JSON with a dry run preview
memtrace import decisions.json --dry-run
```

The Markdown format uses `## [type] first line` headings with a metadata list block, separated by `---`. It is designed to be readable as-is and editable by hand before reimporting.

```markdown
## [decision] We use JWT with RS256 — stateless API, no session storage

- Tags: auth, security
- Confidence: 1.00
- Created: 2026-03-22T10:00:00Z
- Files: src/middleware/auth.go

We use JWT with RS256 for authentication. The API is stateless — no session storage anywhere.
```

---

## Diagnostics

`memtrace doctor` runs a series of health checks and reports any issues:

```
  [ok]   Database:          .memtrace/memtrace.db (24 KB, 42 memories)
  [ok]   Stale memories:    none
  [ok]   Embeddings:        ollama (nomic-embed-text)
  [ok]   Unembedded:        all memories indexed
 [warn]  MCP config:        memtrace not found in any MCP config — run 'claude mcp add memtrace memtrace serve'
  [ok]   CLAUDE.md:         memtrace instructions present

1 issue found.
```

Checks: database health, stale memories, embedding configuration, unindexed memories, MCP wiring, CLAUDE.md instructions.

---

## Session auto-summarization

When an MCP session ends, memtrace automatically saves a compact event memory recording what happened:

```
Session 2026-03-23T14:32Z (45m): saved 3 memories — "We use JWT with RS256" [decision],
"Error handling convention" [convention], "DB schema on v2" [fact]. Recalled 5 times.
```

The summary is only saved if at least one memory was written during the session — read-only sessions produce no noise. Sessions are tagged `session` and can be reviewed with:

```bash
memtrace list --type event
```

---

## Staleness detection

Memories that reference source files can go stale when those files change. Run `memtrace scan` to check:

```bash
memtrace scan
```

```
  stale  01ABCDEF1234  "Auth uses RS256 JWT..." — file modified: src/auth/middleware.go
  stale  01EFGH567890  "Schema migration v3..." — file deleted: db/migrations/003.sql

2 memories marked stale (11 unchanged).
Run 'memtrace list --status stale' to review.
```

A memory is marked stale when any of its `file_paths` has been deleted or modified more recently than the memory was last updated. Run `memtrace list --status stale` to review and either update or archive them.

---

## Storage

All data lives in `.memtrace/memtrace.db` inside your project — SQLite, local-only, no account, no cloud. The `.memtrace/` directory is added to `.gitignore` automatically on init.

---

## Development

```bash
make build      # build binary to bin/memtrace
make install    # build and copy to $GOPATH/bin
make test       # run all tests
make lint       # go vet
make snapshot   # cross-platform build via goreleaser (no publish)
make release VERSION=1.2.3  # tag + push → triggers GitHub release workflow
```

---

## Author

Built by [Sebastian Puchet](https://github.com/SebastianPuchet) — [LinkedIn](https://www.linkedin.com/in/sebastianpuchet/)

---

## License

MIT — see [LICENSE](LICENSE)
