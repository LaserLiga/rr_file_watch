package roadrunner

import (
	"context"
	"encoding/json"
	"github.com/roadrunner-server/errors"
	"github.com/roadrunner-server/sdk/v4/state/process"
	"go.uber.org/zap"
	"os"
	"regexp"
	"sync"
	"time"
)

const (
	RrMode          string = "RR_MODE"
	RrModeFileWatch string = "file_watch"

	PluginName = "file_watch"
)

type Plugin struct {
	mu          sync.RWMutex
	cfg         *Config
	workersPool Pool
	server      Server
	log         *zap.Logger
	metrics     *statsExporter

	// signal channel to stop the pollers
	stopCh chan struct{}
}

func (p *Plugin) Init(cfg Configurer, log Logger, server Server) error {
	const op = errors.Op("file_watch_plugin_init")
	if !cfg.Has(PluginName) {
		return errors.E(op, errors.Disabled)
	}

	err := cfg.UnmarshalKey(PluginName, &p.cfg)
	if err != nil {
		return errors.E(op, err)
	}

	p.cfg.InitDefaults()

	p.server = server

	p.log = new(zap.Logger)
	p.log = log.NamedLogger(PluginName)

	p.metrics = newStatsExporter(p)

	return nil
}

func (p *Plugin) Serve() chan error {
	errCh := make(chan error, 1)
	const op = errors.Op("file_watch_plugin_serve")

	// Validate directory config
	jsonCfg, jsonErr := json.Marshal(p.cfg)
	if p.cfg.Dir == "" {
		if jsonErr == nil {
			errCh <- errors.E(op, errors.Str("Dir is required "+string(jsonCfg)))
		} else {
			errCh <- errors.E(op, jsonErr)
			errCh <- errors.E(op, errors.Str("Dir is required"))
		}
		return errCh
	}
	info, fileErr := os.Stat(p.cfg.Dir)
	if os.IsNotExist(fileErr) {
		if jsonErr == nil {
			errCh <- errors.E(op, errors.Str("Dir does not exist "+string(jsonCfg)))
		} else {
			errCh <- errors.E(op, jsonErr)
			errCh <- errors.E(op, errors.Str("Dir does not exist"))
		}
		return errCh
	} else if fileErr != nil {
		errCh <- errors.E(op, fileErr)
		return errCh
	}
	// Check if the path is a directory
	if !info.IsDir() {
		if jsonErr == nil {
			errCh <- errors.E(op, errors.Str("Dir is not a directory "+string(jsonCfg)))
		} else {
			errCh <- errors.E(op, jsonErr)
			errCh <- errors.E(op, errors.Str("Dir is not a directory"))
		}
		return errCh
	}

	// Validate Regexp
	if p.cfg.Regexp != "" {
		_, err := regexp.Compile(p.cfg.Regexp)
		if err != nil {
			errCh <- errors.E(op, err)
			return errCh
		}
	}

	p.mu.Lock()

	var err error
	p.workersPool, err = p.server.NewPool(context.Background(), p.cfg.Pool, map[string]string{RrMode: RrModeFileWatch}, nil)
	if err != nil {
		p.mu.Unlock()
		errCh <- errors.E(op, err)
		return errCh
	}

	// start listening
	p.listener()

	p.mu.Unlock()
	return errCh
}

func (p *Plugin) Reset() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.workersPool == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	err := p.workersPool.Reset(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (p *Plugin) Stop(ctx context.Context) error {
	// Broadcast stop signal to all pollers
	close(p.stopCh)
	return nil
}

func (p *Plugin) Workers() []*process.State {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.workersPool == nil {
		return nil
	}

	wrk := p.workersPool.Workers()

	ps := make([]*process.State, len(wrk))

	for i := 0; i < len(wrk); i++ {
		if wrk[i] == nil {
			continue
		}
		st, err := process.WorkerProcessState(wrk[i])
		if err != nil {
			p.log.Error("notifications workers state", zap.Error(err))
			return nil
		}

		ps[i] = st
	}

	return ps
}
