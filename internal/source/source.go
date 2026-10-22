// Package source fetches spec files at a commit (CLI-6).
package source

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MajidBabaeifar/spechub/internal/index"
)

// Fetcher returns one file of a repo at a commit SHA.
type Fetcher interface {
	File(repo, sha, path string) ([]byte, error)
}

// RawBase is overridable in tests.
var RawBase = "https://raw.githubusercontent.com"

// GitHub fetches from raw.githubusercontent.com: no API calls, no token.
type GitHub struct{}

func (GitHub) File(repo, sha, path string) ([]byte, error) {
	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("refusing path %q", path)
	}
	return index.Get(fmt.Sprintf("%s/%s/%s/%s", RawBase, repo, sha, path))
}

// Dir serves files from local directories keyed by repo (tests, offline use).
type Dir map[string]string

func (d Dir) File(repo, sha, path string) ([]byte, error) {
	root, ok := d[repo]
	if !ok {
		return nil, fmt.Errorf("no local dir for %s", repo)
	}
	return os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
}
