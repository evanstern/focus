package board

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenWalksAncestors(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	b, err := Open(deep)
	if err != nil {
		t.Fatalf("Open from deep: %v", err)
	}
	if b.Root != root {
		t.Errorf("Root = %q, want %q", b.Root, root)
	}
}

func TestOpenReturnsErrNotInBoard(t *testing.T) {
	dir := t.TempDir()
	_, err := Open(dir)
	if !errors.Is(err, ErrNotInBoard) {
		t.Errorf("err = %v, want ErrNotInBoard", err)
	}
}

func TestInitIdempotent(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root); err != nil {
		t.Fatalf("first Init: %v", err)
	}
	if _, err := Init(root); err != nil {
		t.Errorf("second Init should be a no-op: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".focus", "config.yaml")); err != nil {
		t.Errorf("config.yaml missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".focus", "cards")); err != nil {
		t.Errorf("cards/ missing: %v", err)
	}
	// Per designs/focus-issue-001.md §"`focus init` minimal state",
	// init must NOT create index.json. It's the first `focus new`'s
	// job to write that.
	if _, err := os.Stat(filepath.Join(root, ".focus", "index.json")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("index.json should not exist after init, got err=%v", err)
	}
}

func TestInitDoesNotOverwriteConfig(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	cfgPath := filepath.Join(root, ".focus", "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("wip_limit: 5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(root); err != nil {
		t.Fatalf("re-Init: %v", err)
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "wip_limit: 5\n" {
		t.Errorf("config clobbered by re-Init: %q", string(data))
	}
}
