package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"github.com/fatih/color"
	"github.com/memtrace-dev/memtrace/internal/symbol"
	"github.com/memtrace-dev/memtrace/internal/types"
	"github.com/spf13/cobra"
)

func newLinkCmd() *cobra.Command {
	var dryRun bool
	var memType string

	cmd := &cobra.Command{
		Use:   "link <file> [file...]",
		Short: "Extract symbols from source files and save them as memories",
		Long: `Parses source files and creates one memory per top-level symbol
(functions, types, classes, interfaces, etc.).

Supports Go, TypeScript, JavaScript, Python, and Rust.

Examples:
  memtrace link src/auth/middleware.go
  memtrace link --dry-run src/api/*.go
  memtrace link --type convention src/auth/middleware.go`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			k, projectRoot, err := openKernel()
			if err != nil {
				return err
			}
			defer k.Close()

			mt := types.MemoryType(memType)
			if mt == "" {
				mt = types.MemoryTypeFact
			}

			totalSaved := 0
			totalSkipped := 0

			for _, arg := range args {
				// Expand globs
				paths, err := filepath.Glob(arg)
				if err != nil || len(paths) == 0 {
					paths = []string{arg}
				}

				for _, absPath := range paths {
					// Make path relative to project root for storage.
					relPath, err := filepath.Rel(projectRoot, absPath)
					if err != nil {
						relPath = absPath
					}
					// Normalize to forward slashes.
					relPath = filepath.ToSlash(relPath)

					symbols, err := symbol.ExtractFile(absPath)
					if err != nil {
						fmt.Fprintf(os.Stderr, "warning: %v\n", err)
						continue
					}
					if symbols == nil {
						fmt.Printf("  skip  %s (unsupported file type)\n", relPath)
						continue
					}

					fmt.Printf("\nLinking %s (%d symbols):\n", relPath, len(symbols))

					for _, sym := range symbols {
						content := symbol.MemoryContent(sym, relPath)
						tags := symbol.Tags(sym, relPath)

						if dryRun {
							fmt.Printf("  [dry-run] %s `%s` (line %d)\n", sym.Kind, sym.Name, sym.Line)
							totalSaved++
							continue
						}

						mem, err := k.Save(types.MemorySaveInput{
							Content:   content,
							Type:      mt,
							Tags:      tags,
							FilePaths: []string{relPath},
							Source:    types.MemorySourceUser,
						})
						if err != nil {
							fmt.Fprintf(os.Stderr, "  error saving %s: %v\n", sym.Name, err)
							totalSkipped++
							continue
						}
						shortID := mem.ID
						if len(shortID) > 12 {
							shortID = shortID[:12]
						}
						fmt.Printf("  saved  %s  %s `%s`\n", shortID, sym.Kind, sym.Name)
						totalSaved++
					}
				}
			}

			fmt.Println()
			if dryRun {
				action := "would be saved"
				fmt.Printf("%s\n", color.YellowString("%d %s %s (dry run).", totalSaved, plural(totalSaved, "symbol", "symbols"), action))
			} else {
				msg := fmt.Sprintf("%d %s linked.", totalSaved, plural(totalSaved, "symbol", "symbols"))
				if totalSkipped > 0 {
					msg += fmt.Sprintf(" %d skipped.", totalSkipped)
				}
				fmt.Printf("%s\n", color.GreenString("%s", msg))
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview symbols without saving")
	cmd.Flags().StringVar(&memType, "type", "fact", "Memory type: decision, convention, fact, event")
	return cmd
}

func plural(n int, singular, pluralForm string) string {
	if n == 1 {
		return singular
	}
	return pluralForm
}

