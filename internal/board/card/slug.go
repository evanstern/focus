package card

import (
	"errors"
	"strings"
	"unicode"
)

// MaxSlugLen is the soft cap on slug length. Slugs longer than this
// after normalization get truncated at the nearest hyphen so we don't
// split words. 64 chars is plenty for a folder name and keeps `ls
// cards/` readable on an 80-column terminal.
const MaxSlugLen = 64

// ErrEmptySlug is returned by Slugify when the title produces an empty
// slug (e.g. the title is emoji-only or all whitespace). Callers
// should error out and ask for an explicit --slug; auto-deriving a
// fallback like "card-142" silently would surprise users.
var ErrEmptySlug = errors.New("title produced empty slug; pass --slug explicitly")

// Slugify converts a card title into the slug used in the card's
// directory name. The algorithm follows designs/focus-issue-001.md
// §"Slug rules at creation":
//
//  1. Lowercase.
//  2. ASCII-only (non-ASCII characters are dropped, not transliterated;
//     transliteration would require a heavy dep and the design doc
//     doesn't ask for it).
//  3. Replace runs of non-alphanumeric with single hyphens.
//  4. Trim leading + trailing hyphens.
//  5. Truncate to MaxSlugLen at a hyphen boundary if possible.
//  6. Empty result → ErrEmptySlug.
//
// Slugs are folder-only — they're not stored in frontmatter and never
// renamed after creation (see designs/focus-id-strategy.md §"Slug").
func Slugify(title string) (string, error) {
	var b strings.Builder
	b.Grow(len(title))

	prevHyphen := true // suppresses leading hyphens
	for _, r := range strings.ToLower(title) {
		switch {
		case r > unicode.MaxASCII:
			// Drop non-ASCII. We deliberately don't NFKD-normalize and
			// extract Latin equivalents; that's a transliteration
			// problem and out of scope for v2.
			continue
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevHyphen = false
		default:
			// Whitespace, punctuation, symbols — collapse to a single
			// hyphen.
			if !prevHyphen {
				b.WriteByte('-')
				prevHyphen = true
			}
		}
	}

	slug := strings.TrimRight(b.String(), "-")
	if slug == "" {
		return "", ErrEmptySlug
	}

	if len(slug) > MaxSlugLen {
		slug = truncateAtHyphen(slug, MaxSlugLen)
	}
	return slug, nil
}

// truncateAtHyphen trims s to at most maxLen runes, preferring to cut
// at the last hyphen at or before maxLen so we don't split a word in
// the middle. If the slug has no hyphens at all (one giant word), we
// hard-truncate.
func truncateAtHyphen(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	cut := s[:maxLen]
	if i := strings.LastIndex(cut, "-"); i > 0 {
		return cut[:i]
	}
	return cut
}
