package prometheus

import (
	"context"
	"time"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/go-kit/kit/metrics"
)

// NewMetricsClient returns a new instance of a metrics Client.
func NewMetricsClient(c Client, requestCount metrics.Counter, requestLatency metrics.Histogram) Client {
	return &metricsClient{c, requestCount, requestLatency}
}

type metricsClient struct {
	Client         Client
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
}

func (c *metricsClient) AwaitScrapeInterval(ctx context.Context) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "AwaitScrapeInterval", begin)
	}(time.Now())

	c.Client.AwaitScrapeInterval(ctx)
}

func (c *metricsClient) GetMaxMemoryByPodName(ctx context.Context, podName string) (max float64, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetMaxMemoryByPodName", begin)
	}(time.Now())

	return c.Client.GetMaxMemoryByPodName(ctx, podName)
}

func (c *metricsClient) GetMaxCPUByPodName(ctx context.Context, podName string) (max float64, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetMaxCPUByPodName", begin)
	}(time.Now())

	return c.Client.GetMaxCPUByPodName(ctx, podName)
}
