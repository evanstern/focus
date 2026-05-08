// Package mcp implements the focus MCP server (`focus mcp serve`).
//
// Tools are thin wrappers over internal/board so the MCP surface and
// the CLI surface never drift — designs/focus-issue-001.md §"CLI/MCP
// shared logic placement". The server runs over stdio and resolves
// the .focus/ board the same way the CLI does: ancestor-walk from
// the calling agent's working directory.
package mcp

import (
	"context"
	"os"
	"sync"

	"github.com/evanstern/focus/internal/board"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// serverDefaultFocusDir is the focus dir resolved once at server
// startup using FOCUS_DIR + the upward walk from the server's CWD.
// Per-tool `focus_dir` arguments override this. Empty string means
// "no default available" (resolution failed at startup; the per-tool
// arg becomes mandatory).
var (
	serverDefaultFocusDir   string
	serverDefaultFocusDirMu sync.RWMutex
)

func setServerDefaultFocusDir(s string) {
	serverDefaultFocusDirMu.Lock()
	defer serverDefaultFocusDirMu.Unlock()
	serverDefaultFocusDir = s
}

func getServerDefaultFocusDir() string {
	serverDefaultFocusDirMu.RLock()
	defer serverDefaultFocusDirMu.RUnlock()
	return serverDefaultFocusDir
}

func newImplementation(version string) *mcpsdk.Implementation {
	return &mcpsdk.Implementation{
		Name:    "focus",
		Version: version,
	}
}

// Serve runs the MCP server over stdio until the client disconnects
// or ctx is cancelled. version is stamped into the initialize-time
// implementation info; the CLI passes its own Version constant in.
//
// At startup the server resolves a default focus dir using
// FOCUS_DIR and the upward walk from the server's CWD. Per-tool
// `focus_dir` arguments override that default.
func Serve(ctx context.Context, version string) error {
	if dir, err := board.Resolve("", os.Getenv("FOCUS_DIR")); err == nil {
		setServerDefaultFocusDir(dir)
	}
	srv := mcpsdk.NewServer(newImplementation(version), nil)
	registerTools(srv)
	return srv.Run(ctx, &mcpsdk.StdioTransport{})
}

// resolveBoardWithArg returns the board to use for a single tool
// call. The per-tool focus_dir arg, if set, takes precedence over
// the server default. If neither is set, ErrNotInBoard is returned
// so the agent gets a useful message.
func resolveBoardWithArg(focusDirArg string) (*board.Board, error) {
	if focusDirArg != "" {
		dir, err := board.Resolve(focusDirArg, "")
		if err != nil {
			return nil, err
		}
		return board.OpenAt(dir)
	}
	if def := getServerDefaultFocusDir(); def != "" {
		return board.OpenAt(def)
	}
	return nil, board.ErrNotInBoard
}
