package project

var (
	buildTimestamp = "unknown"
	gitSHA         = "unknown"
	version        = "1.1.1-dev"
)

func BuildTimestamp() string { return buildTimestamp }
func GitSHA() string         { return gitSHA }
func Version() string        { return version }
