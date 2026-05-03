// Package tui implements `focus tui`: a Bubble Tea TUI over the same
// internal/board layer the CLI and MCP server use.
//
// Architecture: state enum + delegated Update/View. The top-level
// model holds a viewMode and routes messages to the active sub-model
// (board / detail / help). This is the pattern from
// happytaoer/cli_kanban referenced in
// wiki/decisions/focus-stack.md.
//
// Vim keybindings are first-class per
// wiki/decisions/focus-tui-keybinds.md. Arrows still work for users
// who don't speak vim.
package tui

import (
	"fmt"

	"github.com/evanstern/focus/internal/board"

	tea "github.com/charmbracelet/bubbletea"
)

// Run boots the TUI program against the given board and runs until
// the user quits. Caller is responsible for board resolution; this
// matches the CLI handler shape (open the board, hand it to the TUI).
func Run(b *board.Board) error {
	m, err := newModel(b)
	if err != nil {
		return err
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

// viewMode is the top-level state enum. Each value names a sub-model
// the root delegates to. Modes are mutually exclusive.
type viewMode int

const (
	viewBoard viewMode = iota
	viewDetail
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

// Model is the root Bubble Tea model. Holds the resolved board, the
// current view + input modes, and references to each sub-model so
// Update can route messages to whichever is active.
type Model struct {
	board *board.Board

	view  viewMode
	input inputMode

	width  int
	height int

	board_  boardModel
	detail  detailModel
	help    helpModel
	search  searchState
	command commandState

	// status is the line at the bottom of the screen used for ephemeral
	// feedback after a transition or error. Cleared on the next key.
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
		detail:  newDetailModel(b),
		help:    newHelpModel(),
		search:  newSearchState(),
		command: newCommandState(),
	}, nil
}

// Init is Bubble Tea's startup hook. We use it to fire off the first
// board reload so the screen has data before any keystroke.
func (m *Model) Init() tea.Cmd {
	return reloadCmd(m.board)
}

// Update routes messages to the active sub-model's Update or handles
// global shortcuts (quit, help, mode switches). Per Bubble Tea's
// pattern, Update returns the (possibly mutated) model and an
// optional command.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.detail.resize(m.width, m.height)
		return m, nil

	case reloadedMsg:
		m.board_.applyReload(msg)
		m.status = ""
		return m, nil

	case statusMsg:
		m.status = string(msg)
		return m, nil

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
	case viewDetail:
		return m.handleDetailKey(msg)
	case viewHelp:
		if msg.String() == "esc" || msg.String() == "q" {
			m.view = viewBoard
		}
		return m, nil
	}
	return m, nil
}

// View dispatches to the active sub-model's View.
func (m *Model) View() string {
	var body string
	switch m.view {
	case viewBoard:
		body = m.board_.view(m.width, m.height-statusBarHeight)
	case viewDetail:
		body = m.detail.view()
	case viewHelp:
		body = m.help.view()
	}
	return body + "\n" + m.statusLine()
}

// statusBarHeight reserves rows at the bottom of the screen for the
// status line + mode indicator. Bubble Tea's lipgloss layout pads
// the body to fit.
const statusBarHeight = 2

// statusLine renders the bottom bar: input-mode indicator, ephemeral
// status message, and (in search/command modes) the current input.
func (m *Model) statusLine() string {
	var mode string
	switch m.input {
	case modeNormal:
		mode = "[NORMAL]"
	case modeSearch:
		mode = "/" + m.search.query
	case modeCommand:
		mode = ":" + m.command.input
	}
	if m.status != "" {
		return fmt.Sprintf("%s  %s", mode, m.status)
	}
	hint := "j/k move  enter open  /search  :command  ?help  q quit"
	return fmt.Sprintf("%s  %s", mode, hint)
}
