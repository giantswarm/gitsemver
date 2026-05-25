package project

var (
	buildTimestamp = "unknown"
	gitSHA         = "unknown"
	version        = "0.0.0"
)

func BuildTimestamp() string { return buildTimestamp }
func GitSHA() string         { return gitSHA }
func Version() string        { return version }
