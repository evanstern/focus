package card

import (
	"strings"
	"testing"
)

func TestNormalizeBodyFoldsParagraphs(t *testing.T) {
	in := "First line of\na paragraph that\nspans three lines.\n\nSecond paragraph\nhas two.\n"
	want := "First line of a paragraph that spans three lines.\n\nSecond paragraph has two.\n"
	if got := NormalizeBody(in); got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestNormalizeBodyPreservesCodeFences(t *testing.T) {
	in := "Para one\nfolded.\n\n```go\nfunc main() {\n    fmt.Println(\"hi\")\n}\n```\n\nPara two\nfolded.\n"
	got := NormalizeBody(in)
	if !strings.Contains(got, "func main() {\n    fmt.Println(\"hi\")\n}") {
		t.Errorf("code fence body got mangled:\n%s", got)
	}
	if !strings.Contains(got, "Para one folded.") {
		t.Errorf("para around fence didn't fold")
	}
}

func TestNormalizeBodyPreservesLists(t *testing.T) {
	in := "Intro\nfolded.\n\n- first\n- second\n- third\n\nOutro\nfolded.\n"
	got := NormalizeBody(in)
	if !strings.Contains(got, "- first\n- second\n- third") {
		t.Errorf("list got folded:\n%s", got)
	}
	if !strings.Contains(got, "Intro folded.") {
		t.Errorf("intro didn't fold")
	}
}

func TestNormalizeBodyPreservesHeadings(t *testing.T) {
	in := "## Heading\n\nLine one\nline two.\n"
	want := "## Heading\n\nLine one line two.\n"
	if got := NormalizeBody(in); got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestNormalizeBodyPreservesBlockquotes(t *testing.T) {
	in := "Para\nfolded.\n\n> quoted\n> still quoted\n\nMore\ntext.\n"
	got := NormalizeBody(in)
	if !strings.Contains(got, "> quoted\n> still quoted") {
		t.Errorf("blockquote folded:\n%s", got)
	}
}

func TestNormalizeBodyHorizontalRule(t *testing.T) {
	in := "Above\nfolded.\n\n---\n\nBelow\nfolded.\n"
	got := NormalizeBody(in)
	if !strings.Contains(got, "\n---\n") {
		t.Errorf("hr lost:\n%s", got)
	}
}

func TestNormalizeBodyOrderedList(t *testing.T) {
	in := "1. one\n2. two\n3. three\n"
	got := NormalizeBody(in)
	if !strings.Contains(got, "1. one\n2. two\n3. three") {
		t.Errorf("ordered list folded:\n%s", got)
	}
}

func TestNormalizeBodyIdempotent(t *testing.T) {
	in := "First paragraph all on one line.\n\nSecond paragraph also one line.\n\n```\ncode\n```\n\n- list\n- items\n"
	once := NormalizeBody(in)
	twice := NormalizeBody(once)
	if once != twice {
		t.Errorf("not idempotent:\nfirst:\n%q\nsecond:\n%q", once, twice)
	}
}

func TestNormalizeBodyEmpty(t *testing.T) {
	if got := NormalizeBody(""); got != "" {
		t.Errorf("empty -> %q", got)
	}
}

func TestNormalizeBodyCollapsesMultipleBlankLines(t *testing.T) {
	in := "para one\n\n\n\npara two\n"
	want := "para one\n\npara two\n"
	if got := NormalizeBody(in); got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestNormalizeBodyIndentedCode(t *testing.T) {
	in := "Above\nfolded.\n\n    code line 1\n    code line 2\n\nBelow\nfolded.\n"
	got := NormalizeBody(in)
	if !strings.Contains(got, "    code line 1\n    code line 2") {
		t.Errorf("indented code folded:\n%s", got)
	}
}

func TestNormalizeBodyFoldsListContinuations(t *testing.T) {
	in := "- first item\n  with continuation\n- second item\n  also wrapped\n  to three lines\n"
	want := "- first item with continuation\n- second item also wrapped to three lines\n"
	if got := NormalizeBody(in); got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestNormalizeBodyPreservesTables(t *testing.T) {
	in := "| col | val |\n|---|---|\n| a | 1 |\n| b | 2 |\n"
	got := NormalizeBody(in)
	if !strings.Contains(got, "| col | val |\n|---|---|\n| a | 1 |\n| b | 2 |") {
		t.Errorf("table got mangled:\n%s", got)
	}
}
