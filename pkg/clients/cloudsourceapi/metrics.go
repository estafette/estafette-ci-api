package cloudsourceapi

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

func (c *metricsClient) GetAccessToken(ctx context.Context) (accesstoken AccessToken, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetAccessToken", begin)
	}(time.Now())

	return c.Client.GetAccessToken(ctx)
}

func (c *metricsClient) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, notification PubSubNotification, gitClone func(string, string, string) error) (valid bool, manifest string, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetEstafetteManifest", begin)
	}(time.Now())

	return c.Client.GetEstafetteManifest(ctx, accesstoken, notification, gitClone)
}

func (c *metricsClient) JobVarsFunc(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
	defer func(begin time.Time) { api.UpdateMetrics(c.requestCount, c.requestLatency, "JobVarsFunc", begin) }(time.Now())

	return c.Client.JobVarsFunc(ctx)
}
