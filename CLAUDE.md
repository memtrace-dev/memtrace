# memtrace

This project has the **memtrace MCP server** connected. Use its tools for **all** memory operations — never use built-in memory tools or write to ~/.claude/projects/.

## Memory tools

- `memory_recall` — search memories by natural language query
- `memory_save` — save a decision, convention, fact, or event
- `memory_get` — fetch full content of a memory by ID
- `memory_update` — patch an existing memory by ID (content, tags, type, confidence)
- `memory_forget` — delete or archive a memory by ID or query
- `memory_context` — get memories relevant to specific file paths
- `memory_prompt` — capture the user's goal at the start of a session

## Rules (always follow)

- **Before every task** — call `memory_recall` with a relevant query, no exceptions. This includes commits, quick fixes, and one-liners.
- **Before committing** — call `memory_recall` to check for commit conventions.
- **Learn something new** — call `memory_save` to persist it.
- **User says forget/delete/remove** — call `memory_forget`.
- **Never** write memory files manually or use built-in memory features.
