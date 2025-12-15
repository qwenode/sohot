package types

import "fmt"

// BuildInfo contains all build-time information.
type BuildInfo struct {
    Version   string `json:"version"`
    Commit    string `json:"commit"`
    Date      string `json:"date"`
    BuiltBy   string `json:"builtBy"`
    GoVersion string `json:"goVersion"`
}

// String returns a formatted string with all build information.
func (bi BuildInfo) String() string {
    return fmt.Sprintf("SoHot %s\nCommit: %s\nBuilt: %s\nBuilt by: %s\nGo version: %s",
        bi.Version, bi.Commit, bi.Date, bi.BuiltBy, bi.GoVersion)
}
