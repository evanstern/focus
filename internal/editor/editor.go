// Package editor builds the *exec.Cmd that launches the user's
// $EDITOR on a given file. It exists as its own package so the CLI
// (focus edit) and the TUI (e/enter keybind) share the same parsing
// of $EDITOR — including the multi-token form like "code -w" or
// "vim -u NONE" that exec.Command can't handle directly.
package editor

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

// DefaultEditor is the fallback when $EDITOR is unset.
const DefaultEditor = "vi"

// Command parses $EDITOR (falling back to DefaultEditor when unset)
// and returns an *exec.Cmd that runs it on path. $EDITOR is split on
// whitespace so values like "code -w" are interpreted as binary +
// args; this matches git's GIT_EDITOR semantics. Use Command.Stdin /
// Stdout / Stderr to wire I/O before calling Run().
//
// Returns an error if $EDITOR is set but contains only whitespace.
func Command(path string) (*exec.Cmd, error) {
	editor := strings.TrimSpace(os.Getenv("EDITOR"))
	if editor == "" {
		editor = DefaultEditor
	}
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return nil, errors.New("EDITOR is empty")
	}
	args := append(parts[1:], path)
	return exec.Command(parts[0], args...), nil
}
