package cli

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/evanstern/focus/internal/board/card"
	"github.com/evanstern/focus/internal/board/index"
)

func makeEntry(id int, title string) index.Entry {
	return index.Entry{
		ID:       id,
		Title:    title,
		Project:  "demo",
		Priority: card.Priority("p2"),
		Owner:    "ash",
		Status:   card.Status("active"),
	}
}

func TestFormatRowWidth_80ColTruncatesAndAligns(t *testing.T) {
	long := "A very long title that absolutely will not fit inside an 80 column terminal row"
	short := "short title"
	rowLong := formatRowWidth(makeEntry(1, long), 80, false)
	rowShort := formatRowWidth(makeEntry(2, short), 80, false)

	if utf8.RuneCountInString(rowLong) != utf8.RuneCountInString(rowShort) {
		t.Fatalf("rows must be the same rune width for column alignment\n  long  (%d): %q\n  short (%d): %q",
			utf8.RuneCountInString(rowLong), rowLong,
			utf8.RuneCountInString(rowShort), rowShort)
	}

	if !strings.ContainsRune(rowLong, '…') {
		t.Errorf("long row should contain ellipsis marker, got %q", rowLong)
	}
	if strings.ContainsRune(rowShort, '…') {
		t.Errorf("short row should NOT contain ellipsis, got %q", rowShort)
	}

	if !strings.Contains(rowLong, "demo") || !strings.Contains(rowLong, "p2") || !strings.Contains(rowLong, "ash") {
		t.Errorf("long row missing fixed columns: %q", rowLong)
	}
}

func TestFormatRowWidth_NarrowTerminalHonorsMinBudget(t *testing.T) {
	long := "An impractically long title for a 40 column terminal"
	row := formatRowWidth(makeEntry(1, long), 40, false)

	titleSegment := strings.SplitN(row, "  ", 3)
	if len(titleSegment) < 2 {
		t.Fatalf("could not isolate title segment in row %q", row)
	}
	titleRunes := utf8.RuneCountInString(strings.TrimRight(titleSegment[1], " "))
	if titleRunes < minTitleBudget {
		t.Errorf("title segment shrank below minTitleBudget: got %d runes in %q", titleRunes, row)
	}

	if !strings.Contains(row, "demo") {
		t.Errorf("project column missing on narrow terminal: %q", row)
	}
}

func TestFormatRowWidth_NoTruncatePreservesFullTitle(t *testing.T) {
	long := "A very long title that absolutely will not fit inside an 80 column terminal row"
	row := formatRowWidth(makeEntry(1, long), 80, true)

	if !strings.Contains(row, long) {
		t.Errorf("--no-truncate should preserve full title, got %q", row)
	}
	if strings.ContainsRune(row, '…') {
		t.Errorf("--no-truncate row should not contain ellipsis, got %q", row)
	}
}

func TestFormatRowWidth_MultiByteTitleTruncatesByRune(t *testing.T) {
	title := strings.Repeat("世", 60)
	row := formatRowWidth(makeEntry(1, title), 80, false)

	if !strings.ContainsRune(row, '…') {
		t.Errorf("expected ellipsis on overflowing CJK title: %q", row)
	}
	for _, r := range row {
		if r == utf8.RuneError {
			t.Fatalf("row contains invalid UTF-8: %q", row)
		}
	}

	short := formatRowWidth(makeEntry(2, "short"), 80, false)
	if utf8.RuneCountInString(row) != utf8.RuneCountInString(short) {
		t.Errorf("CJK row width %d != short row width %d (cols misaligned)",
			utf8.RuneCountInString(row), utf8.RuneCountInString(short))
	}
}

func TestFormatRowWidth_ZeroWidthFallsBackToLegacy(t *testing.T) {
	long := "A very long title that absolutely will not fit inside an 80 column terminal row"
	row := formatRowWidth(makeEntry(1, long), 0, false)
	if !strings.Contains(row, long) {
		t.Errorf("width=0 should behave like --no-truncate, got %q", row)
	}
	if strings.ContainsRune(row, '…') {
		t.Errorf("width=0 should not produce ellipsis, got %q", row)
	}
}

func TestFormatRowWidth_NegativeWidthFloorsToMinBudget(t *testing.T) {
	long := "A very long title that absolutely will not fit anywhere"
	row := formatRowWidth(makeEntry(1, long), -5, false)

	if !strings.ContainsRune(row, '…') {
		t.Errorf("negative width must still truncate (not fall back to legacy), got %q", row)
	}
	if strings.Contains(row, long) {
		t.Errorf("negative width should not preserve full title, got %q", row)
	}
}

func TestTruncateRunes(t *testing.T) {
	cases := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"fits", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"trunc-ascii", "abcdefgh", 5, "abcd…"},
		{"trunc-cjk", "世界世界世界", 4, "世界世…"},
		{"max-zero", "abc", 0, ""},
		{"max-one", "abcdef", 1, "…"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := truncateRunes(tc.in, tc.max)
			if got != tc.want {
				t.Errorf("truncateRunes(%q, %d) = %q, want %q", tc.in, tc.max, got, tc.want)
			}
		})
	}
}

func TestPrintList_NoTruncateRetainsTitle(t *testing.T) {
	long := "A very long title that absolutely will not fit inside an 80 column terminal row"
	entries := []index.Entry{makeEntry(1, long)}

	var buf bytes.Buffer
	printList(&buf, entries, 80, true)
	if !strings.Contains(buf.String(), long) {
		t.Errorf("--no-truncate dropped title: %q", buf.String())
	}

	buf.Reset()
	printList(&buf, entries, 80, false)
	if !strings.ContainsRune(buf.String(), '…') {
		t.Errorf("default list should truncate, got %q", buf.String())
	}
}

func TestListNoTruncateFlag(t *testing.T) {
	root := t.TempDir()
	if code, _, _ := runIn(t, root, "init"); code != 0 {
		t.Fatal("init")
	}
	long := "A very long title that absolutely will not fit inside an 80 column terminal row"
	if code, _, _ := runIn(t, root, "new", long); code != 0 {
		t.Fatal("new")
	}

	code, out, _ := runIn(t, root, "list", "--no-truncate")
	if code != 0 {
		t.Fatalf("list --no-truncate exit %d", code)
	}
	if !strings.Contains(out, long) {
		t.Errorf("--no-truncate dropped title: %q", out)
	}
}

func TestBoardNoTruncateFlag(t *testing.T) {
	root := t.TempDir()
	if code, _, _ := runIn(t, root, "init"); code != 0 {
		t.Fatal("init")
	}
	long := "A very long title that absolutely will not fit inside an 80 column terminal row"
	if code, _, _ := runIn(t, root, "new", long); code != 0 {
		t.Fatal("new")
	}

	code, out, _ := runIn(t, root, "board", "--no-truncate")
	if code != 0 {
		t.Fatalf("board --no-truncate exit %d", code)
	}
	if !strings.Contains(out, long) {
		t.Errorf("--no-truncate dropped title: %q", out)
	}
}

func TestEpicListNoTruncateFlag(t *testing.T) {
	root := t.TempDir()
	if code, _, _ := runIn(t, root, "init"); code != 0 {
		t.Fatal("init")
	}
	long := "An impractically long epic title that will not fit inside an 80 column terminal"
	if code, _, _ := runIn(t, root, "new", long, "--type", "epic"); code != 0 {
		t.Fatal("new epic")
	}

	code, out, _ := runIn(t, root, "epic", "list", "--no-truncate")
	if code != 0 {
		t.Fatalf("epic list --no-truncate exit %d", code)
	}
	if !strings.Contains(out, long) {
		t.Errorf("--no-truncate dropped epic title: %q", out)
	}
}
