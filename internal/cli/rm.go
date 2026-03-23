package cli

import (
	"fmt"
	"strings"

	"github.com/memtrace-dev/memtrace/internal/types"
	"github.com/spf13/cobra"
)

func newRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <id|prefix>",
		Short: "Delete a memory by ID or short prefix",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			k, _, err := openKernel()
			if err != nil {
				return err
			}
			defer k.Close()

			id := args[0]

			// Resolve short prefix to full ID if needed
			if len(id) < 26 {
				all, err := k.List(types.ListOptions{Limit: 1000, Status: "active"})
				if err != nil {
					return err
				}
				var matches []string
				for _, m := range all {
					if strings.HasPrefix(m.ID, strings.ToUpper(id)) {
						matches = append(matches, m.ID)
					}
				}
				if len(matches) == 0 {
					fmt.Printf("Memory %s not found\n", id)
					return nil
				}
				if len(matches) > 1 {
					fmt.Printf("Ambiguous prefix %q matches %d memories — use more characters\n", id, len(matches))
					return nil
				}
				id = matches[0]
			}

			deleted, err := k.Delete(id)
			if err != nil {
				return err
			}
			if !deleted {
				fmt.Printf("Memory %s not found\n", id)
				return nil
			}
			fmt.Printf("Deleted %s\n", id)
			return nil
		},
	}
}
