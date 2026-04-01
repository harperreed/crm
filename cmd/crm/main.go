// ABOUTME: CLI entry point for crm.
// ABOUTME: Initializes and executes root command.

package main

import (
	"fmt"
	"os"
)

var (
	// These variables are set via ldflags during build.
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
