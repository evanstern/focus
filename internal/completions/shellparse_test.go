package completions

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestShellsParseEmbeddedScripts(t *testing.T) {
	cases := []struct {
		shell   string
		flag    string
		ext     string
		content string
	}{
		{"bash", "-n", "sh", bashScript},
		{"zsh", "-n", "zsh", zshScript},
		{"fish", "-n", "fish", fishScript},
	}
	for _, c := range cases {
		t.Run(c.shell, func(t *testing.T) {
			path, err := exec.LookPath(c.shell)
			if err != nil {
				t.Skipf("%s not on PATH", c.shell)
			}
			tmp := filepath.Join(t.TempDir(), "focus."+c.ext)
			if err := os.WriteFile(tmp, []byte(c.content), 0o644); err != nil {
				t.Fatal(err)
			}
			out, err := exec.Command(path, c.flag, tmp).CombinedOutput()
			if err != nil {
				t.Errorf("%s -n: %v\n%s", c.shell, err, out)
			}
		})
	}
}
