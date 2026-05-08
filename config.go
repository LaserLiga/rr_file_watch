package roadrunner

import (
	"errors"
	"time"

	poolImpl "github.com/roadrunner-server/pool/pool"
)

type Config struct {
	// Pool configures roadrunner workers pool.
	Pool   *poolImpl.Config `mapstructure:"pool"`
	Dir    string           `mapstructure:"dir"`
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

	if cfg.Dir == "" {
		cfg.Dir = "./lmx/results"
	}

	if cfg.Debounce == "" {
		cfg.Debounce = "1s"
	}
}

func (cfg *Config) Validate() error {
	if _, err := cfg.DebounceDuration(); err != nil {
		return err
	}
	return nil
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
