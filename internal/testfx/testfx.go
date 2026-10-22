// Package testfx builds an index from the local seeds for tests.
package testfx

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/MajidBabaeifar/spechub/internal/index"
	"github.com/MajidBabaeifar/spechub/internal/source"
	"github.com/MajidBabaeifar/spechub/internal/spec"
)

// SeedsDir is the frozen v0.1.0 copy of the seed specs.
func SeedsDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", "seeds")
}

// Seeds indexes every seed as repo "MajidBabaeifar/spechub-<name>" at version 0.1.0.
func Seeds(t testing.TB) (*index.Index, source.Dir) {
	t.Helper()
	dirs, _ := filepath.Glob(filepath.Join(SeedsDir(), "*", "README.md"))
	if len(dirs) == 0 {
		t.Skip("no seeds in this checkout")
	}
	ix := &index.Index{GeneratedAt: "2026-10-22T00:00:00Z", Unclaimed: []string{}}
	fetch := source.Dir{}
	for _, readme := range dirs {
		dir := filepath.Dir(readme)
		b, err := os.ReadFile(readme)
		if err != nil {
			t.Fatal(err)
		}
		fm, err := spec.ParseFrontMatter(b)
		if err != nil {
			t.Fatal(err)
		}
		repo := "MajidBabaeifar/spechub-" + filepath.Base(dir)
		fetch[repo] = dir
		files, _ := filepath.Glob(filepath.Join(dir, "specs", "*.md"))
		var rel []string
		for _, f := range files {
			if !strings.HasPrefix(filepath.Base(f), "_") && filepath.Base(f) != "README.md" {
				rel = append(rel, "specs/"+filepath.Base(f))
			}
		}
		sort.Strings(rel)
		snap := index.Snapshot{Requires: fm.Requires, Overrides: fm.Overrides, Files: rel}
		ix.Specs = append(ix.Specs, index.Spec{
			Name: fm.Name, Repo: repo, RepoID: int64(len(ix.Specs) + 1), OwnerID: 1, State: "active",
			Summary: fm.Summary, License: fm.License, DerivedFrom: fm.DerivedFrom, HeadSHA: "head",
			Head: snap, Versions: []index.Version{{Version: "0.1.0", SHA: "v0.1.0-sha", Snapshot: snap}}, Implementations: []index.Impl{},
		})
	}
	sort.Slice(ix.Specs, func(i, j int) bool { return ix.Specs[i].Name < ix.Specs[j].Name })
	return ix, fetch
}
