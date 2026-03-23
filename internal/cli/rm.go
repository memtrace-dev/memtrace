package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <id>",
		Short: "Delete a memory by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			k, _, err := openKernel()
			if err != nil {
				return err
			}
			defer k.Close()

			deleted, err := k.Delete(args[0])
			if err != nil {
				return err
			}
			if !deleted {
				fmt.Printf("Memory %s not found\n", args[0])
				return nil
			}
			fmt.Printf("Deleted memory %s\n", args[0])
			return nil
		},
	}
}
