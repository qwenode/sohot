package version

import (
	"fmt"
	"runtime"
)

// Build information - populated by GoReleaser during build
var (
	Version   = "dev"
	Commit    = "none"
	Date      = "unknown"
	BuiltBy   = "unknown"
	GoVersion = runtime.Version()
)

// BuildInfo contains all build-time information
type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	BuiltBy   string `json:"builtBy"`
	GoVersion string `json:"goVersion"`
}

// GetBuildInfo returns the current build information
func GetBuildInfo() BuildInfo {
	return BuildInfo{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		BuiltBy:   BuiltBy,
		GoVersion: GoVersion,
	}
}

// String returns a formatted string with all build information
func (bi BuildInfo) String() string {
	return fmt.Sprintf("SoHot %s\nCommit: %s\nBuilt: %s\nBuilt by: %s\nGo version: %s",
		bi.Version, bi.Commit, bi.Date, bi.BuiltBy, bi.GoVersion)
}

// Short returns a short version string
func (bi BuildInfo) Short() string {
	return fmt.Sprintf("SoHot %s (%s)", bi.Version, bi.Commit[:8])
}