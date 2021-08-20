package dockerhubapi

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

func (c *metricsClient) GetToken(ctx context.Context, repository string) (token DockerHubToken, err error) {
	defer func(begin time.Time) { api.UpdateMetrics(c.requestCount, c.requestLatency, "GetToken", begin) }(time.Now())

	return c.Client.GetToken(ctx, repository)
}

func (c *metricsClient) GetDigest(ctx context.Context, token DockerHubToken, repository string, tag string) (digest DockerImageDigest, err error) {
	defer func(begin time.Time) { api.UpdateMetrics(c.requestCount, c.requestLatency, "GetDigest", begin) }(time.Now())

	return c.Client.GetDigest(ctx, token, repository, tag)
}

func (c *metricsClient) GetDigestCached(ctx context.Context, repository string, tag string) (digest DockerImageDigest, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetDigestCached", begin)
	}(time.Now())

	return c.Client.GetDigestCached(ctx, repository, tag)
}
