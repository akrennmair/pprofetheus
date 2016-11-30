package pprofetheus

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	dto "github.com/prometheus/client_model/go"
)

func TestCPUProfileCollector(t *testing.T) {
	profileCollector, err := NewCPUProfileCollector()
	if err != nil {
		t.Fatal(err)
	}

	cpuProfileCollector, ok := profileCollector.(*cpuProfileCollector)
	if !ok {
		t.Fatalf("returned ProfileCollector is not a *cpuProfileCollector")
	}

	cpuProfileCollector.Start()

	spendSomeTimeComputing()

	metricsChan := make(chan prometheus.Metric)
	go func() {
		cpuProfileCollector.Collect(metricsChan)
		close(metricsChan)
	}()

	metrics := []prometheus.Metric{}

	for m := range metricsChan {
		metrics = append(metrics, m)
	}

	if len(metrics) != 6 {
		t.Fatalf("Expected 6 metrics, got %d instead: %#v", len(metrics), metrics)
	}

	testData := []struct {
		ExpectedMetric   string
		ExpectedFunc     string
		CheckFunc        bool
		ExpectedMinValue float64
		ExpectedMaxValue float64
	}{
		{"pprof_cpu_time_used_ms", "github.com/travelaudience/pprofetheus.spendSomeTimeComputing", true, 990, 1100},
		{"pprof_cpu_time_used_cum_ms", "github.com/travelaudience/pprofetheus.spendSomeTimeComputing", true, 990, 1100},
		{"pprof_cpu_time_used_cum_ms", "testing.tRunner", true, 990, 1100},
		{"pprof_cpu_time_used_cum_ms", "runtime.goexit", true, 990, 1100},
		{"pprof_cpu_started", "", false, 1, 1},
		{"pprof_cpu_stopped", "", false, 0, 0},
	}

	for idx, testEntry := range testData {

		found := false

	innerLoop:
		for _, m := range metrics {
			var metric dto.Metric

			if err := m.Write(&metric); err != nil {
				t.Errorf("%d. writing metric to DTO failed: %v", idx, err)
				continue
			}

			if !strings.Contains(m.Desc().String(), `fqName: "`+testEntry.ExpectedMetric+`"`) {
				continue
			}

			if testEntry.CheckFunc {
				found := false
				for _, l := range metric.Label {
					if l.GetName() == "function" && l.GetValue() == testEntry.ExpectedFunc {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			found = true

			value := metric.GetCounter().GetValue()

			if testEntry.ExpectedMinValue > value || testEntry.ExpectedMaxValue < value {
				t.Errorf("%d. value it outside of range (%f,%f): %f", idx, testEntry.ExpectedMinValue, testEntry.ExpectedMaxValue, value)
				break innerLoop
			}
		}

		if !found {
			t.Errorf("%d. metric %s with function %q not found.", idx, testEntry.ExpectedMetric, testEntry.ExpectedFunc)
		}
	}

	cpuProfileCollector.Stop()
}

func spendSomeTimeComputing() {
	to := time.After(1 * time.Second)

	for {
		select {
		case <-to:
			return
		default:
			x := 0
			for i := 0; i < 100000; i++ {
				if x > i {
					x += i
				}
			}
		}
	}
}
