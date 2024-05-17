package roadrunner

import (
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/roadrunner-server/sdk/v4/metrics"
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

	defaultExporter *metrics.StatsExporter
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
		defaultExporter: &metrics.StatsExporter{
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
