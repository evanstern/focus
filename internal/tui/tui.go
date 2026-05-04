// Package tui implements `focus tui`: a Bubble Tea TUI over the same
// internal/board layer the CLI and MCP server use.
//
// Layout: a split-pane board view. Nav (the card list) sits on the
// left when the terminal is wide enough; otherwise nav stacks on top
// of the preview. Cursor movement in the nav loads the highlighted
// card into the preview pane on every keystroke. There's no separate
// "detail view" mode — `e` and `enter` jump straight to $EDITOR.
//
// Vim keybindings are first-class per
// wiki/decisions/focus-tui-keybinds.md. Arrows still work for users
// who don't speak vim.
package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/evanstern/focus/internal/board"
	"github.com/evanstern/focus/internal/editor"

	"github.com/charmbracelet/colorprofile"

	tea "charm.land/bubbletea/v2"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
)

// Run boots the TUI program against the given board and runs until
// the user quits. Caller is responsible for board resolution; this
// matches the CLI handler shape (open the board, hand it to the TUI).
//
// We pass the color profile via tea.WithColorProfile based on
// TERM/COLORTERM. v2 dropped lipgloss's global default renderer so
// the program-level option is now the way to pin profile detection
// in tmux/screen contexts where auto-detection returns Ascii.
func Run(b *board.Board) error {
	m, err := newModel(b)
	if err != nil {
		return err
	}
	p := tea.NewProgram(m, tea.WithColorProfile(detectProfile()))
	_, err = p.Run()
	return err
}

// detectProfile picks a sensible color profile from environment
// variables. The runtime's auto-detection can return Ascii when the
// stdout TTY check fails during program init; this env-only sniff
// keeps colors deterministic across the common terminals.
func detectProfile() colorprofile.Profile {
	switch os.Getenv("COLORTERM") {
	case "24bit", "truecolor":
		return colorprofile.TrueColor
	case "yes", "true":
		return colorprofile.ANSI256
	}
	term := os.Getenv("TERM")
	switch term {
	case "alacritty", "wezterm", "xterm-kitty", "xterm-ghostty", "rio", "contour":
		return colorprofile.TrueColor
	}
	if strings.Contains(term, "256color") {
		return colorprofile.ANSI256
	}
	if strings.Contains(term, "color") || strings.Contains(term, "ansi") {
		return colorprofile.ANSI
	}
	return colorprofile.ANSI256
}

// viewMode is the top-level state enum. Two modes only: the split
// board (with nav + preview) and a help overlay.
type viewMode int

const (
	viewBoard viewMode = iota
	viewHelp
)

// inputMode is orthogonal to viewMode: it's "what is the keyboard
// currently doing?" rather than "what is on screen?". Search and
// command-mode are input modes that overlay the board view.
type inputMode int

const (
	modeNormal inputMode = iota
	modeSearch
	modeCommand
)

// focusedPane is which pane currently owns the keyboard for
// movement. Tab cycles it; the visual indicator is an accent border.
// Default on TUI entry is focusNav.
type focusedPane int

const (
	focusNav focusedPane = iota
	focusPreview
	numPanes = 2
)

// splitMode controls how the nav and preview panes are arranged.
// 's' cycles auto -> horizontal -> vertical -> auto. Auto picks
// horizontal when the terminal is wide enough for both panes,
// vertical otherwise.
type splitMode int

const (
	splitAuto splitMode = iota
	splitHorizontal
	splitVertical
)

func (s splitMode) label() string {
	switch s {
	case splitHorizontal:
		return "horizontal"
	case splitVertical:
		return "vertical"
	}
	return "auto"
}

func (s splitMode) next() splitMode { return (s + 1) % 3 }

// autoSplitThreshold is the minimum terminal width at which auto-mode
// picks horizontal layout. Below this, auto stacks vertically.
const autoSplitThreshold = 120

// Model is the root Bubble Tea model. Holds the resolved board, the
// current view + input modes, and references to each sub-model.
type Model struct {
	board *board.Board

	view    viewMode
	input   inputMode
	split   splitMode
	focused focusedPane

	width  int
	height int

	keys     KeyMap
	board_   boardModel
	preview  previewModel
	help     help.Model
	search   searchState
	command  commandState
	gPending bool

	// status is the line at the bottom of the screen used for ephemeral
	// feedback after a transition or error. Cleared on the next reload.
	status string
}

func newModel(b *board.Board) (*Model, error) {
	bm, err := newBoardModel(b)
	if err != nil {
		return nil, err
	}
	return &Model{
		board:   b,
		view:    viewBoard,
		input:   modeNormal,
		keys:    DefaultKeyMap(),
		board_:  bm,
		preview: newPreviewModel(b),
		help:    help.New(),
		search:  newSearchState(),
		command: newCommandState(),
	}, nil
}

// Init is Bubble Tea's startup hook. We use it to fire off the first
// board reload so the screen has data before any keystroke.
func (m *Model) Init() tea.Cmd {
	return reloadCmd(m.board, m.board_.filter, 0)
}

// Update routes messages to handlers. Cursor movement in the board
// triggers a preview reload as a side effect.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.SetWidth(msg.Width)
		return m, nil

	case reloadedMsg:
		m.board_.applyReload(msg, msg.preserveID)
		m.board_.applyFilter(m.search.Value())
		m.status = ""
		m.refreshPreview()
		return m, nil

	case statusMsg:
		m.status = string(msg)
		return m, nil

	case editFinishedMsg:
		// $EDITOR exited; reload the card from disk in case the user
		// changed body or frontmatter, and put the cursor back on the
		// card they just edited.
		if msg.id != 0 {
			m.preview.invalidate(msg.id)
		}
		return m, reloadCmd(m.board, m.board_.filter, msg.id)

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.PasteMsg:
		// In v2, paste arrives as its own message type. Hand it to
		// whichever input is open; in normal mode we silently drop it.
		if m.input == modeSearch {
			return m.handleSearchPaste(msg)
		}
		if m.input == modeCommand {
			return m.handleCommandPaste(msg)
		}
		return m, nil
	}
	return m, nil
}

// handleKey is the central key router. Modal first (search /
// command override most other handling), then per-view routing.
func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.input == modeSearch {
		return m.handleSearchKey(msg)
	}
	if m.input == modeCommand {
		return m.handleCommandKey(msg)
	}

	// Global keys regardless of view.
	switch {
	case msg.Code == 'c' && msg.Mod == tea.ModCtrl:
		return m, tea.Quit
	case key.Matches(msg, m.keys.Help):
		if m.view == viewHelp {
			m.view = viewBoard
		} else {
			m.view = viewHelp
		}
		return m, nil
	case key.Matches(msg, m.keys.Command):
		m.input = modeCommand
		m.command.reset()
		return m, m.command.Focus()
	}

	switch m.view {
	case viewBoard:
		return m.handleBoardKey(msg)
	case viewHelp:
		// Esc / q dismiss help. Quit is handled above.
		s := msg.String()
		if s == "esc" || s == "q" {
			m.view = viewBoard
		}
		return m, nil
	}
	return m, nil
}

// refreshPreview asks the preview pane to load whatever card is
// currently under the cursor. Called after the cursor moves and
// after every reload.
func (m *Model) refreshPreview() {
	e := m.board_.selectedCard()
	if e == nil {
		m.preview.card = nil
		return
	}
	if err := m.preview.load(e.ID); err != nil {
		m.status = err.Error()
	}
}

// View renders the screen. Help overlays everything; otherwise we
// render the bordered split board layout above a bordered status bar.
func (m *Model) View() tea.View {
	var content string
	if m.view == viewHelp {
		content = m.help.View(m.keys) + "\n" + m.renderStatusBar()
	} else {
		bodyHeight := m.height - statusBarOuterHeight
		if bodyHeight < 1 {
			bodyHeight = 1
		}
		body := renderSplit(m.split, m.width, bodyHeight, m.focused, &m.board_, &m.preview)
		content = body + "\n" + m.renderStatusBar()
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

// statusBarOuterHeight is the row budget the bordered status bar
// consumes: 1 line of content + 2 lines of border.
const statusBarOuterHeight = 3

// renderStatusBar produces the bottom-bar widget: a single line of
// mode + hint/status, wrapped in the same rounded border as the
// nav and preview panes.
func (m *Model) renderStatusBar() string {
	innerW := m.width - borderChrome
	if innerW < 1 {
		innerW = 1
	}
	return borderedPane(m.statusContent(), innerW, 1, false)
}

// statusContent is the single status-bar content line, without
// border or padding.
func (m *Model) statusContent() string {
	var mode string
	switch m.input {
	case modeNormal:
		mode = fmt.Sprintf("[%s]", m.board_.filter.label())
	case modeSearch:
		mode = "/" + m.search.Value()
	case modeCommand:
		mode = ":" + m.command.Value()
	}
	if m.status != "" {
		return fmt.Sprintf("%s  %s", mode, m.status)
	}
	hint := "tab focus  j/k move/scroll  h/l filter  s layout  e edit  a/p/d/K/r transition  /search  :command  ?help  q quit"
	return fmt.Sprintf("%s  %s", mode, hint)
}

// editFinishedMsg is sent by Bubble Tea after $EDITOR exits. Carries
// the card id so the preview cache can be invalidated.
type editFinishedMsg struct {
	id  int
	err error
}

// editCmd suspends the TUI, runs $EDITOR on the card's INDEX.md,
// then resumes. tea.ExecProcess handles the suspend/resume dance;
// editor.Command parses $EDITOR (including multi-token forms like
// "code -w") so we don't choke on common editor configs.
func (m *Model) editCmd(id int) tea.Cmd {
	dirName, err := m.board.FindCardDir(id)
	if err != nil {
		return func() tea.Msg { return statusMsg("edit: " + err.Error()) }
	}
	path := m.board.CardFile(dirName)

	cmd, err := editor.Command(path)
	if err != nil {
		return func() tea.Msg { return statusMsg("edit: " + err.Error()) }
	}
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editFinishedMsg{id: id, err: err}
	})
}
