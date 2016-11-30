// pprofetheus is a collector for Prometheus that collects CPU profiling data
// for the current process and exports them as metrics. It can be used to monitor,
// visualize, and alert on profiling data from any Go process that imports
// pprofetheus and exports metrics via Prometheus.
//
// In order to use pprofetheus in your Prometheus-enabled Go application, you just
// need to
//
//   go get github.com/travelaudience/pprofetheus
//
// and then import the same package, and set up the collector with Prometheus in
// your code, e.g. like this:
//
//   cpuProfileCollector, err := pprofetheus.NewCPUProfileCollector()
//   if err != nil {
//   	/* handle error */
//   }
//   prometheus.MustRegister(cpuProfileCollector)
//   cpuProfileCollector.Start()
package pprofetheus

import (
	"bytes"
	"runtime"
	"sync"

	"github.com/travelaudience/pprofetheus/internal/objfile"
	"github.com/travelaudience/pprofetheus/internal/pprof/profile"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace          = "pprof"
	cpuSubsystem       = "cpu"
	cpuProfileRate     = 100
	nanoToMilliDivisor = 1000000
)

var (
	labelNames = []string{"function"}
)

// NewCPUProfileCollector creates a new CPU profile collector.
func NewCPUProfileCollector() (ProfileCollector, error) {
	exeFile, err := objfile.Open("/proc/self/exe")
	if err != nil {
		return nil, err
	}

	symbols, err := exeFile.Symbols()
	if err != nil {
		return nil, err
	}

	return &cpuProfileCollector{
		timeUsed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: cpuSubsystem,
				Name:      "time_used_ms",
				Help:      "CPU time used by function in milliseconds",
			},
			labelNames,
		),
		timeUsedCum: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: cpuSubsystem,
				Name:      "time_used_cum_ms",
				Help:      "CPU time used by function in milliseconds (cumulated)",
			},
			labelNames,
		),
		started: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: cpuSubsystem,
				Name:      "started",
				Help:      "counter of pprof start events in CPU profile collector",
			},
		),
		stopped: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: cpuSubsystem,
				Name:      "stopped",
				Help:      "counter of pprof stop events in CPU profile collector",
			},
		),
		symbols: symbols,
	}, nil
}

// ProfileCollector describes a pprofetheus collector. It can act as a prometheus.Collector
// plus it can be Start()ed and Stop()ed to limit profiling to only desired time periods.
type ProfileCollector interface {
	prometheus.Collector
	Start()
	Stop()
}

type cpuProfileCollector struct {
	sync.Mutex
	timeUsed    *prometheus.CounterVec
	timeUsedCum *prometheus.CounterVec
	started     prometheus.Counter
	stopped     prometheus.Counter
	running     bool
	symbols     []objfile.Sym
}

func (c *cpuProfileCollector) Start() {
	c.Lock()
	defer c.Unlock()

	if c.running {
		return
	}
	c.running = true

	runtime.SetCPUProfileRate(cpuProfileRate)

	c.started.Inc()
}

func (c *cpuProfileCollector) Stop() {
	c.Lock()
	defer c.Unlock()

	if !c.running {
		return
	}
	c.running = false

	runtime.SetCPUProfileRate(0)

	c.stopped.Inc()
}

func (c *cpuProfileCollector) Describe(ch chan<- *prometheus.Desc) {
	c.timeUsed.Describe(ch)
	c.timeUsedCum.Describe(ch)
	c.started.Describe(ch)
	c.stopped.Describe(ch)
}

func (c *cpuProfileCollector) Collect(ch chan<- prometheus.Metric) {
	c.Lock()
	defer c.Unlock()
	if c.running {
		runtime.SetCPUProfileRate(0)

		var allData bytes.Buffer
		for {
			data := runtime.CPUProfile()
			if data == nil {
				break
			}
			allData.Write(data)
		}

		p, err := profile.Parse(&allData)
		if err != nil {
			panic(err) // TODO: introduce metric for parse errors.
		}

		locations := mapLocations(p.Location, c.symbols)

		for _, s := range p.Sample {
			if len(s.Location) == 0 || len(s.Value) < 2 {
				continue
			}

			c.timeUsed.WithLabelValues(locations[s.Location[0].ID]).Add(float64(s.Value[1]) / nanoToMilliDivisor)

			for _, l := range s.Location {
				c.timeUsedCum.WithLabelValues(locations[l.ID]).Add(float64(s.Value[1]) / nanoToMilliDivisor)
			}
		}
	}

	c.timeUsed.Collect(ch)
	c.timeUsedCum.Collect(ch)
	c.started.Collect(ch)
	c.stopped.Collect(ch)

	if c.running {
		runtime.SetCPUProfileRate(cpuProfileRate)
	}
}

func mapLocations(locations []*profile.Location, symbols []objfile.Sym) map[uint64]string {
	result := make(map[uint64]string)

	for _, l := range locations {
		for _, s := range symbols {
			if l.Address >= s.Addr && l.Address <= s.Addr+uint64(s.Size) {
				result[l.ID] = s.Name
				break
			}
		}
	}

	return result
}
