package gitrepo

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-errors/errors"
	billy "github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

var tagRegex = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+`)
var stableTagRegex = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

var tagPrefixEnvVarName = "GS_GIT_TAG_PREFIX"
var prefixedTagRegex = regexp.MustCompile(`^[a-zA-Z0-9-_]+/v[0-9]+\.[0-9]+\.[0-9]+`)

var branchEnvVarName = "GS_BRANCH_NAME"

// branchSanitizeRegex replaces one or more consecutive invalid semVer
// pre-release characters with a single hyphen.
var branchSanitizeRegex = regexp.MustCompile(`[^a-zA-Z0-9-]+`)

// nowFunc is the clock used for dev build timestamps; overridable in tests.
var nowFunc = time.Now

const unknownBranch = "unknown"

// currentBranch returns the name of the branch to embed in dev build versions.
// Priority: GS_BRANCH_NAME env var > HEAD branch of CWD git repo > "unknown".
func currentBranch() string {
	if b := strings.TrimSpace(os.Getenv(branchEnvVarName)); b != "" {
		return b
	}
	repo, err := git.PlainOpenWithOptions(".", &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return unknownBranch
	}
	head, err := repo.Head()
	if err != nil {
		return unknownBranch
	}
	return head.Name().Short()
}

func sanitizeBranchName(branch string) string {
	return branchSanitizeRegex.ReplaceAllString(branch, "-")
}

func incrementPatch(version string) (string, error) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return "", &ExecutionFailedError{message: fmt.Sprintf("invalid version %q: expected X.Y.Z", version)}
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", &ExecutionFailedError{message: fmt.Sprintf("invalid patch in version %q: %v", version, err)}
	}
	parts[2] = strconv.Itoa(patch + 1)
	return strings.Join(parts, "."), nil
}

type Config struct {
	AuthBasicToken string
	Dir            string
	URL            string
}

type Repo struct {
	url string

	auth     transport.AuthMethod
	storage  *filesystem.Storage
	worktree billy.Filesystem
}

func New(config Config) (*Repo, error) {
	if config.Dir == "" {
		return nil, &InvalidConfigError{message: fmt.Sprintf("%T.Dir must not be empty", config)}
	}

	var auth transport.AuthMethod
	{
		if config.AuthBasicToken != "" {
			auth = &http.BasicAuth{
				Username: "can-be-anything-but-not-empty",
				Password: config.AuthBasicToken,
			}
		}
	}

	worktree := osfs.New(config.Dir)
	fs := osfs.New(filepath.Join(config.Dir, ".git"))
	storage := filesystem.NewStorageWithOptions(fs, cache.NewObjectLRUDefault(), filesystem.Options{})

	// When URL is not configured assume the repository is cloned on disk
	// and take the URL or origin remote.
	if config.URL == "" {
		repo, err := git.Open(storage, worktree)
		if err != nil {
			return nil, &InvalidConfigError{message: fmt.Sprintf("%T.URL not set and failed to open repository with error %#q", config, err)}
		}

		remoteName := "origin"

		remote, err := repo.Remote(remoteName)
		if err != nil {
			return nil, &InvalidConfigError{message: fmt.Sprintf("%T.URL not set and failed to find remote with name %#q with error %#q", config, remoteName, err)}
		}

		// According to
		// https://godoc.org/gopkg.in/src-d/go-git.v4/config#RemoteConfig:
		//
		//	URLs the URLs of a remote repository. It must be
		//	non-empty. Fetch will always use the first URL, while
		//	push will use all of them.
		//
		config.URL = remote.Config().URLs[0]
	}

	r := &Repo{
		url: config.URL,

		auth:     auth,
		storage:  storage,
		worktree: worktree,
	}

	return r, nil
}

// EnsureUpToDate fetches latest changes from remote.
func (r *Repo) EnsureUpToDate(ctx context.Context) error {
	cloneOpts := &git.CloneOptions{
		Auth:       r.auth,
		URL:        r.url,
		NoCheckout: true,
	}

	_, err := r.worktree.Stat("/")
	if os.IsNotExist(err) {
		// Repo is empty so perform an initial checkout
		cloneOpts.NoCheckout = false
	} else if err != nil {
		return err
	}

	repo, err := git.Clone(r.storage, r.worktree, cloneOpts)
	if errors.Is(err, git.ErrRepositoryAlreadyExists) {
		repo, err = git.Open(r.storage, r.worktree)
		if err != nil {
			return err
		}
	} else if errors.Is(err, transport.ErrRepositoryNotFound) {
		return &RepositoryNotFoundError{message: fmt.Sprintf("%#q", r.url)}
	} else if err != nil {
		return err
	}

	fetchOpts := &git.FetchOptions{
		Auth:  r.auth,
		Force: true,
	}

	err = repo.Fetch(fetchOpts)
	if errors.Is(err, git.NoErrAlreadyUpToDate) {
		// Fall through.
	} else if errors.Is(err, transport.ErrRepositoryNotFound) {
		// This could happen if the repository does not exist, but you already have the folder on the filesystem.
		// In that case Fetch will be the first to realise that repo does not exist since Clone only performs an Open.
		// Also, Clone creates the folder on the filesystem even if it fails, so you end simulate the same situation when
		// you call EnsureUpToDate more that once on the same non-existent repo.
		return &RepositoryNotFoundError{message: fmt.Sprintf("%#q", r.url)}
	} else if err != nil {
		return err
	}

	return nil
}

// HeadBranch returns branch name for the HEAD ref.
func (r *Repo) HeadBranch(ctx context.Context) (string, error) {
	repo, err := git.Open(r.storage, r.worktree)
	if err != nil {
		return "", err
	}

	head, err := repo.Head()
	if err != nil {
		return "", err
	}

	return head.Name().Short(), nil
}

// HeadSHA returns sha for the HEAD ref.
func (r *Repo) HeadSHA(ctx context.Context) (string, error) {
	repo, err := git.Open(r.storage, r.worktree)
	if err != nil {
		return "", err
	}

	head, err := repo.Head()
	if err != nil {
		return "", err
	}

	return head.Hash().String(), nil
}

// HeadTag returns tag for the HEAD ref.
//
// If GS_TAG_PREFIX environment variable is set, it looks for tags prefixed with that.
// For example, when the value is 'module-a', it filters found tags to 'module-a/v1.2.0',
// must match <module_name>/v<semantic_version>.
//
// Note: if GS_TAG_PREFIX is not set, all tags matching the prefixed tag regex are filtered out!
//
// It returns error handled by IsReferenceNotFound if the HEAD ref is not
// tagged.
func (r *Repo) HeadTag(ctx context.Context) (string, error) {
	repo, err := git.Open(r.storage, r.worktree)
	if err != nil {
		return "", err
	}

	head, err := repo.Head()
	if err != nil {
		return "", err
	}

	tagsBySHA, err := r.tags(repo)
	if err != nil {
		return "", err
	}

	tags := tagsBySHA[head.Hash().String()]

	tagPrefix := os.Getenv(tagPrefixEnvVarName)

	var filteredTags []string
	if tagPrefix != "" {
		for _, tag := range tags {
			if strings.HasPrefix(tag, tagPrefix+"/") {
				filteredTags = append(filteredTags, tag)
			}
		}
	} else {
		for _, tag := range tags {
			if !prefixedTagRegex.MatchString(tag) {
				filteredTags = append(filteredTags, tag)
			}
		}
	}

	if len(filteredTags) == 0 {
		return "", &ReferenceNotFoundError{message: fmt.Sprintf("HEAD ref is not tagged (filtered for prefix: '%s')", tagPrefix)}
	}
	if len(filteredTags) > 1 {
		return "", &ExecutionFailedError{message: fmt.Sprintf("HEAD ref has multiple tags %v (filtered for prefix: '%s')", filteredTags, tagPrefix)}
	}

	return filteredTags[0], nil
}

// ResolveVersion resolves the version for a git reference:
//
//   - Stable tag vX.Y.Z on the commit → returns "X.Y.Z"
//   - Pre-release tag vX.Y.Z-<pre> on the commit → returns "X.Y.Z-<pre>" (e.g. "1.2.3-rc.1")
//   - Untagged commit → returns a semVer dev build:
//     "X.Y.(Z+1)-dev.<branch>.<YYYYMMDD>.<HHMMSS>"
//     where X.Y.Z is the most recent stable (non-pre-release) ancestor tag reachable
//     from the reference, or "0.0.0" when no stable ancestor exists. The branch name
//     is resolved from GS_BRANCH_NAME env var, then the HEAD branch of the CWD git
//     repo, then "unknown". Non-reachable tags are never used as the base.
//
// If GS_GIT_TAG_PREFIX is set, only tags prefixed with "<value>/" are considered,
// e.g. "module-a/v1.2.3". The prefix, separator, and "v" are stripped from the result.
//
// It returns an error handled by IsReferenceNotFound if the reference cannot be resolved.
func (r *Repo) ResolveVersion(ctx context.Context, ref string) (string, error) {
	repo, err := git.Open(r.storage, r.worktree)
	if err != nil {
		return "", err
	}

	tagPrefix := os.Getenv(tagPrefixEnvVarName)

	versionsByHash := map[string]string{}
	stableVersionsByHash := map[string]string{}
	{

		tagsByHash, err := r.tags(repo)
		if err != nil {
			return "", err
		}
		for hash, tags := range tagsByHash {
			for _, t := range tags {
				var versionTags []string

				if tagPrefix != "" {
					if prefixedTagRegex.MatchString(t) && strings.HasPrefix(t, tagPrefix+"/") {
						versionTags = append(versionTags, t)
						version := strings.TrimPrefix(strings.TrimPrefix(t, tagPrefix+"/"), "v")
						versionsByHash[hash] = version
						tagWithoutPrefix := strings.TrimPrefix(t, tagPrefix+"/")
						if stableTagRegex.MatchString(tagWithoutPrefix) {
							stableVersionsByHash[hash] = version
						}
					}
				} else {
					if tagRegex.MatchString(t) {
						versionTags = append(versionTags, t)
						version := strings.TrimPrefix(t, "v")
						versionsByHash[hash] = version
						if stableTagRegex.MatchString(t) {
							stableVersionsByHash[hash] = version
						}
					}
				}

				if len(versionTags) > 1 {
					return "", &ExecutionFailedError{message: fmt.Sprintf("multiple version tags %#v found for hash %#q", versionTags, hash)}
				}
			}
		}

	}

	var commit *object.Commit
	{
		hash, err := repo.ResolveRevision(plumbing.Revision(ref))
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return "", &ReferenceNotFoundError{message: fmt.Sprintf("%#q", ref)}
		} else if err != nil {
			return "", err
		}

		commit, err = repo.CommitObject(*hash)
		if err != nil {
			return "", err
		}
	}

	// When the commit is tagged return the tag.
	{
		version, ok := versionsByHash[commit.Hash.String()]
		if ok {
			return version, nil
		}
	}

	// Otherwise walk ancestors to find the most recent stable tagged parent,
	// then return a semVer-compatible dev build version.
	// The queue only ever holds commits reachable from `commit` (parents of parents,
	// etc.), so stableVersionsByHash is only consulted for actual ancestors — tags
	// on unrelated branches are never returned.
	var pseudoVersion string
	{
		var lastStableVersion string

		queue := []*object.Commit{
			commit,
		}

		for {
			if len(queue) == 0 {
				lastStableVersion = "0.0.0"
				break
			}

			// Pop the first element from the queue.
			c := queue[0]
			queue = queue[1:]

			// Check if this commit has a stable tag. RC and other pre-release
			// tags are intentionally skipped so the dev build base is always
			// derived from the last stable release.
			v, ok := stableVersionsByHash[c.Hash.String()]
			if ok {
				lastStableVersion = v
				break
			}

			// Push all the parents to the queue.
			err = c.Parents().ForEach(func(p *object.Commit) error {
				// If the commit is already in the queue skip
				// it. This is possible multiple commits have
				// the same parent. Adding all of them to the
				// queue may lead in exponential growth of the
				// queue resulting in extremely long execution.
				for _, c := range queue {
					if c.Hash == p.Hash {
						return nil
					}
				}

				queue = append(queue, p)
				return nil
			})
			if err != nil {
				return "", err
			}

			// Sort commits in the queue by commit date in
			// descending order to find the most recent tag first.
			sort.Slice(queue, func(i, j int) bool { return queue[i].Committer.When.After(queue[j].Committer.When) })
		}

		base, err := incrementPatch(lastStableVersion)
		if err != nil {
			return "", err
		}
		branch := sanitizeBranchName(currentBranch())
		t := nowFunc().UTC()
		pseudoVersion = fmt.Sprintf("%s-dev.%s.%s.%s",
			base,
			branch,
			t.Format("20060102"),
			t.Format("150405"),
		)
	}

	return pseudoVersion, nil
}

// GetFileContent retrieves content of file stored at path on version specified in ref.
// When empty ref defaults to master branch.
func (r *Repo) GetFileContent(path, ref string) ([]byte, error) {
	worktree, err := r.checkoutRef(ref)
	if err != nil {
		return nil, err
	}

	file, err := worktree.Filesystem.Open(path)
	if os.IsNotExist(err) {
		return nil, &FileNotFoundError{message: fmt.Sprintf("%#q", path)}
	} else if err != nil {
		return nil, err
	}

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return content, nil
}

// GetFolderContent retrieves content of a folder stored at path on version specified in ref.
// When empty ref defaults to master branch.
func (r *Repo) GetFolderContent(path, ref string) ([]os.FileInfo, error) {
	worktree, err := r.checkoutRef(ref)
	if err != nil {
		return nil, err
	}

	files, err := worktree.Filesystem.ReadDir(path)
	if os.IsNotExist(err) {
		return nil, &FolderNotFoundError{message: fmt.Sprintf("%#q", path)}
	} else if err != nil {
		return nil, err
	}

	return files, nil
}

func (r *Repo) checkoutRef(ref string) (*git.Worktree, error) {
	repo, err := git.Open(r.storage, r.worktree)
	if err != nil {
		return nil, err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	// When empty CheckoutOptions defaults to master branch.
	opt := &git.CheckoutOptions{}
	if ref != "" {
		hash, err := repo.ResolveRevision(plumbing.Revision(ref))
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return nil, &ReferenceNotFoundError{message: fmt.Sprintf("%#q", ref)}
		} else if err != nil {
			return nil, err
		}

		head, err := repo.Head()
		if err != nil {
			return nil, err
		}

		if head.Hash() == *hash {
			// We're already at the right ref, no need to checkout
			return worktree, nil
		}

		opt.Hash = *hash
	}

	err = worktree.Checkout(opt)
	if err != nil {
		return nil, err
	}

	err = worktree.Clean(&git.CleanOptions{Dir: true})
	if err != nil {
		return nil, err
	}

	return worktree, nil
}

func (r *Repo) tags(repo *git.Repository) (map[string][]string, error) {
	tags := map[string][]string{}

	// Get lightweight tags.
	{
		tagsIter, err := repo.Tags()
		if err != nil {
			return nil, err
		}
		defer tagsIter.Close()

		err = tagsIter.ForEach(func(tag *plumbing.Reference) error {
			v := tags[tag.Hash().String()]
			if v == nil {
				v = []string{}
			}
			v = append(v, tag.Name().Short())

			tags[tag.Hash().String()] = v

			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// Get tag objects.
	{
		tagObjectsIter, err := repo.TagObjects()
		if err != nil {
			return nil, err
		}
		defer tagObjectsIter.Close()

		err = tagObjectsIter.ForEach(func(tag *object.Tag) error {
			commit, err := tag.Commit()
			if err != nil {
				return err
			}

			v := tags[commit.Hash.String()]
			if v == nil {
				v = []string{}
			}
			v = append(v, tag.Name)

			tags[commit.Hash.String()] = v

			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return tags, nil
}
