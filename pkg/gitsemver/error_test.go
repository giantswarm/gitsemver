package gitsemver

import (
	"errors"
	"testing"
)

func Test_ExecutionFailedError_Error(t *testing.T) {
	t.Parallel()
	e := &ExecutionFailedError{message: "something went wrong"}
	want := "ExecutionFailedError: something went wrong"
	if got := e.Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func Test_ExecutionFailedError_Is(t *testing.T) {
	t.Parallel()
	e := &ExecutionFailedError{message: "a"}
	if !errors.Is(e, &ExecutionFailedError{}) {
		t.Error("errors.Is should match same type with empty target")
	}
	if errors.Is(e, &InvalidConfigError{}) {
		t.Error("errors.Is should not match different type")
	}
}

func Test_InvalidConfigError_Error(t *testing.T) {
	t.Parallel()
	e := &InvalidConfigError{message: "bad config"}
	want := "InvalidConfigError: bad config"
	if got := e.Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func Test_InvalidConfigError_Is(t *testing.T) {
	t.Parallel()
	e := &InvalidConfigError{message: "a"}
	if !errors.Is(e, &InvalidConfigError{}) {
		t.Error("errors.Is should match same type with empty target")
	}
	if errors.Is(e, &ExecutionFailedError{}) {
		t.Error("errors.Is should not match different type")
	}
}

func Test_FileNotFoundError_Error(t *testing.T) {
	t.Parallel()
	e := &FileNotFoundError{message: "file.txt"}
	want := "FileNotFoundError: file.txt"
	if got := e.Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func Test_FileNotFoundError_Is(t *testing.T) {
	t.Parallel()
	e := &FileNotFoundError{message: "a"}
	if !errors.Is(e, &FileNotFoundError{}) {
		t.Error("errors.Is should match same type with empty target")
	}
	if errors.Is(e, &FolderNotFoundError{}) {
		t.Error("errors.Is should not match different type")
	}
}

func Test_FolderNotFoundError_Error(t *testing.T) {
	t.Parallel()
	e := &FolderNotFoundError{message: "dir/"}
	want := "FolderNotFoundError: dir/"
	if got := e.Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func Test_FolderNotFoundError_Is(t *testing.T) {
	t.Parallel()
	e := &FolderNotFoundError{message: "a"}
	if !errors.Is(e, &FolderNotFoundError{}) {
		t.Error("errors.Is should match same type with empty target")
	}
	if errors.Is(e, &FileNotFoundError{}) {
		t.Error("errors.Is should not match different type")
	}
}

func Test_ReferenceNotFoundError_Error(t *testing.T) {
	t.Parallel()
	e := &ReferenceNotFoundError{message: "refs/heads/missing"}
	want := "ReferenceNotFoundError: refs/heads/missing"
	if got := e.Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func Test_ReferenceNotFoundError_Is(t *testing.T) {
	t.Parallel()
	e := &ReferenceNotFoundError{message: "a"}
	if !errors.Is(e, &ReferenceNotFoundError{}) {
		t.Error("errors.Is should match same type with empty target")
	}
	if errors.Is(e, &RepositoryNotFoundError{}) {
		t.Error("errors.Is should not match different type")
	}
}

func Test_RepositoryNotFoundError_Error(t *testing.T) {
	t.Parallel()
	e := &RepositoryNotFoundError{message: "/no/such/path"}
	want := "RepositoryNotFoundError: /no/such/path"
	if got := e.Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func Test_RepositoryNotFoundError_Is(t *testing.T) {
	t.Parallel()
	e := &RepositoryNotFoundError{message: "a"}
	if !errors.Is(e, &RepositoryNotFoundError{}) {
		t.Error("errors.Is should match same type with empty target")
	}
	if errors.Is(e, &ReferenceNotFoundError{}) {
		t.Error("errors.Is should not match different type")
	}
}
