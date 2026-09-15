// Package gitsemver_test exercises the branch transformation through the public
// API only, the way an external consumer imports it. Keeping these cases out of
// package gitsemver is deliberate: it proves the surface a caller building a
// semVer filter for dev builds can actually reach.
package gitsemver_test

import (
	"testing"

	"github.com/giantswarm/gitsemver/v2/pkg/gitsemver"
)

func TestSanitizeBranchName(t *testing.T) {
	t.Parallel()

	// A branch whose sanitized form is well over the 63 character version
	// budget. SanitizeBranchName must not shorten it: truncation depends on the
	// version base and belongs to DevVersionBranch.
	longBranch := "renovate/update-all-the-dependencies-to-their-very-latest-published-versions"
	longSanitized := "renovate-update-all-the-dependencies-to-their-very-latest-published-versions"

	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain branch", "main", "main"},
		{"plain branch with hyphens", "my-feature", "my-feature"},
		{"slashes become hyphens", "feature/my-thing", "feature-my-thing"},
		{"nested slashes", "feat/sub/deep", "feat-sub-deep"},
		{"slashes and capitals", "Feature/MyThing", "feature-mything"},
		{"capitals lowercased", "UPPER", "upper"},
		{"underscores become hyphens", "feature_underscored", "feature-underscored"},
		{"already clean is unchanged", "already-clean-123", "already-clean-123"},
		{"consecutive invalid characters collapse", "feat//double-slash", "feat-double-slash"},
		{"leading invalid characters trimmed", "__leading", "leading"},
		{"trailing invalid characters trimmed", "trailing__", "trailing"},
		{"hyphen runs collapse so -- stays the truncation marker", "a--b", "a-b"},
		{"nothing usable falls back to unknown", "///", "unknown"},
		{"all-digit branch loses leading zeros", "0042", "42"},
		{"all-digit branch loses leading zeros, short", "007", "7"},
		{"all zeros collapse to a single zero", "000", "0"},
		{"numeric without a leading zero is kept", "42", "42"},
		{"a hyphen makes it alphanumeric, leading zero kept", "0-1", "0-1"},
		{"longer than 63 characters is not truncated", longBranch, longSanitized},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := gitsemver.SanitizeBranchName(tc.input)
			if got != tc.expected {
				t.Errorf("SanitizeBranchName(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestDevVersionBranch(t *testing.T) {
	t.Parallel()

	// A branch that fits its budget is carried whole; a longer one loses its
	// middle to the "--" marker. The expected strings are the worked examples
	// documented in the CHANGELOG for the 63 character default.
	cases := []struct {
		name     string
		branch   string
		base     string
		maxLen   int
		expected string
	}{
		{"plain branch fits whole", "main", "1.2.4", 63, "main"},
		{"plain branch with hyphens fits whole", "my-feature", "1.2.4", 63, "my-feature"},
		{"slashes and capitals", "Feature/MyThing", "1.2.4", 63, "feature-mything"},
		{
			name:     "long branch is middle-truncated",
			branch:   "renovate/update-all-dependencies-to-latest",
			base:     "1.2.4",
			maxLen:   63,
			expected: "renovate-up--s-to-latest",
		},
		{
			// 76 characters once sanitized, far past any budget.
			name:     "branch longer than 63 characters",
			branch:   "renovate/update-all-the-dependencies-to-their-very-latest-published-versions",
			base:     "1.2.4",
			maxLen:   63,
			expected: "renovate-up--ed-versions",
		},
		{
			// A wider base leaves fewer characters for the branch.
			name:     "wider version base shrinks the branch",
			branch:   "renovate/update-all-dependencies-to-latest",
			base:     "100.200.3456",
			maxLen:   63,
			expected: "renovat--o-latest",
		},
		{"maxLen <= 0 falls back to the default", "main", "1.2.4", 0, "main"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := gitsemver.DevVersionBranch(tc.branch, tc.base, tc.maxLen)
			if err != nil {
				t.Fatalf("DevVersionBranch(%q, %q, %d) returned an error: %v", tc.branch, tc.base, tc.maxLen, err)
			}
			if got != tc.expected {
				t.Errorf("DevVersionBranch(%q, %q, %d) = %q, want %q", tc.branch, tc.base, tc.maxLen, got, tc.expected)
			}
		})
	}
}

func TestDevVersionBranchErrorsWhenNothingFits(t *testing.T) {
	t.Parallel()

	if _, err := gitsemver.DevVersionBranch("main", "1.2.4", 20); err == nil {
		t.Error("expected an error when the fixed parts of the version already exceed maxLen")
	}
}
