package gitsemver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// testGit runs the git CLI in dir. The layouts under test (linked worktrees,
// extensions.worktreeConfig) are created by git itself, so the tests see the
// files exactly as a person's checkout has them.
func testGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...) // #nosec G204 -- the tests' own git arguments
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
		"GIT_AUTHOR_DATE=2020-01-01T00:00:00Z", "GIT_COMMITTER_DATE=2020-01-01T00:00:00Z",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// testMainCheckout creates a main checkout with an origin remote, a stable tag
// and an untagged commit on top of it.
func testMainCheckout(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	testGit(t, dir, "init", "--initial-branch=main")
	testGit(t, dir, "remote", "add", "origin", "https://example.com/org/repo.git")
	testGit(t, dir, "commit", "--allow-empty", "-m", "first")
	testGit(t, dir, "tag", "v1.2.3")
	testGit(t, dir, "commit", "--allow-empty", "-m", "second")
	return dir
}

// testLinkedWorktree adds a linked worktree of mainDir on a new branch at the
// main checkout's HEAD. Its .git is a file pointing into mainDir/.git/worktrees.
func testLinkedWorktree(t *testing.T, mainDir string) string {
	t.Helper()
	wt := filepath.Join(t.TempDir(), "wt")
	testGit(t, mainDir, "worktree", "add", "-b", "feature", wt)
	return wt
}

func testResolveVersion(t *testing.T, path string) string {
	t.Helper()
	top, err := TopLevel(path)
	if err != nil {
		t.Fatalf("TopLevel(%q): %v", path, err)
	}
	repo, err := New(Config{Dir: top})
	if err != nil {
		t.Fatalf("New(%q): %v", top, err)
	}
	v, err := repo.ResolveVersion(t.Context(), "HEAD")
	if err != nil {
		t.Fatalf("ResolveVersion in %q: %v", top, err)
	}
	return v
}

func Test_TopLevel_linkedWorktree(t *testing.T) {
	wt := testLinkedWorktree(t, testMainCheckout(t))
	sub := filepath.Join(wt, "sub")
	if err := os.Mkdir(sub, 0o750); err != nil {
		t.Fatal(err)
	}
	want := testGit(t, sub, "rev-parse", "--show-toplevel")

	for _, path := range []string{wt, sub} {
		got, err := TopLevel(path)
		if err != nil {
			t.Fatalf("TopLevel(%q): %v", path, err)
		}
		if got != want {
			t.Errorf("TopLevel(%q) = %q, want %q", path, got, want)
		}
	}
}

func Test_ResolveVersion_linkedWorktree(t *testing.T) {
	t.Setenv(branchEnvVarName, "main")
	mainDir := testMainCheckout(t)
	wt := testLinkedWorktree(t, mainDir)

	want := testResolveVersion(t, mainDir)
	if got := testResolveVersion(t, wt); got != want {
		t.Errorf("version in linked worktree = %q, want %q (main checkout at the same commit)", got, want)
	}

	// A tag created after the worktree lives in the common directory.
	testGit(t, mainDir, "tag", "v1.2.4")
	if got := testResolveVersion(t, wt); got != "1.2.4" {
		t.Errorf("version in linked worktree = %q, want %q", got, "1.2.4")
	}
}

func Test_ResolveVersion_worktreeConfigExtension(t *testing.T) {
	t.Setenv(branchEnvVarName, "main")
	mainDir := testMainCheckout(t)
	want := testResolveVersion(t, mainDir)

	testGit(t, mainDir, "config", "extensions.worktreeConfig", "true")
	wt := testLinkedWorktree(t, mainDir)

	for _, dir := range []string{mainDir, wt} {
		if got := testResolveVersion(t, dir); got != want {
			t.Errorf("version in %q with extensions.worktreeConfig = %q, want %q", dir, got, want)
		}
	}

	// gitsemver reads the repository; it never rewrites its config.
	if got := testGit(t, mainDir, "config", "extensions.worktreeConfig"); got != "true" {
		t.Errorf("extensions.worktreeConfig = %q after resolving, want %q", got, "true")
	}
}

func Test_CurrentBranch_linkedWorktree(t *testing.T) {
	t.Setenv(branchEnvVarName, "")
	mainDir := testMainCheckout(t)
	testGit(t, mainDir, "config", "extensions.worktreeConfig", "true")
	t.Chdir(testLinkedWorktree(t, mainDir))

	if got := CurrentBranch(); got != "feature" {
		t.Errorf("CurrentBranch() in linked worktree = %q, want %q", got, "feature")
	}
}
