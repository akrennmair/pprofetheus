# pprofetheus

[![GoDoc](https://godoc.org/github.com/travelaudience/pprofetheus?status.svg)](https://godoc.org/github.com/travelaudience/pprofetheus)

pprofetheus is a collector for [Prometheus](https://prometheus.io/) that 
collects CPU profiling data for the current process and exports them as metrics. 
It can be used to monitor, visualize, and alert on profiling data from any Go 
process that imports pprofetheus and exports metrics via Prometheus.

## Quick start

In order to use pprofetheus in your Prometheus-enabled Go application, you just 
need to

	go get github.com/travelaudience/pprofetheus

and then import the same package, and set up the collector with Prometheus in 
your code, e.g. like this:

	cpuProfileCollector, err := pprofetheus.NewCPUProfileCollector()
	if err != nil {
		/* handle error */
	}
	prometheus.MustRegister(cpuProfileCollector)
	cpuProfileCollector.Start()

After these changes, your application will export the Prometheus metrics 
`pprof_cpu_time_used_ms`, `pprof_cpu_time_used_cum_ms`, `pprof_cpu_started` and 
`pprof_cpu_stopped`.

`pprof_cpu_time_used_ms` contains the amount of milliseconds the program spent 
in the function provided in the label `function`.

`pprof_cpu_time_used_cum_ms` contains the _cumulated_ amount of milliseconds 
the program spent in the function provided in the label `function`. This means 
that the amount of time spent in a function is accounted both for the function 
that spent the time and all functions up the call stack. The cumulated time 
thus includes the total time that a function spent, including all other 
functions that were called by that function.

`pprof_cpu_started` counts how often the `Start` method has been called on the 
collector, while `pprof_cpu_stopped` counts how often the `Stop` method has 
been called on the collector.

## License

Please see the file [LICENSE.md](LICENSE.md) for licensing information.

The files in the subdirectory `internal` are subject to a separate license. See
[internal/LICENSE](internal/LICENSE) for licensing information.
