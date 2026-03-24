# Semantic Search & Embeddings

`memory_recall` and `memtrace search` use **hybrid BM25 + semantic scoring** when an embedder is configured. The pipeline runs full-text search and vector similarity independently, merges the candidate pools, and reranks — so memories that match on meaning but not exact keywords still surface.

Without an embedder, memtrace falls back to BM25-only search, which is still fast and useful.

---

## Zero-config with Ollama

If [Ollama](https://ollama.com) is running locally, memtrace detects it automatically — no configuration needed:

```bash
ollama pull nomic-embed-text
# memtrace picks it up on next start
```

Verify:

```bash
memtrace status
# Embeddings: ollama (nomic-embed-text)
```

---

## OpenAI (or any compatible API)

Store the API key in memtrace's config so both the CLI and MCP server pick it up:

```bash
memtrace config set embed.key sk-...
memtrace config set embed.model text-embedding-3-small   # optional, this is the default
```

Or pass it via environment variable:

```bash
export MEMTRACE_EMBED_KEY=sk-...
```

---

## Custom local server

Any OpenAI-compatible endpoint works without an API key:

```bash
memtrace config set embed.url http://localhost:8080/v1
memtrace config set embed.model my-model
```

Memtrace sends a placeholder auth header that local servers ignore.

---

## Passing the key via your MCP client

Add `env` to your MCP config entry so the key is available when `memtrace serve` runs:

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

## Backfilling existing memories

Memories saved before an embedder was configured have no stored vector. Run `reindex` once to backfill:

```bash
memtrace reindex
```

---

## Disabling semantic search

```bash
memtrace config set embed.provider disabled
# or: MEMTRACE_EMBED_PROVIDER=disabled
```

---

## Environment variables

Environment variables always override config file values.

| Variable | Default | Description |
|---|---|---|
| `MEMTRACE_EMBED_KEY` | — | API key. Falls back to `OPENAI_API_KEY`. |
| `MEMTRACE_EMBED_URL` | `https://api.openai.com/v1` | Base URL of the embeddings API. |
| `MEMTRACE_EMBED_MODEL` | `text-embedding-3-small` | Model name. |
| `MEMTRACE_EMBED_PROVIDER` | `auto` | Set to `disabled` to turn off embeddings entirely. |

---

## How hybrid scoring works

1. **FTS5** — keyword search across content, summary, tags, and file paths. Returns BM25-ranked candidates.
2. **Vector search** — cosine similarity against stored embeddings for all active memories.
3. **Merge** — candidates from both passes are combined. Vector-only matches get a zero BM25 score.
4. **Rerank** — final score combines BM25, semantic similarity, recency, and confidence decay.

This means a memory saved as "use RS256 for tokens" will still surface for a query like "JWT signing algorithm" even if none of those exact words appear in the content.
