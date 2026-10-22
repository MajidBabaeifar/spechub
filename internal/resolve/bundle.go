package resolve

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/specfork/specfork/internal/source"
	"github.com/specfork/specfork/internal/spec"
)

// Bundle is the flattened output (CMP-7..9): file path → content, in order.
type Bundle struct {
	Paths []string
	Files map[string][]byte
}

// Dir is the bundle folder of a spec: "@scope/name" → "scope/name".
func Dir(name string) string { return strings.TrimPrefix(name, "@") }

type override struct {
	value string
	by    string
}

type loaded struct {
	node  *Node
	files map[string][]byte // repo-relative path → content
	ids   map[string]string // local ID → repo-relative file
}

var (
	globalRefRe = regexp.MustCompile("`(@[a-z0-9-]+/[a-z0-9-]+)(?::([A-Z][A-Z0-9]*-[0-9]+))?`")
	idLinkRe    = regexp.MustCompile(`\[([A-Z][A-Z0-9]*-[0-9]+)\]\(([^)\s#]+\.md)\)`)
)

// Build fetches every selected spec and writes the bundle in memory.
func Build(sel *Selection, f source.Fetcher) (*Bundle, error) {
	specs := map[string]*loaded{}
	for _, n := range sel.Order {
		l := &loaded{node: n, files: map[string][]byte{}, ids: map[string]string{}}
		for _, p := range append([]string{"README.md"}, n.Snapshot.Files...) {
			b, err := f.File(n.Repo, n.SHA, p)
			if err != nil {
				return nil, fmt.Errorf("%s@%s %s: %w", n.Name, n.Version, p, err)
			}
			l.files[p] = b
			if p != "README.md" {
				for _, row := range spec.ParseArea(p, b).Rows {
					l.ids[row.ID] = p
				}
			}
		}
		specs[n.Name] = l
	}

	effective, err := overrides(sel, specs)
	if err != nil {
		return nil, err
	}

	b := &Bundle{Files: map[string][]byte{}}
	b.add("BUNDLE.md", bundleMD(sel, effective))
	for _, n := range sel.Order {
		l := specs[n.Name]
		paths := append([]string{"README.md"}, n.Snapshot.Files...)
		for _, p := range paths {
			out := rewrite(string(l.files[p]), n.Name, p, specs, effective)
			b.add(path.Join(Dir(n.Name), p), []byte(out))
		}
	}
	return b, nil
}

func (b *Bundle) add(p string, content []byte) {
	b.Paths = append(b.Paths, p)
	b.Files[p] = content
}

// overrides validates CMP-4 and CMP-5 and returns global ID → effective override.
func overrides(sel *Selection, specs map[string]*loaded) (map[string]override, error) {
	reach := map[string]map[string]bool{}
	var walk func(name string, into map[string]bool)
	walk = func(name string, into map[string]bool) {
		for dep := range specs[name].node.Snapshot.Requires {
			if !into[dep] {
				into[dep] = true
				walk(dep, into)
			}
		}
	}
	for _, n := range sel.Order {
		reach[n.Name] = map[string]bool{}
		walk(n.Name, reach[n.Name])
	}

	byTarget := map[string][]override{}
	for _, n := range sel.Order {
		for _, gid := range sortedKeys(n.Snapshot.Overrides) {
			m := spec.GlobalIDRe.FindStringSubmatch(gid)
			if m == nil {
				return nil, fmt.Errorf("%s overrides %q: not a global ID", n.Name, gid)
			}
			target, id := m[1], m[2]
			if !reach[n.Name][target] {
				return nil, fmt.Errorf("%s overrides %s, but %s is not in its requires", n.Name, gid, target)
			}
			if _, ok := specs[target].ids[id]; !ok {
				return nil, fmt.Errorf("%s overrides %s, which %s@%s doesn't define", n.Name, gid, target, specs[target].node.Version)
			}
			byTarget[gid] = append(byTarget[gid], override{value: n.Snapshot.Overrides[gid], by: n.Name})
		}
	}
	out := map[string]override{}
	for gid, list := range byTarget {
		var rootO *override
		for i := range list {
			if list[i].by == sel.Root {
				rootO = &list[i]
			}
		}
		if rootO != nil {
			out[gid] = *rootO
			continue
		}
		for _, o := range list[1:] {
			if o.value != list[0].value {
				return nil, fmt.Errorf("conflicting overrides of %s by %s and %s; the root spec must override it", gid, list[0].by, o.by)
			}
		}
		out[gid] = list[0]
	}
	return out, nil
}

// rewrite applies CMP-8 and CMP-9 to one file of spec name at repo path p.
func rewrite(content, name, p string, specs map[string]*loaded, eff map[string]override) string {
	here := path.Join(Dir(name), p)
	rel := func(target string) string {
		r, err := filepath.Rel(path.Dir(here), target)
		if err != nil {
			return target
		}
		return filepath.ToSlash(r)
	}
	ids := specs[name].ids
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		cells := spec.SplitRow(line)
		if len(cells) >= 3 && spec.IDRe.MatchString(cells[0]) {
			if _, own := ids[cells[0]]; own {
				gid := name + ":" + cells[0]
				cells[0] = gid
				if o, ok := eff[gid]; ok {
					if strings.EqualFold(strings.TrimSpace(o.value), "removed") {
						cells[1] = fmt.Sprintf("Removed (by %s)", o.by)
					} else {
						cells[1] = fmt.Sprintf("%s (override by %s)", o.value, o.by)
					}
				}
				line = "| " + strings.Join(cells, " | ") + " |"
			}
		}
		line = idLinkRe.ReplaceAllStringFunc(line, func(m string) string {
			sm := idLinkRe.FindStringSubmatch(m)
			if _, own := ids[sm[1]]; !own {
				return m
			}
			return fmt.Sprintf("[%s:%s](%s)", name, sm[1], sm[2])
		})
		line = globalRefRe.ReplaceAllStringFunc(line, func(m string) string {
			sm := globalRefRe.FindStringSubmatch(m)
			target, id := sm[1], sm[2]
			t, ok := specs[target]
			if !ok {
				return m
			}
			if id == "" {
				return fmt.Sprintf("[%s](%s)", m, rel(path.Join(Dir(target), "README.md")))
			}
			f, ok := t.ids[id]
			if !ok {
				return m
			}
			return fmt.Sprintf("[%s](%s)", m, rel(path.Join(Dir(target), f)))
		})
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

func bundleMD(sel *Selection, eff map[string]override) []byte {
	var b strings.Builder
	root := sel.Order[len(sel.Order)-1]
	fmt.Fprintf(&b, "# Bundle: %s@%s\n\n", root.Name, root.Version)
	b.WriteString("Build **" + root.Name + "**. Every spec below is part of it: dependencies come first and the root last. ")
	b.WriteString("Requirement IDs are global (`@scope/name:ID`); each `Verified by` cell is an acceptance check to automate. ")
	b.WriteString("Rows marked `override by` or `Removed (by …)` were changed by a spec that requires them; follow the changed text.\n\n")
	b.WriteString("| # | Spec | Version | Commit | Repo | Licence | Derived from |\n|---|---|---|---|---|---|---|\n")
	for i, n := range sel.Order {
		derived := "–"
		if d := n.Spec.DerivedFrom; d != nil {
			derived = d.Repo
			if d.License != "" {
				derived += " (" + d.License + ")"
			}
		}
		sha := n.SHA
		if len(sha) > 12 {
			sha = sha[:12]
		}
		fmt.Fprintf(&b, "| %d | [%s](%s/README.md) | %s | `%s` | https://github.com/%s | %s | %s |\n",
			i+1, n.Name, Dir(n.Name), n.Version, sha, n.Repo, n.Spec.License, derived)
	}
	if len(eff) > 0 {
		b.WriteString("\n## Overrides applied\n\n")
		var keys []string
		for k := range eff {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "- `%s` by %s: %s\n", k, eff[k].by, eff[k].value)
		}
	}
	b.WriteString("\n## Files\n\n")
	for _, n := range sel.Order {
		fmt.Fprintf(&b, "- %s: `%s/README.md`", n.Name, Dir(n.Name))
		for _, f := range n.Snapshot.Files {
			fmt.Fprintf(&b, ", `%s/%s`", Dir(n.Name), f)
		}
		b.WriteString("\n")
	}
	return []byte(b.String())
}

// Write writes the bundle to dir, replacing an earlier bundle there.
func (b *Bundle) Write(dir string) error {
	if st, err := os.Stat(dir); err == nil && st.IsDir() {
		entries, _ := os.ReadDir(dir)
		if len(entries) > 0 {
			if _, err := os.Stat(filepath.Join(dir, "BUNDLE.md")); err != nil {
				return fmt.Errorf("%s exists and is not a bundle; choose another --out", dir)
			}
			if err := os.RemoveAll(dir); err != nil {
				return err
			}
		}
	}
	for _, p := range b.Paths {
		full := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, b.Files[p], 0o644); err != nil {
			return err
		}
	}
	return nil
}

// Markdown concatenates the bundle into one document, BUNDLE.md first (MCP-2).
func (b *Bundle) Markdown() string {
	var s strings.Builder
	for i, p := range b.Paths {
		if i > 0 {
			fmt.Fprintf(&s, "\n\n---\n\n<!-- file: %s -->\n\n", p)
		}
		s.Write(b.Files[p])
	}
	return s.String()
}
