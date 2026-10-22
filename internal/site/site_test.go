package site

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MajidBabaeifar/spechub/internal/index"
)

// IDX-9
func TestSitePages(t *testing.T) {
	ix := &index.Index{GeneratedAt: "2026-10-22T12:00:00Z", Unclaimed: []string{"eve/thing"}, Specs: []index.Spec{
		{Name: "@spechub/auth", Repo: "o/spechub-auth", State: "active", Summary: "Sign-in <script>", License: "CC-BY-4.0", HeadSHA: "abcdef1234",
			Head:     index.Snapshot{Files: []string{"specs/auth.md"}},
			Versions: []index.Version{{Version: "0.1.0", SHA: "1111111222", Snapshot: index.Snapshot{Files: []string{"specs/auth.md"}}}}},
		{Name: "@spechub/scheduling", Repo: "o/spechub-scheduling", State: "active", Summary: "Booking", License: "CC-BY-4.0", HeadSHA: "ffff",
			Versions:        []index.Version{{Version: "0.1.0", SHA: "2222", Snapshot: index.Snapshot{Requires: map[string]string{"@spechub/auth": "^0.1"}, Files: []string{"specs/slots.md"}}}},
			Implementations: []index.Impl{{Repo: "o/scheduling-rebuild", Stack: []string{"go", "sqlite"}, Version: "0.1.0"}}},
	}}
	dir := t.TempDir()
	if err := Generate(ix, dir); err != nil {
		t.Fatal(err)
	}
	read := func(p string) string {
		b, err := os.ReadFile(filepath.Join(dir, p))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	home := read("index.html")
	for _, want := range []string{`href="s/spechub/scheduling/"`, "Sign-in &lt;script&gt;", "eve/thing", "1 implementation<", `href="style.css"`} {
		if !strings.Contains(home, want) {
			t.Errorf("index.html missing %q", want)
		}
	}
	page := read("s/spechub/scheduling/index.html")
	for _, want := range []string{
		"spechub resolve @spechub/scheduling",
		`href="../../../s/spechub/auth/"`,
		"https://github.com/o/scheduling-rebuild",
		"go, sqlite",
		"https://github.com/o/spechub-scheduling/blob/2222/specs/slots.md",
		`href="../../../style.css"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("scheduling page missing %q", want)
		}
	}
	if auth := read("s/spechub/auth/index.html"); !strings.Contains(auth, `href="../../../s/spechub/scheduling/"`) {
		t.Error("auth page lacks required-by link")
	}
	var back index.Index
	if err := json.Unmarshal([]byte(read("index.json")), &back); err != nil || len(back.Specs) != 2 {
		t.Fatalf("index.json: %v", err)
	}
	read(".nojekyll")
}
