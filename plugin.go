package roadrunner

import (
	"context"
	"github.com/roadrunner-server/errors"
	"github.com/roadrunner-server/sdk/v4/state/process"
	"go.uber.org/zap"
	"os"
	"regexp"
	"sync"
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

	// Validate directory config
	if p.cfg.dir == "" {
		return errors.E(op, errors.Str("dir is required"))
	}
	info, err := os.Stat(p.cfg.dir)
	if os.IsNotExist(err) {
		return errors.E(op, errors.Str("dir does not exist"))
	} else if err != nil {
		return errors.E(op, err)
	}
	// Check if the path is a directory
	if !info.IsDir() {
		return errors.E(op, errors.Str("dir is not a directory"))
	}

	// Validate regexp
	if p.cfg.regexp != "" {
		_, err := regexp.Compile(p.cfg.regexp)
		if err != nil {
			return errors.E(op, err)
		}
	}

	p.server = server

	p.log = new(zap.Logger)
	p.log = log.NamedLogger(PluginName)

	p.metrics = newStatsExporter(p)

	return nil
}

func (p *Plugin) Serve() chan error {
	errCh := make(chan error, 1)
	const op = errors.Op("file_watch_plugin_serve")

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
