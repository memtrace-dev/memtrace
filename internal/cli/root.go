package cli

import "github.com/spf13/cobra"

// NewRootCmd builds and returns the root cobra command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "memtrace",
		Short: "Local-first memory engine for AI coding agents",
		Long:  "Memtrace gives AI coding tools persistent, structured memory across sessions.\nWebsite: https://memtrace.sh",
	}

	root.AddCommand(
		newInitCmd(),
		newSaveCmd(),
		newSearchCmd(),
		newListCmd(),
		newRmCmd(),
		newServeCmd(),
		newStatusCmd(),
	)

	return root
}
