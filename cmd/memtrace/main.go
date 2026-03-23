package main

import (
	"fmt"
	"os"

	"github.com/memtrace-dev/memtrace/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
