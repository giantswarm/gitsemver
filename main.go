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
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/giantswarm/gitsemver/pkg/gitsemver"
)

// errInvalidVersion is returned by runValidate when the version string does
// not match the requested format.  It is not a usage error — the caller
// should print "invalid" and exit 1.
var errInvalidVersion = errors.New("invalid")

var validBumpTypes = []string{
	gitsemver.BumpTypePatch, gitsemver.BumpTypeMinor, gitsemver.BumpTypeMajor,
	gitsemver.BumpTypePatchRC, gitsemver.BumpTypeMinorRC, gitsemver.BumpTypeMajorRC,
	gitsemver.BumpTypeRC, gitsemver.BumpTypeRCRelease,
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		if errors.Is(err, errInvalidVersion) {
			fmt.Println("invalid")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "gitsemver",
		Short:         "Print or validate semVer-compatible version strings for git refs.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newVersionCmd(), newNextCmd(), newValidateCmd())
	return root
}

func newVersionCmd() *cobra.Command {
	var dir, ref string
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Resolve and print the semver version for a git ref.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion(dir, ref)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "path inside the git repository (resolved to the repo root)")
	cmd.Flags().StringVar(&ref, "ref", "HEAD", "git ref to resolve: branch name, tag, or commit SHA")
	return cmd
}

func newNextCmd() *cobra.Command {
	var lastTag string
	cmd := &cobra.Command{
		Use:       "next <bump-type>",
		Short:     "Compute the next semver tag after the last tag reachable from HEAD.",
		Args:      cobra.ExactArgs(1),
		ValidArgs: validBumpTypes,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNext(args[0], lastTag)
		},
	}
	cmd.Flags().StringVar(&lastTag, "last-tag", "", "base tag to compute from; when set, no git repo is needed")
	return cmd
}

func newValidateCmd() *cobra.Command {
	var typFlag string
	cmd := &cobra.Command{
		Use:   "validate <version>",
		Short: "Check whether a version string matches a known format. Exits 0 on success, 1 otherwise.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(typFlag, args[0])
		},
	}
	cmd.Flags().StringVar(&typFlag, "type", "any", "version type to validate: dev, rc, stable, or any")
	_ = cmd.RegisterFlagCompletionFunc("type", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"dev", "rc", "stable", "any"}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func runVersion(dir, ref string) error {
	ctx := context.Background()

	topLevel, err := gitsemver.TopLevel(dir)
	if err != nil {
		return fmt.Errorf("finding git root from %q: %w", dir, err)
	}

	repo, err := openRepo(topLevel)
	if err != nil {
		return err
	}

	version, err := repo.ResolveVersion(ctx, ref)
	if err != nil {
		return fmt.Errorf("resolving version for %q: %w", ref, err)
	}

	fmt.Println(version)
	return nil
}

func openRepo(topLevel string) (*gitsemver.Repo, error) {
	repo, err := gitsemver.New(gitsemver.Config{Dir: topLevel})
	if err != nil {
		// New() fails when there is no origin remote and no URL was given.
		// Fall back to a placeholder URL so we can still read local tags.
		var invalidCfg *gitsemver.InvalidConfigError
		if errors.As(err, &invalidCfg) {
			repo, err = gitsemver.New(gitsemver.Config{Dir: topLevel, URL: "_"})
		}
		if err != nil {
			return nil, fmt.Errorf("opening repository at %q: %w", topLevel, err)
		}
	}
	return repo, nil
}

func runNext(bumpType, lastTag string) error {
	if !slices.Contains(validBumpTypes, bumpType) {
		return fmt.Errorf("unknown bump type %q: must be one of %s", bumpType, strings.Join(validBumpTypes, ", "))
	}

	if lastTag != "" {
		tag := lastTag
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

	topLevel, err := gitsemver.TopLevel(".")
	if err != nil {
		return fmt.Errorf("finding git root: %w", err)
	}

	repo, err := openRepo(topLevel)
	if err != nil {
		return err
	}

	version, err := repo.NextVersion(ctx, bumpType)
	if err != nil {
		return fmt.Errorf("computing next version: %w", err)
	}

	fmt.Println(version)
	return nil
}

func runValidate(typFlag, version string) error {
	var ok bool
	switch typFlag {
	case "stable":
		ok = gitsemver.IsValidStable(version)
	case "rc":
		ok = gitsemver.IsValidRC(version)
	case "dev":
		ok = gitsemver.IsValidDev(version)
	case "any":
		ok = gitsemver.IsValid(version)
	default:
		return fmt.Errorf("unknown --type %q: must be dev, rc, stable, or any", typFlag)
	}

	if ok {
		fmt.Println("valid")
		return nil
	}
	return errInvalidVersion
}
