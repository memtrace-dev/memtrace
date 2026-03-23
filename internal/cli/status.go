package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/memtrace-dev/memtrace/internal/types"
	"github.com/memtrace-dev/memtrace/internal/util"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show project status",
		RunE: func(cmd *cobra.Command, args []string) error {
			k, projectRoot, err := openKernel()
			if err != nil {
				return err
			}
			defer k.Close()

			cfg := util.GetProjectConfig()
			entry := cfg.Projects[projectRoot]
			dbPath := util.GetProjectDbPath(projectRoot)

			// Count by type
			counts := map[string]int{}
			for _, t := range []types.MemoryType{
				types.MemoryTypeDecision,
				types.MemoryTypeConvention,
				types.MemoryTypeFact,
				types.MemoryTypeEvent,
			} {
				n, _ := k.Count(t, "")
				counts[string(t)] = n
			}

			// Count by status
			statusCounts := map[string]int{}
			for _, s := range []types.MemoryStatus{
				types.MemoryStatusActive,
				types.MemoryStatusStale,
				types.MemoryStatusArchived,
			} {
				n, _ := k.Count("", s)
				statusCounts[string(s)] = n
			}
			total := statusCounts["active"] + statusCounts["stale"] + statusCounts["archived"]

			// DB size
			dbSize := dbFileSize(dbPath)

			if asJSON {
				out := map[string]interface{}{
					"project": entry.Name,
					"root":    projectRoot,
					"db_path": dbPath,
					"db_size": dbSize,
					"total":   total,
					"by_type": counts,
					"by_status": statusCounts,
				}
				return json.NewEncoder(os.Stdout).Encode(out)
			}

			bold := color.New(color.Bold)
			dim := color.New(color.Faint)

			bold.Printf("Project:   %s\n", entry.Name)
			dim.Printf("Root:      %s\n", projectRoot)
			dim.Printf("Database:  %s (%s)\n", filepath.Join(".memtrace", "memtrace.db"), dbSize)
			fmt.Println()
			bold.Printf("Memories:  %d total\n", total)
			for _, t := range []string{"decision", "convention", "fact", "event"} {
				if n := counts[t]; n > 0 {
					fmt.Printf("  %-12s %d\n", t+":", n)
				}
			}
			fmt.Println()
			fmt.Printf("Status:    %d active, %d stale, %d archived\n",
				statusCounts["active"], statusCounts["stale"], statusCounts["archived"])
			return nil
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")
	return cmd
}

func dbFileSize(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "unknown"
	}
	kb := info.Size() / 1024
	if kb < 1024 {
		return fmt.Sprintf("%d KB", kb)
	}
	return fmt.Sprintf("%.1f MB", float64(info.Size())/(1024*1024))
}
