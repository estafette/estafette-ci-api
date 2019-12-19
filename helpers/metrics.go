package helpers

import (
	"time"

	foundation "github.com/estafette/estafette-foundation"
	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

func UpdateMetrics(requestCount metrics.Counter, requestLatency metrics.Histogram, funcName string, begin time.Time) {
	funcName = foundation.ToLowerSnakeCase(funcName)

	requestCount.With("func", funcName).Add(1)
	requestLatency.With("func", funcName).Observe(time.Since(begin).Seconds())
}

func NewRequestCounter(subsystem string) metrics.Counter {
	return kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "api",
		Subsystem: subsystem,
		Name:      "request_count",
		Help:      "Number of requests received.",
	}, []string{"func"})
}

func NewRequestHistogram(subsystem string) metrics.Histogram {
	return kitprometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
		Namespace: "api",
		Subsystem: subsystem,
		Name:      "request_latency_seconds",
		Help:      "Total duration of requests in seconds.",
	}, []string{"func"})
}
