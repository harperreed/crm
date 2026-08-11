// ABOUTME: Exercises user-visible behavior of the root CRM command.
// ABOUTME: Verifies Cobra's version flag reports the configured build version.

package main

import (
	"bytes"
	"fmt"
	"testing"
)

func TestRootVersionFlag(t *testing.T) {
	var output bytes.Buffer
	previousOutput := rootCmd.OutOrStdout()
	rootCmd.SetArgs([]string{"--version"})
	rootCmd.SetOut(&output)
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(previousOutput)
	})

	if err := Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	want := fmt.Sprintf("crm version %s\n", version)
	if got := output.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
