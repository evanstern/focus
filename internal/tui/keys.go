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
	query   string
	matches []int
	cursor  int
}

func newSearchState() searchState { return searchState{} }

// commandState mirrors searchState for : mode.
type commandState struct {
	input string
}

func newCommandState() commandState { return commandState{} }

func (c *commandState) reset() { c.input = "" }

// handleBoardKey handles normal-mode keys for the split board.
// Cursor movement triggers a preview reload via refreshPreview() —
// preview always reflects whatever's under the nav cursor.
func (m *Model) handleBoardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// gg sequence: track the previous "g" so a second "g" jumps to top.
	if key == "g" {
		if m.search.cursor == -1 {
			m.board_.gotoFirstCard()
			m.refreshPreview()
			m.search.cursor = 0
			return m, nil
		}
		m.search.cursor = -1
		return m, nil
	}
	m.search.cursor = 0

	switch key {
	case "q":
		return m, tea.Quit
	case "j", "down":
		m.board_.moveCursor(1)
		m.refreshPreview()
	case "k", "up":
		m.board_.moveCursor(-1)
		m.refreshPreview()
	case "G":
		m.board_.gotoLastCard()
		m.refreshPreview()
	case "ctrl+d", "pgdown":
		m.board_.moveCursor(10)
		m.refreshPreview()
	case "ctrl+u", "pgup":
		m.board_.moveCursor(-10)
		m.refreshPreview()
	case "/":
		m.input = modeSearch
		m.search.query = ""
		m.search.matches = nil
	case "enter", "e", "o":
		if e := m.board_.selectedCard(); e != nil {
			return m, m.editCmd(e.ID)
		}
	case "a":
		return m, m.transitionCmd("activate")
	case "p":
		return m, m.transitionCmd("park")
	case "d":
		return m, m.transitionCmd("done")
	case "K":
		return m, m.transitionCmd("kill")
	case "r":
		return m, m.transitionCmd("revive")
	case "n":
		m.advanceSearchMatch(1)
		m.refreshPreview()
	case "N":
		m.advanceSearchMatch(-1)
		m.refreshPreview()
	case "tab":
		next := m.board_.filter.next()
		return m, reloadCmd(m.board, next)
	}
	return m, nil
}

// handleSearchKey handles key events while in / mode. Enter applies
// the filter; esc cancels.
func (m *Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.input = modeNormal
		m.search.query = ""
		m.search.matches = nil
		return m, nil
	case "enter":
		m.computeMatches()
		m.input = modeNormal
		if len(m.search.matches) > 0 {
			m.jumpToMatch(0)
		} else {
			m.status = "no matches"
		}
		return m, nil
	case "backspace":
		if len(m.search.query) > 0 {
			m.search.query = m.search.query[:len(m.search.query)-1]
		}
		return m, nil
	}
	if isPrintable(msg) {
		m.search.query += msg.String()
	}
	return m, nil
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

// isPrintable reports whether a key event is a single printable rune
// we should append to a search/command buffer. Bubble Tea reports
// special keys ("enter", "ctrl+x") as multi-rune strings; we filter
// those out by length.
func isPrintable(msg tea.KeyMsg) bool {
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

// computeMatches builds m.search.matches from m.search.query. We
// match against title, project, owner, and tags — explicitly NOT
// body text, since the index doesn't carry it (designs/focus-issue
// -001.md §"Search behavior").
func (m *Model) computeMatches() {
	q := strings.ToLower(strings.TrimSpace(m.search.query))
	m.search.matches = m.search.matches[:0]
	if q == "" {
		return
	}
	for _, r := range m.board_.rows {
		if !r.isCard() {
			continue
		}
		e := r.entry
		hay := strings.ToLower(e.Title + " " + e.Project + " " + e.Owner + " " + strings.Join(e.Tags, " "))
		if strings.Contains(hay, q) {
			m.search.matches = append(m.search.matches, e.ID)
		}
	}
}

// jumpToMatch moves the cursor to the row of the i-th search match.
func (m *Model) jumpToMatch(i int) {
	if len(m.search.matches) == 0 {
		return
	}
	if i < 0 {
		i = len(m.search.matches) - 1
	} else if i >= len(m.search.matches) {
		i = 0
	}
	id := m.search.matches[i]
	for idx, r := range m.board_.rows {
		if r.isCard() && r.entry.ID == id {
			m.board_.cursor = idx
			return
		}
	}
}

// advanceSearchMatch implements n / N over the matches.
func (m *Model) advanceSearchMatch(delta int) {
	if len(m.search.matches) == 0 {
		return
	}
	curID := -1
	if e := m.board_.selectedCard(); e != nil {
		curID = e.ID
	}
	pos := 0
	for i, id := range m.search.matches {
		if id == curID {
			pos = i
			break
		}
	}
	m.jumpToMatch(pos + delta)
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
		return reloadCmd(board_, filter)()
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
		return func() tea.Msg {
			if _, err := board_.Reindex(); err != nil {
				return statusMsg("reindex: " + err.Error())
			}
			return reloadCmd(board_, filter)()
		}
	case "new":
		if rest == "" {
			return func() tea.Msg { return statusMsg("usage: :new <title>") }
		}
		board_ := m.board
		filter := m.board_.filter
		return func() tea.Msg {
			if _, _, err := board_.NewCard(rest, board.NewCardOpts{}); err != nil {
				return statusMsg("new: " + err.Error())
			}
			return reloadCmd(board_, filter)()
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
