package gitsemver

import (
	"fmt"
	"regexp"
	"strconv"
)

// Bump type constants for use with ComputeNextVersion and NextVersion.
const (
	BumpTypePatch     = "patch"
	BumpTypeMinor     = "minor"
	BumpTypeMajor     = "major"
	BumpTypePatchRC   = "patch-rc"
	BumpTypeMinorRC   = "minor-rc"
	BumpTypeMajorRC   = "major-rc"
	BumpTypeRC        = "rc"
	BumpTypeRCRelease = "rc-release"
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
	var err error
	if pv.major, err = strconv.Atoi(m[1]); err != nil {
		return parsedVersion{}, &ExecutionFailedError{message: fmt.Sprintf("major component of %q overflows int: %v", v, err)}
	}
	if pv.minor, err = strconv.Atoi(m[2]); err != nil {
		return parsedVersion{}, &ExecutionFailedError{message: fmt.Sprintf("minor component of %q overflows int: %v", v, err)}
	}
	if pv.patch, err = strconv.Atoi(m[3]); err != nil {
		return parsedVersion{}, &ExecutionFailedError{message: fmt.Sprintf("patch component of %q overflows int: %v", v, err)}
	}
	if m[4] != "" {
		pv.isRC = true
		if pv.rcNum, err = strconv.Atoi(m[4]); err != nil {
			return parsedVersion{}, &ExecutionFailedError{message: fmt.Sprintf("RC number in %q overflows int: %v", v, err)}
		}
	}
	return pv, nil
}

// compareSemver returns negative, zero, or positive comparing a to b.
// Follows semver §11: stable has higher precedence than RC at the same X.Y.Z.
func compareSemver(a, b parsedVersion) int {
	if a.major != b.major {
		if a.major < b.major {
			return -1
		}
		return 1
	}
	if a.minor != b.minor {
		if a.minor < b.minor {
			return -1
		}
		return 1
	}
	if a.patch != b.patch {
		if a.patch < b.patch {
			return -1
		}
		return 1
	}
	switch {
	case !a.isRC && !b.isRC:
		return 0
	case a.isRC && !b.isRC:
		return -1 // a is RC, b is stable → a < b
	case !a.isRC && b.isRC:
		return 1 // a is stable, b is RC → a > b
	default:
		if a.rcNum != b.rcNum {
			if a.rcNum < b.rcNum {
				return -1
			}
			return 1
		}
		return 0
	}
}

// ComputeNextVersion returns the next version after lastTag for the given
// bumpType. It is a pure function — no git access is required.
//
// Valid from a stable tag (X.Y.Z): patch, minor, major, patch-rc, minor-rc, major-rc.
// Valid from an RC tag (X.Y.Z-rc.N): rc (bump counter), rc-release (finalize to stable).
//
// The returned string never carries a leading "v".
//
// Callers are responsible for stripping the GS_GIT_TAG_PREFIX and the
// separating "/" before passing lastTag to this function. For example,
// "module-a/v1.2.3" must be passed as "v1.2.3" (or "1.2.3").
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
	case BumpTypePatch, BumpTypeMinor, BumpTypeMajor, BumpTypePatchRC, BumpTypeMinorRC, BumpTypeMajorRC:
		if pv.isRC {
			return "", &ExecutionFailedError{message: fmt.Sprintf(
				"bump type %q requires a stable last tag; %q is an RC — use 'rc' to bump the counter or 'rc-release' to finalize",
				bumpType, lastTag,
			)}
		}
		switch bumpType {
		case BumpTypePatch:
			return fmt.Sprintf("%d.%d.%d", pv.major, pv.minor, pv.patch+1), nil
		case BumpTypeMinor:
			return fmt.Sprintf("%d.%d.0", pv.major, pv.minor+1), nil
		case BumpTypeMajor:
			return fmt.Sprintf("%d.0.0", pv.major+1), nil
		case BumpTypePatchRC:
			return fmt.Sprintf("%d.%d.%d-rc.1", pv.major, pv.minor, pv.patch+1), nil
		case BumpTypeMinorRC:
			return fmt.Sprintf("%d.%d.0-rc.1", pv.major, pv.minor+1), nil
		case BumpTypeMajorRC:
			return fmt.Sprintf("%d.0.0-rc.1", pv.major+1), nil
		}
	case BumpTypeRC:
		if !pv.isRC {
			return "", &ExecutionFailedError{message: fmt.Sprintf(
				"bump type 'rc' requires an RC last tag; %q is stable — use 'patch-rc', 'minor-rc', or 'major-rc' to start a new RC series",
				lastTag,
			)}
		}
		return fmt.Sprintf("%d.%d.%d-rc.%d", pv.major, pv.minor, pv.patch, pv.rcNum+1), nil
	case BumpTypeRCRelease:
		if !pv.isRC {
			return "", &ExecutionFailedError{message: fmt.Sprintf(
				"bump type 'rc-release' requires an RC last tag; %q is already a stable release",
				lastTag,
			)}
		}
		return fmt.Sprintf("%d.%d.%d", pv.major, pv.minor, pv.patch), nil
	}
	return "", &ExecutionFailedError{message: fmt.Sprintf(
		"unknown bump type %q: must be one of patch, minor, major, patch-rc, minor-rc, major-rc, rc, rc-release",
		bumpType,
	)}
}
