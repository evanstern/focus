package cli

import (
	"context"
	"fmt"
	"io"
	"os/signal"
	"syscall"

	"github.com/evanstern/focus/internal/mcp"
)

// runMCP dispatches the `focus mcp ...` family. v0.1.0 ships only
// `focus mcp serve` (stdio JSON-RPC). Future subcommands (e.g.
// inspect, validate-config) would slot in here.
func runMCP(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: focus mcp serve")
		return 2
	}
	switch args[0] {
	case "serve":
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		if err := mcp.Serve(ctx, Version); err != nil {
			fmt.Fprintf(stderr, "focus: mcp serve: %v\n", err)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(stderr, "focus: unknown mcp subcommand %q. try `focus mcp serve`.\n", args[0])
		return 2
	}
}
