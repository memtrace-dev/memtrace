package cli

import (
	"fmt"

	"github.com/memtrace-dev/memtrace/internal/ingestion"
	"github.com/memtrace-dev/memtrace/internal/types"
	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	var memType string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "import <file|url>",
		Short: "Import memories from a JSON file or URL",
		Long: `Import memories from a JSON file (memtrace export format) or an HTTP/HTTPS URL.

The source must contain a JSON array of memory objects, or a single memory object.
Use --dry-run to preview what would be imported without saving.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inputs, err := ingestion.ImportJSON(args[0])
			if err != nil {
				return err
			}

			// Apply type filter
			if memType != "" {
				filtered := inputs[:0]
				for _, m := range inputs {
					if string(m.Type) == memType {
						filtered = append(filtered, m)
					}
				}
				inputs = filtered
			}

			if len(inputs) == 0 {
				fmt.Println("No memories to import.")
				return nil
			}

			if dryRun {
				fmt.Printf("Would import %d memories (dry run):\n", len(inputs))
				for i, m := range inputs {
					summary := m.Content
					if len(summary) > 80 {
						summary = summary[:77] + "..."
					}
					fmt.Printf("  [%d] (%s) %s\n", i+1, m.Type, summary)
				}
				return nil
			}

			k, _, err := openKernel()
			if err != nil {
				return err
			}
			defer k.Close()

			saved := 0
			for _, input := range inputs {
				input.Source = types.MemorySourceImport
				if _, err := k.Save(input); err == nil {
					saved++
				}
			}
			fmt.Printf("Imported %d of %d memories\n", saved, len(inputs))
			return nil
		},
	}

	cmd.Flags().StringVar(&memType, "type", "", "Only import memories of this type: decision, convention, fact, event")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview what would be imported without saving")
	return cmd
}
