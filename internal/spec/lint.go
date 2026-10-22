package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/MajidBabaeifar/spechub/internal/semver"
)

// Issue is one lint finding. Warnings don't fail lint (CLI-4).
type Issue struct {
	File    string
	Line    int
	Rule    string
	Message string
	Warning bool
}

func (i Issue) String() string {
	kind := "error"
	if i.Warning {
		kind = "warning"
	}
	loc := i.File
	if i.Line > 0 {
		loc = fmt.Sprintf("%s:%d", i.File, i.Line)
	}
	return fmt.Sprintf("%s: %s %s: %s", loc, kind, i.Rule, i.Message)
}

var (
	licenseRe = regexp.MustCompile(`^[A-Za-z0-9.+-]+( (AND|OR|WITH) [A-Za-z0-9.+-]+)*$`)
	linkRe    = regexp.MustCompile(`\]\(([^)\s#]+)(#[^)]*)?\)`)
	idLinkRe  = regexp.MustCompile(`\[([A-Z][A-Z0-9]*-[0-9]+)\]\(([^)\s#]+\.md)\)`)
)

// Repo is a spec repo loaded from disk or from fetched files.
type Repo struct {
	Readme []byte
	Areas  []Area
}

// LoadDir reads README.md and specs/*.md from dir.
func LoadDir(dir string) (Repo, error) {
	var r Repo
	b, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		return r, err
	}
	r.Readme = b
	files, _ := filepath.Glob(filepath.Join(dir, "specs", "*.md"))
	sort.Strings(files)
	for _, f := range files {
		base := filepath.Base(f)
		if strings.HasPrefix(base, "_") || base == "README.md" {
			continue
		}
		c, err := os.ReadFile(f)
		if err != nil {
			return r, err
		}
		r.Areas = append(r.Areas, ParseArea("specs/"+base, c))
	}
	return r, nil
}

// LintDir lints a spec repo on disk.
func LintDir(dir string) ([]Issue, error) {
	r, err := LoadDir(dir)
	if err != nil {
		return nil, err
	}
	return Lint(r), nil
}

// Lint checks FMT-1..5 and FMT-9.
func Lint(r Repo) []Issue {
	var out []Issue
	add := func(file string, line int, rule, msg string, warn bool) {
		out = append(out, Issue{File: file, Line: line, Rule: rule, Message: msg, Warning: warn})
	}
	fm, err := ParseFrontMatter(r.Readme)
	if err != nil {
		add("README.md", 1, "FMT-1", err.Error(), false)
	} else {
		lintFrontMatter(fm, func(rule, msg string, warn bool) { add("README.md", 0, rule, msg, warn) })
	}

	if len(r.Areas) == 0 {
		add("specs/", 0, "FMT-3", "no specs/*.md areas", false)
	}
	prefixes := map[string]string{}
	ids := map[string]string{}
	for _, a := range r.Areas {
		if a.Status == "" {
			add(a.File, 0, "FMT-3", "missing **Status:**", false)
		}
		if a.Prefix == "" {
			add(a.File, 0, "FMT-3", "missing **Prefix:**", false)
		} else if other, dup := prefixes[a.Prefix]; dup {
			add(a.File, 0, "FMT-4", fmt.Sprintf("prefix %s also used in %s", a.Prefix, other), false)
		} else {
			prefixes[a.Prefix] = a.File
		}
		if !a.Header {
			add(a.File, 0, "FMT-3", "missing table header `| ID | Requirement | Verified by |`", false)
		}
		for _, row := range a.Rows {
			if a.Prefix != "" && !strings.HasPrefix(row.ID, a.Prefix+"-") {
				add(a.File, row.Line, "FMT-4", fmt.Sprintf("%s doesn't use the area prefix %s", row.ID, a.Prefix), false)
			}
			if other, dup := ids[row.ID]; dup {
				add(a.File, row.Line, "FMT-4", fmt.Sprintf("%s also defined in %s", row.ID, other), false)
			}
			ids[row.ID] = a.File
			if strings.TrimSpace(row.Requirement) == "" {
				add(a.File, row.Line, "FMT-3", row.ID+" has no requirement text", false)
			}
			v := strings.TrimSpace(row.VerifiedBy)
			if !row.Removed() && (v == "" || v == "—" || v == "-") {
				add(a.File, row.Line, "FMT-5", row.ID+" has an empty Verified by", false)
			}
		}
	}

	// FMT-9 and in-repo ID links.
	for _, a := range r.Areas {
		for i, line := range strings.Split(a.Raw, "\n") {
			for _, m := range linkRe.FindAllStringSubmatch(line, -1) {
				target := m[1]
				if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
					continue
				}
				resolved := filepath.ToSlash(filepath.Clean(filepath.Join(filepath.Dir(a.File), target)))
				if strings.HasPrefix(resolved, "../") || resolved == ".." {
					add(a.File, i+1, "FMT-9", fmt.Sprintf("link %s leaves the repo; reference other specs by global ID", target), false)
				}
			}
			for _, m := range idLinkRe.FindAllStringSubmatch(line, -1) {
				id, target := m[1], m[2]
				want := filepath.ToSlash(filepath.Clean(filepath.Join(filepath.Dir(a.File), target)))
				if got, ok := ids[id]; ok && got != want {
					add(a.File, i+1, "FMT-9", fmt.Sprintf("%s links to %s but is defined in %s", id, target, got), false)
				} else if !ok && !strings.HasPrefix(want, "../") {
					add(a.File, i+1, "FMT-9", fmt.Sprintf("%s is not defined in this repo", id), false)
				}
			}
		}
	}
	return out
}

func lintFrontMatter(fm FrontMatter, add func(rule, msg string, warn bool)) {
	if fm.Spechub != 1 {
		add("FMT-1", "front matter needs `spechub: 1`", false)
	}
	switch {
	case fm.Name == "":
		add("FMT-2", "`name` is required", false)
	case !NameRe.MatchString(fm.Name) || len(fm.Name) > 64:
		add("FMT-2", fmt.Sprintf("name %q must be @scope/name, [a-z0-9-], ≤ 64 chars", fm.Name), false)
	}
	switch n := len([]rune(fm.Summary)); {
	case n == 0:
		add("FMT-2", "`summary` is required", false)
	case n > 200:
		add("FMT-2", fmt.Sprintf("summary is %d chars, max 200", n), false)
	}
	if fm.License == "" {
		add("FMT-2", "`license` is required (SPDX id)", false)
	} else if !licenseRe.MatchString(fm.License) {
		add("FMT-2", fmt.Sprintf("license %q is not an SPDX expression", fm.License), false)
	}
	for name, rng := range fm.Requires {
		if !NameRe.MatchString(name) {
			add("FMT-2", fmt.Sprintf("requires: bad name %q", name), false)
		}
		if _, err := semver.ParseRange(rng); err != nil {
			add("FMT-2", "requires "+name+": "+err.Error(), false)
		}
	}
	if fm.Base != nil {
		if fm.Base.Relation != "fork" && fm.Base.Relation != "extends" {
			add("FMT-2", "base.relation must be fork or extends", false)
		}
		name, ver := SplitNameVersion(fm.Base.Spec)
		if !NameRe.MatchString(name) || ver == "" {
			add("FMT-2", "base.spec must be @scope/name@version", false)
		} else if _, err := semver.Parse(ver); err != nil {
			add("FMT-2", "base.spec: "+err.Error(), false)
		}
	}
	if fm.DerivedFrom != nil && fm.DerivedFrom.Repo == "" {
		add("FMT-2", "derived_from.repo is required", false)
	}
	for id, v := range fm.Overrides {
		if !GlobalIDRe.MatchString(id) {
			add("FMT-2", fmt.Sprintf("overrides: %q is not a global ID (@scope/name:ID-n)", id), false)
		}
		if strings.TrimSpace(v) == "" {
			add("FMT-2", "overrides "+id+": empty value", false)
		}
	}
	for _, k := range fm.Unknown {
		add("FMT-2", fmt.Sprintf("unknown front matter field %q", k), true)
	}
}
