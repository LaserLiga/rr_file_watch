package roadrunner

import (
	"strconv"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/roadrunner-server/pool/fsm"
)

const (
	namespace = "rr_file_watch"
)

type statsExporter struct {
	events  *uint64
	jobsOk  *uint64
	jobsErr *uint64

	eventsDesc  *prometheus.Desc
	jobsErrDesc *prometheus.Desc
	jobsOkDesc  *prometheus.Desc

	defaultExporter *StatsExporter
}

func (p *Plugin) MetricsCollector() []prometheus.Collector {
	// p - implements Exporter interface (workers)
	return []prometheus.Collector{p.metrics}
}

func (se *statsExporter) CountJobOk() {
	atomic.AddUint64(se.jobsOk, 1)
}

func (se *statsExporter) CountJobErr() {
	atomic.AddUint64(se.jobsErr, 1)
}

func (se *statsExporter) CountEvents() {
	atomic.AddUint64(se.events, 1)
}

func newStatsExporter(stats Informer) *statsExporter {
	return &statsExporter{
		defaultExporter: &StatsExporter{
			TotalWorkersDesc: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "total_workers"), "Total number of workers used by the plugin", nil, nil),
			TotalMemoryDesc:  prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "workers_memory_bytes"), "Memory usage by workers.", nil, nil),
			StateDesc:        prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "worker_state"), "Worker current state", []string{"state", "pid"}, nil),
			WorkerMemoryDesc: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "worker_memory_bytes"), "Worker current memory usage", []string{"pid"}, nil),

			WorkersReady:   prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "workers_ready"), "Workers currently in ready state", nil, nil),
			WorkersWorking: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "workers_working"), "Workers currently in working state", nil, nil),
			WorkersInvalid: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "workers_invalid"), "Workers currently in invalid,killing,destroyed,errored,inactive states", nil, nil),

			Workers: stats,
		},

		events:  toPtr(uint64(0)),
		jobsOk:  toPtr(uint64(0)),
		jobsErr: toPtr(uint64(0)),

		eventsDesc:  prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "events"), "Number of events registered in the directory", nil, nil),
		jobsErrDesc: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "jobs_err"), "Number of notifications error while processing in the worker", nil, nil),
		jobsOkDesc:  prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "jobs_ok"), "Number of successfully processed notifications", nil, nil),
	}
}

func (se *statsExporter) Describe(d chan<- *prometheus.Desc) {
	// send description
	se.defaultExporter.Describe(d)
	d <- se.eventsDesc
	d <- se.jobsErrDesc
	d <- se.jobsOkDesc
}

func (se *statsExporter) Collect(ch chan<- prometheus.Metric) {
	// get the copy of the processes
	se.defaultExporter.Collect(ch)

	// send the values to the prometheus
	ch <- prometheus.MustNewConstMetric(se.jobsOkDesc, prometheus.GaugeValue, float64(atomic.LoadUint64(se.jobsOk)))
	ch <- prometheus.MustNewConstMetric(se.jobsErrDesc, prometheus.GaugeValue, float64(atomic.LoadUint64(se.jobsErr)))
	ch <- prometheus.MustNewConstMetric(se.eventsDesc, prometheus.GaugeValue, float64(atomic.LoadUint64(se.events)))
}

func toPtr[T any](v T) *T {
	return &v
}

// ---- Default exporter ----

type StatsExporter struct {
	TotalMemoryDesc  *prometheus.Desc
	StateDesc        *prometheus.Desc
	WorkerMemoryDesc *prometheus.Desc
	TotalWorkersDesc *prometheus.Desc

	WorkersReady   *prometheus.Desc
	WorkersWorking *prometheus.Desc
	WorkersInvalid *prometheus.Desc

	Workers Informer
}

func (s *StatsExporter) Describe(d chan<- *prometheus.Desc) {
	// send description
	d <- s.TotalWorkersDesc
	d <- s.TotalMemoryDesc
	d <- s.StateDesc
	d <- s.WorkerMemoryDesc

	d <- s.WorkersReady
	d <- s.WorkersWorking
	d <- s.WorkersInvalid
}

func (s *StatsExporter) Collect(ch chan<- prometheus.Metric) {
	// get the copy of the processes
	workerStates := s.Workers.Workers()

	// cumulative RSS memory in bytes
	var cum float64

	var ready float64
	var working float64
	var invalid float64

	// collect the memory
	for i := 0; i < len(workerStates); i++ {
		cum += float64(workerStates[i].MemoryUsage)

		ch <- prometheus.MustNewConstMetric(s.StateDesc, prometheus.GaugeValue, 0, workerStates[i].StatusStr, strconv.Itoa(int(workerStates[i].Pid)))
		ch <- prometheus.MustNewConstMetric(s.WorkerMemoryDesc, prometheus.GaugeValue, float64(workerStates[i].MemoryUsage), strconv.Itoa(int(workerStates[i].Pid)))

		// sync with sdk/worker/state.go
		switch workerStates[i].Status {
		case fsm.StateReady:
			ready++
		case fsm.StateWorking:
			working++
		default:
			invalid++
		}
	}

	ch <- prometheus.MustNewConstMetric(s.WorkersReady, prometheus.GaugeValue, ready)
	ch <- prometheus.MustNewConstMetric(s.WorkersWorking, prometheus.GaugeValue, working)
	ch <- prometheus.MustNewConstMetric(s.WorkersInvalid, prometheus.GaugeValue, invalid)

	// send the values to the prometheus
	ch <- prometheus.MustNewConstMetric(s.TotalWorkersDesc, prometheus.GaugeValue, float64(len(workerStates)))
	ch <- prometheus.MustNewConstMetric(s.TotalMemoryDesc, prometheus.GaugeValue, cum)
}
