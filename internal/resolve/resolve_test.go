package resolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MajidBabaeifar/spechub/internal/index"
	"github.com/MajidBabaeifar/spechub/internal/source"
)

type fx struct {
	ix  *index.Index
	dir source.Dir
}

// repo writes a spec repo: one area file with the given rows.
func (f *fx) repo(t *testing.T, name, prefix string, rows ...string) string {
	t.Helper()
	repo := "o/" + strings.ReplaceAll(strings.TrimPrefix(name, "@"), "/", "-")
	d := filepath.Join(t.TempDir(), repo)
	os.MkdirAll(filepath.Join(d, "specs"), 0o755)
	os.WriteFile(filepath.Join(d, "README.md"), []byte("---\nspechub: 1\nname: \""+name+"\"\n---\n# "+name+"\n"), 0o644)
	area := "# A\n\n**Status:** Draft · **Prefix:** " + prefix + "\n\n| ID | Requirement | Verified by |\n|---|---|---|\n" + strings.Join(rows, "\n") + "\n"
	os.WriteFile(filepath.Join(d, "specs", "a.md"), []byte(area), 0o644)
	f.dir[repo] = d
	return repo
}

func (f *fx) spec(name, repo string, versions ...index.Version) *index.Spec {
	s := index.Spec{Name: name, Repo: repo, State: "active", License: "CC-BY-4.0", Versions: versions}
	f.ix.Specs = append(f.ix.Specs, s)
	return &f.ix.Specs[len(f.ix.Specs)-1]
}

func ver(v string, req map[string]string, ov map[string]string) index.Version {
	return index.Version{Version: v, SHA: "sha-" + v, Snapshot: index.Snapshot{Requires: req, Overrides: ov, Files: []string{"specs/a.md"}}}
}

func newFx() *fx { return &fx{ix: &index.Index{}, dir: source.Dir{}} }

func names(sel *Selection) string {
	var out []string
	for _, n := range sel.Order {
		out = append(out, n.Name+"@"+n.Version.String())
	}
	return strings.Join(out, " ")
}

// CMP-2
func TestMVS(t *testing.T) {
	f := newFx()
	f.spec("@x/c", "o/c", ver("0.2.0", nil, nil), ver("0.1.5", nil, nil), ver("0.1.2", nil, nil), ver("0.1.0", nil, nil))
	f.spec("@x/b", "o/b", ver("0.1.0", map[string]string{"@x/c": "^0.1.2"}, nil))
	f.spec("@x/a", "o/a", ver("1.0.0", map[string]string{"@x/b": "^0.1", "@x/c": "^0.1"}, nil))
	sel, err := Select(f.ix, "@x/a", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := names(sel); got != "@x/c@0.1.2 @x/b@0.1.0 @x/a@1.0.0" {
		t.Fatalf("got %s", got)
	}
}

// CMP-3
func TestCycleError(t *testing.T) {
	f := newFx()
	f.spec("@x/a", "o/a", ver("0.1.0", map[string]string{"@x/b": "^0.1"}, nil))
	f.spec("@x/b", "o/b", ver("0.1.0", map[string]string{"@x/a": "^0.1"}, nil))
	_, err := Select(f.ix, "@x/a", "", nil)
	if err == nil || !strings.Contains(err.Error(), "cycle: @x/a → @x/b → @x/a") {
		t.Fatalf("got %v", err)
	}
	f = newFx()
	f.spec("@x/a", "o/a", ver("0.1.0", map[string]string{"@x/b": "^0.2"}, nil))
	f.spec("@x/b", "o/b", ver("0.1.0", nil, nil))
	if _, err := Select(f.ix, "@x/a", "", nil); err == nil || !strings.Contains(err.Error(), "unsatisfiable") {
		t.Fatalf("got %v", err)
	}
	// Two ranges no single version satisfies.
	f = newFx()
	f.spec("@x/c", "o/c", ver("0.2.0", nil, nil), ver("0.1.0", nil, nil))
	f.spec("@x/b", "o/b", ver("0.1.0", map[string]string{"@x/c": "^0.2"}, nil))
	f.spec("@x/a", "o/a", ver("0.1.0", map[string]string{"@x/b": "^0.1", "@x/c": "^0.1"}, nil))
	if _, err := Select(f.ix, "@x/a", "", nil); err == nil || !strings.Contains(err.Error(), "@x/a requires @x/c ^0.1, but 0.2.0 is selected") {
		t.Fatalf("got %v", err)
	}
}

func TestFrozenNotResolvable(t *testing.T) {
	f := newFx()
	s := f.spec("@x/a", "o/a", ver("0.1.0", nil, nil))
	s.State = "frozen"
	if _, err := Select(f.ix, "@x/a", "", nil); err == nil || !strings.Contains(err.Error(), "frozen") {
		t.Fatalf("got %v", err)
	}
}

func build(t *testing.T, f *fx, root string) (*Bundle, error) {
	t.Helper()
	sel, err := Select(f.ix, root, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	return Build(sel, f.dir)
}

// CMP-4
func TestOverrideTargets(t *testing.T) {
	f := newFx()
	rb := f.repo(t, "@x/b", "B", "| B-1 | b one | ok |")
	ra := f.repo(t, "@x/a", "A", "| A-1 | a one | ok |")
	rc := f.repo(t, "@x/c", "C", "| C-1 | c one | ok |")
	f.spec("@x/b", rb, ver("0.1.0", nil, nil))
	f.spec("@x/c", rc, ver("0.1.0", nil, nil))
	f.spec("@x/a", ra, ver("0.1.0", map[string]string{"@x/b": "^0.1"}, map[string]string{"@x/c:C-1": "removed"}))
	if _, err := build(t, f, "@x/a"); err == nil || !strings.Contains(err.Error(), "not in its requires") {
		t.Fatalf("got %v", err)
	}
	f.ix.Specs[2].Versions[0].Overrides = map[string]string{"@x/b:B-9": "removed"}
	if _, err := build(t, f, "@x/a"); err == nil || !strings.Contains(err.Error(), "doesn't define") {
		t.Fatalf("got %v", err)
	}
}

// CMP-5
func TestOverrideConflict(t *testing.T) {
	f := newFx()
	rd := f.repo(t, "@x/d", "D", "| D-1 | d one | ok |")
	rb := f.repo(t, "@x/b", "B", "| B-1 | b | ok |")
	rc := f.repo(t, "@x/c", "C", "| C-1 | c | ok |")
	ra := f.repo(t, "@x/a", "A", "| A-1 | a | ok |")
	f.spec("@x/d", rd, ver("0.1.0", nil, nil))
	f.spec("@x/b", rb, ver("0.1.0", map[string]string{"@x/d": "^0.1"}, map[string]string{"@x/d:D-1": "from b"}))
	f.spec("@x/c", rc, ver("0.1.0", map[string]string{"@x/d": "^0.1"}, map[string]string{"@x/d:D-1": "from c"}))
	a := f.spec("@x/a", ra, ver("0.1.0", map[string]string{"@x/b": "^0.1", "@x/c": "^0.1"}, nil))
	if _, err := build(t, f, "@x/a"); err == nil || !strings.Contains(err.Error(), "conflicting overrides of @x/d:D-1") {
		t.Fatalf("got %v", err)
	}
	a.Versions[0].Overrides = map[string]string{"@x/d:D-1": "from root"}
	b, err := build(t, f, "@x/a")
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b.Files["x/d/specs/a.md"]); !strings.Contains(got, "| @x/d:D-1 | from root (override by @x/a) | ok |") {
		t.Fatalf("root override not applied:\n%s", got)
	}
}

// CMP-6
func TestLockfile(t *testing.T) {
	f := newFx()
	f.spec("@x/c", "o/c", ver("0.1.1", nil, nil), ver("0.1.0", nil, nil))
	f.spec("@x/a", "o/a", ver("0.1.0", map[string]string{"@x/c": "^0.1"}, nil))
	sel, err := Select(f.ix, "@x/a", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	lock := sel.Lock()
	p := filepath.Join(t.TempDir(), "spechub.lock")
	if err := WriteLock(p, lock); err != nil {
		t.Fatal(err)
	}
	read, _ := ReadLock(p)
	if read.Specs["@x/c"].Version != "0.1.0" || read.Specs["@x/c"].SHA != "sha-0.1.0" {
		t.Fatalf("lock %+v", read)
	}
	// A pinned SHA wins even if the index moved.
	read.Specs["@x/c"] = LockEntry{Version: "0.1.1", SHA: "pinned", Repo: "o/c"}
	sel2, err := Select(f.ix, "@x/a", "", read)
	if err != nil {
		t.Fatal(err)
	}
	if n := sel2.Node("@x/c"); n.SHA != "pinned" || n.Version.String() != "0.1.1" {
		t.Fatalf("got %+v", n)
	}
	if missing, _ := ReadLock(filepath.Join(t.TempDir(), "none")); missing != nil {
		t.Fatal("missing lock should be nil")
	}
}

// CMP-7, CMP-8, CMP-9
func TestBundleLayoutAndRewrite(t *testing.T) {
	f := newFx()
	rb := f.repo(t, "@x/auth", "AUTH", "| AUTH-1 | users exist | ok |", "| AUTH-2 | magic links, see [AUTH-1](a.md) | ok |", "| AUTH-3 | passwords | ok |")
	ra := f.repo(t, "@x/app", "APP", "| APP-1 | uses `@x/auth:AUTH-1` and `@x/auth` | ok |")
	f.spec("@x/auth", rb, ver("0.1.0", nil, nil))
	f.spec("@x/app", ra, ver("0.2.0", map[string]string{"@x/auth": "^0.1"}, map[string]string{"@x/auth:AUTH-2": "removed", "@x/auth:AUTH-3": "passkeys only"}))
	b, err := build(t, f, "@x/app")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"BUNDLE.md", "x/auth/README.md", "x/auth/specs/a.md", "x/app/README.md", "x/app/specs/a.md"}
	if strings.Join(b.Paths, ",") != strings.Join(want, ",") {
		t.Fatalf("paths %v", b.Paths)
	}
	auth := string(b.Files["x/auth/specs/a.md"])
	for _, s := range []string{
		"| @x/auth:AUTH-1 | users exist | ok |",
		"| @x/auth:AUTH-2 | Removed (by @x/app) | ok |",
		"| @x/auth:AUTH-3 | passkeys only (override by @x/app) | ok |",
	} {
		if !strings.Contains(auth, s) {
			t.Errorf("auth bundle missing %q:\n%s", s, auth)
		}
	}
	app := string(b.Files["x/app/specs/a.md"])
	if !strings.Contains(app, "[`@x/auth:AUTH-1`](../../auth/specs/a.md)") || !strings.Contains(app, "[`@x/auth`](../../auth/README.md)") {
		t.Errorf("global refs not linked:\n%s", app)
	}
	md := string(b.Files["BUNDLE.md"])
	if !strings.Contains(md, "# Bundle: @x/app@0.2.0") || strings.Index(md, "@x/auth](") > strings.Index(md, "@x/app](") {
		t.Errorf("BUNDLE.md order:\n%s", md)
	}
	out := filepath.Join(t.TempDir(), "spechub-bundle")
	if err := b.Write(out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "x", "app", "specs", "a.md")); err != nil {
		t.Fatal(err)
	}
	if err := b.Write(out); err != nil {
		t.Fatalf("rewrite over an old bundle: %v", err)
	}
	other := t.TempDir()
	os.WriteFile(filepath.Join(other, "keep.txt"), nil, 0o644)
	if err := b.Write(other); err == nil {
		t.Fatal("must not overwrite a non-bundle dir")
	}
	if doc := b.Markdown(); !strings.HasPrefix(doc, "# Bundle:") || !strings.Contains(doc, "<!-- file: x/app/specs/a.md -->") {
		t.Fatal("markdown document")
	}
}
