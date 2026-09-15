package gitsemver

import (
	"strings"
	"testing"
	"time"
)

func Test_truncateBranch(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		branch   string
		budget   int
		expected string
	}{
		{"fits unchanged", "my-feature", 33, "my-feature"},
		{"fits exactly", "my-feature", 10, "my-feature"},
		{
			name:     "middle dropped, head and tail kept",
			branch:   "renovate-update-all-dependencies-to-latest",
			budget:   24,
			expected: "renovate-up--s-to-latest",
		},
		{
			name:     "tail biased larger on odd budget",
			branch:   "aaaaaaaaaabbbbbbbbbbcccccccccc",
			budget:   13,
			expected: "aaaaa--cccccc", // keep=11 → head 5, tail 6
		},
		{"tiny budget falls back to head cut", "abcdefghij", 3, "abc"},
	}

	for _, tc := range cases {
		got := truncateBranch(tc.branch, tc.budget)
		if got != tc.expected {
			t.Errorf("%s: truncateBranch(%q, %d) = %q, want %q", tc.name, tc.branch, tc.budget, got, tc.expected)
		}
		if len(got) > tc.budget {
			t.Errorf("%s: result %q exceeds budget %d", tc.name, got, tc.budget)
		}
		if strings.HasPrefix(got, "-") || strings.HasSuffix(got, "-") {
			t.Errorf("%s: result %q must not start or end with a hyphen", tc.name, got)
		}
	}
}

func Test_buildDevVersion(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 1, 27, 9, 49, 59, 0, time.UTC)
	sha := "1a2b3c4d5e6f7a8b9c0d"

	t.Run("short branch keeps the full name", func(t *testing.T) {
		got, err := buildDevVersion("1.2.4", "my-feature", sha, ts, 63)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "1.2.4-dev.my-feature.2026-01-27.09-49-59.h1a2b3c4"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
		if !IsValidDev(got) {
			t.Errorf("%q is not a valid dev version", got)
		}
	})

	t.Run("long branch is middle-truncated to fit", func(t *testing.T) {
		got, err := buildDevVersion("1.2.4", "renovate-update-all-dependencies-to-latest", sha, ts, 63)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "1.2.4-dev.renovate-up--s-to-latest.2026-01-27.09-49-59.h1a2b3c4"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
		if len(got) > 63 {
			t.Errorf("result %q (len %d) exceeds the 63 char budget", got, len(got))
		}
		if !IsValidDev(got) {
			t.Errorf("%q is not a valid dev version", got)
		}
	})

	t.Run("multi-digit version base shrinks the branch budget", func(t *testing.T) {
		branch := "renovate-update-all-dependencies-to-latest"
		// The overhead is derived from len(base), so a wider base leaves less
		// room for the branch. Every result must still be valid and within 63.
		for _, base := range []string{"1.2.4", "10.12.346", "100.200.3456"} {
			got, err := buildDevVersion(base, branch, sha, ts, 63)
			if err != nil {
				t.Fatalf("base %s: unexpected error: %v", base, err)
			}
			if len(got) > 63 {
				t.Errorf("base %s: %q (len %d) exceeds the 63 char budget", base, got, len(got))
			}
			if !IsValidDev(got) {
				t.Errorf("base %s: %q is not a valid dev version", base, got)
			}
		}
	})

	t.Run("commit hash uses the 7-char short form", func(t *testing.T) {
		got, err := buildDevVersion("0.0.0", "main", sha, ts, 63)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.HasSuffix(got, ".h1a2b3c4") {
			t.Errorf("got %q, want trailing .h1a2b3c4", got)
		}
	})

	t.Run("errors when the fixed parts do not fit", func(t *testing.T) {
		_, err := buildDevVersion("1.2.4", "main", sha, ts, 20)
		if err == nil {
			t.Errorf("expected an error for an impossibly small budget")
		}
	})

	t.Run("per-branch order follows the timestamp", func(t *testing.T) {
		older, _ := buildDevVersion("1.2.4", "feature", sha, ts, 63)
		newer, _ := buildDevVersion("1.2.4", "feature", sha, ts.Add(time.Hour), 63)
		// Same branch and hash, so the timestamp segment decides order and the
		// lexical comparison of the full strings matches chronological order.
		if older >= newer {
			t.Errorf("expected %q < %q", older, newer)
		}
	})
}

func Test_resolveMaxVersionLength(t *testing.T) {
	cases := []struct {
		name       string
		configured int
		env        string
		expected   int
	}{
		{"default when nothing set", 0, "", defaultMaxVersionLength},
		{"configured value wins", 80, "100", 80},
		{"env used when not configured", 0, "100", 100},
		{"invalid env falls back to default", 0, "not-a-number", defaultMaxVersionLength},
		{"non-positive env falls back to default", 0, "0", defaultMaxVersionLength},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// An empty value is treated as unset by resolveMaxVersionLength.
			t.Setenv(maxVersionLengthEnvVarName, tc.env)
			got := resolveMaxVersionLength(tc.configured)
			if got != tc.expected {
				t.Errorf("resolveMaxVersionLength(%d) with env %q = %d, want %d", tc.configured, tc.env, got, tc.expected)
			}
		})
	}
}

// Test_DevVersionBranch_matchesBuildDevVersion is the anti-drift check that the
// whole export exists for: the segment DevVersionBranch hands a caller must be
// the segment a real dev version carries. A filter built from a segment that
// differs by one character matches no tag, and nothing reports it.
func Test_DevVersionBranch_matchesBuildDevVersion(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 1, 27, 9, 49, 59, 0, time.UTC)
	sha := "1a2b3c4d5e6f7a8b9c0d"

	branches := []string{
		"main",
		"my-feature",
		"Feature/MyThing",
		"renovate/update-all-dependencies-to-latest",
		"renovate/update-all-the-dependencies-to-their-very-latest-published-versions",
		"0042",
	}
	bases := []string{"0.0.0", "1.2.4", "10.12.346", "100.200.3456"}

	for _, base := range bases {
		for _, branch := range branches {
			segment, err := DevVersionBranch(branch, base, 63)
			if err != nil {
				t.Fatalf("DevVersionBranch(%q, %q): unexpected error: %v", branch, base, err)
			}
			// ResolveVersion sanitizes before it builds, so mirror that here.
			version, err := buildDevVersion(base, SanitizeBranchName(branch), sha, ts, 63)
			if err != nil {
				t.Fatalf("buildDevVersion(%q, %q): unexpected error: %v", base, branch, err)
			}
			want := "-dev." + segment + "."
			if !strings.Contains(version, want) {
				t.Errorf("branch %q base %q: version %q does not carry the segment %q from DevVersionBranch",
					branch, base, version, segment)
			}
		}
	}
}
