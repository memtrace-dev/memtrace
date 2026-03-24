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

```bash
go install github.com/memtrace-dev/memtrace/cmd/memtrace@latest
```

Or build from source:

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

# 2. Wire up the MCP server (Claude Code)
claude mcp add memtrace memtrace serve

# 3. Start a new Claude Code session — memory is live
```

That's it. Your agent now has `memory_save`, `memory_recall`, `memory_forget`, `memory_update`, and `memory_context` available in every session.

---

## MCP Configuration

### Claude Code

```bash
# Project-scoped (recommended — one setup per project)
claude mcp add memtrace memtrace serve

# User-scoped (available in all projects that have been initialized)
claude mcp add --scope user memtrace memtrace serve
```

`memtrace init` automatically adds instructions to `CLAUDE.md` so Claude routes memory operations to memtrace instead of its built-in memory tools.

### Cursor

Add to `.cursor/mcp.json` in your project:

```json
{
  "mcpServers": {
    "memtrace": {
      "command": "memtrace",
      "args": ["serve"]
    }
  }
}
```

### Other MCP clients

```json
{
  "mcpServers": {
    "memtrace": {
      "command": "memtrace",
      "args": ["serve"]
    }
  }
}
```

---

## MCP Tools

### `memory_save`

Save something worth remembering across sessions.

```
memory_save(
  content:    "We use JWT with RS256 — stateless API, no session storage",
  type:       "decision",           // decision | convention | fact | event
  tags:       ["auth", "security"],
  file_paths: ["src/middleware/auth.go"]
)
```

### `memory_recall`

Search memories by natural language query.

```
memory_recall(
  query: "authentication approach",
  limit: 10,
  type:  "decision"   // optional filter
)
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
memtrace save    <content> [--type decision|convention|fact|event] [--tags auth,api] [--files src/auth.go] [--confidence 0.9]
memtrace update  <id|prefix> [--content "..."] [--type ...] [--tags ...] [--files ...] [--confidence 0.9]
memtrace edit    <id|prefix>
memtrace search  <query> [--limit 10] [--type decision] [--json]
memtrace list    [--limit 20] [--type convention] [--status active] [--json]
memtrace rm      <id|prefix>
memtrace export  [--output memories.json] [--type decision] [--status active]
memtrace import  <file|url> [--type decision] [--dry-run]
memtrace serve   [--dir <path>]
memtrace status  [--json]
memtrace reindex
memtrace scan
memtrace config  get
memtrace config  set <key> <value>
memtrace config  unset <key>
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
# Build
make install

# Test
go test ./...
```

---

## Author

Built by [Sebastian Puchet](https://github.com/SebastianPuchet) — [LinkedIn](https://www.linkedin.com/in/sebastianpuchet/)

---

## License

MIT — see [LICENSE](LICENSE)
