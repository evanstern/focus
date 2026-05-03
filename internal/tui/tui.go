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
	"os/exec"
	"strings"

	"github.com/evanstern/focus/internal/board"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Run boots the TUI program against the given board and runs until
// the user quits. Caller is responsible for board resolution; this
// matches the CLI handler shape (open the board, hand it to the TUI).
//
// We pin the lipgloss default renderer's color profile based on
// TERM/COLORTERM. termenv's auto-detection sometimes returns Ascii
// in tmux/screen contexts where colors actually work fine; the pin
// makes the TUI's colors deterministic.
func Run(b *board.Board) error {
	lipgloss.SetColorProfile(detectProfile())

	m, err := newModel(b)
	if err != nil {
		return err
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

// detectProfile picks a sensible color profile from environment
// variables. termenv's own ColorProfile() is gated on isTTY(stdout)
// which can return false when bubbletea is initializing the program
// before our renderer takes over. We replicate the env-only pieces
// of termenv's detection here so we always get color in the common
// terminals.
func detectProfile() termenv.Profile {
	switch os.Getenv("COLORTERM") {
	case "24bit", "truecolor":
		return termenv.TrueColor
	case "yes", "true":
		return termenv.ANSI256
	}
	term := os.Getenv("TERM")
	switch term {
	case "alacritty", "wezterm", "xterm-kitty", "xterm-ghostty", "rio", "contour":
		return termenv.TrueColor
	}
	if strings.Contains(term, "256color") {
		return termenv.ANSI256
	}
	if strings.Contains(term, "color") || strings.Contains(term, "ansi") {
		return termenv.ANSI
	}
	return termenv.ANSI256
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

	view  viewMode
	input inputMode
	split splitMode

	width  int
	height int

	board_  boardModel
	preview previewModel
	help    helpModel
	search  searchState
	command commandState

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
		board_:  bm,
		preview: newPreviewModel(b),
		help:    newHelpModel(),
		search:  newSearchState(),
		command: newCommandState(),
	}, nil
}

// Init is Bubble Tea's startup hook. We use it to fire off the first
// board reload so the screen has data before any keystroke.
func (m *Model) Init() tea.Cmd {
	return reloadCmd(m.board, m.board_.filter)
}

// Update routes messages to handlers. Cursor movement in the board
// triggers a preview reload as a side effect.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case reloadedMsg:
		m.board_.applyReload(msg)
		m.status = ""
		m.refreshPreview()
		return m, nil

	case statusMsg:
		m.status = string(msg)
		return m, nil

	case editFinishedMsg:
		// $EDITOR exited; reload the card from disk in case the user
		// changed body or frontmatter.
		if msg.id != 0 {
			m.preview.invalidate(msg.id)
		}
		return m, reloadCmd(m.board, m.board_.filter)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// handleKey is the central key router. Modal first (search /
// command override most other handling), then per-view routing.
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.input == modeSearch {
		return m.handleSearchKey(msg)
	}
	if m.input == modeCommand {
		return m.handleCommandKey(msg)
	}

	// Global keys regardless of view.
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "?":
		if m.view == viewHelp {
			m.view = viewBoard
		} else {
			m.view = viewHelp
		}
		return m, nil
	case ":":
		m.input = modeCommand
		m.command.reset()
		return m, nil
	}

	switch m.view {
	case viewBoard:
		return m.handleBoardKey(msg)
	case viewHelp:
		if msg.String() == "esc" || msg.String() == "q" {
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
// render the split board layout.
func (m *Model) View() string {
	if m.view == viewHelp {
		return m.help.view() + "\n" + m.statusLine()
	}
	bodyHeight := m.height - statusBarHeight
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	body := renderSplit(m.split, m.width, bodyHeight, &m.board_, &m.preview)
	return body + "\n" + m.statusLine()
}

// statusBarHeight reserves rows at the bottom of the screen for the
// status line + mode indicator.
const statusBarHeight = 2

// statusLine renders the bottom bar.
func (m *Model) statusLine() string {
	var mode string
	switch m.input {
	case modeNormal:
		mode = fmt.Sprintf("[%s]", m.board_.filter.label())
	case modeSearch:
		mode = "/" + m.search.query
	case modeCommand:
		mode = ":" + m.command.input
	}
	if m.status != "" {
		return fmt.Sprintf("%s  %s", mode, m.status)
	}
	hint := "j/k move  tab filter  s layout  e edit  a/p/d/K/r transition  /search  :command  ?help  q quit"
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
// we just supply the *exec.Cmd.
func (m *Model) editCmd(id int) tea.Cmd {
	dirName, err := m.board.FindCardDir(id)
	if err != nil {
		return func() tea.Msg { return statusMsg("edit: " + err.Error()) }
	}
	path := m.board.CardFile(dirName)

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, path)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editFinishedMsg{id: id, err: err}
	})
}
