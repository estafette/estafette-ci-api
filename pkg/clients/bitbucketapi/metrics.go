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

func (c *metricsClient) GetAccessTokenByInstallation(ctx context.Context, installation BitbucketAppInstallation) (accesstoken AccessToken, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetAccessTokenByInstallation", begin)
	}(time.Now())

	return c.Client.GetAccessTokenByInstallation(ctx, installation)
}

func (c *metricsClient) GetAccessTokenBySlug(ctx context.Context, workspaceSlug string) (accesstoken AccessToken, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetAccessTokenBySlug", begin)
	}(time.Now())

	return c.Client.GetAccessTokenBySlug(ctx, workspaceSlug)
}

func (c *metricsClient) GetAccessTokenByUUID(ctx context.Context, uuid string) (accesstoken AccessToken, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetAccessTokenByUUID", begin)
	}(time.Now())

	return c.Client.GetAccessTokenByUUID(ctx, uuid)
}

func (c *metricsClient) GetAccessTokenByJWTToken(ctx context.Context, jwtToken string) (accesstoken AccessToken, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetAccessTokenByJWTToken", begin)
	}(time.Now())

	return c.Client.GetAccessTokenByJWTToken(ctx, jwtToken)
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

func (c *metricsClient) ValidateInstallationJWT(ctx context.Context, authorizationHeader string) (installation *BitbucketAppInstallation, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "ValidateInstallationJWT", begin)
	}(time.Now())

	return c.Client.ValidateInstallationJWT(ctx, authorizationHeader)
}

func (c *metricsClient) GenerateJWTBySlug(ctx context.Context, workspaceSlug string) (tokenString string, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GenerateJWTBySlug", begin)
	}(time.Now())

	return c.Client.GenerateJWTBySlug(ctx, workspaceSlug)
}

func (c *metricsClient) GenerateJWTByUUID(ctx context.Context, uuid string) (tokenString string, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GenerateJWTByUUID", begin)
	}(time.Now())

	return c.Client.GenerateJWTByUUID(ctx, uuid)
}

func (c *metricsClient) GenerateJWTByInstallation(ctx context.Context, installation BitbucketAppInstallation) (tokenString string, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GenerateJWTByInstallation", begin)
	}(time.Now())

	return c.Client.GenerateJWTByInstallation(ctx, installation)
}

func (c *metricsClient) GetInstallationBySlug(ctx context.Context, workspaceSlug string) (installation *BitbucketAppInstallation, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetInstallationBySlug", begin)
	}(time.Now())

	return c.Client.GetInstallationBySlug(ctx, workspaceSlug)
}

func (c *metricsClient) GetInstallationByUUID(ctx context.Context, uuid string) (installation *BitbucketAppInstallation, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetInstallationByUUID", begin)
	}(time.Now())

	return c.Client.GetInstallationByUUID(ctx, uuid)
}

func (c *metricsClient) GetInstallations(ctx context.Context) (installations []*BitbucketAppInstallation, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetInstallations", begin)
	}(time.Now())

	return c.Client.GetInstallations(ctx)
}

func (c *metricsClient) AddInstallation(ctx context.Context, installation BitbucketAppInstallation) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "AddInstallation", begin)
	}(time.Now())

	return c.Client.AddInstallation(ctx, installation)
}

func (c *metricsClient) RemoveInstallation(ctx context.Context, installation BitbucketAppInstallation) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "RemoveInstallation", begin)
	}(time.Now())

	return c.Client.RemoveInstallation(ctx, installation)
}

func (c *metricsClient) GetWorkspace(ctx context.Context, uuid string) (workspace *Workspace, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetWorkspace", begin)
	}(time.Now())

	return c.Client.GetWorkspace(ctx, uuid)
}
