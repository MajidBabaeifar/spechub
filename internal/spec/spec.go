// Package spec parses Specfork spec repos: README front matter (FMT-1, FMT-2, FMT-8)
// and requirement areas (FMT-3..5).
package spec

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	NameRe     = regexp.MustCompile(`^@[a-z0-9-]+/[a-z0-9-]+$`)
	GlobalIDRe = regexp.MustCompile(`^(@[a-z0-9-]+/[a-z0-9-]+):([A-Z][A-Z0-9]*-[0-9]+)$`)
	IDRe       = regexp.MustCompile(`^([A-Z][A-Z0-9]*)-([0-9]+)$`)
)

// Base names the spec this one forks or extends.
type Base struct {
	Spec     string `yaml:"spec" json:"spec"`
	Relation string `yaml:"relation" json:"relation"`
}

// DerivedFrom credits the source a spec was extracted from.
type DerivedFrom struct {
	Repo    string `yaml:"repo" json:"repo"`
	Commit  string `yaml:"commit" json:"commit,omitempty"`
	URL     string `yaml:"url" json:"url,omitempty"`
	License string `yaml:"license" json:"license,omitempty"`
}

// FrontMatter is the YAML block at the top of a spec or implementation README.
type FrontMatter struct {
	Specfork    int               `yaml:"specfork"`
	Name        string            `yaml:"name"`
	Summary     string            `yaml:"summary"`
	License     string            `yaml:"license"`
	Requires    map[string]string `yaml:"requires"`
	Base        *Base             `yaml:"base"`
	DerivedFrom *DerivedFrom      `yaml:"derived_from"`
	Overrides   map[string]string `yaml:"overrides"`
	Tags        []string          `yaml:"tags"`
	// Implementation repos (FMT-8).
	Implements string   `yaml:"implements"`
	Stack      []string `yaml:"stack"`
	URL        string   `yaml:"url"`

	Unknown []string `yaml:"-"`
}

var knownKeys = map[string]bool{
	"specfork": true, "name": true, "summary": true, "license": true, "requires": true,
	"base": true, "derived_from": true, "overrides": true, "tags": true,
	"implements": true, "stack": true, "url": true,
}

var ErrNoFrontMatter = errors.New("README has no front matter")

// SplitFrontMatter returns the YAML between a leading "---" line and the next "---" line.
func SplitFrontMatter(readme []byte) (yamlPart, body []byte, err error) {
	s := bytes.ReplaceAll(readme, []byte("\r\n"), []byte("\n"))
	if !bytes.HasPrefix(s, []byte("---\n")) {
		return nil, s, ErrNoFrontMatter
	}
	rest := s[4:]
	end := bytes.Index(rest, []byte("\n---\n"))
	if end < 0 {
		if bytes.HasSuffix(rest, []byte("\n---")) {
			return rest[:len(rest)-4], nil, nil
		}
		return nil, s, ErrNoFrontMatter
	}
	return rest[:end+1], rest[end+5:], nil
}

// ParseFrontMatter parses a README's front matter.
func ParseFrontMatter(readme []byte) (FrontMatter, error) {
	y, _, err := SplitFrontMatter(readme)
	if err != nil {
		return FrontMatter{}, err
	}
	var fm FrontMatter
	if err := yaml.Unmarshal(y, &fm); err != nil {
		return FrontMatter{}, fmt.Errorf("front matter: %w", err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(y, &raw); err != nil {
		return FrontMatter{}, fmt.Errorf("front matter: %w", err)
	}
	for k := range raw {
		if !knownKeys[k] {
			fm.Unknown = append(fm.Unknown, k)
		}
	}
	sort.Strings(fm.Unknown)
	return fm, nil
}

// Row is one requirement.
type Row struct {
	ID          string
	Requirement string
	VerifiedBy  string
	Line        int
}

// Area is one specs/*.md file.
type Area struct {
	File   string
	Title  string
	Status string
	Prefix string
	Rows   []Row
	Header bool
	Raw    string
}

var (
	statusRe = regexp.MustCompile(`\*\*Status:\*\*\s*([^·*\n]+)`)
	prefixRe = regexp.MustCompile(`\*\*Prefix:\*\*\s*([A-Z][A-Z0-9]*)`)
)

// SplitRow splits a Markdown table row on unescaped pipes, trimming cells.
func SplitRow(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") || len(line) < 2 {
		return nil
	}
	inner := line[1 : len(line)-1]
	var cells []string
	var cur strings.Builder
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		if c == '\\' && i+1 < len(inner) && inner[i+1] == '|' {
			cur.WriteString(`\|`)
			i++
			continue
		}
		if c == '|' {
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}
		cur.WriteByte(c)
	}
	return append(cells, strings.TrimSpace(cur.String()))
}

// ParseArea reads one area file.
func ParseArea(file string, content []byte) Area {
	raw := strings.ReplaceAll(string(content), "\r\n", "\n")
	a := Area{File: file, Raw: raw}
	lines := strings.Split(raw, "\n")
	for i, l := range lines {
		if a.Title == "" && strings.HasPrefix(l, "# ") {
			a.Title = strings.TrimSpace(l[2:])
		}
		if m := statusRe.FindStringSubmatch(l); m != nil && a.Status == "" {
			a.Status = strings.TrimSpace(m[1])
		}
		if m := prefixRe.FindStringSubmatch(l); m != nil && a.Prefix == "" {
			a.Prefix = m[1]
		}
		cells := SplitRow(l)
		if len(cells) == 3 && cells[0] == "ID" && cells[1] == "Requirement" && cells[2] == "Verified by" {
			a.Header = true
			continue
		}
		if len(cells) >= 1 && IDRe.MatchString(cells[0]) {
			r := Row{ID: cells[0], Line: i + 1}
			if len(cells) > 1 {
				r.Requirement = cells[1]
			}
			if len(cells) > 2 {
				r.VerifiedBy = strings.Join(cells[2:], " | ")
			}
			a.Rows = append(a.Rows, r)
		}
	}
	return a
}

// Removed reports whether a row is a dropped requirement (FMT-4).
func (r Row) Removed() bool { return strings.HasPrefix(r.Requirement, "Removed") }

// SplitNameVersion splits "@scope/name@1.2.3" (or "@scope/name@^1.2") into name and version.
func SplitNameVersion(s string) (name, version string) {
	if i := strings.LastIndex(s, "@"); i > 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}
