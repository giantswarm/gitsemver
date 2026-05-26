package gitsemver

import (
	"fmt"
	"regexp"
	"strconv"
)

// parsedVersion holds the numeric components of a semver string.
type parsedVersion struct {
	major, minor, patch int
	isRC                bool
	rcNum               int
}

// semverParseRegex matches X.Y.Z or X.Y.Z-rc.N with an optional leading v.
// Reuses numID from validate.go to enforce the no-leading-zeros rule (semver §2).
var semverParseRegex = regexp.MustCompile(
	`^v?(` + numID + `)\.(` + numID + `)\.(` + numID + `)(?:-rc\.(` + numID + `))?$`,
)

func parseVersionString(v string) (parsedVersion, error) {
	m := semverParseRegex.FindStringSubmatch(v)
	if m == nil {
		return parsedVersion{}, &ExecutionFailedError{
			message: fmt.Sprintf("cannot parse %q: expected X.Y.Z or X.Y.Z-rc.N (no leading zeros, no other pre-release labels)", v),
		}
	}
	pv := parsedVersion{}
	pv.major, _ = strconv.Atoi(m[1])
	pv.minor, _ = strconv.Atoi(m[2])
	pv.patch, _ = strconv.Atoi(m[3])
	if m[4] != "" {
		pv.isRC = true
		pv.rcNum, _ = strconv.Atoi(m[4])
	}
	return pv, nil
}

// compareSemver returns negative, zero, or positive comparing a to b.
// Follows semver §11: stable has higher precedence than RC at the same X.Y.Z.
func compareSemver(a, b parsedVersion) int {
	if d := a.major - b.major; d != 0 {
		return d
	}
	if d := a.minor - b.minor; d != 0 {
		return d
	}
	if d := a.patch - b.patch; d != 0 {
		return d
	}
	switch {
	case !a.isRC && !b.isRC:
		return 0
	case a.isRC && !b.isRC:
		return -1 // a is RC, b is stable → a < b
	case !a.isRC && b.isRC:
		return 1 // a is stable, b is RC → a > b
	default:
		return a.rcNum - b.rcNum
	}
}

// ComputeNextVersion returns the next version after lastTag for the given
// bumpType. It is a pure function — no git access is required.
//
// Valid from a stable tag (X.Y.Z): patch, minor, major, patch-rc, minor-rc, major-rc.
// Valid from an RC tag (X.Y.Z-rc.N): rc (bump counter), rc-release (finalize to stable).
//
// The returned string never carries a leading "v".
func ComputeNextVersion(lastTag, bumpType string) (string, error) {
	if lastTag == "" {
		return "", &ExecutionFailedError{message: "lastTag must not be empty"}
	}
	if bumpType == "" {
		return "", &ExecutionFailedError{message: "bumpType must not be empty"}
	}

	pv, err := parseVersionString(lastTag)
	if err != nil {
		return "", err
	}

	switch bumpType {
	case "patch", "minor", "major", "patch-rc", "minor-rc", "major-rc":
		if pv.isRC {
			return "", &ExecutionFailedError{message: fmt.Sprintf(
				"bump type %q requires a stable last tag; %q is an RC — use 'rc' to bump the counter or 'rc-release' to finalize",
				bumpType, lastTag,
			)}
		}
		return applyStableBump(pv, bumpType), nil
	case "rc":
		if !pv.isRC {
			return "", &ExecutionFailedError{message: fmt.Sprintf(
				"bump type 'rc' requires an RC last tag; %q is stable — use 'patch-rc', 'minor-rc', or 'major-rc' to start a new RC series",
				lastTag,
			)}
		}
		return fmt.Sprintf("%d.%d.%d-rc.%d", pv.major, pv.minor, pv.patch, pv.rcNum+1), nil
	case "rc-release":
		if !pv.isRC {
			return "", &ExecutionFailedError{message: fmt.Sprintf(
				"bump type 'rc-release' requires an RC last tag; %q is already a stable release",
				lastTag,
			)}
		}
		return fmt.Sprintf("%d.%d.%d", pv.major, pv.minor, pv.patch), nil
	default:
		return "", &ExecutionFailedError{message: fmt.Sprintf(
			"unknown bump type %q: must be one of patch, minor, major, patch-rc, minor-rc, major-rc, rc, rc-release",
			bumpType,
		)}
	}
}

func applyStableBump(pv parsedVersion, bumpType string) string {
	switch bumpType {
	case "patch":
		return fmt.Sprintf("%d.%d.%d", pv.major, pv.minor, pv.patch+1)
	case "minor":
		return fmt.Sprintf("%d.%d.0", pv.major, pv.minor+1)
	case "major":
		return fmt.Sprintf("%d.0.0", pv.major+1)
	case "patch-rc":
		return fmt.Sprintf("%d.%d.%d-rc.1", pv.major, pv.minor, pv.patch+1)
	case "minor-rc":
		return fmt.Sprintf("%d.%d.0-rc.1", pv.major, pv.minor+1)
	case "major-rc":
		return fmt.Sprintf("%d.0.0-rc.1", pv.major+1)
	}
	panic("unreachable: applyStableBump called with " + bumpType)
}
