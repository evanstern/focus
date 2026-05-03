package board

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds per-board configuration loaded from .focus/config.yaml.
// Empty file → zero-value Config which the rest of the package reads
// as "use defaults". v0.1.0 only supports wip_limit; future fields
// (default project, theme override, etc.) get added here.
type Config struct {
	WIPLimit int `yaml:"wip_limit"`
}

// DefaultWIPLimit is the WIP cap applied when no config override is
// present. 3 is the v1 default and matches the kanban literature on
// solo developers; experimentally it's "you can only really focus on
// 3 things in flight at once".
const DefaultWIPLimit = 3

// EffectiveWIPLimit returns the WIP limit for this board: the config
// override if positive, else DefaultWIPLimit.
func (c Config) EffectiveWIPLimit() int {
	if c.WIPLimit > 0 {
		return c.WIPLimit
	}
	return DefaultWIPLimit
}

// LoadConfig reads .focus/config.yaml. An empty or missing file
// returns a zero-value Config — that's the supported "no override"
// state, not an error.
func (b *Board) LoadConfig() (Config, error) {
	path := filepath.Join(b.Dir, ConfigFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, err
	}
	if len(data) == 0 {
		return Config{}, nil
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return c, nil
}
