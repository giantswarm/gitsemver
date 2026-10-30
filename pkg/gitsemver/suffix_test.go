package gitsemver

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// A fork that builds on upstream's releases tags its own releases
// vX.Y.Z-<suffix>.N, where vX.Y.Z is the upstream release it builds on. The
// upstream tags are in the same repository.

func Test_Repo_ResolveVersion_suffix(t *testing.T) {
	ctx := context.Background()
	t.Setenv(branchEnvVarName, "test-branch")
	t.Setenv(tagSuffixEnvVarName, "gs")
	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	// tags[i] go on commit i of a line of commits; want is the version of the
	// last commit, with the dev version's "h<sha>" left off.
	testCases := []struct {
		name string
		tags [][]string
		want string
	}{
		{
			name: "release tag on the commit",
			tags: [][]string{{"v2.46.1"}, {"v2.46.1-gs.1"}},
			want: "2.46.1-gs.1",
		},
		{
			name: "highest release tag of one core on the commit",
			tags: [][]string{{"v2.46.1"}, {"v2.46.1-gs.2", "v2.46.1-gs.10", "v2.46.1-gs.1"}},
			want: "2.46.1-gs.10",
		},
		{
			name: "our release of an unchanged upstream release on its commit",
			tags: [][]string{{"v2.46.1", "v2.46.1-gs.1"}},
			want: "2.46.1-gs.1",
		},
		{
			name: "dev after the upstream release, before our first",
			tags: [][]string{{"v2.46.1"}, nil},
			want: "2.46.2-r57a8dd33t20200101010000h",
		},
		{
			name: "dev after our release",
			tags: [][]string{{"v2.46.1"}, {"v2.46.1-gs.1"}, nil},
			want: "2.46.2-r57a8dd33t20200101020000h",
		},
		{
			name: "upstream's release candidates and other suffixes are no base",
			tags: [][]string{{"v2.46.1"}, {"v2.47.0-rc.1"}, {"v2.47.0-ab.1"}, nil},
			want: "2.46.2-r57a8dd33t20200101030000h",
		},
		{
			name: "an upstream release on the commit is not ours",
			tags: [][]string{{"v2.46.1"}},
			want: "2.46.2-r57a8dd33t20200101000000h",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo, gitRepo := newTestRepo(t)
			var head plumbing.Hash
			for i, tags := range tc.tags {
				head = testCreateCommit(t, gitRepo, fmt.Sprintf("f%d.txt", i), base.Add(time.Duration(i)*time.Hour))
				for _, tag := range tags {
					if _, err := gitRepo.CreateTag(tag, head, nil); err != nil {
						t.Fatal(err)
					}
				}
			}

			got, err := repo.ResolveVersion(ctx, head.String())

			if err != nil {
				t.Fatalf("ResolveVersion: unexpected error %v", err)
			}
			want := tc.want
			if want[len(want)-1] == 'h' {
				want += head.String()[:devShortSHALen]
			}
			if got != want {
				t.Errorf("ResolveVersion = %q, want %q", got, want)
			}
		})
	}
}

func Test_Repo_HeadTag_suffix(t *testing.T) {
	ctx := context.Background()
	t.Setenv(tagSuffixEnvVarName, "gs")
	repo, gitRepo := newTestRepo(t)
	h := testCreateCommit(t, gitRepo, "a.txt", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	for _, tag := range []string{"v2.46.1", "v2.46.1-gs.2", "v2.46.1-gs.1"} {
		if _, err := gitRepo.CreateTag(tag, h, nil); err != nil {
			t.Fatal(err)
		}
	}

	tag, err := repo.HeadTag(ctx)

	if err != nil {
		t.Fatalf("HeadTag: unexpected error %v", err)
	}
	if tag != "v2.46.1-gs.2" {
		t.Errorf("HeadTag = %q, want %q", tag, "v2.46.1-gs.2")
	}
}

func Test_Repo_NextVersion_suffixRefused(t *testing.T) {
	ctx := context.Background()
	t.Setenv(tagSuffixEnvVarName, "gs")
	repo, gitRepo := newTestRepo(t)
	testCreateCommit(t, gitRepo, "a.txt", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))

	got, err := repo.NextVersion(ctx, BumpTypePatch)

	if !errors.Is(err, &InvalidConfigError{}) {
		t.Errorf("NextVersion = %q, %v, want an InvalidConfigError", got, err)
	}
}

func Test_tagSchemeFromEnv_invalid(t *testing.T) {
	testCases := []struct {
		name, prefix, suffix string
	}{
		{name: "prefix and suffix", prefix: "module-a", suffix: "gs"},
		{name: "suffix rc", suffix: "rc"},
		{name: "suffix with a dot", suffix: "gs.1"},
		{name: "numeric suffix", suffix: "1"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(tagPrefixEnvVarName, tc.prefix)
			t.Setenv(tagSuffixEnvVarName, tc.suffix)

			_, err := tagSchemeFromEnv()

			if !errors.Is(err, &InvalidConfigError{}) {
				t.Errorf("tagSchemeFromEnv: got %v, want an InvalidConfigError", err)
			}
		})
	}
}

// Upstream's newer release merged into the fork after the fork's own release:
// the dev version counts up from the highest of the two, not from the one
// with the newer commit date.
func Test_Repo_ResolveVersion_suffixMergedUpstream(t *testing.T) {
	ctx := context.Background()
	t.Setenv(branchEnvVarName, "test-branch")
	t.Setenv(tagSuffixEnvVarName, "gs")
	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	repo, gitRepo := newTestRepo(t)
	w, err := gitRepo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	tag := func(name string, h plumbing.Hash) {
		t.Helper()
		if _, err := gitRepo.CreateTag(name, h, nil); err != nil {
			t.Fatal(err)
		}
	}
	checkout := func(h plumbing.Hash) {
		t.Helper()
		if err := w.Checkout(&git.CheckoutOptions{Hash: h}); err != nil {
			t.Fatal(err)
		}
	}
	upstream := testCreateCommit(t, gitRepo, "a.txt", base)
	tag("v2.46.1", upstream)
	newerUpstream := testCreateCommit(t, gitRepo, "b.txt", base.Add(time.Hour))
	tag("v2.47.0", newerUpstream)
	checkout(upstream)
	ours := testCreateCommit(t, gitRepo, "c.txt", base.Add(2*time.Hour))
	tag("v2.46.1-gs.3", ours)
	sig := &object.Signature{Name: "test", Email: "t@t.com", When: base.Add(3 * time.Hour)}
	merge, err := w.Commit("merge", &git.CommitOptions{Author: sig, Committer: sig, Parents: []plumbing.Hash{ours, newerUpstream}, AllowEmptyCommits: true})
	if err != nil {
		t.Fatal(err)
	}

	got, err := repo.ResolveVersion(ctx, merge.String())

	if err != nil {
		t.Fatalf("ResolveVersion: unexpected error %v", err)
	}
	want := "2.47.1-r57a8dd33t20200101030000h" + merge.String()[:devShortSHALen]
	if got != want {
		t.Errorf("ResolveVersion = %q, want %q", got, want)
	}
}
