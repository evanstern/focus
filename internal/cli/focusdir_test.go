package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runArgs(t *testing.T, wd string, env map[string]string, args ...string) (int, string, string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(wd); err != nil {
		t.Fatalf("chdir %s: %v", wd, err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	for k, v := range env {
		t.Setenv(k, v)
	}

	var out, errb bytes.Buffer
	code := Run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestFocusDirFlagPointsAtBoard(t *testing.T) {
	board := t.TempDir()
	other := t.TempDir()
	if code, _, errb := runArgs(t, board, nil, "init"); code != 0 {
		t.Fatalf("init: %d %s", code, errb)
	}
	if code, _, errb := runArgs(t, board, nil, "--focus-dir", board, "new", "from-flag"); code != 0 {
		t.Fatalf("new: %d %s", code, errb)
	}
	code, out, errb := runArgs(t, other, nil, "--focus-dir", board, "list")
	if code != 0 {
		t.Fatalf("list from %s: code=%d stderr=%s", other, code, errb)
	}
	if !strings.Contains(out, "from-flag") {
		t.Errorf("--focus-dir did not target the right board: out=%q", out)
	}
}

func TestFocusDirFlagWithEqualsSyntax(t *testing.T) {
	board := t.TempDir()
	other := t.TempDir()
	if code, _, _ := runArgs(t, board, nil, "init"); code != 0 {
		t.Fatal("init")
	}
	if code, _, _ := runArgs(t, board, nil, "--focus-dir="+board, "new", "eq-syntax"); code != 0 {
		t.Fatal("new")
	}
	code, out, _ := runArgs(t, other, nil, "--focus-dir="+board, "list")
	if code != 0 || !strings.Contains(out, "eq-syntax") {
		t.Errorf("equals syntax broken: code=%d out=%q", code, out)
	}
}

func TestFocusDirEnvVar(t *testing.T) {
	board := t.TempDir()
	other := t.TempDir()
	if code, _, _ := runArgs(t, board, nil, "init"); code != 0 {
		t.Fatal("init")
	}
	if code, _, _ := runArgs(t, board, map[string]string{"FOCUS_DIR": board}, "new", "from-env"); code != 0 {
		t.Fatal("new")
	}
	code, out, errb := runArgs(t, other, map[string]string{"FOCUS_DIR": board}, "list")
	if code != 0 {
		t.Fatalf("list: code=%d errb=%s", code, errb)
	}
	if !strings.Contains(out, "from-env") {
		t.Errorf("FOCUS_DIR did not target the right board: out=%q", out)
	}
}

func TestFlagBeatsEnv(t *testing.T) {
	flagBoard := t.TempDir()
	envBoard := t.TempDir()
	other := t.TempDir()
	for _, b := range []string{flagBoard, envBoard} {
		if code, _, _ := runArgs(t, b, nil, "init"); code != 0 {
			t.Fatalf("init %s", b)
		}
	}
	if code, _, _ := runArgs(t, flagBoard, nil, "new", "flag-card"); code != 0 {
		t.Fatal("new flag-card")
	}
	if code, _, _ := runArgs(t, envBoard, nil, "new", "env-card"); code != 0 {
		t.Fatal("new env-card")
	}
	code, out, _ := runArgs(t, other, map[string]string{"FOCUS_DIR": envBoard},
		"--focus-dir", flagBoard, "list")
	if code != 0 {
		t.Fatalf("list code=%d", code)
	}
	if !strings.Contains(out, "flag-card") {
		t.Errorf("flag did not win: out=%q", out)
	}
	if strings.Contains(out, "env-card") {
		t.Errorf("env board leaked: out=%q", out)
	}
}

func TestFocusDirAcceptsFocusDirDirectly(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	if code, _, _ := runArgs(t, root, nil, "init"); code != 0 {
		t.Fatal("init")
	}
	if code, _, _ := runArgs(t, root, nil, "new", "card-direct"); code != 0 {
		t.Fatal("new")
	}
	focusDir := filepath.Join(root, ".focus")
	code, out, _ := runArgs(t, other, nil, "--focus-dir", focusDir, "list")
	if code != 0 || !strings.Contains(out, "card-direct") {
		t.Errorf("--focus-dir <.focus> failed: code=%d out=%q", code, out)
	}
}

func TestFocusDirErrorMessage(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	cwd := t.TempDir()
	code, _, errb := runArgs(t, cwd, nil, "--focus-dir", missing, "board")
	if code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	want := "focus: no .focus/ found at " + missing + "\n"
	if errb != want {
		t.Errorf("stderr = %q, want %q", errb, want)
	}
}

func TestFocusDirInitTargetsFlagPath(t *testing.T) {
	cwd := t.TempDir()
	target := t.TempDir()
	if code, _, errb := runArgs(t, cwd, nil, "--focus-dir", target, "init"); code != 0 {
		t.Fatalf("init: code=%d errb=%s", code, errb)
	}
	if _, err := os.Stat(filepath.Join(target, ".focus", "config.yaml")); err != nil {
		t.Errorf("init did not create .focus at flag path: %v", err)
	}
}

func TestFocusDirInitDoesNotNestWhenPathEndsInFocusDir(t *testing.T) {
	cwd := t.TempDir()
	target := t.TempDir()
	focusDir := filepath.Join(target, ".focus")
	if code, _, errb := runArgs(t, cwd, nil, "--focus-dir", focusDir, "init"); code != 0 {
		t.Fatalf("init: code=%d errb=%s", code, errb)
	}
	if _, err := os.Stat(filepath.Join(target, ".focus", "config.yaml")); err != nil {
		t.Errorf("init did not create .focus at the parent of the supplied .focus path: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, ".focus", ".focus")); err == nil {
		t.Errorf("init created nested .focus/.focus when given a path ending in .focus")
	}
}

func TestExtractFocusDirHonorsDoubleDashTerminator(t *testing.T) {
	focusDirFlag = ""
	got, err := extractFocusDir([]string{"new", "--", "--focus-dir", "literal"})
	if err != nil {
		t.Fatalf("extractFocusDir: %v", err)
	}
	want := []string{"new", "--", "--focus-dir", "literal"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if focusDirFlag != "" {
		t.Errorf("focusDirFlag set after --: %q", focusDirFlag)
	}
}
