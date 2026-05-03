// Package board is the shared logic layer of focus.
//
// Both the CLI (internal/cli) and the MCP server (internal/mcp) are
// thin wrappers over this package. Card mutations, status transitions,
// next_id allocation, and index updates all live here so the two
// surfaces never drift — see designs/focus-issue-001.md §"CLI/MCP
// shared logic placement".
//
// Operations that mutate state acquire the .focus/.lock flock for the
// duration of the read-modify-write cycle. The Open() entry point
// returns a *Board which carries the .focus/ path for the resolved
// board; callers ask the Board to do things rather than wiring paths
// through every call.
package board

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// FocusDirName is the per-project board directory that focus walks
// for, the way git walks for ".git". See designs/focus-v2.md
// §"Project-local boards".
const FocusDirName = ".focus"

// CardsDirName is the per-board directory under .focus/ that holds
// each card folder. Always relative to .focus/.
const CardsDirName = "cards"

// CardFileName is the required markdown file inside a card directory.
// Optional artifacts (designs, screenshots, logs) sit alongside it.
const CardFileName = "INDEX.md"

// ConfigFileName is the per-board config file under .focus/. v0.1.0
// writes it as an empty placeholder; future features fill it in.
const ConfigFileName = "config.yaml"

// ErrNotInBoard is returned by Open when no .focus/ directory is
// found in $PWD or any ancestor. CLI prints a helpful message; MCP
// surfaces it as a tool error.
var ErrNotInBoard = errors.New("not in a focus board (no .focus/ found in this directory or any ancestor); run `focus init`")

// Board is a resolved focus board. Holds the absolute path to the
// .focus/ directory. Cheap to construct; safe to pass around.
type Board struct {
	// Root is the project root — the directory that contains .focus/.
	Root string
	// Dir is the absolute path to .focus/ itself.
	Dir string
}

// Open walks from startDir up the directory tree looking for a
// .focus/ directory. Returns ErrNotInBoard if none is found before
// hitting the filesystem root.
//
// startDir may be relative; we resolve to an absolute path first so
// the walk terminates predictably.
func Open(startDir string) (*Board, error) {
	abs, err := filepath.Abs(startDir)
	if err != nil {
		return nil, err
	}
	cur := abs
	for {
		candidate := filepath.Join(cur, FocusDirName)
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return &Board{Root: cur, Dir: candidate}, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return nil, ErrNotInBoard
		}
		cur = parent
	}
}

// Init creates a .focus/ directory at root with the bare-minimum
// layout: empty config.yaml + empty cards/ dir. Per
// designs/focus-issue-001.md §"`focus init` minimal state", we
// deliberately do NOT create index.json (first `focus new` writes it),
// .lock (created on demand), starter cards, or a README.
//
// Idempotent: running Init on an existing board is a no-op that
// returns the existing Board.
func Init(root string) (*Board, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(abs, FocusDirName)
	if err := os.MkdirAll(filepath.Join(dir, CardsDirName), 0o755); err != nil {
		return nil, fmt.Errorf("create %s: %w", dir, err)
	}
	cfg := filepath.Join(dir, ConfigFileName)
	if _, err := os.Stat(cfg); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(cfg, nil, 0o644); err != nil {
			return nil, fmt.Errorf("create %s: %w", cfg, err)
		}
	} else if err != nil {
		return nil, err
	}
	return &Board{Root: abs, Dir: dir}, nil
}

// CardsDir returns the absolute path to <board>/.focus/cards/.
func (b *Board) CardsDir() string {
	return filepath.Join(b.Dir, CardsDirName)
}

// CardDir returns the absolute path to a card's directory given its
// id and slug. The slug is folder-only signage and is taken verbatim;
// the caller is responsible for having normalized it via
// card.Slugify.
func (b *Board) CardDir(id int, slug string) string {
	return filepath.Join(b.CardsDir(), padDirName(id, slug))
}

// CardFile returns the absolute path to a card's INDEX.md given the
// already-known directory name. Used by handlers that have already
// looked up the dir from the index.
func (b *Board) CardFile(dirName string) string {
	return filepath.Join(b.CardsDir(), dirName, CardFileName)
}

// FindCardDir locates a card's directory on disk by id. The directory
// name is "<padded-id>-<slug>" but the slug is unknown to the caller
// (we don't store it in frontmatter), so we glob for the prefix.
//
// Returns the directory name (e.g. "0142-ship-the-feature") relative
// to the cards/ dir, suitable for passing to CardFile.
func (b *Board) FindCardDir(id int) (string, error) {
	pattern := filepath.Join(b.CardsDir(), fmt.Sprintf("%04d-*", id))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("card %d not found", id)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("card %d has %d matching directories: %v", id, len(matches), matches)
	}
	return filepath.Base(matches[0]), nil
}

// padDirName is the inverse of card.DirName but kept private to
// avoid an import cycle. The card package owns the public
// PaddedID/DirName helpers.
func padDirName(id int, slug string) string {
	return fmt.Sprintf("%04d-%s", id, slug)
}
