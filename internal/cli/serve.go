package cli

import (
	"fmt"
	"os"

	mcpserver "github.com/memtrace-dev/memtrace/internal/mcp"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP server (stdio transport)",
		RunE: func(cmd *cobra.Command, args []string) error {
			k, _, err := openKernel()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			defer k.Close()
			return mcpserver.Serve(k)
		},
	}
}
