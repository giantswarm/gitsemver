// gitrepo-version prints the semVer-compatible version for a git ref.
//
// For a ref that carries a stable tag (vX.Y.Z) it prints X.Y.Z.
// For a pre-release tag (vX.Y.Z-rc.N) it prints X.Y.Z-rc.N.
// For an untagged ref it prints a dev build:
//
//	X.Y.(Z+1)-dev.<branch>.<YYYYMMDD>.<HHMMSS>
//
// where X.Y.Z is the most recent stable ancestor tag reachable from the ref,
// or 0.0.0 when none exists.
//
// Environment variables:
//
//	GS_BRANCH_NAME      Override the branch name embedded in dev builds.
//	                    Defaults to the HEAD branch of the repo, then "unknown".
//	GS_GIT_TAG_PREFIX   Monorepo support: only consider tags prefixed with
//	                    "<value>/", e.g. "module-a/v1.2.3".
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/giantswarm/gitrepo/pkg/gitrepo"
)

func main() {
	dir := flag.String("dir", ".", "path inside the git repository (resolved to the repo root)")
	ref := flag.String("ref", "HEAD", "git ref to resolve: branch name, tag, or commit SHA")
	flag.Parse()

	if err := run(*dir, *ref); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(dir, ref string) error {
	ctx := context.Background()

	topLevel, err := gitrepo.TopLevel(ctx, dir)
	if err != nil {
		return fmt.Errorf("finding git root from %q: %w", dir, err)
	}

	repo, err := gitrepo.New(gitrepo.Config{Dir: topLevel})
	if err != nil {
		// New() fails when there is no origin remote and no URL was given.
		// Fall back to a placeholder URL so we can still read local tags.
		var invalidCfg *gitrepo.InvalidConfigError
		if errors.As(err, &invalidCfg) {
			repo, err = gitrepo.New(gitrepo.Config{Dir: topLevel, URL: "_"})
		}
		if err != nil {
			return fmt.Errorf("opening repository at %q: %w", topLevel, err)
		}
	}

	version, err := repo.ResolveVersion(ctx, ref)
	if err != nil {
		return fmt.Errorf("resolving version for %q: %w", ref, err)
	}

	fmt.Println(version)
	return nil
}
