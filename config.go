package roadrunner

import poolImpl "github.com/roadrunner-server/sdk/v4/pool"

type Config struct {
	// Pool configures roadrunner workers pool.
	Pool   *poolImpl.Config `mapstructure:"pool"`
	dir    string           `mapstructure:"dir"`
	regexp string           `mapstructure:"regexp"`
}

func (cfg *Config) InitDefaults() {
	if cfg.Pool == nil {
		cfg.Pool = &poolImpl.Config{}
	}

	cfg.Pool.InitDefaults()
}
