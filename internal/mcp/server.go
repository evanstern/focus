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
	"fmt"
	"os"

	"github.com/evanstern/focus/internal/board"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Implementation describes the focus MCP server to clients during
// the initialize handshake. Surfaces the binary version we were
// built with so clients can correlate server behavior to the
// release.
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
// The .focus/ board is resolved lazily inside each tool handler from
// $PWD, which matches CLI semantics and means agents see the board
// for the project they're working in (designs/focus-v2.md §"MCP
// server").
func Serve(ctx context.Context, version string) error {
	srv := mcpsdk.NewServer(newImplementation(version), nil)
	registerTools(srv)
	return srv.Run(ctx, &mcpsdk.StdioTransport{})
}

// resolveBoard is the per-tool entry point: walk for .focus/ from
// $PWD, surface ErrNotInBoard as a tool-level error so the agent
// gets a useful message instead of a transport error.
func resolveBoard() (*board.Board, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getwd: %w", err)
	}
	return board.Open(cwd)
}
