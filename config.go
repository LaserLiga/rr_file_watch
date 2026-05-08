package roadrunner

import (
	"errors"
	"slices"
	"time"

	poolImpl "github.com/roadrunner-server/pool/pool"
)

type Config struct {
	// Pool configures roadrunner workers pool.
	Pool   *poolImpl.Config `mapstructure:"pool"`
	Dir    string           `mapstructure:"dir"`
	Dirs   []string         `mapstructure:"dirs"`
	Regexp string           `mapstructure:"regexp"`
	// Debounce delays worker dispatch for repeated events on the same path until the file is quiet.
	// Configure it as a Go duration string, for example "500ms", "1s", or "0s" to disable coalescing.
	Debounce string `mapstructure:"debounce"`
}

func (cfg *Config) InitDefaults() {
	if cfg.Pool == nil {
		cfg.Pool = &poolImpl.Config{}
	}

	cfg.Pool.InitDefaults()

	if cfg.Dir == "" && len(cfg.Dirs) == 0 {
		cfg.Dir = "./lmx/results"
	}

	if cfg.Debounce == "" {
		cfg.Debounce = "1s"
	}
}

func (cfg *Config) Validate() error {
	if len(cfg.WatchDirs()) == 0 {
		return errors.New("at least one watch directory is required")
	}
	if _, err := cfg.DebounceDuration(); err != nil {
		return err
	}
	return nil
}

func (cfg *Config) WatchDirs() []string {
	dirs := make([]string, 0, len(cfg.Dirs)+1)
	if cfg.Dir != "" {
		dirs = append(dirs, cfg.Dir)
	}
	for _, dir := range cfg.Dirs {
		if dir == "" || slices.Contains(dirs, dir) {
			continue
		}
		dirs = append(dirs, dir)
	}
	return dirs
}

func (cfg *Config) DebounceDuration() (time.Duration, error) {
	debounce, err := time.ParseDuration(cfg.Debounce)
	if err != nil {
		return 0, err
	}
	if debounce < 0 {
		return 0, errors.New("debounce must not be negative")
	}
	return debounce, nil
}
