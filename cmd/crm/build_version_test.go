// ABOUTME: Verifies how CRM selects a user-visible version for each build type.
// ABOUTME: Covers release linker values and module versions embedded by go install.

package main

import (
	"runtime/debug"
	"testing"
)

func TestVersionForBuild(t *testing.T) {
	tests := []struct {
		name          string
		linkerVersion string
		buildInfo     *debug.BuildInfo
		buildInfoOK   bool
		want          string
	}{
		{
			name:          "linker version wins over embedded version",
			linkerVersion: "2.3.0",
			buildInfo:     &debug.BuildInfo{Main: debug.Module{Version: "v9.9.9"}},
			buildInfoOK:   true,
			want:          "2.3.0",
		},
		{
			name:          "embedded module version supports go install",
			linkerVersion: "dev",
			buildInfo:     &debug.BuildInfo{Main: debug.Module{Version: "v2.3.0"}},
			buildInfoOK:   true,
			want:          "2.3.0",
		},
		{
			name:          "development build keeps dev version",
			linkerVersion: "dev",
			buildInfo:     &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}},
			buildInfoOK:   true,
			want:          "dev",
		},
		{
			name:          "empty build info keeps dev version",
			linkerVersion: "dev",
			buildInfo:     &debug.BuildInfo{},
			buildInfoOK:   true,
			want:          "dev",
		},
		{
			name:          "unavailable build info keeps dev version",
			linkerVersion: "dev",
			buildInfo:     nil,
			buildInfoOK:   false,
			want:          "dev",
		},
		{
			name:          "nil build info keeps dev version",
			linkerVersion: "dev",
			buildInfo:     nil,
			buildInfoOK:   true,
			want:          "dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := versionForBuild(tt.linkerVersion, tt.buildInfo, tt.buildInfoOK); got != tt.want {
				t.Fatalf("versionForBuild(%q, %#v, %t) = %q, want %q", tt.linkerVersion, tt.buildInfo, tt.buildInfoOK, got, tt.want)
			}
		})
	}
}
