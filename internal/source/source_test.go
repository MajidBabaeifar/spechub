package source

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// CLI-6
func TestFetchBySHA(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Path
		if r.Header.Get("Authorization") != "" {
			t.Error("raw fetches must not send a token")
		}
		w.Write([]byte("# spec"))
	}))
	defer srv.Close()
	old := RawBase
	RawBase = srv.URL
	defer func() { RawBase = old }()
	b, err := GitHub{}.File("o/r", "0123abc", "specs/a.md")
	if err != nil || string(b) != "# spec" || got != "/o/r/0123abc/specs/a.md" {
		t.Fatalf("%q %v %q", b, err, got)
	}
	if _, err := (GitHub{}).File("o/r", "sha", "../x"); err == nil {
		t.Fatal("path traversal accepted")
	}
}
