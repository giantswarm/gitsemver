package gitsemver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	billy "github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/storage/filesystem/dotgit"
)

// ignoredExtensions are git extensions that go-git refuses to open a
// repository with, but that do not change how gitsemver reads one.
// worktreeConfig only adds a per-worktree config.worktree file on top of the
// shared config; objects, refs and HEAD are laid out as without it.
// gitsemver reads only the shared config, so a per-worktree override in
// config.worktree, such as remote.origin.url, is not seen by New. Version
// resolution never uses the remote URL, so the version is unaffected.
var ignoredExtensions = []string{"worktreeConfig"}

// repoStorage is the filesystem storage of a checkout. Its config hides
// ignoredExtensions, so go-git opens the repository the way git does.
// gitsemver never writes the config of an existing repository, so the hidden
// extensions are never removed from disk.
type repoStorage struct {
	*filesystem.Storage
}

func (s *repoStorage) Config() (*config.Config, error) {
	cfg, err := s.Storage.Config()
	if err != nil {
		return nil, err
	}
	if cfg.Raw != nil && cfg.Raw.HasSection("extensions") {
		section := cfg.Raw.Section("extensions")
		for _, name := range ignoredExtensions {
			section.RemoveOption(name)
		}
	}
	return cfg, nil
}

// newRepoStorage returns the storage of the checkout whose top-level
// directory is dir, following a .git file the way git does. A missing .git
// yields an empty storage at dir/.git that a clone can fill.
func newRepoStorage(dir string) (*repoStorage, error) {
	gitDir, commonDir, err := gitDirs(dir)
	if err != nil {
		return nil, err
	}

	var fs billy.Filesystem
	if commonDir == "" {
		fs = osfs.New(gitDir)
	} else {
		fs = dotgit.NewRepositoryFilesystem(osfs.New(gitDir), osfs.New(commonDir))
	}

	return &repoStorage{filesystem.NewStorageWithOptions(fs, cache.NewObjectLRUDefault(), filesystem.Options{})}, nil
}

// gitDirs returns the git directory of the checkout at dir and, for a linked
// worktree, the common directory of the main repository that holds its
// objects, refs and config.
//
// A linked worktree or a submodule has a .git file reading
// "gitdir: <path>", relative to dir unless absolute. A linked worktree's git
// directory holds a commondir file naming the common directory, relative to
// the git directory unless absolute.
func gitDirs(dir string) (gitDir, commonDir string, err error) {
	dotGit := filepath.Join(dir, ".git")

	f, err := os.Stat(dotGit)
	if os.IsNotExist(err) || (err == nil && f.IsDir()) {
		return dotGit, "", nil
	} else if err != nil {
		return "", "", err
	}

	content, err := os.ReadFile(dotGit) // #nosec G304 -- the .git file of the checkout being read
	if err != nil {
		return "", "", err
	}
	const prefix = "gitdir: "
	line, _, _ := strings.Cut(string(content), "\n")
	if !strings.HasPrefix(line, prefix) {
		return "", "", &ExecutionFailedError{message: fmt.Sprintf("%#q has no %#q prefix", dotGit, prefix)}
	}
	gitDir = resolvePath(dir, strings.TrimPrefix(line, prefix))

	content, err = os.ReadFile(filepath.Join(gitDir, "commondir")) // #nosec G304 G703 -- inside the checkout's git directory
	if os.IsNotExist(err) {
		return gitDir, "", nil
	} else if err != nil {
		return "", "", err
	}

	return gitDir, resolvePath(gitDir, string(content)), nil
}

// resolvePath trims path and resolves it against base unless it is absolute.
func resolvePath(base, path string) string {
	path = strings.TrimSpace(path)
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(base, path)
}
