package mcp

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"

	"github.com/evanstern/focus/internal/board"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// boardCtx sets up a tempdir board, chdirs into it, and returns a
// connected client + server pair on an in-memory transport. The
// chdir is necessary because resolveBoard() uses os.Getwd() — the
// MCP server is intentionally CWD-scoped per the design doc.
func boardCtx(t *testing.T) (*mcpsdk.ClientSession, *mcpsdk.ServerSession, func()) {
	t.Helper()
	root := t.TempDir()
	if _, err := board.Init(root); err != nil {
		t.Fatalf("board.Init: %v", err)
	}
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	srv := mcpsdk.NewServer(newImplementation("test"), nil)
	registerTools(srv)

	ctx := context.Background()
	ct, st := mcpsdk.NewInMemoryTransports()
	ss, err := srv.Connect(ctx, st, nil)
	if err != nil {
		_ = os.Chdir(prev)
		t.Fatalf("server.Connect: %v", err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = ss.Wait()
	}()

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		_ = os.Chdir(prev)
		t.Fatalf("client.Connect: %v", err)
	}

	cleanup := func() {
		_ = cs.Close()
		wg.Wait()
		_ = os.Chdir(prev)
	}
	return cs, ss, cleanup
}

func callTool(t *testing.T, cs *mcpsdk.ClientSession, name string, args any) *mcpsdk.CallToolResult {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool %s: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("CallTool %s tool-error: %v", name, res.Content)
	}
	return res
}

func TestToolNewAndBoard(t *testing.T) {
	cs, _, cleanup := boardCtx(t)
	defer cleanup()

	res := callTool(t, cs, "focus_new", map[string]any{"title": "Hello"})
	if res.StructuredContent == nil {
		t.Fatal("focus_new: no structured content")
	}
	data, _ := json.Marshal(res.StructuredContent)
	var nr NewResult
	if err := json.Unmarshal(data, &nr); err != nil {
		t.Fatalf("decode NewResult: %v", err)
	}
	if nr.ID != 1 {
		t.Errorf("first id = %d, want 1", nr.ID)
	}
	if nr.UUID == "" {
		t.Error("uuid empty")
	}

	res = callTool(t, cs, "focus_board", map[string]any{})
	data, _ = json.Marshal(res.StructuredContent)
	var br BoardResult
	if err := json.Unmarshal(data, &br); err != nil {
		t.Fatalf("decode BoardResult: %v", err)
	}
	if len(br.Backlog) != 1 || br.Backlog[0].Title != "Hello" {
		t.Errorf("Backlog = %v", br.Backlog)
	}
}

func TestToolTransitionFlow(t *testing.T) {
	cs, _, cleanup := boardCtx(t)
	defer cleanup()
	_ = callTool(t, cs, "focus_new", map[string]any{"title": "task"})

	res := callTool(t, cs, "focus_activate", map[string]any{"id": 1})
	data, _ := json.Marshal(res.StructuredContent)
	var tr TransitionResult
	_ = json.Unmarshal(data, &tr)
	if tr.Status != "active" {
		t.Errorf("activate result = %+v", tr)
	}

	_ = callTool(t, cs, "focus_done", map[string]any{"id": 1})
	res = callTool(t, cs, "focus_show", map[string]any{"id": 1})
	data, _ = json.Marshal(res.StructuredContent)
	var sr ShowResult
	_ = json.Unmarshal(data, &sr)
	if sr.Status != "done" {
		t.Errorf("show after done: status = %q", sr.Status)
	}
}

func TestToolEditBody(t *testing.T) {
	cs, _, cleanup := boardCtx(t)
	defer cleanup()
	_ = callTool(t, cs, "focus_new", map[string]any{"title": "task"})

	const newBody = "## Changed\n\nReplaced via MCP.\n"
	_ = callTool(t, cs, "focus_edit_body", map[string]any{"id": 1, "body": newBody})

	res := callTool(t, cs, "focus_show", map[string]any{"id": 1})
	data, _ := json.Marshal(res.StructuredContent)
	var sr ShowResult
	_ = json.Unmarshal(data, &sr)
	if sr.Body != newBody {
		t.Errorf("body = %q, want %q", sr.Body, newBody)
	}
}

func TestToolEpicFlow(t *testing.T) {
	cs, _, cleanup := boardCtx(t)
	defer cleanup()

	_ = callTool(t, cs, "focus_new", map[string]any{"title": "Launch", "type": "epic"})
	_ = callTool(t, cs, "focus_new", map[string]any{"title": "Ship feature"})
	_ = callTool(t, cs, "focus_epic_add", map[string]any{"epic_id": 1, "card_id": 2})

	res := callTool(t, cs, "focus_epic_show", map[string]any{"id": 1})
	data, _ := json.Marshal(res.StructuredContent)
	var ep EpicProgress
	_ = json.Unmarshal(data, &ep)
	if ep.Total != 1 || ep.Backlog != 1 {
		t.Errorf("epic progress = %+v", ep)
	}
}

func TestToolReindex(t *testing.T) {
	cs, _, cleanup := boardCtx(t)
	defer cleanup()
	_ = callTool(t, cs, "focus_new", map[string]any{"title": "card"})
	res := callTool(t, cs, "focus_reindex", map[string]any{})
	data, _ := json.Marshal(res.StructuredContent)
	var rr ReindexResult
	_ = json.Unmarshal(data, &rr)
	if rr.Cards != 1 || rr.NextID != 2 {
		t.Errorf("reindex = %+v", rr)
	}
}
