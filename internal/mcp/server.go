package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/memtrace-dev/memtrace/internal/kernel"
	"github.com/memtrace-dev/memtrace/internal/types"
)

// Serve starts the MCP server over stdio. Blocks until the connection closes.
func Serve(k *kernel.MemoryKernel) error {
	s := server.NewMCPServer(
		"memtrace",
		"0.1.0",
		server.WithToolCapabilities(true),
	)

	registerTools(s, k)

	return server.ServeStdio(s)
}

func registerTools(s *server.MCPServer, k *kernel.MemoryKernel) {
	// Tool 1: memory_save
	s.AddTool(
		mcp.NewTool("memory_save",
			mcp.WithDescription("Save a memory (decision, convention, fact, or event) to the local memory store. Use this when you learn something important about the project that should persist across sessions."),
			mcp.WithString("content",
				mcp.Required(),
				mcp.Description("The memory content to save. Be specific and self-contained."),
			),
			mcp.WithString("type",
				mcp.Description("Memory type: decision, convention, fact, event. Default: fact"),
			),
			mcp.WithArray("tags",
				mcp.Description(`Tags for categorization, e.g. ["auth", "database"]`),
			),
			mcp.WithArray("file_paths",
				mcp.Description(`Related file paths relative to project root, e.g. ["src/auth/middleware.go"]`),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			content, _ := args["content"].(string)
			memType, _ := args["type"].(string)
			tags := extractStringSlice(args, "tags")
			filePaths := extractStringSlice(args, "file_paths")

			mem, err := k.Save(types.MemorySaveInput{
				Content:   content,
				Type:      types.MemoryType(memType),
				Source:    types.MemorySourceAgent,
				Tags:      tags,
				FilePaths: filePaths,
			})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			text := fmt.Sprintf("Saved memory %s (%s): %s", mem.ID, mem.Type, mem.Summary)
			return mcp.NewToolResultText(text), nil
		},
	)

	// Tool 2: memory_recall
	s.AddTool(
		mcp.NewTool("memory_recall",
			mcp.WithDescription("Search the memory store for relevant past memories. Use this at the start of tasks and when you need project context, conventions, or past decisions."),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description(`Natural language search query, e.g. "authentication approach" or "database conventions"`),
			),
			mcp.WithNumber("limit",
				mcp.Description("Max results to return. Default: 10, max: 50"),
			),
			mcp.WithString("type",
				mcp.Description("Filter by memory type: decision, convention, fact, event"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			query, _ := args["query"].(string)
			limit := 10
			if l, ok := args["limit"].(float64); ok {
				limit = int(l)
			}
			memType, _ := args["type"].(string)

			results, err := k.Recall(types.MemoryRecallInput{
				Query: query,
				Limit: limit,
				Type:  types.MemoryType(memType),
			})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if len(results) == 0 {
				return mcp.NewToolResultText("No relevant memories found."), nil
			}

			var buf strings.Builder
			fmt.Fprintf(&buf, "Found %d memories:\n\n", len(results))
			for i, r := range results {
				m := r.Memory
				fmt.Fprintf(&buf, "[%d] (%s, %s, confidence: %.1f) %s",
					i+1, m.Type, formatAge(m.CreatedAt), m.Confidence, m.Content)
				if len(m.Tags) > 0 {
					fmt.Fprintf(&buf, "\n   tags: %s", strings.Join(m.Tags, ", "))
				}
				if len(m.FilePaths) > 0 {
					fmt.Fprintf(&buf, "\n   files: %s", strings.Join(m.FilePaths, ", "))
				}
				if i < len(results)-1 {
					buf.WriteString("\n\n")
				}
			}
			return mcp.NewToolResultText(buf.String()), nil
		},
	)

	// Tool 3: memory_forget
	s.AddTool(
		mcp.NewTool("memory_forget",
			mcp.WithDescription("Delete a specific memory by ID, or archive the top memory matching a query. Use to remove outdated or incorrect memories."),
			mcp.WithString("id",
				mcp.Description("Specific memory ID to delete"),
			),
			mcp.WithString("query",
				mcp.Description("Search query — archives the top match"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			id, _ := args["id"].(string)
			query, _ := args["query"].(string)

			if id != "" {
				deleted, err := k.Delete(id)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				if !deleted {
					return mcp.NewToolResultText(fmt.Sprintf("Memory %s not found", id)), nil
				}
				return mcp.NewToolResultText(fmt.Sprintf("Deleted memory %s", id)), nil
			}

			if query != "" {
				results, err := k.Recall(types.MemoryRecallInput{Query: query, Limit: 1})
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				if len(results) == 0 {
					return mcp.NewToolResultText("No matching memory found."), nil
				}
				m := results[0].Memory
				archived := types.MemoryStatusArchived
				k.Update(m.ID, types.MemoryUpdateInput{Status: &archived}) //nolint:errcheck
				return mcp.NewToolResultText(
					fmt.Sprintf("Archived memory %s: %s", m.ID, truncateStr(m.Content, 100)),
				), nil
			}

			return mcp.NewToolResultText("Provide either id or query."), nil
		},
	)
}

func extractStringSlice(args map[string]interface{}, key string) []string {
	raw, ok := args[key].([]interface{})
	if !ok {
		return []string{}
	}
	result := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/(24*30)))
	}
}

func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}
