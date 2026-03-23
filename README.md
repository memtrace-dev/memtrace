# memtrace

Persistent memory for AI coding agents — local, structured, zero-config.

Memtrace gives Claude Code, Cursor, and any MCP-compatible agent a memory that survives every new session. Architectural decisions, project conventions, and codebase knowledge — all there the next time you open a chat.

---

## The problem

Every AI coding session starts from zero. You've explained your conventions. You've told it why you chose PostgreSQL, how your auth works, what patterns to follow. Tomorrow it's gone.

Copy-pasting context into every chat doesn't scale. `CLAUDE.md` helps, but it's static. Memtrace gives your agent a living, searchable memory it builds over time.

---

## How it works

Memtrace runs as a local MCP server backed by SQLite inside your project. Your agent calls three tools:

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

That's it. Your agent now has `memory_save`, `memory_recall`, and `memory_forget` available in every session.

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

Delete or archive a memory.

```
memory_forget(id: "01KMDX71NT...")    // delete by ID
memory_forget(query: "old approach")  // archive top match
```

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
memtrace init [--name <name>] [--no-import]
memtrace save <content> [--type decision|convention|fact|event] [--tags auth,api] [--files src/auth.go]
memtrace search <query> [--limit 10] [--type decision] [--json]
memtrace list [--limit 20] [--type convention] [--status active] [--json]
memtrace rm <id|prefix>
memtrace serve [--dir <path>]
memtrace status [--json]
```

---

## Storage

All data lives in `.memtrace/memtrace.db` inside your project — SQLite, local-only, no account, no cloud. The `.memtrace/` directory is added to `.gitignore` automatically on init.

---

## License

MIT — see [LICENSE](LICENSE)
