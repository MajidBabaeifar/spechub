// Package crawl builds index.json from GitHub (IDX-1..8).
package crawl

import (
	"errors"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/specfork/specfork/internal/ghapi"
	"github.com/specfork/specfork/internal/index"
	"github.com/specfork/specfork/internal/registry"
	"github.com/specfork/specfork/internal/semver"
	"github.com/specfork/specfork/internal/spec"
)

// GitHub is what the crawler reads.
type GitHub interface {
	SearchRepos(q string) ([]ghapi.Repo, error)
	RepoByID(id int64) (ghapi.Repo, error)
	BranchSHA(full, branch string) (string, error)
	Tags(full string) ([]ghapi.Tag, error)
	TreeFiles(full, sha string) ([]string, error)
	Raw(full, sha, path string) ([]byte, error)
}

const (
	SpecQuery = "topic:specfork fork:true"
	ImplQuery = "topic:specfork-impl fork:true"
)

// Stats reports what a crawl did.
type Stats struct {
	Read, Reused, Skipped int
	RateLimited           bool
	Errors                []string
}

type crawler struct {
	gh    GitHub
	names *registry.Names
	prev  map[int64]index.Spec
	stats Stats
}

func (c *crawler) stopped(err error) bool {
	if errors.Is(err, ghapi.ErrRateLimited) {
		c.stats.RateLimited = true
		return true
	}
	return c.stats.RateLimited
}

func (c *crawler) note(repo string, err error) {
	c.stats.Errors = append(c.stats.Errors, repo+": "+err.Error())
}

// Run crawls GitHub. prev may be nil.
func Run(gh GitHub, names *registry.Names, prev *index.Index, now time.Time) (*index.Index, Stats) {
	c := &crawler{gh: gh, names: names, prev: map[int64]index.Spec{}}
	if prev != nil {
		for _, s := range prev.Specs {
			c.prev[s.RepoID] = s
		}
	}
	out := &index.Index{GeneratedAt: now.UTC().Format(time.RFC3339), Specs: []index.Spec{}, Unclaimed: []string{}}

	repos := map[int64]ghapi.Repo{}
	gone := map[string]registry.Entry{}
	for _, name := range names.Sorted() {
		e := names.Names[name]
		r, err := gh.RepoByID(e.RepoID)
		switch {
		case err == nil:
			repos[r.ID] = r
		case errors.Is(err, ghapi.ErrNotFound):
			gone[name] = e
		case c.stopped(err):
		default:
			c.note(e.Repo, err)
		}
	}
	if !c.stats.RateLimited {
		found, err := gh.SearchRepos(SpecQuery)
		if err != nil && !c.stopped(err) {
			c.note("search", err)
		}
		for _, r := range found {
			if _, ok := repos[r.ID]; !ok {
				repos[r.ID] = r
			}
		}
	}
	// Registered repos the search and lookups couldn't reach keep their old entry.
	if c.stats.RateLimited && prev != nil {
		for _, s := range prev.Specs {
			if _, ok := repos[s.RepoID]; !ok {
				if _, isGone := gone[s.Name]; !isGone {
					out.Specs = append(out.Specs, s)
				}
			}
		}
	}

	ids := make([]int64, 0, len(repos))
	for id := range repos {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return repos[ids[i]].FullName < repos[ids[j]].FullName })
	seen := map[string]bool{}
	for _, id := range ids {
		r := repos[id]
		s, ok, claimed := c.spec(r)
		if !ok {
			continue
		}
		if !claimed {
			out.Unclaimed = append(out.Unclaimed, r.FullName)
			continue
		}
		if seen[s.Name] {
			continue
		}
		seen[s.Name] = true
		out.Specs = append(out.Specs, s)
	}
	for name, e := range gone {
		s := index.Spec{Name: name, Repo: e.Repo, RepoID: e.RepoID, OwnerID: e.OwnerID, Versions: []index.Version{}, Implementations: []index.Impl{}}
		if p, ok := c.prev[e.RepoID]; ok {
			s = p
		}
		s.State = "frozen"
		out.Specs = append(out.Specs, s)
	}

	c.attachImpls(out, prev)
	sort.Slice(out.Specs, func(i, j int) bool { return out.Specs[i].Name < out.Specs[j].Name })
	sort.Strings(out.Unclaimed)
	return out, c.stats
}

// spec reads one repo. ok=false: not a spec repo or unreadable; claimed=false: unclaimed.
func (c *crawler) spec(r ghapi.Repo) (index.Spec, bool, bool) {
	prev, hadPrev := c.prev[r.ID]
	if c.stats.RateLimited {
		if hadPrev {
			c.stats.Reused++
			return c.refresh(prev, r), true, true
		}
		c.stats.Skipped++
		return index.Spec{}, false, false
	}
	head, err := c.gh.BranchSHA(r.FullName, r.DefaultBranch)
	if err != nil {
		if c.stopped(err) && hadPrev {
			c.stats.Reused++
			return c.refresh(prev, r), true, true
		}
		c.note(r.FullName, err)
		return index.Spec{}, false, false
	}
	tags, err := c.gh.Tags(r.FullName)
	if err != nil {
		if c.stopped(err) && hadPrev {
			c.stats.Reused++
			return c.refresh(prev, r), true, true
		}
		c.note(r.FullName, err)
		return index.Spec{}, false, false
	}
	var releases []ghapi.Tag
	for _, t := range tags {
		if strings.HasPrefix(t.Name, "v") {
			if _, err := semver.Parse(t.Name); err == nil {
				releases = append(releases, t)
			}
		}
	}
	if hadPrev && prev.HeadSHA == head && sameVersions(prev.Versions, releases) {
		c.stats.Reused++
		return c.refresh(prev, r), true, true
	}

	fm, snap, err := c.snapshot(r.FullName, head)
	if err != nil {
		if !errors.Is(err, spec.ErrNoFrontMatter) && !errors.Is(err, ghapi.ErrNotFound) {
			c.note(r.FullName, err)
		}
		return index.Spec{}, false, false
	}
	if fm.Specfork != 1 || fm.Name == "" {
		return index.Spec{}, false, false
	}
	c.stats.Read++
	s := index.Spec{
		Name: fm.Name, Repo: r.FullName, RepoID: r.ID, OwnerID: r.Owner.ID,
		Summary: fm.Summary, License: fm.License, Tags: fm.Tags, Stars: r.Stars, PushedAt: r.PushedAt,
		DerivedFrom: fm.DerivedFrom, Base: fm.Base, HeadSHA: head, Head: snap,
		Versions: []index.Version{}, Implementations: []index.Impl{},
	}
	for _, t := range releases {
		vfm, vs, err := c.snapshot(r.FullName, t.SHA)
		if err != nil {
			c.note(r.FullName+"@"+t.Name, err)
			continue
		}
		if vfm.Specfork != 1 || vfm.Name != fm.Name {
			continue // a tag from before a rename, or not a spec (IDX-3)
		}
		v, _ := semver.Parse(t.Name)
		s.Versions = append(s.Versions, index.Version{Version: v.String(), SHA: t.SHA, Snapshot: vs})
	}
	sortVersions(s.Versions)
	return c.claim(s, r)
}

// claim applies the registry: IDX-4, REG-5, REG-6.
func (c *crawler) claim(s index.Spec, r ghapi.Repo) (index.Spec, bool, bool) {
	name, e, ok := c.names.ByRepoID(r.ID)
	if !ok {
		return s, true, false
	}
	declared := s.Name
	s.Name = name
	s.Repo = r.FullName
	s.OwnerID = r.Owner.ID
	s.State = e.State
	if e.OwnerID != r.Owner.ID || declared != name {
		s.State = "frozen"
	}
	return s, true, true
}

func (c *crawler) refresh(prev index.Spec, r ghapi.Repo) index.Spec {
	prev.Stars = r.Stars
	prev.PushedAt = r.PushedAt
	s, _, _ := c.claim(prev, r)
	return s
}

func (c *crawler) snapshot(full, sha string) (spec.FrontMatter, index.Snapshot, error) {
	readme, err := c.gh.Raw(full, sha, "README.md")
	if err != nil {
		return spec.FrontMatter{}, index.Snapshot{}, err
	}
	fm, err := spec.ParseFrontMatter(readme)
	if err != nil {
		return fm, index.Snapshot{}, err
	}
	paths, err := c.gh.TreeFiles(full, sha)
	if err != nil {
		return fm, index.Snapshot{}, err
	}
	snap := index.Snapshot{Requires: fm.Requires, Overrides: fm.Overrides, Files: []string{}}
	for _, p := range paths {
		dir, file := path.Split(p)
		if dir == "specs/" && strings.HasSuffix(file, ".md") && !strings.HasPrefix(file, "_") && file != "README.md" {
			snap.Files = append(snap.Files, p)
		}
	}
	sort.Strings(snap.Files)
	return fm, snap, nil
}

// attachImpls reads implementation repos (FMT-8, IDX-7).
func (c *crawler) attachImpls(out *index.Index, prev *index.Index) {
	byName := map[string]*index.Spec{}
	for i := range out.Specs {
		out.Specs[i].Implementations = []index.Impl{}
		byName[out.Specs[i].Name] = &out.Specs[i]
	}
	keepPrev := func() {
		if prev == nil {
			return
		}
		for _, p := range prev.Specs {
			if s, ok := byName[p.Name]; ok {
				s.Implementations = p.Implementations
			}
		}
	}
	if c.stats.RateLimited {
		keepPrev()
		return
	}
	impls, err := c.gh.SearchRepos(ImplQuery)
	if err != nil {
		if !c.stopped(err) {
			c.note("impl search", err)
		}
		keepPrev()
		return
	}
	sort.Slice(impls, func(i, j int) bool { return impls[i].FullName < impls[j].FullName })
	for _, r := range impls {
		head, err := c.gh.BranchSHA(r.FullName, r.DefaultBranch)
		if err != nil {
			if c.stopped(err) {
				keepPrev()
				return
			}
			c.note(r.FullName, err)
			continue
		}
		readme, err := c.gh.Raw(r.FullName, head, "README.md")
		if err != nil {
			continue
		}
		fm, err := spec.ParseFrontMatter(readme)
		if err != nil || fm.Implements == "" {
			continue
		}
		name, ver := spec.SplitNameVersion(fm.Implements)
		s, ok := byName[name]
		if !ok {
			continue
		}
		s.Implementations = append(s.Implementations, index.Impl{Repo: r.FullName, Stack: fm.Stack, URL: fm.URL, Version: ver})
	}
}

func sameVersions(have []index.Version, tags []ghapi.Tag) bool {
	if len(have) != len(tags) {
		return false
	}
	m := map[string]string{}
	for _, v := range have {
		m[v.Version] = v.SHA
	}
	for _, t := range tags {
		v, _ := semver.Parse(t.Name)
		if m[v.String()] != t.SHA {
			return false
		}
	}
	return true
}

func sortVersions(vs []index.Version) {
	sort.Slice(vs, func(i, j int) bool {
		a, _ := semver.Parse(vs[i].Version)
		b, _ := semver.Parse(vs[j].Version)
		return semver.Compare(a, b) > 0
	})
}
