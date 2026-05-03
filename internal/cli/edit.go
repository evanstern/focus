package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
)

func runEdit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("edit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprintln(stderr, "usage: focus edit <id>") }
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return 2
	}
	id, err := strconv.Atoi(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "focus: invalid id %q\n", fs.Arg(0))
		return 2
	}

	b, code := openBoard(stderr)
	if code != 0 {
		return code
	}
	dirName, err := b.FindCardDir(id)
	if err != nil {
		fmt.Fprintf(stderr, "focus: %v\n", err)
		return 1
	}
	path := b.CardFile(dirName)

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	// In non-tty contexts (CI, MCP, scripts) opening an editor is the
	// wrong behavior; print the path so the caller can do something
	// useful with it. v1 grew this affordance organically; v2 ships
	// with it from day one.
	if !isTTY(os.Stdout) {
		fmt.Fprintln(stdout, path)
		return 0
	}

	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(stderr, "focus: editor failed: %v\n", err)
		return 1
	}
	return 0
}

// isTTY reports whether f is a character device (terminal). Used by
// runEdit to decide whether to spawn $EDITOR or print the card path.
//
// We avoid the golang.org/x/term dep for a one-line decision; Stat()
// + ModeCharDevice is the stdlib pattern.
func isTTY(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
