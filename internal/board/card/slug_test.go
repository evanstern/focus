package card

import (
	"errors"
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := []struct {
		title string
		want  string
	}{
		{"Ship the feature", "ship-the-feature"},
		{"Ship the feature!", "ship-the-feature"},
		{"  Spaces  around  ", "spaces-around"},
		{"MIXED Case Title", "mixed-case-title"},
		{"hyphen--collapse", "hyphen-collapse"},
		{"keeps123numbers", "keeps123numbers"},
		{"!!!only-punctuation-then-words", "only-punctuation-then-words"},
		{"underscores_become_hyphens", "underscores-become-hyphens"},
		{"slash/and:colon", "slash-and-colon"},
	}
	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			got, err := Slugify(tc.title)
			if err != nil {
				t.Fatalf("Slugify(%q) err = %v", tc.title, err)
			}
			if got != tc.want {
				t.Errorf("Slugify(%q) = %q, want %q", tc.title, got, tc.want)
			}
		})
	}
}

func TestSlugifyDropsNonASCII(t *testing.T) {
	got, err := Slugify("Café résumé naïve")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != "caf-rsum-nave" {
		t.Errorf("Slugify dropped accents but result was %q", got)
	}
}

func TestSlugifyEmptyResultErrors(t *testing.T) {
	cases := []string{"", "   ", "!!!", "🔥🚀💯", "✨ ✨ ✨"}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			_, err := Slugify(in)
			if !errors.Is(err, ErrEmptySlug) {
				t.Errorf("Slugify(%q) err = %v, want ErrEmptySlug", in, err)
			}
		})
	}
}

func TestSlugifyTruncatesAtHyphen(t *testing.T) {
	long := strings.Repeat("word-", 30) + "end"
	got, err := Slugify(long)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) > MaxSlugLen {
		t.Errorf("len(got) = %d, exceeds MaxSlugLen %d", len(got), MaxSlugLen)
	}
	if strings.HasSuffix(got, "-") {
		t.Errorf("trailing hyphen after truncation: %q", got)
	}
}

func TestSlugifyHardTruncatesUnhyphenated(t *testing.T) {
	long := strings.Repeat("a", MaxSlugLen+10)
	got, err := Slugify(long)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != MaxSlugLen {
		t.Errorf("len(got) = %d, want %d", len(got), MaxSlugLen)
	}
}
