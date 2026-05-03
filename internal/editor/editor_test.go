package editor

import (
	"reflect"
	"testing"
)

func TestCommand(t *testing.T) {
	cases := []struct {
		name    string
		envEDIT string
		path    string
		wantBin string
		wantArg []string
	}{
		{"unset uses default", "", "/tmp/x", "vi", []string{"/tmp/x"}},
		{"plain editor", "nano", "/tmp/x", "nano", []string{"/tmp/x"}},
		{"editor with flag", "code -w", "/tmp/x", "code", []string{"-w", "/tmp/x"}},
		{"editor with multiple flags", "vim -u NONE -p", "/tmp/x", "vim", []string{"-u", "NONE", "-p", "/tmp/x"}},
		{"trims surrounding whitespace", "  emacs  ", "/tmp/x", "emacs", []string{"/tmp/x"}},
		{"path with spaces preserved as one arg", "vim", "/tmp/has space/x", "vim", []string{"/tmp/has space/x"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("EDITOR", tc.envEDIT)
			cmd, err := Command(tc.path)
			if err != nil {
				t.Fatalf("Command err: %v", err)
			}
			gotBin := cmd.Path
			if base := lastSegment(gotBin); base != tc.wantBin {
				t.Errorf("binary = %q, want %q (full path %q)", base, tc.wantBin, gotBin)
			}
			if !reflect.DeepEqual(cmd.Args[1:], tc.wantArg) {
				t.Errorf("args = %v, want %v", cmd.Args[1:], tc.wantArg)
			}
		})
	}
}

func TestCommandEmptyEditorErrors(t *testing.T) {
	t.Setenv("EDITOR", "    ")
	cmd, err := Command("/tmp/x")
	if err != nil {
		t.Fatalf("blank EDITOR should fall back to default, got err: %v", err)
	}
	if cmd == nil {
		t.Fatal("got nil cmd")
	}
}

func lastSegment(s string) string {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return s[i+1:]
		}
	}
	return s
}
