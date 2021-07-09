package api

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

var requestCounters map[string]metrics.Counter = map[string]metrics.Counter{}

func NewRequestCounter(subsystem string) metrics.Counter {

	if _, ok := requestCounters[subsystem]; !ok {
		requestCounters[subsystem] = kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "api",
			Subsystem: subsystem,
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"func"})
	}

	return requestCounters[subsystem]
}

var requestHistograms map[string]metrics.Histogram = map[string]metrics.Histogram{}

func NewRequestHistogram(subsystem string) metrics.Histogram {

	if _, ok := requestHistograms[subsystem]; !ok {
		requestHistograms[subsystem] = kitprometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: "api",
			Subsystem: subsystem,
			Name:      "request_latency_seconds",
			Help:      "Total duration of requests in seconds.",
		}, []string{"func"})
	}

	return requestHistograms[subsystem]
}
