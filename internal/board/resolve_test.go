package board

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveFlagPrecedence(t *testing.T) {
	flagRoot := t.TempDir()
	envRoot := t.TempDir()
	if _, err := Init(flagRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(envRoot); err != nil {
		t.Fatal(err)
	}

	got, err := Resolve(flagRoot, envRoot)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := filepath.Join(flagRoot, FocusDirName)
	if got != want {
		t.Errorf("flag did not win: got %q, want %q", got, want)
	}
}

func TestResolveEnvWhenNoFlag(t *testing.T) {
	envRoot := t.TempDir()
	if _, err := Init(envRoot); err != nil {
		t.Fatal(err)
	}
	got, err := Resolve("", envRoot)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := filepath.Join(envRoot, FocusDirName)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveWalksWhenNeitherSet(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}
	got, err := Resolve("", "")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := filepath.Join(root, FocusDirName)
	if !sameFile(t, got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveAcceptsFocusDirDirectly(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root); err != nil {
		t.Fatal(err)
	}
	focusDir := filepath.Join(root, FocusDirName)
	got, err := Resolve(focusDir, "")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != focusDir {
		t.Errorf("got %q, want %q", got, focusDir)
	}
}

func TestResolveAcceptsProjectRoot(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root); err != nil {
		t.Fatal(err)
	}
	got, err := Resolve(root, "")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := filepath.Join(root, FocusDirName)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveErrorMessageFormat(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	_, err := Resolve(missing, "")
	if err == nil {
		t.Fatal("expected error")
	}
	want := "focus: no .focus/ found at " + missing
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
	var fnf *FocusDirNotFoundError
	if !errors.As(err, &fnf) {
		t.Errorf("err is not *FocusDirNotFoundError: %T", err)
	}
	if strings.HasSuffix(err.Error(), ".") {
		t.Errorf("error must not end with a period: %q", err.Error())
	}
}

func TestResolveEnvErrorMessageUsesEnvPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "envnope")
	_, err := Resolve("", missing)
	if err == nil {
		t.Fatal("expected error")
	}
	want := "focus: no .focus/ found at " + missing
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestResolveWalkReturnsErrNotInBoard(t *testing.T) {
	root := t.TempDir()
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	_, err := Resolve("", "")
	if !errors.Is(err, ErrNotInBoard) {
		t.Errorf("got %v, want ErrNotInBoard", err)
	}
}

func TestOpenAtReturnsBoard(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root); err != nil {
		t.Fatal(err)
	}
	focusDir := filepath.Join(root, FocusDirName)
	b, err := OpenAt(focusDir)
	if err != nil {
		t.Fatalf("OpenAt: %v", err)
	}
	if b.Dir != focusDir {
		t.Errorf("Dir = %q, want %q", b.Dir, focusDir)
	}
	if b.Root != root {
		t.Errorf("Root = %q, want %q", b.Root, root)
	}
}

func sameFile(t *testing.T, a, b string) bool {
	t.Helper()
	ra, err := filepath.EvalSymlinks(a)
	if err != nil {
		ra = a
	}
	rb, err := filepath.EvalSymlinks(b)
	if err != nil {
		rb = b
	}
	return ra == rb
}
