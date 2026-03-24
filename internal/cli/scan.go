package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Mark memories stale when their referenced files have changed",
		Long: `Checks every active memory that has file_paths set.
A memory is marked stale when any referenced file has been deleted or
modified more recently than the memory was last updated.

Run 'memtrace list --status stale' to review stale memories.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			k, projectRoot, err := openKernel()
			if err != nil {
				return err
			}
			defer k.Close()

			dim := color.New(color.Faint)
			warn := color.New(color.FgYellow)

			res, err := k.ScanStaleness(projectRoot)
			if err != nil {
				return err
			}

			if res.Checked == 0 {
				fmt.Println("No memories with file paths — nothing to scan.")
				return nil
			}

			for _, d := range res.Details {
				shortID := d.MemoryID
				if len(shortID) > 12 {
					shortID = shortID[:12]
				}
				warn.Printf("  stale  ")
				fmt.Printf("%s  %q — %s\n", shortID, d.Summary, d.Reason)
			}

			if res.Marked > 0 {
				fmt.Println()
			}

			unchanged := res.Checked - res.Marked
			switch res.Marked {
			case 0:
				dim.Printf("Scanned %d memories — all up to date.\n", res.Checked)
			case 1:
				fmt.Printf("1 memory marked stale")
				if unchanged > 0 {
					dim.Printf(" (%d unchanged)", unchanged)
				}
				fmt.Println(".")
				fmt.Println("Run 'memtrace list --status stale' to review.")
			default:
				fmt.Printf("%d memories marked stale", res.Marked)
				if unchanged > 0 {
					dim.Printf(" (%d unchanged)", unchanged)
				}
				fmt.Println(".")
				fmt.Println("Run 'memtrace list --status stale' to review.")
			}
			return nil
		},
	}
}
