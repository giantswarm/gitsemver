package gitrepo

import "regexp"

var validStableRegex = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+$`)
var validRCRegex = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+-rc\.[0-9]+$`)
var validDevRegex = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+-dev\.[a-zA-Z0-9-]+\.[0-9]{4}-[0-9]{2}-[0-9]{2}\.[0-9]{2}-[0-9]{2}-[0-9]{2}$`)

// IsValidStable reports whether version is a valid stable release version
// (X.Y.Z or vX.Y.Z).
func IsValidStable(version string) bool {
	return validStableRegex.MatchString(version)
}

// IsValidRC reports whether version is a valid release-candidate version
// (X.Y.Z-rc.N or vX.Y.Z-rc.N).
func IsValidRC(version string) bool {
	return validRCRegex.MatchString(version)
}

// IsValidDev reports whether version is a valid dev-build version
// (X.Y.Z-dev.<branch>.<YYYY-MM-DD>.<HH-MM-SS> or with a leading v).
func IsValidDev(version string) bool {
	return validDevRegex.MatchString(version)
}

// IsValid reports whether version is a valid version string in any of the
// three recognised formats: stable, RC, or dev build.
func IsValid(version string) bool {
	return IsValidStable(version) || IsValidRC(version) || IsValidDev(version)
}
