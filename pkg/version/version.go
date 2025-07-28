package version

import (
	"fmt"
	"runtime"
)

// Build information that can be set via ldflags during build
var (
	// Version is the main version number that is being run at the moment.
	Version = "dev"

	// GitCommit is the git commit hash this binary was built from
	GitCommit = "unknown"

	// BuildDate is the date this binary was built
	BuildDate = "unknown"

	// GoVersion is the version of Go this was compiled with
	GoVersion = runtime.Version()
)

// BuildInfo contains all build-related information
type BuildInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
}

// GetVersion returns the current version
func GetVersion() string {
	if Version == "dev" {
		if len(GitCommit) >= 8 {
			return fmt.Sprintf("dev-%s", GitCommit[:8])
		} else if len(GitCommit) > 0 {
			return fmt.Sprintf("dev-%s", GitCommit)
		}
		return "dev-unknown"
	}
	return Version
}

// GetFullVersion returns a detailed version string
func GetFullVersion() string {
	return fmt.Sprintf("%s (commit: %s, built: %s, go: %s)",
		GetVersion(), GitCommit, BuildDate, GoVersion)
}

// GetBuildInfo returns all build information
func GetBuildInfo() *BuildInfo {
	return &BuildInfo{
		Version:   GetVersion(),
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
	}
}

// IsDevBuild returns true if this is a development build
func IsDevBuild() bool {
	return Version == "dev"
}
