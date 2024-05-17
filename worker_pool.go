package roadrunner

import (
	"go.uber.org/zap"
	"sync"
)

type processor struct {
	wg         sync.WaitGroup
	mu         sync.Mutex
	consumers  *sync.Map
	runners    *map[string]struct{}
	log        *zap.Logger
	queueCh    chan *pjob
	maxWorkers int
	errs       []error
}

type pjob struct {
	configKey string
	timeout   int
}

func newPipesProc(log *zap.Logger, consumers *sync.Map, runners *map[string]struct{}, maxWorkers int) *processor {
	p := &processor{
		log:        log,
		queueCh:    make(chan *pjob, 100),
		maxWorkers: maxWorkers,
		consumers:  consumers,
		runners:    runners,
		wg:         sync.WaitGroup{},
		mu:         sync.Mutex{},
		errs:       make([]error, 0, 1),
	}

	// start the processor
	p.run()

	return p
}

func (p *processor) run() {
}

func (p *processor) add(pjob *pjob) {
	p.wg.Add(1)
	p.queueCh <- pjob
}

func (p *processor) errors() []error {
	p.mu.Lock()
	defer p.mu.Unlock()
	errs := make([]error, len(p.errs))
	copy(errs, p.errs)
	return errs
}

func (p *processor) wait() {
	p.wg.Wait()
}

func (p *processor) stop() {
	close(p.queueCh)
}
