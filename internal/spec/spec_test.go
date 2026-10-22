package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FMT-1, FMT-2: front matter is read and checked.
func TestFrontMatterSchema(t *testing.T) {
	fm, err := ParseFrontMatter([]byte("---\nspecfork: 1\nname: \"@a/b\"\nsummary: s\nlicense: MIT\nrequires:\n  \"@a/c\": \"^0.1\"\nwhat: 1\n---\n# body\n"))
	if err != nil {
		t.Fatal(err)
	}
	if fm.Name != "@a/b" || fm.Requires["@a/c"] != "^0.1" || len(fm.Unknown) != 1 || fm.Unknown[0] != "what" {
		t.Fatalf("parsed %+v", fm)
	}
	if _, err := ParseFrontMatter([]byte("# no front matter\n")); err == nil {
		t.Fatal("want error without front matter")
	}
}

// FMT-8: implementation front matter.
func TestDetectImplRepo(t *testing.T) {
	fm, err := ParseFrontMatter([]byte("---\nimplements: \"@specfork/scheduling@0.1.0\"\nstack: [go, sqlite]\nurl: https://x.example\n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	name, ver := SplitNameVersion(fm.Implements)
	if name != "@specfork/scheduling" || ver != "0.1.0" || len(fm.Stack) != 2 {
		t.Fatalf("got %q %q %v", name, ver, fm.Stack)
	}
}

func TestSplitRowEscapedPipe(t *testing.T) {
	c := SplitRow("| A-1 | a `x\\|y` b | t |")
	if len(c) != 3 || c[1] != "a `x\\|y` b" {
		t.Fatalf("%q", c)
	}
}

// The published seeds lint clean.
func TestLintSeeds(t *testing.T) {
	dirs, _ := filepath.Glob("../testfx/testdata/seeds/*/README.md")
	if len(dirs) == 0 {
		t.Skip("no seeds in this checkout")
	}
	for _, readme := range dirs {
		issues, err := LintDir(filepath.Dir(readme))
		if err != nil {
			t.Fatal(err)
		}
		for _, i := range issues {
			t.Errorf("%s: %s", filepath.Dir(readme), i)
		}
	}
}

// FMT-3, FMT-4, FMT-5, FMT-9: each broken rule is reported.
func TestLintAreas(t *testing.T) {
	issues, err := LintDir("testdata/bad")
	if err != nil {
		t.Fatal(err)
	}
	var all []string
	for _, i := range issues {
		all = append(all, i.String())
	}
	got := strings.Join(all, "\n")
	for _, want := range []string{
		"FMT-1: front matter needs `specfork: 1`",
		"FMT-2: name \"@Bad/Name\"",
		"FMT-2: `summary` is required",
		"is not an SPDX expression",
		"requires @specfork/auth",
		"\"AUTH-1\" is not a global ID",
		"warning FMT-2: unknown front matter field \"colour\"",
		"one.md:10: error FMT-4: ONE-1 also defined",
		"one.md:11: error FMT-4: TWO-9 doesn't use the area prefix ONE",
		"one.md:12: error FMT-5: ONE-2 has an empty Verified by",
		"one.md:14: error FMT-9: link ../../other/specs/x.md leaves the repo",
		"one.md:9: error FMT-9: TWO-1 is not defined in this repo",
		"two.md: error FMT-3: missing **Status:**",
		"two.md: error FMT-4: prefix ONE also used",
		"two.md: error FMT-3: missing table header",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "ONE-3 has an empty") {
		t.Error("removed rows need no check")
	}
}

func TestLintVerifiedByRequired(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "specs"), 0o755)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("---\nspecfork: 1\nname: \"@a/b\"\nsummary: s\nlicense: MIT\n---\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "specs", "a.md"), []byte("# A\n\n**Status:** Draft · **Prefix:** A\n\n| ID | Requirement | Verified by |\n|---|---|---|\n| A-1 | x | y |\n"), 0o644)
	issues, err := LintDir(dir)
	if err != nil || len(issues) != 0 {
		t.Fatalf("clean repo: %v %v", issues, err)
	}
}

// FMT-2: tags.
func TestLintTags(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "specs"), 0o755)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("---\nspecfork: 1\nname: \"@a/b\"\nsummary: s\nlicense: MIT\ntags: [saas, Bad_Tag, a, b, c, d]\n---\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "specs", "a.md"), []byte("# A\n\n**Status:** Draft · **Prefix:** A\n\n| ID | Requirement | Verified by |\n|---|---|---|\n| A-1 | x | y |\n"), 0o644)
	issues, _ := LintDir(dir)
	var all []string
	for _, i := range issues {
		all = append(all, i.String())
	}
	got := strings.Join(all, "\n")
	if !strings.Contains(got, "6 tags, max 5") || !strings.Contains(got, `tag "Bad_Tag"`) {
		t.Fatalf("got:\n%s", got)
	}
}
