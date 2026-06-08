package gitsemver

import "regexp"

// numID is a semver-compliant numeric identifier: zero, or a positive integer
// with no leading zeros (semver spec Rule 9).
const numID = `(?:0|[1-9][0-9]*)`

// vXYZ matches the optional "v" prefix and three dot-separated numeric
// identifiers, each without leading zeros.
const vXYZ = `v?` + numID + `\.` + numID + `\.` + numID

var validStableRegex = regexp.MustCompile(`^` + vXYZ + `$`)
var validRCRegex = regexp.MustCompile(`^` + vXYZ + `-rc\.` + numID + `$`)

// The trailing ".h<7-hex>" commit-hash segment is optional so that dev tags
// generated before it was introduced still validate; generation always emits it.
var validDevRegex = regexp.MustCompile(`^` + vXYZ + `-dev\.[a-zA-Z0-9-]+\.[0-9]{4}-[0-9]{2}-[0-9]{2}\.[0-9]{2}-[0-9]{2}-[0-9]{2}(?:\.h[0-9a-f]{7})?$`)

// IsValidStable reports whether version is a valid stable release version
// (X.Y.Z or vX.Y.Z, no leading zeros in any component).
func IsValidStable(version string) bool {
	return validStableRegex.MatchString(version)
}

// IsValidRC reports whether version is a valid release-candidate version
// (X.Y.Z-rc.N or vX.Y.Z-rc.N, no leading zeros in any numeric component).
func IsValidRC(version string) bool {
	return validRCRegex.MatchString(version)
}

// IsValidDev reports whether version is a valid dev-build version
// (X.Y.Z-dev.<branch>.<YYYY-MM-DD>.<HH-MM-SS>[.h<7-hex>] or with a leading v,
// no leading zeros in X.Y.Z components). The trailing commit-hash segment is
// optional. This checks the format only, not the length budget enforced when a
// dev version is generated.
func IsValidDev(version string) bool {
	return validDevRegex.MatchString(version)
}

// IsValid reports whether version is a valid version string in any of the
// three recognised formats: stable, RC, or dev build.
func IsValid(version string) bool {
	return IsValidStable(version) || IsValidRC(version) || IsValidDev(version)
}
