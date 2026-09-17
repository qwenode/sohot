package version

import (
	"runtime"

	"github.com/qwenode/sohot/types"
)

var (
	Version   = "dev"
	Commit    = "none"
	Date      = "unknown"
	BuiltBy   = "unknown"
	GoVersion = runtime.Version()
)

func GetBuildInfo() types.BuildInfo {
	return types.BuildInfo{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		BuiltBy:   BuiltBy,
		GoVersion: GoVersion,
	}
}
