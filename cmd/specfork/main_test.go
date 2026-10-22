package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/specfork/specfork/internal/index"
	"github.com/specfork/specfork/internal/resolve"
	"github.com/specfork/specfork/internal/testfx"
)

func newApp(t *testing.T) (*app, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	ix, fetch := testfx.Seeds(t)
	var out, errb bytes.Buffer
	return &app{stdout: &out, stderr: &errb, fetcher: fetch, loadIndex: func() (*index.Index, error) { return ix, nil }}, &out, &errb
}

// CLI-1
func TestSearch(t *testing.T) {
	a, out, _ := newApp(t)
	if code := a.run([]string{"search", "team", "roles"}); code != exitOK {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "@specfork/teams") || strings.Contains(out.String(), "@specfork/auth ") {
		t.Fatalf("output:\n%s", out)
	}
	out.Reset()
	a.run([]string{"search", "--json", "SIGN-IN"})
	var hits []map[string]any
	if err := json.Unmarshal(out.Bytes(), &hits); err != nil || len(hits) != 1 || hits[0]["name"] != "@specfork/auth" {
		t.Fatalf("json %s %v", out, err)
	}
}

// CLI-2
func TestInfo(t *testing.T) {
	a, out, _ := newApp(t)
	if code := a.run([]string{"info", "@specfork/scheduling"}); code != exitOK {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{"@specfork/scheduling 0.1.0 (active)", "requires:    @specfork/auth ^0.1, @specfork/teams ^0.1, @specfork/webhooks ^0.1", "github.com/calcom/cal.com (MIT)"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

// CLI-3
func TestResolve(t *testing.T) {
	a, out, _ := newApp(t)
	dir := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)
	if code := a.run([]string{"resolve", "@specfork/scheduling"}); code != exitOK {
		t.Fatalf("exit %d: %s", code, out)
	}
	for _, p := range []string{"specfork-bundle/BUNDLE.md", "specfork-bundle/specfork/scheduling/specs/slots.md", "specfork-bundle/specfork/auth/README.md", "specfork.lock"} {
		if _, err := os.Stat(filepath.Join(dir, p)); err != nil {
			t.Errorf("missing %s", p)
		}
	}
	lock, _ := resolve.ReadLock(filepath.Join(dir, "specfork.lock"))
	if lock.Root != "@specfork/scheduling" || len(lock.Specs) != 4 || lock.Specs["@specfork/auth"].SHA != "v0.1.0-sha" {
		t.Fatalf("lock %+v", lock)
	}
	slots, _ := os.ReadFile(filepath.Join(dir, "specfork-bundle/specfork/scheduling/specs/bookings.md"))
	if !strings.Contains(string(slots), "| @specfork/scheduling:BKG-1 |") {
		t.Fatal("IDs not global in bundle")
	}
	if code := a.run([]string{"resolve", "@specfork/scheduling@^0.1"}); code != exitUsage {
		t.Fatalf("range with a lock present: exit %d", code)
	}
	if code := a.run([]string{"resolve", "--update", "@specfork/scheduling@^0.1"}); code != exitOK {
		t.Fatalf("--update: exit %d", code)
	}
}

// CLI-4
func TestLint(t *testing.T) {
	a, out, _ := newApp(t)
	if code := a.run([]string{"lint", filepath.Join(testfx.SeedsDir(), "scheduling")}); code != exitOK {
		t.Fatalf("seed: exit %d\n%s", code, out)
	}
	out.Reset()
	if code := a.run([]string{"lint", "../../internal/spec/testdata/bad"}); code != exitFail || !strings.Contains(out.String(), "FMT-5") {
		t.Fatalf("bad: exit %d\n%s", code, out)
	}
}

// CLI-9
func TestExitCodes(t *testing.T) {
	a, _, _ := newApp(t)
	if code := a.run(nil); code != exitUsage {
		t.Errorf("no args: %d", code)
	}
	if code := a.run([]string{"frobnicate"}); code != exitUsage {
		t.Errorf("unknown: %d", code)
	}
	if code := a.run([]string{"info", "@nobody/x"}); code != exitFail {
		t.Errorf("not found: %d", code)
	}
	a.loadIndex = func() (*index.Index, error) { return nil, netError{errors.New("offline")} }
	if code := a.run([]string{"search", "x"}); code != exitNetwork {
		t.Errorf("network: %d", code)
	}
}

// CLI-5
func TestIndexFetch(t *testing.T) {
	t.Setenv("SPECFORK_INDEX", "")
	if index.Source() != index.DefaultURL {
		t.Fatal("default URL")
	}
	p := filepath.Join(t.TempDir(), "ix.json")
	os.WriteFile(p, []byte(`{"generated_at":"x","specs":[{"name":"@a/b","state":"active"}],"unclaimed":[]}`), 0o644)
	t.Setenv("SPECFORK_INDEX", p)
	ix, err := index.Load(index.Source())
	if err != nil || len(ix.Specs) != 1 {
		t.Fatalf("%v %v", ix, err)
	}
}
