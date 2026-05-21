[![Go Reference](https://pkg.go.dev/badge/github.com/giantswarm/gitrepo.svg)](https://pkg.go.dev/github.com/giantswarm/gitrepo)
[![CircleCI](https://dl.circleci.com/status-badge/img/gh/giantswarm/gitrepo/tree/main.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/gh/giantswarm/gitrepo/tree/main)

# gitrepo

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

## CLI — `gitrepo-version`

```
go install github.com/giantswarm/gitrepo@latest
```

```
Usage: gitrepo-version [flags]

  -dir string   path inside the git repository (default ".", resolved to repo root)
  -ref string   git ref to resolve: branch name, tag, or commit SHA (default "HEAD")
```

Example — print the version for the current working tree:

```sh
$ GS_BRANCH_NAME=my-feature gitrepo-version
1.2.4-dev.my-feature.2026-01-27.09-49-59

$ gitrepo-version --ref v1.2.3
1.2.3
```

## Go library

```go
c := gitrepo.Config{
    Dir: "/path/to/some-repo",
    URL: "git@github.com:giantswarm/some-repo.git",
}
repo, err := gitrepo.New(c)
version, err := repo.ResolveVersion(ctx, "HEAD")
// e.g. "1.2.4-dev.my-feature.2026-01-27.09-49-59"
```
