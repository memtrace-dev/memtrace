# Agent Setup

`memtrace setup` writes the MCP config entry for your agent. It is idempotent — running it again is safe and merges into existing configs without overwriting other entries.

```bash
memtrace setup   # auto-detect from .claude/, .cursor/, .vscode/, opencode.json, .gemini/
```

---

## Claude Code

**Project scope** (recommended — memory is scoped to this project):

```bash
memtrace setup claude-code
```

Writes to `.claude/mcp.json`:

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

**User scope** (all projects):

```bash
memtrace setup --global
```

Writes to `~/.claude/mcp.json`.

---

## Cursor

```bash
memtrace setup cursor
```

Writes to `.cursor/mcp.json` (same format as Claude Code).

---

## VS Code (Copilot)

```bash
memtrace setup vscode
```

Writes to `.vscode/mcp.json`:

```json
{
  "servers": {
    "memtrace": {
      "type": "stdio",
      "command": "memtrace",
      "args": ["serve"]
    }
  }
}
```

---

## OpenCode

```bash
memtrace setup opencode
```

Writes to `opencode.json` in the project root:

```json
{
  "mcp": {
    "memtrace": {
      "type": "local",
      "command": ["memtrace", "serve"]
    }
  }
}
```

---

## Windsurf

```bash
memtrace setup windsurf
```

Writes to `~/.codeium/windsurf/mcp_config.json` (global — Windsurf doesn't support project-scoped MCP config):

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

## Gemini CLI

```bash
memtrace setup gemini
```

Writes to `.gemini/settings.json` in the project root:

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

## Passing environment variables

To configure the embeddings API key through the MCP client (so you don't need `memtrace config set`), add `env` to the config entry manually:

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

---

## CLAUDE.md instructions

`memtrace init` automatically appends instructions to `CLAUDE.md` directing Claude to use memtrace tools instead of its built-in memory. If you skip init or use a different agent, add this to your project's system prompt or rules file:

```
This project has the memtrace MCP server connected. Use memory_save, memory_recall,
memory_get, memory_forget, memory_update, memory_context, and memory_prompt for all
memory operations — do not use built-in memory tools.
```
