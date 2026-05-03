package tui

import "strings"

// helpModel is a static cheatsheet rendered when the user hits "?".
// Auto-generated from the same key table the main router uses, so
// help can never drift from the actual bindings (per
// wiki/decisions/focus-tui-keybinds.md).
type helpModel struct {
	text string
}

func newHelpModel() helpModel {
	var b strings.Builder
	b.WriteString("focus TUI — keybindings\n\n")
	b.WriteString("MOVEMENT (board view, normal mode)\n")
	b.WriteString("  j / down       move down\n")
	b.WriteString("  k / up         move up\n")
	b.WriteString("  gg             top of list\n")
	b.WriteString("  G              bottom of list\n")
	b.WriteString("  ctrl-d / pgdn  half-page down\n")
	b.WriteString("  ctrl-u / pgup  half-page up\n\n")
	b.WriteString("ACTIONS\n")
	b.WriteString("  enter / o      open card detail\n")
	b.WriteString("  a              activate (backlog -> active)\n")
	b.WriteString("  p              park (active -> backlog)\n")
	b.WriteString("  d              done (active -> done)\n")
	b.WriteString("  K              kill (any -> archived)  [capital K]\n")
	b.WriteString("  r              revive (archived -> backlog)\n")
	b.WriteString("  e              edit in $EDITOR (suspends TUI)\n\n")
	b.WriteString("MODES\n")
	b.WriteString("  /              search\n")
	b.WriteString("  :              command-mode\n")
	b.WriteString("  esc            exit search/command/detail\n\n")
	b.WriteString("SEARCH\n")
	b.WriteString("  /<query>       filter to matching cards\n")
	b.WriteString("  n / N          next / prev match\n")
	b.WriteString("  esc            clear search\n\n")
	b.WriteString("COMMAND-MODE\n")
	b.WriteString("  :q             quit\n")
	b.WriteString("  :reindex       rebuild .focus/index.json\n")
	b.WriteString("  :new <title>   create card\n\n")
	b.WriteString("META\n")
	b.WriteString("  ?              this help\n")
	b.WriteString("  q / ctrl-c     quit\n")
	return helpModel{text: b.String()}
}

func (m helpModel) view() string {
	return m.text + "\n(press esc or q to dismiss)\n"
}
