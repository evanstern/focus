package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/evanstern/focus/internal/board"
	"github.com/evanstern/focus/internal/board/card"

	tea "github.com/charmbracelet/bubbletea"
)

// searchState owns the input buffer for / mode and the matched ids.
// We don't use bubbles/textinput; the model is small enough that a
// plain string + a few key handlers is more direct than wiring up
// a sub-model.
type searchState struct {
	query string
}

func newSearchState() searchState { return searchState{} }

// commandState mirrors searchState for : mode.
type commandState struct {
	input string
}

func newCommandState() commandState { return commandState{} }

func (c *commandState) reset() { c.input = "" }

// handleBoardKey handles normal-mode keys for the split board.
// Movement keys (j/k/gg/G/ctrl+d/u/f/b) route to whichever pane has
// focus: nav when m.focused == focusNav (cursor moves and preview
// reloads), preview when m.focused == focusPreview (viewport scrolls).
// Filter cycle, transitions, search, command-mode, edit and quit work
// regardless of focused pane.
func (m *Model) handleBoardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// gg sequence: track whether the previous keystroke was "g" so
	// a second "g" jumps to top of whichever pane is focused. Any
	// other key clears the flag.
	if key == "g" {
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

	switch key {
	case "q":
		return m, tea.Quit
	case "tab":
		m.focused = (m.focused + 1) % numPanes
		return m, nil
	case "shift+tab":
		m.focused = (m.focused + numPanes - 1) % numPanes
		return m, nil
	case "j", "down":
		if m.focused == focusPreview {
			m.preview.scrollLineDown()
		} else {
			m.board_.moveCursor(1)
			m.refreshPreview()
		}
	case "k", "up":
		if m.focused == focusPreview {
			m.preview.scrollLineUp()
		} else {
			m.board_.moveCursor(-1)
			m.refreshPreview()
		}
	case "G":
		if m.focused == focusPreview {
			m.preview.scrollToBottom()
		} else {
			m.board_.gotoLastCard()
			m.refreshPreview()
		}
	case "ctrl+d", "pgdown":
		if m.focused == focusPreview {
			m.preview.scrollHalfPageDown()
		} else {
			m.board_.moveCursor(10)
			m.refreshPreview()
		}
	case "ctrl+u", "pgup":
		if m.focused == focusPreview {
			m.preview.scrollHalfPageUp()
		} else {
			m.board_.moveCursor(-10)
			m.refreshPreview()
		}
	case "ctrl+f":
		if m.focused == focusPreview {
			m.preview.scrollPageDown()
		}
	case "ctrl+b":
		if m.focused == focusPreview {
			m.preview.scrollPageUp()
		}
	case "/":
		m.input = modeSearch
		m.search.query = ""
		m.board_.applyFilter("")
	case "enter", "e", "o":
		if e := m.board_.selectedCard(); e != nil {
			return m, m.editCmd(e.ID)
		}
	case "a":
		return m, m.transitionCmd("activate")
	case "p":
		return m, m.transitionCmd("park")
	case "d":
		if m.preview.card != nil && len(m.preview.card.Contract) > 0 {
			m.status = fmt.Sprintf("contract has %d item(s); use `focus done %d` from CLI", len(m.preview.card.Contract), m.preview.card.ID)
			return m, nil
		}
		return m, m.transitionCmd("done")
	case "K":
		return m, m.transitionCmd("kill")
	case "r":
		return m, m.transitionCmd("revive")
	case "l":
		next := m.board_.filter.next()
		return m, reloadCmd(m.board, next, 0)
	case "h":
		prev := m.board_.filter.prev()
		return m, reloadCmd(m.board, prev, 0)
	case "s":
		m.split = m.split.next()
		m.status = "layout: " + m.split.label()
	}
	return m, nil
}

// handleSearchKey handles key events while in / mode. Search is
// live: every keystroke filters the visible nav rows. Esc clears
// the query and exits search mode; enter just exits search mode but
// keeps the filter applied so the user can keep navigating with
// j/k over the matched set.
func (m *Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.input = modeNormal
		m.search.query = ""
		m.status = ""
		m.board_.applyFilter("")
		m.refreshPreview()
		return m, nil
	case "enter":
		m.input = modeNormal
		m.status = ""
		return m, nil
	case "backspace":
		if len(m.search.query) > 0 {
			m.search.query = m.search.query[:len(m.search.query)-1]
		}
		m.board_.applyFilter(m.search.query)
		m.updateNoMatchesStatus()
		m.refreshPreview()
		return m, nil
	}
	if isPrintable(msg) {
		m.search.query += msg.String()
		m.board_.applyFilter(m.search.query)
		m.updateNoMatchesStatus()
		m.refreshPreview()
	}
	return m, nil
}

// updateNoMatchesStatus sets m.status to a feedback message based on
// the current filtered row count, or clears it if at least one match
// is visible. Cleared on every keystroke so the message tracks the
// live query rather than sticking around forever.
func (m *Model) updateNoMatchesStatus() {
	if m.search.query == "" {
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
func (m *Model) handleCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.input = modeNormal
		return m, nil
	case "enter":
		cmd := m.runCommandLine(m.command.input)
		m.input = modeNormal
		m.command.reset()
		return m, cmd
	case "backspace":
		if len(m.command.input) > 0 {
			m.command.input = m.command.input[:len(m.command.input)-1]
		}
		return m, nil
	}
	if isPrintable(msg) {
		m.command.input += msg.String()
	}
	return m, nil
}

// isPrintable reports whether a key event is a printable character
// we should append to a search/command buffer. Bubble Tea reports
// most printable characters as KeyRunes but space gets its own
// dedicated KeySpace type, which we accept too.
func isPrintable(msg tea.KeyMsg) bool {
	if msg.Type == tea.KeySpace {
		return true
	}
	if msg.Type != tea.KeyRunes {
		return false
	}
	for _, r := range msg.Runes {
		if r < 0x20 {
			return false
		}
	}
	return true
}

// transitionCmd is the command issued when the user presses a, p,
// d, K, or r in board view. Runs the transition off the
// message-handling goroutine, then triggers a reload so the screen
// reflects reality.
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
// always false from the TUI; users who need --force can drop to the
// CLI.
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
		board_ := m.board
		filter := m.board_.filter
		return func() tea.Msg {
			c, _, err := board_.NewCard(rest, board.NewCardOpts{})
			if err != nil {
				return statusMsg("new: " + err.Error())
			}
			return reloadCmd(board_, filter, c.ID)()
		}
	}

	// Unknown commands: silently no-op rather than yelling, matching
	// vim's behavior on unknown ":foo".
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
