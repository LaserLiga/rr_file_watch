package roadrunner

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"sync"

	"github.com/radovskyb/watcher"
	"github.com/roadrunner-server/errors"
	"github.com/roadrunner-server/pool/pool/static_pool"
	"github.com/roadrunner-server/pool/state/process"
	"go.uber.org/zap"
)

const (
	RrMode          string = "RR_MODE"
	RrModeFileWatch string = "file_watch"

	PluginName = "file_watch"
)

type Plugin struct {
	mu          sync.RWMutex
	cfg         *Config
	workersPool *static_pool.Pool
	watcher     *watcher.Watcher
	server      Server
	log         *zap.Logger
	metrics     *statsExporter

	// signal channel to stop the pollers
	stopCh   chan struct{}
	stopOnce sync.Once
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
	if err = p.cfg.Validate(); err != nil {
		return errors.E(op, err)
	}

	p.server = server

	p.stopOnce = sync.Once{}

	p.log = new(zap.Logger)
	p.log = log.NamedLogger(PluginName)

	p.metrics = newStatsExporter(p)

	return nil
}

func (p *Plugin) Serve() chan error {
	errCh := make(chan error, 1)
	const op = errors.Op("file_watch_plugin_serve")

	// Validate directory config
	validDirs, dirErr := p.validWatchDirs()
	if dirErr != nil {
		errCh <- errors.E(op, dirErr)
		return errCh
	}
	p.cfg.Dirs = validDirs
	p.cfg.Dir = ""

	// Validate Regexp
	if p.cfg.Regexp != "" {
		_, err := regexp.Compile(p.cfg.Regexp)
		if err != nil {
			errCh <- errors.E(op, err)
			return errCh
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.stopCh = make(chan struct{})
	p.stopOnce = sync.Once{}

	var err error
	p.workersPool, err = p.server.NewPool(context.Background(), p.cfg.Pool, map[string]string{RrMode: RrModeFileWatch}, nil)
	if err != nil {
		errCh <- errors.E(op, err)
		return errCh
	}

	// start listening
	if err = p.listener(); err != nil {
		p.workersPool.Destroy(context.Background())
		p.workersPool = nil
		errCh <- errors.E(op, err)
		return errCh
	}

	return errCh
}

func (p *Plugin) validWatchDirs() ([]string, error) {
	jsonCfg, jsonErr := json.Marshal(p.cfg)
	validDirs := make([]string, 0, len(p.cfg.WatchDirs()))
	for _, dir := range p.cfg.WatchDirs() {
		info, fileErr := os.Stat(dir)
		if os.IsNotExist(fileErr) {
			if jsonErr != nil {
				p.log.Warn("configured watch directory does not exist", zap.String("dir", dir), zap.Error(jsonErr))
			} else {
				p.log.Warn("configured watch directory does not exist", zap.String("dir", dir), zap.ByteString("config", jsonCfg))
			}
			continue
		} else if fileErr != nil {
			p.log.Warn("configured watch directory cannot be inspected", zap.String("dir", dir), zap.Error(fileErr))
			continue
		}
		if !info.IsDir() {
			if jsonErr != nil {
				p.log.Warn("configured watch path is not a directory", zap.String("dir", dir), zap.Error(jsonErr))
			} else {
				p.log.Warn("configured watch path is not a directory", zap.String("dir", dir), zap.ByteString("config", jsonCfg))
			}
			continue
		}
		validDirs = append(validDirs, dir)
	}
	if len(validDirs) == 0 {
		return nil, errors.Str("no configured watch directories exist or are directories")
	}
	return validDirs, nil
}

func (p *Plugin) Name() string {
	return PluginName
}

func (p *Plugin) Reset() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	const op = errors.Op("file_watch_plugin_reset")
	p.log.Info("reset signal was received")
	if p.workersPool == nil {
		return errors.E(op, errors.Str("worker pool is not initialized"))
	}
	err := p.workersPool.Reset(context.Background())
	if err != nil {
		return errors.E(op, err)
	}
	p.log.Info("plugin was successfully reset")

	return nil
}

func (p *Plugin) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.watcher != nil {
		p.watcher.Close()
		p.watcher = nil
	}

	if p.stopCh != nil {
		stopCh := p.stopCh
		p.stopOnce.Do(func() {
			// Broadcast stop signal to all pollers.
			close(stopCh)
		})
	}

	return nil
}

func (p *Plugin) Workers() []*process.State {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.workersPool == nil {
		return nil
	}

	wrk := p.workersPool.Workers()

	ps := make([]*process.State, 0, len(wrk))
	for i := range wrk {
		state, err := process.WorkerProcessState(wrk[i])
		if err != nil {
			return nil
		}
		ps = append(ps, state)
	}

	return ps
}
