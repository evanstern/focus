package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/evanstern/focus/internal/board/card"
	"github.com/evanstern/focus/internal/board/index"
)

func TestPadOrTruncShorterPads(t *testing.T) {
	got := padOrTrunc("abc", 6)
	if got != "abc   " {
		t.Errorf("got %q", got)
	}
}

func TestPadOrTruncLongerEllipses(t *testing.T) {
	got := padOrTrunc("a long title that overflows", 12)
	if ansi.StringWidth(got) != 12 {
		t.Errorf("width = %d, want 12", ansi.StringWidth(got))
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("expected ellipsis, got %q", got)
	}
}

func TestComputeColumnWidthsWideAllVisible(t *testing.T) {
	c := computeColumnWidths(80)
	if c.titleW < 30 {
		t.Errorf("title too narrow at 80 cols: %d", c.titleW)
	}
	if c.ownerW == 0 {
		t.Errorf("owner column dropped at 80 cols")
	}
	if c.projectW == 0 || c.priorityW == 0 {
		t.Errorf("project/priority columns dropped at 80 cols")
	}
}

func TestComputeColumnWidthsNarrowDropsOwner(t *testing.T) {
	c := computeColumnWidths(40)
	if c.ownerW != 0 {
		t.Errorf("owner should drop at 40 cols, got %d", c.ownerW)
	}
	if c.titleW < 1 {
		t.Errorf("title got too narrow: %d", c.titleW)
	}
}

func TestComputeColumnWidthsVeryNarrowDropsAll(t *testing.T) {
	c := computeColumnWidths(20)
	if c.projectW != 0 || c.priorityW != 0 || c.ownerW != 0 {
		t.Errorf("very-narrow should drop all but id+title, got %+v", c)
	}
}

func TestFormatCardRowFitsWidth(t *testing.T) {
	e := index.Entry{
		ID:       42,
		Title:    "A pretty long title that will need to be truncated",
		Project:  "myproject",
		Priority: card.PriorityP1,
		Owner:    "alice",
	}
	for _, w := range []int{40, 60, 80, 120} {
		t.Run("", func(t *testing.T) {
			cols := computeColumnWidths(w)
			row := formatCardRow(e, cols)
			got := ansi.StringWidth(row)
			if got > w {
				t.Errorf("width=%d row width=%d, exceeds limit", w, got)
			}
		})
	}
}
