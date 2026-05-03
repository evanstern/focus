package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runIn pushes wd into os.Getwd's view via os.Chdir, runs the CLI,
// and restores the previous wd. We bake this into a helper because
// every CLI test wants to operate inside a tempdir board.
func runIn(t *testing.T, wd string, args ...string) (int, string, string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(wd); err != nil {
		t.Fatalf("chdir %s: %v", wd, err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	var out, errb bytes.Buffer
	code := Run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestVersion(t *testing.T) {
	dir := t.TempDir()
	code, out, _ := runIn(t, dir, "version")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.TrimSpace(out) != "dev" {
		t.Errorf("out = %q, want %q", out, "dev")
	}
}

func TestUnknownCommandExit2(t *testing.T) {
	dir := t.TempDir()
	code, _, errb := runIn(t, dir, "wat")
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if !strings.Contains(errb, "unknown command") {
		t.Errorf("stderr = %q", errb)
	}
}

func TestNotInBoardErrorMessage(t *testing.T) {
	dir := t.TempDir()
	code, _, errb := runIn(t, dir, "board")
	if code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if !strings.Contains(errb, "not in a focus board") {
		t.Errorf("stderr = %q", errb)
	}
}

func TestInitNewBoardListShowFlow(t *testing.T) {
	root := t.TempDir()

	if code, _, errb := runIn(t, root, "init"); code != 0 {
		t.Fatalf("init: exit=%d stderr=%q", code, errb)
	}
	if _, err := os.Stat(filepath.Join(root, ".focus", "config.yaml")); err != nil {
		t.Errorf("init didn't create config.yaml: %v", err)
	}

	code, out, errb := runIn(t, root, "new", "Wire", "up", "auth")
	if code != 0 {
		t.Fatalf("new: exit=%d stderr=%q", code, errb)
	}
	if strings.TrimSpace(out) != "0001" {
		t.Errorf("new stdout = %q, want 0001", out)
	}
	if !strings.Contains(errb, "0001-wire-up-auth") {
		t.Errorf("new stderr = %q", errb)
	}

	code, out, _ = runIn(t, root, "board")
	if code != 0 {
		t.Fatalf("board: exit %d", code)
	}
	if !strings.Contains(out, "Wire up auth") {
		t.Errorf("board out = %q", out)
	}
	if !strings.Contains(out, "ACTIVE (0/3)") {
		t.Errorf("default WIP not 3: %q", out)
	}

	code, out, _ = runIn(t, root, "list")
	if code != 0 || !strings.Contains(out, "Wire up auth") {
		t.Errorf("list: exit=%d out=%q", code, out)
	}

	code, out, _ = runIn(t, root, "show", "1")
	if code != 0 {
		t.Fatalf("show: exit %d", code)
	}
	if !strings.Contains(out, "Wire up auth") {
		t.Errorf("show out = %q", out)
	}
	if !strings.Contains(out, "uuid:") {
		t.Errorf("show didn't render uuid: %q", out)
	}
}

func TestTransitionFlow(t *testing.T) {
	root := t.TempDir()
	if code, _, _ := runIn(t, root, "init"); code != 0 {
		t.Fatal("init")
	}
	if code, _, _ := runIn(t, root, "new", "task"); code != 0 {
		t.Fatal("new")
	}

	code, out, _ := runIn(t, root, "activate", "1")
	if code != 0 {
		t.Fatalf("activate exit %d out=%q", code, out)
	}
	if !strings.Contains(out, "active") {
		t.Errorf("activate out = %q", out)
	}

	code, _, _ = runIn(t, root, "park", "1")
	if code != 0 {
		t.Errorf("park exit %d", code)
	}

	if code, _, _ := runIn(t, root, "activate", "1"); code != 0 {
		t.Fatal("re-activate")
	}
	code, _, _ = runIn(t, root, "done", "1")
	if code != 0 {
		t.Errorf("done exit %d", code)
	}

	code, _, _ = runIn(t, root, "kill", "1")
	if code != 0 {
		t.Errorf("kill exit %d", code)
	}

	code, _, _ = runIn(t, root, "revive", "1")
	if code != 0 {
		t.Errorf("revive exit %d", code)
	}
}

func TestWIPLimitBlocksThirdActivate(t *testing.T) {
	root := t.TempDir()
	if code, _, _ := runIn(t, root, "init"); code != 0 {
		t.Fatal("init")
	}
	cfg := filepath.Join(root, ".focus", "config.yaml")
	if err := os.WriteFile(cfg, []byte("wip_limit: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if code, _, _ := runIn(t, root, "new", "t"); code != 0 {
			t.Fatalf("new %d", i)
		}
	}
	if code, _, _ := runIn(t, root, "activate", "1"); code != 0 {
		t.Fatal("first activate")
	}
	code, _, errb := runIn(t, root, "activate", "2")
	if code != 1 {
		t.Errorf("second activate exit = %d, want 1", code)
	}
	if !strings.Contains(errb, "WIP limit") {
		t.Errorf("WIP error not mentioned: %q", errb)
	}
}

func TestReindexWorks(t *testing.T) {
	root := t.TempDir()
	if code, _, _ := runIn(t, root, "init"); code != 0 {
		t.Fatal("init")
	}
	if code, _, _ := runIn(t, root, "new", "t"); code != 0 {
		t.Fatal("new")
	}
	if err := os.Remove(filepath.Join(root, ".focus", "index.json")); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runIn(t, root, "reindex")
	if code != 0 {
		t.Fatalf("reindex exit %d", code)
	}
	if !strings.Contains(out, "1 cards") {
		t.Errorf("reindex out = %q", out)
	}
}

func TestEpicSubcommands(t *testing.T) {
	root := t.TempDir()
	if code, _, _ := runIn(t, root, "init"); code != 0 {
		t.Fatal("init")
	}
	if code, _, _ := runIn(t, root, "new", "Launch v2", "--type", "epic"); code != 0 {
		t.Fatal("new epic")
	}
	if code, _, _ := runIn(t, root, "new", "Ship feature"); code != 0 {
		t.Fatal("new card")
	}
	if code, _, _ := runIn(t, root, "epic", "add", "1", "2"); code != 0 {
		t.Fatal("epic add")
	}
	code, out, _ := runIn(t, root, "epic", "1")
	if code != 0 {
		t.Fatalf("epic show exit %d", code)
	}
	if !strings.Contains(out, "0/1 done") {
		t.Errorf("epic show out = %q", out)
	}
	code, out, _ = runIn(t, root, "epic", "list")
	if code != 0 {
		t.Errorf("epic list exit %d", code)
	}
	if !strings.Contains(out, "Launch v2") {
		t.Errorf("epic list out = %q", out)
	}
}

func TestEditNonTTYReturnsPath(t *testing.T) {
	// stdin/stdout in `go test` are pipes, not TTYs, so runEdit's
	// non-tty branch fires. The test asserts we get the card path
	// back — same affordance as v1's PR #5.
	root := t.TempDir()
	if code, _, _ := runIn(t, root, "init"); code != 0 {
		t.Fatal("init")
	}
	if code, _, _ := runIn(t, root, "new", "card"); code != 0 {
		t.Fatal("new")
	}
	code, out, _ := runIn(t, root, "edit", "1")
	if code != 0 {
		t.Errorf("edit exit %d", code)
	}
	if !strings.Contains(out, "INDEX.md") {
		t.Errorf("edit out = %q", out)
	}
}
