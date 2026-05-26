// gitsemver prints or validates semVer-compatible version strings for git refs.
//
// Usage:
//
//	gitsemver version [--dir <path>] [--ref <ref>]
//	gitsemver next <bump-type> [--last-tag <tag>]
//	gitsemver validate [--type dev|rc|stable|any] <version>
//
// The "version" subcommand resolves and prints the version for a git ref:
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
// The "next" subcommand computes the next semver tag from the highest-semver tag
// reachable from HEAD. Valid bump types from a stable tag: patch, minor, major,
// patch-rc, minor-rc, major-rc. Valid from an RC tag: rc, rc-release.
// Use --last-tag to skip git and compute directly from a supplied base tag.
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
//	                    "<value>/", e.g. "module-a/v1.2.3". The "next --last-tag"
//	                    flag accepts both prefixed ("module-a/v1.2.3") and bare
//	                    ("v1.2.3") forms; the prefix is stripped automatically.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/giantswarm/gitsemver/pkg/gitsemver"
)

// errInvalidVersion is returned by runValidate when the version string does
// not match the requested format.  It is not a usage error — the caller
// should print "invalid" and exit 1.
var errInvalidVersion = errors.New("invalid")

func main() {
	usage := func() {
		_, _ = fmt.Fprintf(os.Stderr, `Usage:
  gitsemver version [--dir <path>] [--ref <ref>]
  gitsemver next <patch|minor|major|patch-rc|minor-rc|major-rc|rc|rc-release> [--last-tag <tag>]
  gitsemver validate [--type dev|rc|stable|any] <version>

Subcommands:
  version     Resolve and print the semver version for a git ref.
  next        Compute the next semver tag after the last tag reachable from HEAD.
              Bump types from a stable tag: patch, minor, major, patch-rc, minor-rc, major-rc.
              Bump types from an RC tag:    rc (bump counter), rc-release (finalize to stable).
              Use --last-tag to supply the base tag explicitly (no git repo needed).
  validate    Check whether a version string matches a known format.
              Exits 0 and prints "valid" on success, 1 and "invalid" otherwise.
`)
	}

	sub := ""
	if len(os.Args) >= 2 {
		sub = os.Args[1]
	}

	switch sub {
	case "version":
		fs := flag.NewFlagSet("version", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		dir := fs.String("dir", ".", "path inside the git repository (resolved to the repo root)")
		ref := fs.String("ref", "HEAD", "git ref to resolve: branch name, tag, or commit SHA")
		if err := fs.Parse(os.Args[2:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				os.Exit(0)
			}
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(2)
		}
		if err := runResolve(*dir, *ref); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "next":
		if err := runNext(os.Args[2:]); err != nil {
			switch {
			case errors.Is(err, flag.ErrHelp):
				os.Exit(0)
			default:
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		}
	case "validate":
		if err := runValidate(os.Args[2:]); err != nil {
			switch {
			case errors.Is(err, flag.ErrHelp):
				os.Exit(0)
			case errors.Is(err, errInvalidVersion):
				fmt.Println("invalid")
				os.Exit(1)
			default:
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(2)
			}
		}
	default:
		usage()
		os.Exit(2)
	}
}

func runResolve(dir, ref string) error {
	ctx := context.Background()

	topLevel, err := gitsemver.TopLevel(ctx, dir)
	if err != nil {
		return fmt.Errorf("finding git root from %q: %w", dir, err)
	}

	repo, err := gitsemver.New(gitsemver.Config{Dir: topLevel})
	if err != nil {
		// New() fails when there is no origin remote and no URL was given.
		// Fall back to a placeholder URL so we can still read local tags.
		var invalidCfg *gitsemver.InvalidConfigError
		if errors.As(err, &invalidCfg) {
			repo, err = gitsemver.New(gitsemver.Config{Dir: topLevel, URL: "_"})
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

// validBumpTypes lists all accepted values for the "next" bump type argument.
var validBumpTypes = map[string]bool{
	"patch": true, "minor": true, "major": true,
	"patch-rc": true, "minor-rc": true, "major-rc": true,
	"rc": true, "rc-release": true,
}

func runNext(args []string) error {
	fs := flag.NewFlagSet("next", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	lastTag := fs.String("last-tag", "", "base tag to compute from; when set, no git repo is needed")

	// Extract the bump type by value before flag parsing so that both
	//   gitsemver next patch --last-tag v1.2.3
	//   gitsemver next --last-tag v1.2.3 patch
	// are accepted.
	var bumpType string
	var flagArgs []string
	for i, a := range args {
		prevIsLastTagFlag := i > 0 && (args[i-1] == "--last-tag" || args[i-1] == "-last-tag")
		if validBumpTypes[a] && bumpType == "" && !prevIsLastTagFlag {
			bumpType = a
		} else {
			flagArgs = append(flagArgs, a)
		}
	}

	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	switch {
	case bumpType == "" && fs.NArg() > 0:
		return fmt.Errorf("unknown bump type %q: must be one of patch, minor, major, patch-rc, minor-rc, major-rc, rc, rc-release", fs.Arg(0))
	case bumpType == "":
		return fmt.Errorf("next requires exactly one bump type argument\nusage: gitsemver next <patch|minor|major|patch-rc|minor-rc|major-rc|rc|rc-release> [--last-tag <tag>]")
	case fs.NArg() > 0:
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}

	if *lastTag != "" {
		tag := *lastTag
		if prefix := strings.TrimSpace(os.Getenv("GS_GIT_TAG_PREFIX")); prefix != "" {
			tag = strings.TrimPrefix(tag, prefix+"/")
		}
		next, err := gitsemver.ComputeNextVersion(tag, bumpType)
		if err != nil {
			return err
		}
		fmt.Println(next)
		return nil
	}

	ctx := context.Background()

	topLevel, err := gitsemver.TopLevel(ctx, ".")
	if err != nil {
		return fmt.Errorf("finding git root: %w", err)
	}

	repo, err := gitsemver.New(gitsemver.Config{Dir: topLevel})
	if err != nil {
		var invalidCfg *gitsemver.InvalidConfigError
		if errors.As(err, &invalidCfg) {
			repo, err = gitsemver.New(gitsemver.Config{Dir: topLevel, URL: "_"})
		}
		if err != nil {
			return fmt.Errorf("opening repository at %q: %w", topLevel, err)
		}
	}

	version, err := repo.NextVersion(ctx, bumpType)
	if err != nil {
		return err
	}

	fmt.Println(version)
	return nil
}

func runValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	typFlag := fs.String("type", "any", "version type to validate: dev, rc, stable, or any")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("validate requires exactly one version argument\nusage: gitsemver validate [--type dev|rc|stable|any] <version>")
	}
	version := fs.Arg(0)

	var ok bool
	switch *typFlag {
	case "stable":
		ok = gitsemver.IsValidStable(version)
	case "rc":
		ok = gitsemver.IsValidRC(version)
	case "dev":
		ok = gitsemver.IsValidDev(version)
	case "any":
		ok = gitsemver.IsValid(version)
	default:
		return fmt.Errorf("unknown --type %q: must be dev, rc, stable, or any", *typFlag)
	}

	if ok {
		fmt.Println("valid")
		return nil
	}
	return errInvalidVersion
}
