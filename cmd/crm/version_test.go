// ABOUTME: Verifies the detailed version command's exact user-visible output.
// ABOUTME: Ensures all build metadata is written through Cobra's output stream.

package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestVersionCommandOutput(t *testing.T) {
	var output bytes.Buffer
	versionCmd.SetOut(&output)
	t.Cleanup(func() {
		versionCmd.SetOut(nil)
	})

	if versionCmd.RunE == nil {
		t.Fatal("versionCmd.RunE is nil, want an error-returning command")
	}
	if err := versionCmd.RunE(versionCmd, nil); err != nil {
		t.Fatalf("versionCmd.RunE() error = %v", err)
	}

	want := fmt.Sprintf("crm version %s\n  commit: %s\n  built:  %s\n", displayVersion(), commit, date)
	if got := output.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestVersionCommandReturnsWriteError(t *testing.T) {
	output, err := os.CreateTemp(t.TempDir(), "crm-version-*")
	if err != nil {
		t.Fatalf("create temp output: %v", err)
	}
	if err := output.Close(); err != nil {
		t.Fatalf("close temp output: %v", err)
	}
	versionCmd.SetOut(output)
	t.Cleanup(func() {
		versionCmd.SetOut(nil)
	})

	err = versionCmd.RunE(versionCmd, nil)
	if err == nil {
		t.Fatal("versionCmd.RunE() error = nil, want write error")
	}
	if !strings.Contains(err.Error(), "write version") {
		t.Fatalf("versionCmd.RunE() error = %q, want write version context", err)
	}
}
