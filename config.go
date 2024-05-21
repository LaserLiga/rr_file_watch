package roadrunner

import poolImpl "github.com/roadrunner-server/sdk/v4/pool"

type Config struct {
	// Pool configures roadrunner workers pool.
	Pool   *poolImpl.Config `mapstructure:"pool"`
	Dir    string           `mapstructure:"dir"`
	Regexp string           `mapstructure:"regexp"`
}

func (cfg *Config) InitDefaults() {
	if cfg.Pool == nil {
		cfg.Pool = &poolImpl.Config{}
	}

	cfg.Pool.InitDefaults()

	if cfg.Dir == "" {
		cfg.Dir = "./lmx/results"
	}
}
