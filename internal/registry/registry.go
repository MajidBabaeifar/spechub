// Package registry reads registry/names.json and checks claim PRs (REG).
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/specfork/specfork/internal/ghapi"
	"github.com/specfork/specfork/internal/spec"
)

// Entry binds one name to a repo (REG-1, REG-4).
type Entry struct {
	RepoID    int64  `json:"repo_id"`
	OwnerID   int64  `json:"owner_id"`
	Repo      string `json:"repo"`
	ClaimedAt string `json:"claimed_at"`
	State     string `json:"state"`
}

// Names is names.json.
type Names struct {
	Names map[string]Entry `json:"names"`
}

// ReservedScope is claimable only by maintainers (REG-3).
const ReservedScope = "@specfork"

func Parse(b []byte) (*Names, error) {
	var n Names
	if err := json.Unmarshal(b, &n); err != nil {
		return nil, fmt.Errorf("names.json: %w", err)
	}
	if n.Names == nil {
		n.Names = map[string]Entry{}
	}
	return &n, nil
}

func Load(path string) (*Names, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(b)
}

// Validate checks the file's own shape.
func (n *Names) Validate() []string {
	var out []string
	for _, name := range n.Sorted() {
		e := n.Names[name]
		if !spec.NameRe.MatchString(name) || len(name) > 64 {
			out = append(out, name+": bad name")
		}
		if e.RepoID <= 0 || e.OwnerID <= 0 || e.Repo == "" || e.ClaimedAt == "" {
			out = append(out, name+": repo_id, owner_id, repo and claimed_at are required")
		}
		if e.State != "active" && e.State != "frozen" {
			out = append(out, name+": state must be active or frozen")
		}
	}
	return out
}

// Sorted returns the names in order.
func (n *Names) Sorted() []string {
	out := make([]string, 0, len(n.Names))
	for k := range n.Names {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ByRepoID finds the name bound to a repo.
func (n *Names) ByRepoID(id int64) (string, Entry, bool) {
	for name, e := range n.Names {
		if e.RepoID == id {
			return name, e, true
		}
	}
	return "", Entry{}, false
}

// Scope returns "@x" of "@x/y".
func Scope(name string) string {
	s, _, _ := strings.Cut(name, "/")
	return s
}

// GitHub is what the claim check needs from the API.
type GitHub interface {
	RepoByID(id int64) (ghapi.Repo, error)
	BranchSHA(full, branch string) (string, error)
	Raw(full, sha, path string) ([]byte, error)
	Permission(full, user string) (string, error)
}

// Verdict is the outcome of a claim check.
type Verdict struct {
	Added    []string
	Problems []string
	Manual   bool // needs a maintainer: reserved scope or a change to existing entries
}

func (v Verdict) OK() bool { return len(v.Problems) == 0 && !v.Manual && len(v.Added) > 0 }

// CheckClaim compares the PR's names.json with the base one (REG-2, REG-3, REG-7).
func CheckClaim(base, head *Names, author string, gh GitHub) Verdict {
	var v Verdict
	for _, name := range base.Sorted() {
		he, ok := head.Names[name]
		if !ok {
			v.Problems = append(v.Problems, name+": a claim PR may not remove names")
			v.Manual = true
		} else if he != base.Names[name] {
			v.Problems = append(v.Problems, name+": a claim PR may not change existing names (REG-7)")
			v.Manual = true
		}
	}
	v.Problems = append(v.Problems, head.Validate()...)
	for _, name := range head.Sorted() {
		if _, ok := base.Names[name]; ok {
			continue
		}
		v.Added = append(v.Added, name)
		e := head.Names[name]
		bad := func(format string, args ...any) {
			v.Problems = append(v.Problems, name+": "+fmt.Sprintf(format, args...))
		}
		if e.State != "active" {
			bad("new claims start `active`")
		}
		if Scope(name) == ReservedScope {
			v.Manual = true
			bad("the %s scope is reserved for seeds; a maintainer merges these", ReservedScope)
		}
		if other, _, taken := base.ByRepoID(e.RepoID); taken {
			bad("repo %d is already claimed as %s", e.RepoID, other)
		}
		repo, err := gh.RepoByID(e.RepoID)
		if err != nil {
			bad("repo_id %d: %v", e.RepoID, err)
			continue
		}
		if !strings.EqualFold(repo.FullName, e.Repo) {
			bad("repo_id %d is %s, not %s", e.RepoID, repo.FullName, e.Repo)
		}
		if repo.Owner.ID != e.OwnerID {
			bad("owner_id is %d, not %d", repo.Owner.ID, e.OwnerID)
		}
		if Scope(name) != ReservedScope && Scope(name) != "@"+strings.ToLower(repo.Owner.Login) {
			bad("scope must be @%s, the repo owner's login", strings.ToLower(repo.Owner.Login))
		}
		if !repo.HasTopic("specfork") {
			bad("%s lacks the `specfork` topic", repo.FullName)
		}
		if sha, err := gh.BranchSHA(repo.FullName, repo.DefaultBranch); err != nil {
			bad("can't read %s: %v", repo.FullName, err)
		} else if readme, err := gh.Raw(repo.FullName, sha, "README.md"); err != nil {
			bad("can't read README.md: %v", err)
		} else if fm, err := spec.ParseFrontMatter(readme); err != nil {
			bad("README.md: %v", err)
		} else if fm.Name != name {
			bad("README front matter name is %q", fm.Name)
		}
		switch repo.Owner.Type {
		case "User":
			if !strings.EqualFold(author, repo.Owner.Login) {
				bad("only %s can claim names for this repo", repo.Owner.Login)
			}
		default:
			if perm, err := gh.Permission(repo.FullName, author); err != nil || perm != "admin" {
				bad("%s needs admin on %s (got %q, %v)", author, repo.FullName, perm, err)
			}
		}
	}
	if len(v.Added) == 0 && len(v.Problems) == 0 {
		v.Problems = append(v.Problems, "no new names")
	}
	return v
}
