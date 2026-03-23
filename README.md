# memtrace

Local-first memory engine for AI coding agents.

Memtrace gives tools like Claude Code, Cursor, and any MCP-compatible agent persistent, structured memory across sessions — decisions, conventions, and codebase knowledge that survives every new chat.

---

## The problem

Every AI coding session starts from zero. You've told your agent your conventions, your architectural decisions, why you chose PostgreSQL over MySQL — and it's gone the next day. Memtrace fixes that.

## How it works

Memtrace runs as a local MCP server backed by a SQLite database inside your project. When your agent saves a memory (a decision, a convention, a fact), it's there the next session — and in every other tool you use.

```
Agent → memory_save("We use JWT with RS256 for auth")
Agent → memory_recall("authentication")  → returns the JWT decision
```

---

## Install

```bash
go install github.com/memtrace-dev/memtrace/cmd/memtrace@latest
```

Or build from source:

```bash
git clone https://github.com/memtrace-dev/memtrace
cd memtrace
make build
make install
```

---

## Quickstart

```bash
# Initialize memtrace in your project
cd your-project
memtrace init

# Save a memory manually
memtrace save "We use PostgreSQL as the primary database" --type decision --tags "database"

# Search memories
memtrace search "database"

# List all memories
memtrace list

# Project status
memtrace status
```

---

## MCP Configuration

### Claude Code

Register the server with the Claude CLI:

```bash
# Project-scoped (recommended)
claude mcp add memtrace memtrace serve

# Or user-scoped (all projects)
claude mcp add --scope user memtrace memtrace serve
```

`memtrace init` automatically adds instructions to `CLAUDE.md` so Claude uses the memtrace tools instead of its built-in memory. If you skipped init or want to add it manually:

```markdown
## memtrace (memory)
This project uses the memtrace MCP server. When connected (mcp__memtrace__* tools available):
call memory_recall at task start, memory_save for important facts, memory_forget to remove memories.
```

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

### Any MCP-compatible agent

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

Once configured, your agent has access to three tools:

### `memory_save`

Save a memory that should persist across sessions.

```
memory_save(
  content: "We chose JWT over session tokens for the auth service",
  type: "decision",          // decision | convention | fact | event
  tags: ["auth", "security"],
  file_paths: ["src/middleware/auth.go"]
)
```

### `memory_recall`

Search for relevant memories.

```
memory_recall(
  query: "authentication approach",
  limit: 10,
  type: "decision"   // optional filter
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

| Type | When to use |
|------|-------------|
| `decision` | A choice made — architecture, tooling, approach |
| `convention` | A project standard — naming, structure, style |
| `fact` | A durable truth about the codebase |
| `event` | Something that happened — migration, incident, refactor |

---

## CLI Reference

```
memtrace init [--name <name>] [--no-import]
memtrace save <content> [--type decision|convention|fact|event] [--tags auth,api] [--files src/auth.go] [--confidence 0.9]
memtrace search <query> [--limit 10] [--type decision] [--json]
memtrace list [--limit 20] [--type convention] [--status active] [--json]
memtrace rm <id>
memtrace serve
memtrace status [--json]
```

---

## What gets imported on `init`

By default, `memtrace init` auto-imports:

- **Claude Code memories** — from `~/.claude/projects/<project>/memory/`
- **Cursor rules** — from `.cursorrules` in your project root
- **Git history** — recent commits containing decisions, migrations, refactors

Skip with `--no-import`.

---

## Storage

All data lives in `.memtrace/memtrace.db` inside your project directory. SQLite, local, no cloud, no account required.

The `.memtrace/` directory is automatically added to `.gitignore` on init.

---

## License

MIT — see [LICENSE](LICENSE)
