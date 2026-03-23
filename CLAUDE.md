# memtrace

This project has the **memtrace MCP server** connected. Use its tools for all memory operations — do not use built-in memory tools.

## Memory tools (use these)

- `memory_save` — save a decision, convention, fact, or event
- `memory_recall` — search for relevant memories before starting any task
- `memory_forget` — delete or archive a memory by ID or query

## When to use them

- Start of any task → call `memory_recall` with a relevant query
- Learn something about the project → call `memory_save`
- User says "forget" / "delete" / "remove" a memory → call `memory_forget`
