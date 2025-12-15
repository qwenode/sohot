package version

import (
	"runtime"

	"github.com/qwenode/sohot/types"
)

// Build information - populated by GoReleaser during build
var (
	Version   = "dev"
	Commit    = "none"
	Date      = "unknown"
	BuiltBy   = "unknown"
	GoVersion = runtime.Version()
)

// GetBuildInfo returns the current build information
func GetBuildInfo() types.BuildInfo {
	return types.BuildInfo{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		BuiltBy:   BuiltBy,
		GoVersion: GoVersion,
	}
}
