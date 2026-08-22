// ABOUTME: Resolves the version CRM shows for release and source-installed builds.
// ABOUTME: Prefers linker metadata, then uses the module version embedded by Go.

package main

import (
	"runtime/debug"
	"strings"
)

func displayVersion() string {
	buildInfo, ok := debug.ReadBuildInfo()
	return versionForBuild(version, buildInfo, ok)
}

func versionForBuild(linkerVersion string, buildInfo *debug.BuildInfo, buildInfoOK bool) string {
	if linkerVersion != "dev" {
		return linkerVersion
	}
	if !buildInfoOK || buildInfo == nil || buildInfo.Main.Version == "" || buildInfo.Main.Version == "(devel)" {
		return "dev"
	}
	return strings.TrimPrefix(buildInfo.Main.Version, "v")
}
