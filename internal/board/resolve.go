package board

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// FocusDirNotFoundError is returned by Resolve when an explicitly
// provided path (via flag or env) does not contain a .focus/
// directory. The error string matches the spec exactly:
//
//	focus: no .focus/ found at <path>
//
// (no trailing period). Callers print it via %v.
type FocusDirNotFoundError struct {
	Path string
}

func (e *FocusDirNotFoundError) Error() string {
	return fmt.Sprintf("focus: no .focus/ found at %s", e.Path)
}

// Resolve picks the .focus/ directory to operate on, with three-tier
// precedence: flag > env > upward walk from $PWD. The returned path
// is the absolute path to the .focus/ directory itself.
//
//   - flagValue: value of the --focus-dir flag (empty if unset).
//   - envValue:  value of the FOCUS_DIR env var (empty if unset).
//
// For the explicit tiers (flag/env) the path is accepted in two
// forms: pointing directly at a .focus/ directory, or pointing at a
// project root that contains one. If neither holds, Resolve returns
// a *FocusDirNotFoundError naming the path the user supplied so the
// error message stays grounded in their input.
//
// When neither flagValue nor envValue is set, Resolve falls back to
// the original Open() walk semantics and returns ErrNotInBoard if no
// ancestor of $PWD has a .focus/.
func Resolve(flagValue, envValue string) (string, error) {
	if flagValue != "" {
		return resolveExplicit(flagValue)
	}
	if envValue != "" {
		return resolveExplicit(envValue)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	b, err := Open(cwd)
	if err != nil {
		return "", err
	}
	return b.Dir, nil
}

// resolveExplicit handles a user-supplied path (flag or env). The
// path is accepted as either a project root containing .focus/ or
// the .focus/ directory itself. We prefer the project-root form for
// error messaging so the user sees the path they typed.
func resolveExplicit(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	candidate := filepath.Join(abs, FocusDirName)
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return candidate, nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if info, err := os.Stat(abs); err == nil && info.IsDir() && filepath.Base(abs) == FocusDirName {
		return abs, nil
	}
	return "", &FocusDirNotFoundError{Path: path}
}

// OpenAt returns a *Board rooted at the given absolute .focus/
// directory. Useful when callers have already resolved the focus dir
// via Resolve and want the same Board shape Open() returns.
func OpenAt(focusDir string) (*Board, error) {
	abs, err := filepath.Abs(focusDir)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", abs)
	}
	return &Board{Root: filepath.Dir(abs), Dir: abs}, nil
}
