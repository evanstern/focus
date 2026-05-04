// Package completions implements `focus completions <shell>` and the
// hidden `focus _complete <kind>` candidate producer used by the
// embedded shell scripts.
//
// The three shell scripts (bash.sh, zsh.zsh, fish.fish) are embedded
// at build time via go:embed and printed verbatim; users pipe the
// output into `eval` (bash/zsh) or redirect it to fish's completion
// dir.
package completions

import (
	_ "embed"
	"fmt"
	"io"

	"github.com/evanstern/focus/internal/board"
	"github.com/evanstern/focus/internal/board/card"
)

//go:embed bash.sh
var bashScript string

//go:embed zsh.zsh
var zshScript string

//go:embed fish.fish
var fishScript string

// Script returns the embedded completion script for the given shell.
// Returns an empty string and false for unknown shells.
func Script(shell string) (string, bool) {
	switch shell {
	case "bash":
		return bashScript, true
	case "zsh":
		return zshScript, true
	case "fish":
		return fishScript, true
	}
	return "", false
}

// PublicSubcommands is the list of subcommands the user can invoke.
// Hidden commands (like `_complete`) are deliberately omitted.
var PublicSubcommands = []string{
	"init", "new", "show", "edit", "board", "list",
	"activate", "park", "done", "kill", "revive",
	"reindex", "epic", "mcp", "tui", "completions",
	"version", "help",
}

// Priorities is the canonical priority list. p0 highest.
var Priorities = []string{"p0", "p1", "p2", "p3"}

// Types is the canonical card type list.
var Types = []string{"card", "epic"}

// Statuses is the canonical status list.
var Statuses = []string{"active", "backlog", "done", "archived"}

// PrintSubcommands writes the public subcommand list one per line.
func PrintSubcommands(w io.Writer) {
	for _, s := range PublicSubcommands {
		fmt.Fprintln(w, s)
	}
}

// PrintPriorities writes p0..p3 one per line.
func PrintPriorities(w io.Writer) {
	for _, p := range Priorities {
		fmt.Fprintln(w, p)
	}
}

// PrintTypes writes card/epic one per line.
func PrintTypes(w io.Writer) {
	for _, t := range Types {
		fmt.Fprintln(w, t)
	}
}

// PrintStatuses writes the four statuses one per line.
func PrintStatuses(w io.Writer) {
	for _, s := range Statuses {
		fmt.Fprintln(w, s)
	}
}

// IDFilter narrows which card ids the producer emits.
type IDFilter struct {
	Status card.Status
	Type   card.Type
}

// PrintIDs writes one bare card id per line, ordered by id, filtered
// by f. Bare ints (e.g. "1", not "0001") because that's what the
// shell user types: `focus done 1`.
func PrintIDs(w io.Writer, b *board.Board, f IDFilter) error {
	entries, err := b.List(board.ListOpts{Status: f.Status, Type: f.Type})
	if err != nil {
		return err
	}
	for _, e := range entries {
		fmt.Fprintln(w, e.ID)
	}
	return nil
}
