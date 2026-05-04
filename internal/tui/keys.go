package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/evanstern/focus/internal/board"
	"github.com/evanstern/focus/internal/board/card"

	tea "charm.land/bubbletea/v2"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
)

// KeyMap collects every keystroke the TUI responds to. The router
// dispatches via key.Matches against this struct rather than
// switching on raw key strings, and bubbles/help renders the help
// view from the same struct so the two can never drift.
type KeyMap struct {
	// Pane focus
	FocusNext, FocusPrev key.Binding

	// Nav movement (also re-used for preview when preview is focused)
	Up, Down, Top, Bottom key.Binding
	JumpDown, JumpUp      key.Binding

	// Preview-only scroll (full-page)
	ScrollPgDown, ScrollPgUp key.Binding

	// Filter cycle
	FilterNext, FilterPrev key.Binding

	// Layout cycle
	LayoutCycle key.Binding

	// Actions
	Edit, Activate, Park, Done, Kill, Revive key.Binding

	// Modes
	Search, Command, Help, Quit key.Binding
}

// DefaultKeyMap returns the canonical keybinding set, matching the
// contract in wiki/decisions/focus-tui-keybinds.md.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		FocusNext: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "focus next pane"),
		),
		FocusPrev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "focus prev pane"),
		),
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "down"),
		),
		Top: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("gg", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "bottom"),
		),
		JumpDown: key.NewBinding(
			key.WithKeys("ctrl+d", "pgdown"),
			key.WithHelp("ctrl+d", "jump down"),
		),
		JumpUp: key.NewBinding(
			key.WithKeys("ctrl+u", "pgup"),
			key.WithHelp("ctrl+u", "jump up"),
		),
		ScrollPgDown: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("ctrl+f", "page down (preview)"),
		),
		ScrollPgUp: key.NewBinding(
			key.WithKeys("ctrl+b"),
			key.WithHelp("ctrl+b", "page up (preview)"),
		),
		FilterNext: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "next filter"),
		),
		FilterPrev: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "prev filter"),
		),
		LayoutCycle: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "cycle layout"),
		),
		Edit: key.NewBinding(
			key.WithKeys("enter", "e", "o"),
			key.WithHelp("enter/e/o", "edit in $EDITOR"),
		),
		Activate: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "activate"),
		),
		Park: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "park"),
		),
		Done: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "done"),
		),
		Kill: key.NewBinding(
			key.WithKeys("K"),
			key.WithHelp("K", "kill (archive)"),
		),
		Revive: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "revive"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Command: key.NewBinding(
			key.WithKeys(":"),
			key.WithHelp(":", "command-mode"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ShortHelp / FullHelp implement help.KeyMap so bubbles/help renders
// the keybind table directly from the KeyMap struct above.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.FocusNext, k.Up, k.Down, k.Edit, k.FilterNext, k.Search, k.Command, k.Help, k.Quit,
	}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.FocusNext, k.FocusPrev},
		{k.Up, k.Down, k.Top, k.Bottom, k.JumpDown, k.JumpUp, k.ScrollPgDown, k.ScrollPgUp},
		{k.Edit, k.Activate, k.Park, k.Done, k.Kill, k.Revive},
		{k.FilterNext, k.FilterPrev, k.LayoutCycle},
		{k.Search, k.Command, k.Help, k.Quit},
	}
}

// searchState owns the / mode input. Wraps bubbles/textinput so we
// get paste, ctrl-w, ctrl-a/e, etc. for free; the query string is
// just textinput.Value().
type searchState struct {
	textinput.Model
}

func newSearchState() searchState {
	ti := textinput.New()
	ti.Prompt = ""
	return searchState{Model: ti}
}

// Value returns the current search query. Convenience so callers
// don't need to call .Model.Value().
func (s *searchState) Value() string { return s.Model.Value() }

// commandState mirrors searchState for : mode.
type commandState struct {
	textinput.Model
}

func newCommandState() commandState {
	ti := textinput.New()
	ti.Prompt = ""
	return commandState{Model: ti}
}

func (c *commandState) Value() string { return c.Model.Value() }

func (c *commandState) reset() {
	c.Model.SetValue("")
}

// handleBoardKey handles normal-mode keys for the split board.
// Movement keys route by focused pane:
//   - Up/Down/Top/Bottom/JumpDown/JumpUp: both panes — move the nav
//     cursor (and reload the preview) when nav is focused, scroll
//     the preview viewport when preview is focused.
//   - ScrollPgDown/ScrollPgUp: preview only — full-page scroll.
//
// Filter cycle, transitions, search, command-mode, edit and quit
// work regardless of focused pane.
func (m *Model) handleBoardKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// gg sequence: track whether the previous keystroke was "g" so
	// a second "g" jumps to top of whichever pane is focused. Any
	// other key clears the flag.
	if key.Matches(msg, m.keys.Top) {
		if m.gPending {
			if m.focused == focusPreview {
				m.preview.scrollToTop()
			} else {
				m.board_.gotoFirstCard()
				m.refreshPreview()
			}
			m.gPending = false
			return m, nil
		}
		m.gPending = true
		return m, nil
	}
	m.gPending = false

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.FocusNext):
		m.focused = (m.focused + 1) % numPanes
		return m, nil
	case key.Matches(msg, m.keys.FocusPrev):
		m.focused = (m.focused + numPanes - 1) % numPanes
		return m, nil
	case key.Matches(msg, m.keys.Down):
		if m.focused == focusPreview {
			m.preview.scrollLineDown()
		} else {
			m.board_.moveCursor(1)
			m.refreshPreview()
		}
	case key.Matches(msg, m.keys.Up):
		if m.focused == focusPreview {
			m.preview.scrollLineUp()
		} else {
			m.board_.moveCursor(-1)
			m.refreshPreview()
		}
	case key.Matches(msg, m.keys.Bottom):
		if m.focused == focusPreview {
			m.preview.scrollToBottom()
		} else {
			m.board_.gotoLastCard()
			m.refreshPreview()
		}
	case key.Matches(msg, m.keys.JumpDown):
		if m.focused == focusPreview {
			m.preview.scrollHalfPageDown()
		} else {
			m.board_.moveCursor(10)
			m.refreshPreview()
		}
	case key.Matches(msg, m.keys.JumpUp):
		if m.focused == focusPreview {
			m.preview.scrollHalfPageUp()
		} else {
			m.board_.moveCursor(-10)
			m.refreshPreview()
		}
	case key.Matches(msg, m.keys.ScrollPgDown):
		if m.focused == focusPreview {
			m.preview.scrollPageDown()
		}
	case key.Matches(msg, m.keys.ScrollPgUp):
		if m.focused == focusPreview {
			m.preview.scrollPageUp()
		}
	case key.Matches(msg, m.keys.Search):
		m.input = modeSearch
		m.search.reset()
		m.board_.applyFilter("")
		return m, m.search.Focus()
	case key.Matches(msg, m.keys.Edit):
		if e := m.board_.selectedCard(); e != nil {
			return m, m.editCmd(e.ID)
		}
	case key.Matches(msg, m.keys.Activate):
		return m, m.transitionCmd("activate")
	case key.Matches(msg, m.keys.Park):
		return m, m.transitionCmd("park")
	case key.Matches(msg, m.keys.Done):
		if m.preview.card != nil && len(m.preview.card.Contract) > 0 {
			m.status = fmt.Sprintf("contract has %d item(s); use `focus done %d` from CLI", len(m.preview.card.Contract), m.preview.card.ID)
			return m, nil
		}
		return m, m.transitionCmd("done")
	case key.Matches(msg, m.keys.Kill):
		return m, m.transitionCmd("kill")
	case key.Matches(msg, m.keys.Revive):
		return m, m.transitionCmd("revive")
	case key.Matches(msg, m.keys.FilterNext):
		next := m.board_.filter.next()
		return m, reloadCmd(m.board, next, 0)
	case key.Matches(msg, m.keys.FilterPrev):
		prev := m.board_.filter.prev()
		return m, reloadCmd(m.board, prev, 0)
	case key.Matches(msg, m.keys.LayoutCycle):
		m.split = m.split.next()
		m.status = "layout: " + m.split.label()
	}
	return m, nil
}

// reset clears the search buffer.
func (s *searchState) reset() {
	s.Model.SetValue("")
}

// handleSearchKey handles key events while in / mode. Search is
// live: every keystroke filters the visible nav rows. Esc clears
// the query and exits search mode; enter just exits search mode but
// keeps the filter applied.
func (m *Model) handleSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.input = modeNormal
		m.search.reset()
		m.search.Blur()
		m.status = ""
		m.board_.applyFilter("")
		m.refreshPreview()
		return m, nil
	case "enter":
		m.input = modeNormal
		m.search.Blur()
		m.status = ""
		return m, nil
	}
	var cmd tea.Cmd
	m.search.Model, cmd = m.search.Model.Update(msg)
	m.board_.applyFilter(m.search.Value())
	m.updateNoMatchesStatus()
	m.refreshPreview()
	return m, cmd
}

// handleSearchPaste handles tea.PasteMsg in search mode.
func (m *Model) handleSearchPaste(msg tea.PasteMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.search.Model, cmd = m.search.Model.Update(msg)
	m.board_.applyFilter(m.search.Value())
	m.updateNoMatchesStatus()
	m.refreshPreview()
	return m, cmd
}

// updateNoMatchesStatus sets m.status to a feedback message based on
// the current filtered row count, or clears it if at least one match
// is visible.
func (m *Model) updateNoMatchesStatus() {
	if m.search.Value() == "" {
		m.status = ""
		return
	}
	cards := 0
	for _, r := range m.board_.rows {
		if r.isCard() {
			cards++
		}
	}
	if cards == 0 {
		m.status = "no matches"
	} else {
		m.status = ""
	}
}

// handleCommandKey handles : mode.
func (m *Model) handleCommandKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.input = modeNormal
		m.command.Blur()
		return m, nil
	case "enter":
		cmd := m.runCommandLine(m.command.Value())
		m.input = modeNormal
		m.command.reset()
		m.command.Blur()
		return m, cmd
	}
	var cmd tea.Cmd
	m.command.Model, cmd = m.command.Model.Update(msg)
	return m, cmd
}

// handleCommandPaste handles tea.PasteMsg in command mode.
func (m *Model) handleCommandPaste(msg tea.PasteMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.command.Model, cmd = m.command.Model.Update(msg)
	return m, cmd
}

// transitionCmd is the command issued when the user presses a, p,
// d, K, or r in board view.
func (m *Model) transitionCmd(name string) tea.Cmd {
	e := m.board_.selectedCard()
	if e == nil {
		return nil
	}
	id := e.ID
	board_ := m.board
	filter := m.board_.filter
	return func() tea.Msg {
		if err := runTransition(board_, name, id); err != nil {
			return statusMsg(err.Error())
		}
		return reloadCmd(board_, filter, id)()
	}
}

// runTransition dispatches by name to the right board op. Force is
// always false from the TUI.
func runTransition(b *board.Board, name string, id int) error {
	switch name {
	case "activate":
		_, err := b.Activate(id, false)
		return err
	case "park":
		_, err := b.Park(id, false)
		return err
	case "done":
		_, err := b.Done(id, false)
		return err
	case "kill":
		_, err := b.Kill(id, false)
		return err
	case "revive":
		_, err := b.Revive(id, false)
		return err
	default:
		return fmt.Errorf("unknown transition %q", name)
	}
}

// runCommandLine parses the buffer of : mode and executes whatever
// the user typed. Recognized: :q, :reindex, :new <title>.
func (m *Model) runCommandLine(line string) tea.Cmd {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	parts := strings.SplitN(line, " ", 2)
	cmd := parts[0]
	rest := ""
	if len(parts) > 1 {
		rest = parts[1]
	}
	switch cmd {
	case "q", "quit":
		return tea.Quit
	case "reindex":
		board_ := m.board
		filter := m.board_.filter
		preserveID := 0
		if e := m.board_.selectedCard(); e != nil {
			preserveID = e.ID
		}
		return func() tea.Msg {
			if _, err := board_.Reindex(); err != nil {
				return statusMsg("reindex: " + err.Error())
			}
			return reloadCmd(board_, filter, preserveID)()
		}
	case "new":
		if rest == "" {
			return func() tea.Msg { return statusMsg("usage: :new <title>") }
		}
		title := strings.Trim(rest, `"'`)
		board_ := m.board
		filter := m.board_.filter
		return func() tea.Msg {
			c, _, err := board_.NewCard(title, board.NewCardOpts{})
			if err != nil {
				return statusMsg("new: " + err.Error())
			}
			return reloadCmd(board_, filter, c.ID)()
		}
	}

	return func() tea.Msg { return statusMsg("unknown command: " + cmd) }
}

// idForRow extracts a card id from the cursor row label format. Used
// by the help view's mock interactions in tests; not on a hot path.
func idForRow(label string) (int, bool) {
	if !strings.HasPrefix(label, "#") {
		return 0, false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(label, "#"))
	tok := strings.SplitN(rest, " ", 2)[0]
	id, err := strconv.Atoi(tok)
	if err != nil {
		return 0, false
	}
	return id, true
}

var _ = card.PaddedID
