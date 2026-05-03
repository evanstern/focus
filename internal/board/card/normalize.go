package card

import "strings"

// NormalizeBody rewrites a card body so paragraphs flow as single
// long lines. Authored line breaks within a paragraph become spaces;
// blank lines stay; structural markdown (code fences, lists,
// headings, blockquotes, indented code, horizontal rules, HTML-ish
// blocks) is preserved verbatim.
//
// The function is idempotent: running it on already-normalized
// content is a no-op.
func NormalizeBody(body string) string {
	if body == "" {
		return ""
	}

	lines := strings.Split(body, "\n")
	out := make([]string, 0, len(lines))

	inFence := false
	fenceMarker := ""

	flushPara := func(para []string) []string {
		if len(para) == 0 {
			return nil
		}
		var b strings.Builder
		for i, line := range para {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(strings.TrimRight(line, " \t"))
		}
		out = append(out, b.String())
		return nil
	}

	var para []string
	prevWasListItem := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if marker, ok := fenceOpener(trimmed); ok && !inFence {
			para = flushPara(para)
			inFence = true
			fenceMarker = marker
			prevWasListItem = false
			out = append(out, line)
			continue
		}
		if inFence {
			out = append(out, line)
			if isFenceCloser(trimmed, fenceMarker) {
				inFence = false
				fenceMarker = ""
			}
			continue
		}

		if trimmed == "" {
			para = flushPara(para)
			prevWasListItem = false
			out = append(out, "")
			continue
		}

		// Continuation of a list item: leading whitespace 1-3 spaces
		// and a non-empty trimmed body. Fold into the previous output
		// line so "- item\n  more" becomes "- item more".
		if prevWasListItem && isListContinuation(line) {
			out[len(out)-1] = strings.TrimRight(out[len(out)-1], " \t") + " " + trimmed
			continue
		}

		if isStructural(line, trimmed) {
			para = flushPara(para)
			out = append(out, line)
			prevWasListItem = isListItem(trimmed)
			continue
		}

		para = append(para, line)
		prevWasListItem = false
	}
	_ = flushPara(para)

	result := strings.Join(out, "\n")
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}
	return result
}

func fenceOpener(trimmed string) (string, bool) {
	if strings.HasPrefix(trimmed, "```") {
		return "```", true
	}
	if strings.HasPrefix(trimmed, "~~~") {
		return "~~~", true
	}
	return "", false
}

func isFenceCloser(trimmed, marker string) bool {
	return strings.HasPrefix(trimmed, marker)
}

// isStructural reports whether the line should be kept on its own
// rather than folded into a paragraph. Covers headings, lists,
// blockquotes, horizontal rules, indented code, table rows, and
// HTML-ish lines.
func isStructural(line, trimmed string) bool {
	if trimmed == "" {
		return false
	}
	if strings.HasPrefix(line, "\t") {
		return true
	}
	if strings.HasPrefix(line, "    ") {
		return true
	}
	if strings.HasPrefix(trimmed, "#") {
		return true
	}
	if strings.HasPrefix(trimmed, ">") {
		return true
	}
	if strings.HasPrefix(trimmed, "<") {
		return true
	}
	if strings.HasPrefix(trimmed, "|") {
		return true
	}
	if isHorizontalRule(trimmed) {
		return true
	}
	if isListItem(trimmed) {
		return true
	}
	return false
}

func isHorizontalRule(s string) bool {
	if len(s) < 3 {
		return false
	}
	c := s[0]
	if c != '-' && c != '*' && c != '_' {
		return false
	}
	count := 0
	for i := 0; i < len(s); i++ {
		if s[i] != c && s[i] != ' ' {
			return false
		}
		if s[i] == c {
			count++
		}
	}
	return count >= 3
}

// isListItem reports whether trimmed begins with a bullet or
// ordered-list marker. Used to flag the previous line so the next
// indented line can be folded into it.
func isListItem(trimmed string) bool {
	if len(trimmed) >= 2 {
		switch trimmed[0] {
		case '-', '*', '+':
			if trimmed[1] == ' ' || trimmed[1] == '\t' {
				return true
			}
		}
	}
	return isOrderedListItem(trimmed)
}

// isListContinuation reports whether line is an indented continuation
// of a list item: 1-3 leading spaces, then non-empty content. Tabs
// and 4+ spaces are indented code, not continuations.
func isListContinuation(line string) bool {
	if line == "" {
		return false
	}
	spaces := 0
	for spaces < len(line) && line[spaces] == ' ' {
		spaces++
	}
	if spaces == 0 || spaces >= 4 {
		return false
	}
	return spaces < len(line)
}

func isOrderedListItem(s string) bool {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 || i >= len(s) {
		return false
	}
	if s[i] != '.' && s[i] != ')' {
		return false
	}
	i++
	if i >= len(s) {
		return false
	}
	return s[i] == ' ' || s[i] == '\t'
}
