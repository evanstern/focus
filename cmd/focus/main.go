// Command focus is a project-local kanban board for solo developers
// and their orchestrator agents.
//
// See README.md for an overview. Implementation phases are tracked
// in IMPLEMENT.md at the repo root during the v2 build-out.
package main

import (
	"os"

	"github.com/evanstern/focus/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
