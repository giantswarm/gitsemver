// gitrepo-version prints or validates semVer-compatible version strings for git refs.
//
// Usage:
//
//	gitrepo-version [flags]
//	gitrepo-version validate [--type dev|rc|stable|any] <version>
//
// Without a subcommand it resolves and prints the version for a git ref:
//
//	For a ref that carries a stable tag (vX.Y.Z) it prints X.Y.Z.
//	For a pre-release tag (vX.Y.Z-rc.N) it prints X.Y.Z-rc.N.
//	For an untagged ref it prints a dev build:
//
//	    X.Y.(Z+1)-dev.<branch>.<YYYY-MM-DD>.<HH-MM-SS>
//
//	where X.Y.Z is the most recent stable ancestor tag reachable from the ref,
//	or 0.0.0 when none exists.
//
// The "validate" subcommand checks whether a version string matches the
// expected format.  It exits 0 and prints "valid" on success, exits 1 and
// prints "invalid" otherwise.
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
	if len(os.Args) >= 2 && os.Args[1] == "validate" {
		if err := runValidate(os.Args[2:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				os.Exit(0)
			}
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(2)
		}
		return
	}

	dir := flag.String("dir", ".", "path inside the git repository (resolved to the repo root)")
	ref := flag.String("ref", "HEAD", "git ref to resolve: branch name, tag, or commit SHA")
	flag.Parse()

	if err := runResolve(*dir, *ref); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runResolve(dir, ref string) error {
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

func runValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	typFlag := fs.String("type", "any", "version type to validate: dev, rc, stable, or any")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("validate requires exactly one version argument\nusage: gitrepo-version validate [--type dev|rc|stable|any] <version>")
	}
	version := fs.Arg(0)

	var ok bool
	switch *typFlag {
	case "stable":
		ok = gitrepo.IsValidStable(version)
	case "rc":
		ok = gitrepo.IsValidRC(version)
	case "dev":
		ok = gitrepo.IsValidDev(version)
	case "any":
		ok = gitrepo.IsValid(version)
	default:
		return fmt.Errorf("unknown --type %q: must be dev, rc, stable, or any", *typFlag)
	}

	if ok {
		fmt.Println("valid")
	} else {
		fmt.Println("invalid")
		os.Exit(1)
	}
	return nil
}
