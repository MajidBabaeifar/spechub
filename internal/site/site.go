// Package site renders index.json into a static site for GitHub Pages (IDX-9).
package site

import (
	"embed"
	"encoding/json"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/MajidBabaeifar/spechub/internal/index"
	"github.com/MajidBabaeifar/spechub/internal/resolve"
)

//go:embed templates/*.html templates/style.css
var files embed.FS

var tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"short": func(s string) string {
		if len(s) > 7 {
			return s[:7]
		}
		return s
	},
	"page":  func(name string) string { return "s/" + resolve.Dir(name) + "/" },
	"lower": strings.ToLower,
	"join":  strings.Join,
	"base":  filepath.Base,
	"dict": func(kv ...any) map[string]any {
		m := map[string]any{}
		for i := 0; i+1 < len(kv); i += 2 {
			m[kv[i].(string)] = kv[i+1]
		}
		return m
	},
}).ParseFS(files, "templates/*.html"))

// Repo is where SpecHub's own code lives.
const Repo = "MajidBabaeifar/spechub"

type specPage struct {
	Root       string
	Spec       index.Spec
	Snap       index.Snapshot
	SnapSHA    string
	RequiredBy []string
	Known      map[string]bool
	Repo       string
}

type indexPage struct {
	Root  string
	Index *index.Index
	Repo  string
}

// Generate writes index.json, index.html, one page per spec, and the stylesheet into dir.
func Generate(ix *index.Index, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(ix, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "index.json"), append(b, '\n'), 0o644); err != nil {
		return err
	}
	css, _ := files.ReadFile("templates/style.css")
	if err := os.WriteFile(filepath.Join(dir, "style.css"), css, 0o644); err != nil {
		return err
	}
	os.WriteFile(filepath.Join(dir, ".nojekyll"), nil, 0o644)
	if err := render(filepath.Join(dir, "index.html"), "index.html", indexPage{Root: "", Index: ix, Repo: Repo}); err != nil {
		return err
	}
	known := map[string]bool{}
	for _, s := range ix.Specs {
		known[s.Name] = true
	}
	for _, s := range ix.Specs {
		snap, sha := s.Head, s.HeadSHA
		if len(s.Versions) > 0 {
			snap, sha = s.Versions[0].Snapshot, s.Versions[0].SHA
		}
		p := specPage{Root: "../../../", Spec: s, Snap: snap, SnapSHA: sha, RequiredBy: ix.RequiredBy(s.Name), Known: known, Repo: Repo}
		out := filepath.Join(dir, "s", filepath.FromSlash(resolve.Dir(s.Name)), "index.html")
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		if err := render(out, "spec.html", p); err != nil {
			return err
		}
	}
	return nil
}

func render(path, name string, data any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return tmpl.ExecuteTemplate(f, name, data)
}
