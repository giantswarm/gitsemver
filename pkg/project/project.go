package project

import "strings"

var (
	buildTimestamp = "n/a"
	gitSHA         = "n/a"
	version        = "n/a"
)

func BuildTimestamp() string { return buildTimestamp }
func GitSHA() string         { return gitSHA }

func Version() string {
	if strings.Contains(version, "-dev.") {
		return "devel"
	}
	return version
}
