package project

var (
	buildTimestamp = "unknown"
	gitSHA         = "unknown"
	version        = "1.1.2"
)

func BuildTimestamp() string { return buildTimestamp }
func GitSHA() string         { return gitSHA }
func Version() string        { return version }
