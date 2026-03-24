package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newSetupCmd() *cobra.Command {
	var global bool

	cmd := &cobra.Command{
		Use:   "setup [agent]",
		Short: "Wire memtrace into your AI coding agent's MCP config",
		Long: `Adds memtrace to your agent's MCP configuration so it is available in every session.

Supported agents:
  claude-code   Writes to .claude/mcp.json (or ~/.claude/mcp.json with --global)
  cursor        Writes to .cursor/mcp.json
  vscode        Writes to .vscode/mcp.json

If no agent is specified, memtrace auto-detects which agents are configured in the
current directory (by checking for .claude/, .cursor/, and .vscode/ directories)
and sets up all detected ones. Falls back to claude-code if none are detected.

The command is idempotent — running it again is safe.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			projectRoot := cwd

			var agents []string
			if len(args) == 1 {
				agents = []string{normalizeAgent(args[0])}
			} else {
				agents = detectAgents(projectRoot)
			}

			any := false
			for _, agent := range agents {
				done, err := setupAgent(agent, projectRoot, global)
				if err != nil {
					fmt.Fprintf(os.Stderr, "  [error] %s: %v\n", agent, err)
					continue
				}
				if done {
					fmt.Printf("  [ok] %-12s configured\n", agent)
				} else {
					fmt.Printf("  [ok] %-12s already configured\n", agent)
				}
				any = true
			}

			if !any {
				fmt.Println("No agents set up. Specify an agent:")
				fmt.Println("  memtrace setup claude-code")
				fmt.Println("  memtrace setup cursor")
				fmt.Println("  memtrace setup vscode")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "Install at user scope (~/.claude/mcp.json) instead of project scope (claude-code only)")
	return cmd
}

func normalizeAgent(s string) string {
	switch strings.ToLower(s) {
	case "claude", "claude-code", "claudecode":
		return "claude-code"
	case "cursor":
		return "cursor"
	case "vscode", "vs-code", "code":
		return "vscode"
	default:
		return s
	}
}

func detectAgents(projectRoot string) []string {
	var found []string
	checks := []struct {
		dir   string
		agent string
	}{
		{".claude", "claude-code"},
		{".cursor", "cursor"},
		{".vscode", "vscode"},
	}
	for _, c := range checks {
		if info, err := os.Stat(filepath.Join(projectRoot, c.dir)); err == nil && info.IsDir() {
			found = append(found, c.agent)
		}
	}
	if len(found) == 0 {
		return []string{"claude-code"}
	}
	return found
}

// setupAgent writes the memtrace MCP entry into the agent's config file.
// Returns (true, nil) if the entry was written, (false, nil) if it was already present.
func setupAgent(agent, projectRoot string, global bool) (bool, error) {
	switch agent {
	case "claude-code":
		var configPath string
		if global {
			home, err := os.UserHomeDir()
			if err != nil {
				return false, fmt.Errorf("could not find home directory: %w", err)
			}
			configPath = filepath.Join(home, ".claude", "mcp.json")
		} else {
			configPath = filepath.Join(projectRoot, ".claude", "mcp.json")
		}
		return writeMCPEntry(configPath, "mcpServers", map[string]interface{}{
			"command": "memtrace",
			"args":    []string{"serve"},
		})

	case "cursor":
		configPath := filepath.Join(projectRoot, ".cursor", "mcp.json")
		return writeMCPEntry(configPath, "mcpServers", map[string]interface{}{
			"command": "memtrace",
			"args":    []string{"serve"},
		})

	case "vscode":
		configPath := filepath.Join(projectRoot, ".vscode", "mcp.json")
		return writeMCPEntry(configPath, "servers", map[string]interface{}{
			"type":    "stdio",
			"command": "memtrace",
			"args":    []string{"serve"},
		})

	default:
		return false, fmt.Errorf("unknown agent %q — supported: claude-code, cursor, vscode", agent)
	}
}

// writeMCPEntry reads (or creates) the JSON config at path, merges the memtrace
// entry under the given key, and writes it back. Returns false if already present.
func writeMCPEntry(path, serversKey string, entry map[string]interface{}) (bool, error) {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return false, fmt.Errorf("creating directory: %w", err)
	}

	// Read existing config or start fresh
	var cfg map[string]interface{}
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return false, fmt.Errorf("parsing %s: %w", path, err)
		}
	}
	if cfg == nil {
		cfg = make(map[string]interface{})
	}

	// Ensure the servers key exists as a map
	servers, _ := cfg[serversKey].(map[string]interface{})
	if servers == nil {
		servers = make(map[string]interface{})
	}

	// Already present — nothing to do
	if _, exists := servers["memtrace"]; exists {
		return false, nil
	}

	servers["memtrace"] = entry
	cfg[serversKey] = servers

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return false, err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		return false, fmt.Errorf("writing %s: %w", path, err)
	}
	return true, nil
}
