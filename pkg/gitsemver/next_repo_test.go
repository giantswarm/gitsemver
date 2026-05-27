package gitsemver

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// baseTime is a fixed timestamp used across next_repo tests.
var nextTestBaseTime = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

func Test_Repo_NextVersion_stableAncestor_patch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo, gitRepo := newTestRepo(t)

	h := testCreateCommit(t, gitRepo, "stable.txt", nextTestBaseTime)
	if _, err := gitRepo.CreateTag("v1.2.3", h, nil); err != nil {
		t.Fatal(err)
	}
	testCreateCommit(t, gitRepo, "next.txt", nextTestBaseTime.Add(time.Hour))

	got, err := repo.NextVersion(ctx, "patch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1.2.4" {
		t.Errorf("NextVersion = %q, want %q", got, "1.2.4")
	}
}

func Test_Repo_NextVersion_stableOnHEAD_minorRC(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo, gitRepo := newTestRepo(t)

	h := testCreateCommit(t, gitRepo, "stable.txt", nextTestBaseTime)
	if _, err := gitRepo.CreateTag("v1.2.3", h, nil); err != nil {
		t.Fatal(err)
	}

	got, err := repo.NextVersion(ctx, "minor-rc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1.3.0-rc.1" {
		t.Errorf("NextVersion = %q, want %q", got, "1.3.0-rc.1")
	}
}

func Test_Repo_NextVersion_noTags_patch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo, gitRepo := newTestRepo(t)

	testCreateCommit(t, gitRepo, "a.txt", nextTestBaseTime)

	got, err := repo.NextVersion(ctx, "patch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "0.0.1" {
		t.Errorf("NextVersion = %q, want %q", got, "0.0.1")
	}
}

func Test_Repo_NextVersion_noTags_minor(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo, gitRepo := newTestRepo(t)

	testCreateCommit(t, gitRepo, "a.txt", nextTestBaseTime)

	got, err := repo.NextVersion(ctx, "minor")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "0.1.0" {
		t.Errorf("NextVersion = %q, want %q", got, "0.1.0")
	}
}

func Test_Repo_NextVersion_noTags_majorRC(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo, gitRepo := newTestRepo(t)

	testCreateCommit(t, gitRepo, "a.txt", nextTestBaseTime)

	got, err := repo.NextVersion(ctx, "major-rc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1.0.0-rc.1" {
		t.Errorf("NextVersion = %q, want %q", got, "1.0.0-rc.1")
	}
}

func Test_Repo_NextVersion_RCAncestor_rcBump(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo, gitRepo := newTestRepo(t)

	h := testCreateCommit(t, gitRepo, "rc.txt", nextTestBaseTime)
	if _, err := gitRepo.CreateTag("v1.2.3-rc.2", h, nil); err != nil {
		t.Fatal(err)
	}
	testCreateCommit(t, gitRepo, "next.txt", nextTestBaseTime.Add(time.Hour))

	got, err := repo.NextVersion(ctx, "rc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1.2.3-rc.3" {
		t.Errorf("NextVersion = %q, want %q", got, "1.2.3-rc.3")
	}
}

func Test_Repo_NextVersion_RCAncestor_rcRelease(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo, gitRepo := newTestRepo(t)

	h := testCreateCommit(t, gitRepo, "rc.txt", nextTestBaseTime)
	if _, err := gitRepo.CreateTag("v1.2.3-rc.2", h, nil); err != nil {
		t.Fatal(err)
	}
	testCreateCommit(t, gitRepo, "next.txt", nextTestBaseTime.Add(time.Hour))

	got, err := repo.NextVersion(ctx, "rc-release")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1.2.3" {
		t.Errorf("NextVersion = %q, want %q", got, "1.2.3")
	}
}

// RC tag is directly on HEAD (not an ancestor) — rc bump must still work.
func Test_Repo_NextVersion_rcOnHEAD_rcBump(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo, gitRepo := newTestRepo(t)

	h := testCreateCommit(t, gitRepo, "rc.txt", nextTestBaseTime)
	if _, err := gitRepo.CreateTag("v1.2.3-rc.2", h, nil); err != nil {
		t.Fatal(err)
	}

	got, err := repo.NextVersion(ctx, "rc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1.2.3-rc.3" {
		t.Errorf("NextVersion = %q, want %q", got, "1.2.3-rc.3")
	}
}

// RC tag is directly on HEAD (not an ancestor) — rc-release must still work.
func Test_Repo_NextVersion_rcOnHEAD_rcRelease(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo, gitRepo := newTestRepo(t)

	h := testCreateCommit(t, gitRepo, "rc.txt", nextTestBaseTime)
	if _, err := gitRepo.CreateTag("v1.2.3-rc.2", h, nil); err != nil {
		t.Fatal(err)
	}

	got, err := repo.NextVersion(ctx, "rc-release")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1.2.3" {
		t.Errorf("NextVersion = %q, want %q", got, "1.2.3")
	}
}

// Both a stable tag (v1.2.3) and a higher RC tag (v1.3.0-rc.1) are reachable.
// The RC has higher semver precedence, so it is used as the base.
func Test_Repo_NextVersion_stableAndRC_rcBump(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo, gitRepo := newTestRepo(t)

	stableH := testCreateCommit(t, gitRepo, "stable.txt", nextTestBaseTime)
	if _, err := gitRepo.CreateTag("v1.2.3", stableH, nil); err != nil {
		t.Fatal(err)
	}
	rcH := testCreateCommit(t, gitRepo, "rc.txt", nextTestBaseTime.Add(time.Hour))
	if _, err := gitRepo.CreateTag("v1.3.0-rc.1", rcH, nil); err != nil {
		t.Fatal(err)
	}
	testCreateCommit(t, gitRepo, "next.txt", nextTestBaseTime.Add(2*time.Hour))

	got, err := repo.NextVersion(ctx, "rc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1.3.0-rc.2" {
		t.Errorf("NextVersion = %q, want %q", got, "1.3.0-rc.2")
	}
}

// Both a stable tag (v1.2.3) and a higher RC tag (v1.3.0-rc.1) are reachable.
// patch/minor/major bump types are invalid when the highest reachable tag is an RC.
func Test_Repo_NextVersion_stableAndRC_patchErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo, gitRepo := newTestRepo(t)

	stableH := testCreateCommit(t, gitRepo, "stable.txt", nextTestBaseTime)
	if _, err := gitRepo.CreateTag("v1.2.3", stableH, nil); err != nil {
		t.Fatal(err)
	}
	rcH := testCreateCommit(t, gitRepo, "rc.txt", nextTestBaseTime.Add(time.Hour))
	if _, err := gitRepo.CreateTag("v1.3.0-rc.1", rcH, nil); err != nil {
		t.Fatal(err)
	}
	testCreateCommit(t, gitRepo, "next.txt", nextTestBaseTime.Add(2*time.Hour))

	_, err := repo.NextVersion(ctx, "patch")
	var execErr *ExecutionFailedError
	if !errors.As(err, &execErr) {
		t.Fatalf("err = %T (%v), want *ExecutionFailedError", err, err)
	}
	if !strings.Contains(execErr.Error(), "requires a stable last tag") {
		t.Errorf("unexpected error message: %v", execErr)
	}
}

// A tag on a side branch that is not reachable from HEAD must be ignored.
func Test_Repo_NextVersion_nonReachableTagIgnored(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo, gitRepo := newTestRepo(t)

	base := testCreateCommit(t, gitRepo, "base.txt", nextTestBaseTime)
	if _, err := gitRepo.CreateTag("v1.0.0", base, nil); err != nil {
		t.Fatal(err)
	}

	// Side branch with a higher tag — not reachable from main HEAD.
	w, err := gitRepo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Checkout(&git.CheckoutOptions{
		Hash:   base,
		Branch: plumbing.NewBranchReferenceName("side"),
		Create: true,
	}); err != nil {
		t.Fatal(err)
	}
	sideH := testCreateCommit(t, gitRepo, "side.txt", nextTestBaseTime.Add(time.Hour))
	if _, err := gitRepo.CreateTag("v9.9.9", sideH, nil); err != nil {
		t.Fatal(err)
	}

	// Back to the default branch, add a commit past v1.0.0.
	if err := w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("master"),
	}); err != nil {
		t.Fatal(err)
	}
	testCreateCommit(t, gitRepo, "next.txt", nextTestBaseTime.Add(2*time.Hour))

	got, err := repo.NextVersion(ctx, "patch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1.0.1" {
		t.Errorf("NextVersion = %q, want %q — non-reachable v9.9.9 must be ignored", got, "1.0.1")
	}
}

// When multiple stable ancestors are reachable, the highest semver wins (not the nearest commit).
// v1.3.0 is on the OLDER commit and v1.0.0 on the NEWER one so that log order (newest-first)
// disagrees with semver order — a naive "first tagged commit found" implementation would return
// v1.0.0, while the correct implementation returns v1.3.0.
func Test_Repo_NextVersion_multipleStable_highestSemverWins(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo, gitRepo := newTestRepo(t)

	// Older commit carries the HIGHER semver tag.
	h1 := testCreateCommit(t, gitRepo, "a.txt", nextTestBaseTime)
	if _, err := gitRepo.CreateTag("v1.3.0", h1, nil); err != nil {
		t.Fatal(err)
	}
	// Newer commit carries the LOWER semver tag.
	h2 := testCreateCommit(t, gitRepo, "b.txt", nextTestBaseTime.Add(time.Hour))
	if _, err := gitRepo.CreateTag("v1.0.0", h2, nil); err != nil {
		t.Fatal(err)
	}
	testCreateCommit(t, gitRepo, "c.txt", nextTestBaseTime.Add(2*time.Hour))

	got, err := repo.NextVersion(ctx, "patch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1.3.1" {
		t.Errorf("NextVersion = %q, want %q — highest semver v1.3.0 must win even when on older commit", got, "1.3.1")
	}
}

// With GS_GIT_TAG_PREFIX set, only prefixed tags are considered.
func Test_Repo_NextVersion_monorepoPrefixFiltering(t *testing.T) {
	ctx := context.Background()
	repo, gitRepo := newTestRepo(t)

	h1 := testCreateCommit(t, gitRepo, "a.txt", nextTestBaseTime)
	// This tag should be ignored — wrong prefix.
	if _, err := gitRepo.CreateTag("other/v9.9.9", h1, nil); err != nil {
		t.Fatal(err)
	}
	h2 := testCreateCommit(t, gitRepo, "b.txt", nextTestBaseTime.Add(time.Hour))
	// This tag should be used.
	if _, err := gitRepo.CreateTag("module-a/v1.2.3", h2, nil); err != nil {
		t.Fatal(err)
	}
	testCreateCommit(t, gitRepo, "c.txt", nextTestBaseTime.Add(2*time.Hour))

	t.Setenv(tagPrefixEnvVarName, "module-a")

	got, err := repo.NextVersion(ctx, "patch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1.2.4" {
		t.Errorf("NextVersion = %q, want %q", got, "1.2.4")
	}
}

// A commit carrying two version tags must produce an error, not silently
// return a randomly chosen version (regression test for the dead versionTags
// guard that was scoped inside the inner per-tag loop).
func Test_Repo_NextVersion_multipleTagsOnSameCommit_errors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo, gitRepo := newTestRepo(t)

	h := testCreateCommit(t, gitRepo, "a.txt", nextTestBaseTime)
	if _, err := gitRepo.CreateTag("v1.0.0", h, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := gitRepo.CreateTag("v1.0.1", h, nil); err != nil {
		t.Fatal(err)
	}

	_, err := repo.NextVersion(ctx, "patch")
	var execErr *ExecutionFailedError
	if !errors.As(err, &execErr) {
		t.Fatalf("err = %T (%v), want *ExecutionFailedError", err, err)
	}
	if !strings.Contains(execErr.Error(), "multiple version tags") {
		t.Errorf("unexpected error message: %v", execErr)
	}
}

// Invalid bump type always errors regardless of repo state.
func Test_Repo_NextVersion_invalidBumpType(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo, gitRepo := newTestRepo(t)

	testCreateCommit(t, gitRepo, "a.txt", nextTestBaseTime)

	_, err := repo.NextVersion(ctx, "bad")
	var execErr *ExecutionFailedError
	if !errors.As(err, &execErr) {
		t.Fatalf("err = %T (%v), want *ExecutionFailedError", err, err)
	}
	if !strings.Contains(execErr.Error(), "unknown bump type") {
		t.Errorf("unexpected error message: %v", execErr)
	}
}
