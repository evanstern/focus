package cli

import (
	"strings"
	"testing"
)

func TestCompletionsBashZshFish(t *testing.T) {
	dir := t.TempDir()
	for _, shell := range []string{"bash", "zsh", "fish"} {
		code, out, errb := runIn(t, dir, "completions", shell)
		if code != 0 {
			t.Errorf("completions %s: exit=%d stderr=%q", shell, code, errb)
		}
		if strings.TrimSpace(out) == "" {
			t.Errorf("completions %s: empty stdout", shell)
		}
	}
}

func TestCompletionsUnknownShell(t *testing.T) {
	dir := t.TempDir()
	code, _, errb := runIn(t, dir, "completions", "nushell")
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if !strings.Contains(errb, "bash") || !strings.Contains(errb, "zsh") || !strings.Contains(errb, "fish") {
		t.Errorf("stderr should hint at supported shells: %q", errb)
	}
}

func TestCompletionsMissingArg(t *testing.T) {
	dir := t.TempDir()
	code, _, _ := runIn(t, dir, "completions")
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
}

func TestCompletionsExtraArgs(t *testing.T) {
	dir := t.TempDir()
	code, _, _ := runIn(t, dir, "completions", "bash", "extra")
	if code != 2 {
		t.Errorf("exit = %d, want 2 (extra args should be rejected)", code)
	}
}

func TestCompletePriorities(t *testing.T) {
	dir := t.TempDir()
	code, out, _ := runIn(t, dir, "_complete", "priorities")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if out != "p0\np1\np2\np3\n" {
		t.Errorf("out = %q", out)
	}
}

func TestCompleteStatuses(t *testing.T) {
	dir := t.TempDir()
	code, out, _ := runIn(t, dir, "_complete", "statuses")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if out != "active\nbacklog\ndone\narchived\n" {
		t.Errorf("out = %q", out)
	}
}

func TestCompleteTypes(t *testing.T) {
	dir := t.TempDir()
	code, out, _ := runIn(t, dir, "_complete", "types")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if out != "card\nepic\n" {
		t.Errorf("out = %q", out)
	}
}

func TestCompleteSubcommandsHidesUnderscoreCommand(t *testing.T) {
	dir := t.TempDir()
	code, out, _ := runIn(t, dir, "_complete", "subcommands")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.Contains(out, "_complete") {
		t.Errorf("subcommands list leaked hidden _complete: %q", out)
	}
	for _, want := range []string{"init", "completions", "board"} {
		if !strings.Contains(out, want+"\n") {
			t.Errorf("subcommands missing %q: %q", want, out)
		}
	}
}

func TestCompleteIDs(t *testing.T) {
	root := t.TempDir()
	if code, _, errb := runIn(t, root, "init"); code != 0 {
		t.Fatalf("init: %d %s", code, errb)
	}
	if code, _, errb := runIn(t, root, "new", "first card"); code != 0 {
		t.Fatalf("new 1: %d %s", code, errb)
	}
	if code, _, errb := runIn(t, root, "new", "second card"); code != 0 {
		t.Fatalf("new 2: %d %s", code, errb)
	}
	if code, _, errb := runIn(t, root, "new", "third card"); code != 0 {
		t.Fatalf("new 3: %d %s", code, errb)
	}

	code, out, _ := runIn(t, root, "_complete", "ids")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if out != "1\n2\n3\n" {
		t.Errorf("ids = %q, want %q", out, "1\n2\n3\n")
	}
}

func TestCompleteIDsFilterStatus(t *testing.T) {
	root := t.TempDir()
	if code, _, errb := runIn(t, root, "init"); code != 0 {
		t.Fatalf("init: %d %s", code, errb)
	}
	if code, _, errb := runIn(t, root, "new", "alpha"); code != 0 {
		t.Fatalf("new 1: %d %s", code, errb)
	}
	if code, _, errb := runIn(t, root, "new", "beta"); code != 0 {
		t.Fatalf("new 2: %d %s", code, errb)
	}
	if code, _, errb := runIn(t, root, "activate", "1"); code != 0 {
		t.Fatalf("activate: %d %s", code, errb)
	}

	code, out, _ := runIn(t, root, "_complete", "ids", "--status", "active")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if out != "1\n" {
		t.Errorf("active ids = %q, want %q", out, "1\n")
	}

	code, out, _ = runIn(t, root, "_complete", "ids", "--status", "backlog")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if out != "2\n" {
		t.Errorf("backlog ids = %q, want %q", out, "2\n")
	}
}

func TestCompleteIDsOutsideBoardIsSilent(t *testing.T) {
	dir := t.TempDir()
	code, out, errb := runIn(t, dir, "_complete", "ids")
	if code != 0 {
		t.Errorf("exit = %d, want 0 (silent outside board)", code)
	}
	if out != "" {
		t.Errorf("out = %q, want empty", out)
	}
	if errb != "" {
		t.Errorf("stderr = %q, want empty", errb)
	}
}
