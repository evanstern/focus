package completions

import (
	"bytes"
	"strings"
	"testing"
)

func TestEmbeddedScriptsNonEmpty(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish"} {
		s, ok := Script(shell)
		if !ok {
			t.Errorf("Script(%q) ok=false", shell)
			continue
		}
		if strings.TrimSpace(s) == "" {
			t.Errorf("Script(%q) is empty", shell)
		}
	}
}

func TestScriptUnknown(t *testing.T) {
	if _, ok := Script("nushell"); ok {
		t.Error("Script(nushell) ok=true, want false")
	}
}

func TestPrintPriorities(t *testing.T) {
	var buf bytes.Buffer
	PrintPriorities(&buf)
	want := "p0\np1\np2\np3\n"
	if buf.String() != want {
		t.Errorf("PrintPriorities = %q, want %q", buf.String(), want)
	}
}

func TestPrintStatuses(t *testing.T) {
	var buf bytes.Buffer
	PrintStatuses(&buf)
	want := "active\nbacklog\ndone\narchived\n"
	if buf.String() != want {
		t.Errorf("PrintStatuses = %q, want %q", buf.String(), want)
	}
}

func TestPrintTypes(t *testing.T) {
	var buf bytes.Buffer
	PrintTypes(&buf)
	want := "card\nepic\n"
	if buf.String() != want {
		t.Errorf("PrintTypes = %q, want %q", buf.String(), want)
	}
}

func TestPrintSubcommands(t *testing.T) {
	var buf bytes.Buffer
	PrintSubcommands(&buf)
	out := buf.String()
	for _, want := range []string{"init", "new", "show", "edit", "board", "list", "activate", "park", "done", "kill", "revive", "completions"} {
		if !strings.Contains(out, want+"\n") {
			t.Errorf("PrintSubcommands missing %q", want)
		}
	}
	if strings.Contains(out, "_complete") {
		t.Error("PrintSubcommands leaked the hidden _complete subcommand")
	}
}
