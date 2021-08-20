package bitbucketapi

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

func (c *metricsClient) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event RepositoryPushEvent) (valid bool, manifest string, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetEstafetteManifest", begin)
	}(time.Now())

	return c.Client.GetEstafetteManifest(ctx, accesstoken, event)
}

func (c *metricsClient) JobVarsFunc(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
	defer func(begin time.Time) { api.UpdateMetrics(c.requestCount, c.requestLatency, "JobVarsFunc", begin) }(time.Now())

	return c.Client.JobVarsFunc(ctx)
}

func (c *metricsClient) GenerateJWT() (tokenString string, err error) {
	return c.Client.GenerateJWT()
}

func (c *metricsClient) GetInstallations(ctx context.Context) (installations []*BitbucketAppInstallation, err error) {
	return c.Client.GetInstallations(ctx)
}

func (c *metricsClient) AddInstallation(ctx context.Context, installation BitbucketAppInstallation) (err error) {
	return c.Client.AddInstallation(ctx, installation)
}

func (c *metricsClient) RemoveInstallation(ctx context.Context, installation BitbucketAppInstallation) (err error) {
	return c.Client.RemoveInstallation(ctx, installation)
}
