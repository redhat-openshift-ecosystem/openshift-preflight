package version

// TODO restructure this to be local only these packages
// TODO dynamically generate these at release
var version = "unknown"
var commit = "unknown"

var Version = VersionContext{
	Version: version,
	Commit:  commit,
}

type VersionContext struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
}
