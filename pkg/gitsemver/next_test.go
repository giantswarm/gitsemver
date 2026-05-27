package gitsemver

import (
	"testing"
)

func Test_parseVersionString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input   string
		want    parsedVersion
		wantErr bool
	}{
		// stable, with v prefix
		{"v1.2.3", parsedVersion{major: 1, minor: 2, patch: 3}, false},
		// stable, no v prefix
		{"1.2.3", parsedVersion{major: 1, minor: 2, patch: 3}, false},
		// stable, zeros
		{"v0.0.0", parsedVersion{major: 0, minor: 0, patch: 0}, false},
		// stable, large numbers
		{"v10.20.30", parsedVersion{major: 10, minor: 20, patch: 30}, false},
		// RC, with v prefix
		{"v1.2.3-rc.1", parsedVersion{major: 1, minor: 2, patch: 3, isRC: true, rcNum: 1}, false},
		// RC, no v prefix
		{"1.2.3-rc.1", parsedVersion{major: 1, minor: 2, patch: 3, isRC: true, rcNum: 1}, false},
		// RC, large rc number
		{"v1.0.0-rc.99", parsedVersion{major: 1, minor: 0, patch: 0, isRC: true, rcNum: 99}, false},
		// RC, zero base
		{"v0.0.0-rc.1", parsedVersion{major: 0, minor: 0, patch: 0, isRC: true, rcNum: 1}, false},
		// empty string
		{"", parsedVersion{}, true},
		// not semver
		{"invalid", parsedVersion{}, true},
		// non-numeric patch
		{"v1.2.x", parsedVersion{}, true},
		// missing patch
		{"v1.2", parsedVersion{}, true},
		// dev build — rejected
		{"v1.2.3-dev.main.2026-01-01.10-00-00", parsedVersion{}, true},
		// arbitrary pre-release — rejected
		{"v1.2.3-alpha.1", parsedVersion{}, true},
		// leading zeros in minor — rejected (semver §2)
		{"v1.02.3", parsedVersion{}, true},
	}

	for _, tc := range cases {
		got, err := parseVersionString(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseVersionString(%q): expected error, got %+v", tc.input, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseVersionString(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseVersionString(%q) = %+v, want %+v", tc.input, got, tc.want)
		}
	}
}

func Test_ComputeNextVersion(t *testing.T) {
	t.Parallel()

	cases := []struct {
		lastTag  string
		bumpType string
		want     string
		wantErr  bool
	}{
		// --- stable base → stable ---
		{"v1.2.3", "patch", "1.2.4", false},
		{"v1.2.3", "minor", "1.3.0", false},
		{"v1.2.3", "major", "2.0.0", false},
		// no v prefix accepted
		{"1.2.3", "patch", "1.2.4", false},
		// zero base
		{"v0.0.0", "patch", "0.0.1", false},
		{"v0.0.0", "minor", "0.1.0", false},
		{"v0.0.0", "major", "1.0.0", false},
		// large numbers
		{"v10.20.30", "patch", "10.20.31", false},
		{"v10.20.30", "minor", "10.21.0", false},
		{"v10.20.30", "major", "11.0.0", false},

		// --- stable base → RC ---
		{"v1.2.3", "patch-rc", "1.2.4-rc.1", false},
		{"v1.2.3", "minor-rc", "1.3.0-rc.1", false},
		{"v1.2.3", "major-rc", "2.0.0-rc.1", false},
		{"v0.0.0", "patch-rc", "0.0.1-rc.1", false},

		// --- RC base → RC (bump counter) ---
		{"v1.2.3-rc.1", "rc", "1.2.3-rc.2", false},
		{"v1.2.3-rc.5", "rc", "1.2.3-rc.6", false},
		{"v1.0.0-rc.99", "rc", "1.0.0-rc.100", false},

		// --- RC base → stable (finalize) ---
		{"v1.2.3-rc.1", "rc-release", "1.2.3", false},
		{"v1.3.0-rc.2", "rc-release", "1.3.0", false},
		{"v2.0.0-rc.1", "rc-release", "2.0.0", false},

		// --- invalid: rc from stable ---
		{"v1.2.3", "rc", "", true},
		// --- invalid: rc-release from stable ---
		{"v1.2.3", "rc-release", "", true},

		// --- invalid: stable bump types from RC ---
		{"v1.2.3-rc.1", "patch", "", true},
		{"v1.2.3-rc.1", "minor", "", true},
		{"v1.2.3-rc.1", "major", "", true},
		{"v1.2.3-rc.1", "patch-rc", "", true},
		{"v1.2.3-rc.1", "minor-rc", "", true},
		{"v1.2.3-rc.1", "major-rc", "", true},

		// --- bad inputs ---
		{"", "patch", "", true},
		{"invalid", "patch", "", true},
		{"v1.2.x", "patch", "", true},
		{"v1.2.3-dev.main.2026-01-01.10-00-00", "patch", "", true},
		{"v1.2.3", "", "", true},
		{"v1.2.3", "bad", "", true},
	}

	for _, tc := range cases {
		got, err := ComputeNextVersion(tc.lastTag, tc.bumpType)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ComputeNextVersion(%q, %q): expected error, got %q", tc.lastTag, tc.bumpType, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ComputeNextVersion(%q, %q): unexpected error: %v", tc.lastTag, tc.bumpType, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ComputeNextVersion(%q, %q) = %q, want %q", tc.lastTag, tc.bumpType, got, tc.want)
		}
	}
}
