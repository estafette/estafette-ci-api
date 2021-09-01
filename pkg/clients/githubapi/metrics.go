package githubapi

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

func (c *metricsClient) GetGithubAppToken(ctx context.Context) (token string, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetGithubAppToken", begin)
	}(time.Now())

	return c.Client.GetGithubAppToken(ctx)
}

func (c *metricsClient) GetInstallationID(ctx context.Context, repoOwner string) (installationID int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetInstallationID", begin)
	}(time.Now())

	return c.Client.GetInstallationID(ctx, repoOwner)
}

func (c *metricsClient) GetInstallationToken(ctx context.Context, installationID int) (token AccessToken, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetInstallationToken", begin)
	}(time.Now())

	return c.Client.GetInstallationToken(ctx, installationID)
}

func (c *metricsClient) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event PushEvent) (valid bool, manifest string, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetEstafetteManifest", begin)
	}(time.Now())

	return c.Client.GetEstafetteManifest(ctx, accesstoken, event)
}

func (c *metricsClient) JobVarsFunc(ctx context.Context) func(context.Context, string, string, string) (string, error) {
	defer func(begin time.Time) { api.UpdateMetrics(c.requestCount, c.requestLatency, "JobVarsFunc", begin) }(time.Now())

	return c.Client.JobVarsFunc(ctx)
}

func (c *metricsClient) ConvertAppManifestCode(ctx context.Context, code string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "ConvertAppManifestCode", begin)
	}(time.Now())

	return c.Client.ConvertAppManifestCode(ctx, code)
}

func (c *metricsClient) GetApps(ctx context.Context) (apps []*GithubApp, err error) {
	defer func(begin time.Time) { api.UpdateMetrics(c.requestCount, c.requestLatency, "GetApps", begin) }(time.Now())

	return c.Client.GetApps(ctx)
}

func (c *metricsClient) GetAppByID(ctx context.Context, id int) (app *GithubApp, err error) {
	defer func(begin time.Time) { api.UpdateMetrics(c.requestCount, c.requestLatency, "GetAppByID", begin) }(time.Now())

	return c.Client.GetAppByID(ctx, id)
}

func (c *metricsClient) AddApp(ctx context.Context, app GithubApp) (err error) {
	defer func(begin time.Time) { api.UpdateMetrics(c.requestCount, c.requestLatency, "AddApp", begin) }(time.Now())

	return c.Client.AddApp(ctx, app)
}

func (c *metricsClient) RemoveApp(ctx context.Context, app GithubApp) (err error) {
	defer func(begin time.Time) { api.UpdateMetrics(c.requestCount, c.requestLatency, "RemoveApp", begin) }(time.Now())

	return c.Client.RemoveApp(ctx, app)
}

func (c *metricsClient) AddInstallation(ctx context.Context, installation Installation) (err error) {
	defer func(begin time.Time) { api.UpdateMetrics(c.requestCount, c.requestLatency, "AddInstallation", begin) }(time.Now())

	return c.Client.AddInstallation(ctx, installation)
}

func (c *metricsClient) RemoveInstallation(ctx context.Context, installation Installation) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "RemoveInstallation", begin)
	}(time.Now())

	return c.Client.RemoveInstallation(ctx, installation)
}
