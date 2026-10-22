// Package resolve turns a spec and everything it requires into one bundle (CMP).
package resolve

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/MajidBabaeifar/spechub/internal/index"
	"github.com/MajidBabaeifar/spechub/internal/semver"
)

// Node is one selected spec version.
type Node struct {
	Name     string
	Version  semver.Version
	SHA      string
	Repo     string
	Spec     *index.Spec
	Snapshot index.Snapshot
}

// Selection is the result of version selection, dependencies first, root last.
type Selection struct {
	Root  string
	Order []*Node
}

// Node returns the selected node for name.
func (s *Selection) Node(name string) *Node {
	for _, n := range s.Order {
		if n.Name == name {
			return n
		}
	}
	return nil
}

// Lock is spechub.lock (CMP-6).
type Lock struct {
	Root  string               `json:"root"`
	Specs map[string]LockEntry `json:"specs"`
}

// LockEntry pins one spec.
type LockEntry struct {
	Version string `json:"version"`
	SHA     string `json:"sha"`
	Repo    string `json:"repo"`
}

// ReadLock reads a lock file; a missing file is not an error.
func ReadLock(path string) (*Lock, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var l Lock
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &l, nil
}

// Lock builds the lock for a selection.
func (s *Selection) Lock() *Lock {
	l := &Lock{Root: s.Root, Specs: map[string]LockEntry{}}
	for _, n := range s.Order {
		l.Specs[n.Name] = LockEntry{Version: n.Version.String(), SHA: n.SHA, Repo: n.Repo}
	}
	return l
}

// WriteLock writes the lock as indented JSON.
func WriteLock(path string, l *Lock) error {
	b, _ := json.MarshalIndent(l, "", "  ")
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

type key struct {
	name string
	ver  string
}

// Select runs minimal version selection from root (CMP-2, CMP-3). A lock for the
// same root pins every version and SHA (CMP-6).
func Select(ix *index.Index, root, rootRange string, lock *Lock) (*Selection, error) {
	if lock != nil && lock.Root == root {
		return fromLock(ix, lock)
	}
	if rootRange == "" {
		rootRange = "latest"
	}
	rootSpec, err := lookup(ix, root)
	if err != nil {
		return nil, err
	}
	var rootVer semver.Version
	if rootRange == "latest" {
		rootVer, _ = semver.Parse(rootSpec.Latest())
	} else {
		r, err := semver.ParseRange(rootRange)
		if err != nil {
			return nil, err
		}
		v, ok := highest(r, rootSpec.Available())
		if !ok {
			return nil, fmt.Errorf("%s has no version matching %s", root, rootRange)
		}
		rootVer = v
	}

	picks := map[string]semver.Version{root: rootVer}
	type edge struct{ from, to, rng string }
	var edges []edge
	seen := map[key]bool{}
	queue := []key{{root, rootVer.String()}}
	for len(queue) > 0 {
		k := queue[0]
		queue = queue[1:]
		if seen[k] {
			continue
		}
		seen[k] = true
		s, err := lookup(ix, k.name)
		if err != nil {
			return nil, err
		}
		v, _ := semver.Parse(k.ver)
		snap, _, ok := s.At(v)
		if !ok {
			return nil, fmt.Errorf("%s@%s is not in the index", k.name, k.ver)
		}
		for _, dep := range sortedKeys(snap.Requires) {
			rng := snap.Requires[dep]
			edges = append(edges, edge{k.name, dep, rng})
			r, err := semver.ParseRange(rng)
			if err != nil {
				return nil, fmt.Errorf("%s requires %s %q: %w", k.name, dep, rng, err)
			}
			ds, err := lookup(ix, dep)
			if err != nil {
				return nil, fmt.Errorf("%s requires %s: %w", k.name, dep, err)
			}
			low, ok := r.Lowest(ds.Available())
			if !ok {
				return nil, fmt.Errorf("unsatisfiable: %s requires %s %s; available: %v", k.name, dep, rng, ds.Available())
			}
			if cur, ok := picks[dep]; !ok || semver.Compare(low, cur) > 0 {
				picks[dep] = low
			}
			queue = append(queue, key{dep, low.String()})
		}
	}

	// Every range declared by a selected version must allow the selection.
	selected := map[string]bool{}
	for name, v := range picks {
		selected[name+"@"+v.String()] = true
	}
	graph := map[string][]string{}
	for _, e := range edges {
		if !selected[e.from+"@"+picks[e.from].String()] {
			continue
		}
		r, _ := semver.ParseRange(e.rng)
		if !r.Allows(picks[e.to]) {
			return nil, fmt.Errorf("unsatisfiable: %s requires %s %s, but %s is selected", e.from, e.to, e.rng, picks[e.to])
		}
		graph[e.from] = append(graph[e.from], e.to)
	}

	order, err := topo(root, graph)
	if err != nil {
		return nil, err
	}
	sel := &Selection{Root: root}
	for _, name := range order {
		s, _ := lookup(ix, name)
		snap, sha, _ := s.At(picks[name])
		sel.Order = append(sel.Order, &Node{Name: name, Version: picks[name], SHA: sha, Repo: s.Repo, Spec: s, Snapshot: snap})
	}
	return sel, nil
}

func fromLock(ix *index.Index, lock *Lock) (*Selection, error) {
	graph := map[string][]string{}
	nodes := map[string]*Node{}
	for name, e := range lock.Specs {
		s, err := lookup(ix, name)
		if err != nil {
			return nil, err
		}
		v, err := semver.Parse(e.Version)
		if err != nil {
			return nil, fmt.Errorf("spechub.lock %s: %w", name, err)
		}
		snap, _, ok := s.At(v)
		if !ok {
			return nil, fmt.Errorf("spechub.lock pins %s@%s, which the index no longer has; use --update", name, e.Version)
		}
		nodes[name] = &Node{Name: name, Version: v, SHA: e.SHA, Repo: s.Repo, Spec: s, Snapshot: snap}
		for dep := range snap.Requires {
			if _, ok := lock.Specs[dep]; !ok {
				return nil, fmt.Errorf("spechub.lock misses %s (required by %s); use --update", dep, name)
			}
			graph[name] = append(graph[name], dep)
		}
	}
	if _, ok := nodes[lock.Root]; !ok {
		return nil, fmt.Errorf("spechub.lock doesn't pin its root %s", lock.Root)
	}
	order, err := topo(lock.Root, graph)
	if err != nil {
		return nil, err
	}
	sel := &Selection{Root: lock.Root}
	for _, name := range order {
		sel.Order = append(sel.Order, nodes[name])
	}
	return sel, nil
}

func lookup(ix *index.Index, name string) (*index.Spec, error) {
	s, ok := ix.Find(name)
	if !ok {
		return nil, fmt.Errorf("%s: not found in the index", name)
	}
	if !s.Resolvable() {
		return nil, fmt.Errorf("%s is %s and can't be resolved", name, s.State)
	}
	return s, nil
}

func highest(r semver.Range, avail []semver.Version) (semver.Version, bool) {
	var best semver.Version
	found := false
	for _, v := range avail {
		if v.Head && !r.IsHead() {
			continue
		}
		if r.Allows(v) && (!found || semver.Compare(v, best) > 0) {
			best, found = v, true
		}
	}
	return best, found
}

// topo orders names reachable from root, dependencies first; a cycle is an error (CMP-3).
func topo(root string, graph map[string][]string) ([]string, error) {
	const (
		white = iota
		grey
		black
	)
	color := map[string]int{}
	var order, stack []string
	var visit func(string) error
	visit = func(n string) error {
		switch color[n] {
		case grey:
			i := len(stack) - 1
			for i >= 0 && stack[i] != n {
				i--
			}
			return fmt.Errorf("cycle: %s → %s", strings.Join(stack[i:], " → "), n)
		case black:
			return nil
		}
		color[n] = grey
		stack = append(stack, n)
		deps := append([]string(nil), graph[n]...)
		sort.Strings(deps)
		for _, d := range deps {
			if err := visit(d); err != nil {
				return err
			}
		}
		stack = stack[:len(stack)-1]
		color[n] = black
		order = append(order, n)
		return nil
	}
	if err := visit(root); err != nil {
		return nil, err
	}
	return order, nil
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
