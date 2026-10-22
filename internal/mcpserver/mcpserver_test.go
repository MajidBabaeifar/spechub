package mcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/MajidBabaeifar/spechub/internal/index"
	"github.com/MajidBabaeifar/spechub/internal/testfx"
)

func connect(t *testing.T) *mcp.ClientSession {
	t.Helper()
	ix, fetch := testfx.Seeds(t)
	srv := New(Deps{LoadIndex: func() (*index.Index, error) { return ix, nil }, Fetcher: fetch, Version: "test"})
	st, ct := mcp.NewInMemoryTransports()
	ctx := context.Background()
	if _, err := srv.Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "t", Version: "1"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

// MCP-1
func TestMCPHandshake(t *testing.T) {
	cs := connect(t)
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tl := range res.Tools {
		names = append(names, tl.Name)
	}
	if strings.Join(names, ",") != "get_spec,resolve_spec,search_specs" {
		t.Fatalf("tools %v", names)
	}
}

func text(t *testing.T, r *mcp.CallToolResult) string {
	t.Helper()
	if r.IsError {
		t.Fatalf("tool error: %v", r.Content)
	}
	return r.Content[0].(*mcp.TextContent).Text
}

// MCP-2, MCP-3
func TestMCPTools(t *testing.T) {
	cs := connect(t)
	ctx := context.Background()
	r, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "search_specs", Arguments: map[string]any{"query": "booking"}})
	if err != nil {
		t.Fatal(err)
	}
	var found SearchOut
	json.Unmarshal([]byte(text(t, r)), &found)
	if len(found.Specs) != 1 || found.Specs[0].Name != "@spechub/scheduling" {
		t.Fatalf("search %+v", found)
	}
	r, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "get_spec", Arguments: map[string]any{"name": "@spechub/auth"}})
	if err != nil {
		t.Fatal(err)
	}
	var info GetOut
	json.Unmarshal([]byte(text(t, r)), &info)
	if info.Version != "0.1.0" || strings.Join(info.RequiredBy, ",") != "@spechub/scheduling,@spechub/teams" {
		t.Fatalf("info %+v", info)
	}
	r, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "resolve_spec", Arguments: map[string]any{"name": "@spechub/scheduling"}})
	if err != nil {
		t.Fatal(err)
	}
	doc := text(t, r)
	if !strings.HasPrefix(doc, "# Bundle: @spechub/scheduling@0.1.0") || !strings.Contains(doc, "| @spechub/scheduling:SLT-1 |") || !strings.Contains(doc, "<!-- file: spechub/auth/specs/auth.md -->") {
		t.Fatalf("bundle starts %q", doc[:200])
	}
	r, _ = cs.CallTool(ctx, &mcp.CallToolParams{Name: "resolve_spec", Arguments: map[string]any{"name": "@nobody/x"}})
	if !r.IsError {
		t.Fatal("unknown spec should be a tool error")
	}
}
