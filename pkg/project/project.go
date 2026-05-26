package project

var (
	buildTimestamp = "unknown"
	gitSHA         = "unknown"
	version        = "1.0.3-dev"
)

func BuildTimestamp() string { return buildTimestamp }
func GitSHA() string         { return gitSHA }
func Version() string        { return version }
