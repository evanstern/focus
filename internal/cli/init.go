package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/evanstern/focus/internal/board"
)

func runInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprintln(stderr, "usage: focus init [path]") }
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return 2
	}

	path := ""
	if fs.NArg() > 0 {
		path = fs.Arg(0)
	} else if focusDirFlag != "" {
		path = focusDirFlag
	} else if env := os.Getenv("FOCUS_DIR"); env != "" {
		path = env
	}

	if path == "" || path == "." {
		var err error
		path, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "focus: %v\n", err)
			return 1
		}
	}
	if filepath.Base(path) == board.FocusDirName {
		path = filepath.Dir(path)
	}

	b, err := board.Init(path)
	if err != nil {
		fmt.Fprintf(stderr, "focus: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "initialized focus board at %s\n", b.Dir)
	return 0
}
