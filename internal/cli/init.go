package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/memtrace-dev/memtrace/internal/ingestion"
	"github.com/memtrace-dev/memtrace/internal/kernel"
	"github.com/memtrace-dev/memtrace/internal/util"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var name string
	var noImport bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize memtrace for the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			// Find project root (prefer .git/ if no .memtrace/ yet)
			projectRoot := util.FindProjectRoot(cwd)
			if projectRoot == "" {
				projectRoot = cwd
			}

			memtraceDir := filepath.Join(projectRoot, ".memtrace")

			// Check if already initialized
			if info, err := os.Stat(memtraceDir); err == nil && info.IsDir() {
				fmt.Printf("memtrace is already initialized in %s\n", projectRoot)
				return nil
			}

			// Create .memtrace directory
			if err := os.MkdirAll(memtraceDir, 0755); err != nil {
				return fmt.Errorf("creating .memtrace directory: %w", err)
			}

			// Determine project name
			projectName := name
			if projectName == "" {
				projectName = filepath.Base(projectRoot)
			}

			// Generate project ID and register
			projectID := util.GenerateID()
			cfg := util.GetProjectConfig()
			cfg.Projects[projectRoot] = util.ProjectEntry{
				ID:        projectID,
				Name:      projectName,
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
			}
			if err := util.SaveProjectConfig(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			// Open database and apply schema
			dbPath := util.GetProjectDbPath(projectRoot)
			k := kernel.New(dbPath, projectID)
			if err := k.Open(); err != nil {
				return fmt.Errorf("initializing database: %w", err)
			}
			defer k.Close()

			// Add .memtrace/ to .gitignore
			addToGitignore(projectRoot)

			// Add memtrace instructions to CLAUDE.md (Claude Code only)
			addToClaudeMd(projectRoot)

			// Run importers unless --no-import
			var result *ingestion.IngestResult
			if !noImport {
				pipeline := ingestion.New(k)
				result = pipeline.IngestOnInit(projectRoot)
			}

			fmt.Printf("Initialized memtrace in %s\n", projectRoot)
			if result != nil && result.Total > 0 {
				parts := []string{}
				for src, n := range result.Sources {
					parts = append(parts, fmt.Sprintf("%s: %d", src, n))
				}
				fmt.Printf("Imported %d memories (%s)\n", result.Total, strings.Join(parts, ", "))
			}

			fmt.Println("\nNext: run 'memtrace setup' to wire the MCP server into your agent.")
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Project name (default: directory name)")
	cmd.Flags().BoolVar(&noImport, "no-import", false, "Skip auto-importing from Claude/Cursor/git")
	return cmd
}

const claudeMdSnippet = `
## memtrace (memory)

This project has the **memtrace MCP server** connected. Use its tools for **all** memory operations — never use built-in memory tools or write to ~/.claude/projects/.

Memory tools: ` + "`memory_recall`" + `, ` + "`memory_save`" + `, ` + "`memory_get`" + `, ` + "`memory_update`" + `, ` + "`memory_forget`" + `, ` + "`memory_context`" + `, ` + "`memory_prompt`" + `

Rules:
- **Before every task** — call ` + "`memory_recall`" + ` with a relevant query, no exceptions. This includes commits, quick fixes, and one-liners.
- **Before committing** — call ` + "`memory_recall`" + ` to check for commit conventions.
- **Learn something new** — call ` + "`memory_save`" + ` to persist it.
- **User says forget/delete/remove** — call ` + "`memory_forget`" + `.
- **Never** write memory files manually or use built-in memory features.
`

// addToClaudeMd appends memtrace instructions to CLAUDE.md if not already present.
func addToClaudeMd(projectRoot string) {
	claudePath := filepath.Join(projectRoot, "CLAUDE.md")
	data, _ := os.ReadFile(claudePath)
	if strings.Contains(string(data), "memtrace") {
		return
	}
	f, err := os.OpenFile(claudePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(claudeMdSnippet)
}

// addToGitignore appends .memtrace/ to .gitignore if the file exists and doesn't already contain it.
func addToGitignore(projectRoot string) {
	gitignorePath := filepath.Join(projectRoot, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		return // .gitignore doesn't exist, skip
	}
	if strings.Contains(string(data), ".memtrace") {
		return // already present
	}
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	content := string(data)
	prefix := "\n"
	if strings.HasSuffix(content, "\n") {
		prefix = ""
	}
	fmt.Fprintf(f, "%s.memtrace/\n", prefix)
}
