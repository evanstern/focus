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
	b.WriteString("MOVEMENT\n")
	b.WriteString("  j / down       move down\n")
	b.WriteString("  k / up         move up\n")
	b.WriteString("  gg             top of list\n")
	b.WriteString("  G              bottom of list\n")
	b.WriteString("  ctrl-d / pgdn  jump down 10\n")
	b.WriteString("  ctrl-u / pgup  jump up 10\n\n")
	b.WriteString("ACTIONS (on highlighted card)\n")
	b.WriteString("  enter / e / o  open in $EDITOR\n")
	b.WriteString("  a              activate (backlog -> active)\n")
	b.WriteString("  p              park (active -> backlog)\n")
	b.WriteString("  d              done (active -> done)\n")
	b.WriteString("  K              kill (any -> archived)  [capital K]\n")
	b.WriteString("  r              revive (archived -> backlog)\n\n")
	b.WriteString("FILTER\n")
	b.WriteString("  l / tab        next filter (in-flight \u2192 all \u2192 done \u2192 archived)\n")
	b.WriteString("  h / shift-tab  prev filter\n\n")
	b.WriteString("LAYOUT\n")
	b.WriteString("  s              cycle auto \u2192 horizontal \u2192 vertical\n\n")
	b.WriteString("MODES\n")
	b.WriteString("  /              search\n")
	b.WriteString("  :              command-mode\n")
	b.WriteString("  esc            exit search/command\n\n")
	b.WriteString("SEARCH\n")
	b.WriteString("  /              start live filter (matches id, title, project, owner, tags)\n")
	b.WriteString("  enter          accept filter, return to navigation\n")
	b.WriteString("  esc            cancel and clear filter\n\n")
	b.WriteString("COMMAND-MODE\n")
	b.WriteString("  :q             quit\n")
	b.WriteString("  :reindex       rebuild .focus/index.json\n")
	b.WriteString("  :new <title>   create card\n\n")
	b.WriteString("META\n")
	b.WriteString("  ?              toggle help\n")
	b.WriteString("  q / ctrl-c     quit\n")
	return helpModel{text: b.String()}
}

func (m helpModel) view() string {
	return m.text + "\n(press esc or q to dismiss)\n"
}
