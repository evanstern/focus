// Package lock wraps gofrs/flock to provide an advisory file lock on
// .focus/.lock for the duration of a mutating CLI/MCP operation.
//
// Per designs/focus-issue-001.md §"Concurrency", every mutating
// command acquires this lock to serialize next_id allocation and
// index.json writes across concurrent invocations. Read-only commands
// do NOT take the lock — atomic-rename on the writer side guarantees
// they see a consistent snapshot.
//
// Stale lock files are not a concern: flock is kernel-managed, so a
// dying process releases its lock when its file descriptor closes.
// The .lock file itself stays on disk; that's expected and harmless.
package lock

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

// FileName is the lock file's name relative to .focus/.
const FileName = ".lock"

// Lock is a held lock on .focus/.lock. Always Release() what you
// Acquire(). Use With() if you can; it's harder to forget.
type Lock struct {
	fl *flock.Flock
}

// Acquire takes an exclusive lock on .focus/.lock, blocking until the
// lock is available. Creates the .lock file if it doesn't exist.
func Acquire(focusDir string) (*Lock, error) {
	if err := os.MkdirAll(focusDir, 0o755); err != nil {
		return nil, fmt.Errorf("ensure %s: %w", focusDir, err)
	}
	path := filepath.Join(focusDir, FileName)
	fl := flock.New(path)
	if err := fl.Lock(); err != nil {
		return nil, fmt.Errorf("acquire %s: %w", path, err)
	}
	return &Lock{fl: fl}, nil
}

// Release releases the lock and closes the underlying file
// descriptor. Safe to call multiple times — the second call is a
// no-op.
func (l *Lock) Release() error {
	if l == nil || l.fl == nil {
		return nil
	}
	err := l.fl.Unlock()
	l.fl = nil
	return err
}

// With acquires the .focus/.lock for the duration of fn and releases
// it on return, regardless of whether fn errors or panics. The
// preferred API for mutating commands.
func With(focusDir string, fn func() error) (err error) {
	l, err := Acquire(focusDir)
	if err != nil {
		return err
	}
	defer func() {
		if relErr := l.Release(); relErr != nil && err == nil {
			err = relErr
		}
	}()
	return fn()
}
