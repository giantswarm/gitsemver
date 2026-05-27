[![Go Reference](https://pkg.go.dev/badge/github.com/giantswarm/gitsemver.svg)](https://pkg.go.dev/github.com/giantswarm/gitsemver)
[![CircleCI](https://dl.circleci.com/status-badge/img/gh/giantswarm/gitsemver/tree/main.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/gh/giantswarm/gitsemver/tree/main)

# gitsemver

Library and CLI tool for computing a semVer-compatible version from a git reference.

## Version format

| Situation | Returned version |
|---|---|
| HEAD carries stable tag `vX.Y.Z` | `X.Y.Z` |
| HEAD carries pre-release tag `vX.Y.Z-rc.N` | `X.Y.Z-rc.N` |
| HEAD is untagged, stable ancestor `vX.Y.Z` reachable | `X.Y.(Z+1)-dev.<branch>.<YYYY-MM-DD>.<HH-MM-SS>` |
| HEAD is untagged, no stable ancestor reachable | `0.0.0-dev.<branch>.<YYYY-MM-DD>.<HH-MM-SS>` |

For untagged commits the base is the most recent **stable** ancestor tag reachable from the ref (RC and other pre-release tags are skipped). When no stable ancestor exists the version prefix is `0.0.0` with no patch increment.

## Environment variables

| Variable | Effect |
|---|---|
| `GS_BRANCH_NAME` | Override the branch name embedded in dev build versions. Defaults to the HEAD branch of the repo, then `"unknown"`. |
| `GS_GIT_TAG_PREFIX` | Monorepo support: only consider tags prefixed with `"<value>/"`, e.g. `module-a/v1.2.3`. |

## CLI — `gitsemver`

```
go install github.com/giantswarm/gitsemver@latest
```

```
Usage:
  gitsemver version [--dir <path>] [--ref <ref>]
  gitsemver next <patch|minor|major|patch-rc|minor-rc|major-rc|rc|rc-release> [--last-tag <tag>]
  gitsemver validate [--type dev|rc|stable|any] <version>
```

### version

Print the version for a git ref:

```
  --dir string   path inside the git repository (default ".", resolved to repo root)
  --ref string   git ref to resolve: branch name, tag, or commit SHA (default "HEAD")
```

```sh
$ GS_BRANCH_NAME=my-feature gitsemver version
1.2.4-dev.my-feature.2026-01-27.09-49-59

$ gitsemver version --ref v1.2.3
1.2.3
```

### next

Compute the next semver release tag after the highest-semver tag reachable from HEAD:

```sh
$ gitsemver next patch          # v1.2.3 ancestor → prints 1.2.4
$ gitsemver next minor-rc       # v1.2.3 ancestor → prints 1.3.0-rc.1
$ gitsemver next rc             # v1.3.0-rc.1 ancestor → prints 1.3.0-rc.2
$ gitsemver next rc-release     # v1.3.0-rc.1 ancestor → prints 1.3.0
```

Use `--last-tag` to supply the base explicitly — no git repository needed:

```sh
$ gitsemver next patch --last-tag v1.2.3
1.2.4
```

Valid bump types:

| Base tag | Bump type | Result |
|---|---|---|
| Stable `X.Y.Z` | `patch` | `X.Y.Z+1` |
| Stable `X.Y.Z` | `minor` | `X.Y+1.0` |
| Stable `X.Y.Z` | `major` | `X+1.0.0` |
| Stable `X.Y.Z` | `patch-rc` | `X.Y.Z+1-rc.1` |
| Stable `X.Y.Z` | `minor-rc` | `X.Y+1.0-rc.1` |
| Stable `X.Y.Z` | `major-rc` | `X+1.0.0-rc.1` |
| RC `X.Y.Z-rc.N` | `rc` | `X.Y.Z-rc.N+1` |
| RC `X.Y.Z-rc.N` | `rc-release` | `X.Y.Z` |

When no version tag is reachable from HEAD, `0.0.0` is used as the base.
Note: `rc` and `rc-release` require an actual reachable RC tag and cannot be used from the implicit `0.0.0` base — start a new RC series with `patch-rc`, `minor-rc`, or `major-rc` instead.

## Go library

```go
c := gitsemver.Config{
    Dir: "/path/to/some-repo",
    URL: "git@github.com:giantswarm/some-repo.git",
}
repo, err := gitsemver.New(c)
version, err := repo.ResolveVersion(ctx, "HEAD")
// e.g. "1.2.4-dev.my-feature.2026-01-27.09-49-59"
```
