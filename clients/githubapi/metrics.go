package githubapi

import (
	"context"
	"time"

	"github.com/estafette/estafette-ci-api/helpers"
	"github.com/go-kit/kit/metrics"
)

// NewMetricsClient returns a new instance of a metrics Client.
func NewMetricsClient(c Client, requestCount metrics.Counter, requestLatency metrics.Histogram) Client {
	return &metricsClient{c, requestCount, requestLatency}
}

type metricsClient struct {
	Client
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
}

func (c *metricsClient) GetGithubAppToken(ctx context.Context) (token string, err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(c.requestCount, c.requestLatency, "GetGithubAppToken", begin)
	}(time.Now())

	return c.Client.GetGithubAppToken(ctx)
}

func (c *metricsClient) GetInstallationID(ctx context.Context, repoOwner string) (installationID int, err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(c.requestCount, c.requestLatency, "GetInstallationID", begin)
	}(time.Now())

	return c.Client.GetInstallationID(ctx, repoOwner)
}

func (c *metricsClient) GetInstallationToken(ctx context.Context, installationID int) (token AccessToken, err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(c.requestCount, c.requestLatency, "GetInstallationToken", begin)
	}(time.Now())

	return c.Client.GetInstallationToken(ctx, installationID)
}

func (c *metricsClient) GetAuthenticatedRepositoryURL(ctx context.Context, accesstoken AccessToken, htmlURL string) (url string, err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(c.requestCount, c.requestLatency, "GetAuthenticatedRepositoryURL", begin)
	}(time.Now())

	return c.Client.GetAuthenticatedRepositoryURL(ctx, accesstoken, htmlURL)
}

func (c *metricsClient) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event PushEvent) (valid bool, manifest string, err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(c.requestCount, c.requestLatency, "GetEstafetteManifest", begin)
	}(time.Now())

	return c.Client.GetEstafetteManifest(ctx, accesstoken, event)
}

func (c *metricsClient) JobVarsFunc(ctx context.Context) func(context.Context, string, string, string) (string, string, error) {
	defer func(begin time.Time) { helpers.UpdateMetrics(c.requestCount, c.requestLatency, "JobVarsFunc", begin) }(time.Now())

	return c.Client.JobVarsFunc(ctx)
}
