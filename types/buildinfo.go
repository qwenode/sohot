package types

import "fmt"

type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	BuiltBy   string `json:"builtBy"`
	GoVersion string `json:"goVersion"`
}

func (bi BuildInfo) String() string {
	return fmt.Sprintf("SoHot %s\nCommit: %s\nBuilt: %s\nBuilt by: %s\nGo version: %s",
		bi.Version, bi.Commit, bi.Date, bi.BuiltBy, bi.GoVersion)
}
