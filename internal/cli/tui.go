package cli

import (
	"fmt"
	"io"

	"github.com/evanstern/focus/internal/tui"
)

func runTUI(_ []string, _, stderr io.Writer) int {
	b, code := openBoard(stderr)
	if code != 0 {
		return code
	}
	if err := tui.Run(b); err != nil {
		fmt.Fprintf(stderr, "focus: tui: %v\n", err)
		return 1
	}
	return 0
}
