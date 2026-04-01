// ABOUTME: CLI output helpers that wrap fmt.Fprint* for stdout.
// ABOUTME: Discards write errors since stdout failures in a CLI are unrecoverable.

package main

import (
	"fmt"
	"os"
)

// out writes a formatted string to stdout.
func out(format string, a ...any) {
	_, _ = fmt.Fprintf(os.Stdout, format, a...)
}

// outln writes a line to stdout.
func outln(a ...any) {
	_, _ = fmt.Fprintln(os.Stdout, a...)
}
