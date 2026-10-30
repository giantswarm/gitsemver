package gitsemver

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var tagSuffixEnvVarName = "GS_GIT_TAG_SUFFIX"

// A suffix becomes a semver pre-release label, so it can't be numeric, and
// "rc" would make a fork's releases look like release candidates.
var tagSuffixRegex = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

// tagScheme is how a repository names its version tags:
//
//   - vX.Y.Z and vX.Y.Z-rc.N, the default.
//   - <prefix>/vX.Y.Z and <prefix>/vX.Y.Z-rc.N, one module of a mono repo
//     (GS_GIT_TAG_PREFIX).
//   - vX.Y.Z-<suffix>.N, a fork that builds on upstream's release vX.Y.Z and
//     keeps upstream's tags (GS_GIT_TAG_SUFFIX). Upstream's vX.Y.Z is then no
//     version of the fork, but a dev version still counts up from it.
type tagScheme struct {
	prefix    string
	suffix    string
	suffixTag *regexp.Regexp
}

func tagSchemeFromEnv() (tagScheme, error) {
	s := tagScheme{
		prefix: os.Getenv(tagPrefixEnvVarName),
		suffix: strings.TrimSpace(os.Getenv(tagSuffixEnvVarName)),
	}
	if s.suffix == "" {
		return s, nil
	}
	if s.prefix != "" {
		return tagScheme{}, &InvalidConfigError{message: fmt.Sprintf("%s and %s can't be used together", tagPrefixEnvVarName, tagSuffixEnvVarName)}
	}
	if !tagSuffixRegex.MatchString(s.suffix) || s.suffix == "rc" {
		return tagScheme{}, &InvalidConfigError{message: fmt.Sprintf("%s=%q: want lowercase letters and digits starting with a letter, and not \"rc\"", tagSuffixEnvVarName, s.suffix)}
	}
	s.suffixTag = regexp.MustCompile(`^v(` + numID + `)\.(` + numID + `)\.(` + numID + `)-` + s.suffix + `\.(` + numID + `)$`)
	return s, nil
}

// isVersionTag reports whether t names a version of this repository, the
// version ResolveVersion returns for the commit it is on.
func (s tagScheme) isVersionTag(t string) bool {
	switch {
	case s.suffix != "":
		return s.suffixTag.MatchString(t)
	case s.prefix != "":
		return prefixedTagRegex.MatchString(t) && strings.HasPrefix(t, s.prefix+"/")
	default:
		return tagRegex.MatchString(t)
	}
}

// isBaseTag reports whether t counts as the last release a dev version counts
// up from.
func (s tagScheme) isBaseTag(t string) bool {
	switch {
	case s.suffix != "":
		return s.suffixTag.MatchString(t) || stableTagRegex.MatchString(t)
	case s.prefix != "":
		return s.isVersionTag(t) && stableTagRegex.MatchString(strings.TrimPrefix(t, s.prefix+"/"))
	default:
		return stableTagRegex.MatchString(t)
	}
}

// version maps a tag to the version string: the prefix, its "/" separator and
// the leading "v" stripped.
func (s tagScheme) version(t string) string {
	if s.prefix != "" {
		t = strings.TrimPrefix(t, s.prefix+"/")
	}
	return strings.TrimPrefix(t, "v")
}

// parse returns a tag's version for precedence.
func (s tagScheme) parse(t string) (parsedVersion, error) {
	if s.suffix != "" {
		if m := s.suffixTag.FindStringSubmatch(t); m != nil {
			pv, err := parseVersionString(fmt.Sprintf("%s.%s.%s", m[1], m[2], m[3]))
			if err != nil {
				return parsedVersion{}, err
			}
			pv.isPre = true
			if pv.preNum, err = strconv.Atoi(m[4]); err != nil {
				return parsedVersion{}, &ExecutionFailedError{message: fmt.Sprintf("%s number in %q overflows int: %v", s.suffix, t, err)}
			}
			return pv, nil
		}
	}
	return parseVersionString(s.version(t))
}
