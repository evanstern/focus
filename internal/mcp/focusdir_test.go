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

func clientForServer(t *testing.T) (*mcpsdk.ClientSession, func()) {
	t.Helper()
	srv := mcpsdk.NewServer(newImplementation("test"), nil)
	registerTools(srv)

	ctx := context.Background()
	ct, st := mcpsdk.NewInMemoryTransports()
	ss, err := srv.Connect(ctx, st, nil)
	if err != nil {
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
		t.Fatalf("client.Connect: %v", err)
	}
	cleanup := func() {
		_ = cs.Close()
		wg.Wait()
	}
	return cs, cleanup
}

func TestPerToolFocusDirOverridesServerDefault(t *testing.T) {
	defaultRoot := t.TempDir()
	overrideRoot := t.TempDir()
	if _, err := board.Init(defaultRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := board.Init(overrideRoot); err != nil {
		t.Fatal(err)
	}

	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(defaultRoot); err != nil {
		t.Fatal(err)
	}
	dir, err := board.Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	setServerDefaultFocusDir(dir)
	t.Cleanup(func() { setServerDefaultFocusDir("") })

	cs, cleanup := clientForServer(t)
	defer cleanup()

	_ = callTool(t, cs, "focus_new", map[string]any{"title": "default-card"})
	_ = callTool(t, cs, "focus_new", map[string]any{
		"title":     "override-card",
		"focus_dir": overrideRoot,
	})

	res := callTool(t, cs, "focus_board", map[string]any{})
	data, _ := json.Marshal(res.StructuredContent)
	var br BoardResult
	_ = json.Unmarshal(data, &br)
	if len(br.Backlog) != 1 || br.Backlog[0].Title != "default-card" {
		t.Errorf("default board = %v", br.Backlog)
	}

	res = callTool(t, cs, "focus_board", map[string]any{"focus_dir": overrideRoot})
	data, _ = json.Marshal(res.StructuredContent)
	br = BoardResult{}
	_ = json.Unmarshal(data, &br)
	if len(br.Backlog) != 1 || br.Backlog[0].Title != "override-card" {
		t.Errorf("override board = %v", br.Backlog)
	}
}

func TestServerDefaultResolvedAtStartup(t *testing.T) {
	root := t.TempDir()
	if _, err := board.Init(root); err != nil {
		t.Fatal(err)
	}

	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	dir, err := board.Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	setServerDefaultFocusDir(dir)
	t.Cleanup(func() { setServerDefaultFocusDir("") })

	other := t.TempDir()
	if err := os.Chdir(other); err != nil {
		t.Fatal(err)
	}

	cs, cleanup := clientForServer(t)
	defer cleanup()

	_ = callTool(t, cs, "focus_new", map[string]any{"title": "still-original"})
	res := callTool(t, cs, "focus_board", map[string]any{})
	data, _ := json.Marshal(res.StructuredContent)
	var br BoardResult
	_ = json.Unmarshal(data, &br)
	if len(br.Backlog) != 1 || br.Backlog[0].Title != "still-original" {
		t.Errorf("server did not stay on startup-resolved board: %v", br.Backlog)
	}
}

func TestPerToolFocusDirAcceptsFocusDirDirectly(t *testing.T) {
	root := t.TempDir()
	if _, err := board.Init(root); err != nil {
		t.Fatal(err)
	}
	setServerDefaultFocusDir("")
	t.Cleanup(func() { setServerDefaultFocusDir("") })

	cs, cleanup := clientForServer(t)
	defer cleanup()

	res, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "focus_new",
		Arguments: map[string]any{
			"title":     "via-focus-dir",
			"focus_dir": root + "/.focus",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool error: %v", res.Content)
	}
}
