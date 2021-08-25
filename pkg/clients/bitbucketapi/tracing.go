package bitbucketapi

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/opentracing/opentracing-go"
)

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c, "bitbucketapi"}
}

type tracingClient struct {
	Client Client
	prefix string
}

func (c *tracingClient) GetAccessToken(ctx context.Context) (accesstoken AccessToken, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetAccessToken"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetAccessToken(ctx)
}

func (c *tracingClient) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event RepositoryPushEvent) (valid bool, manifest string, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetEstafetteManifest"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetEstafetteManifest(ctx, accesstoken, event)
}

func (c *tracingClient) JobVarsFunc(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "JobVarsFunc"))
	defer func() { api.FinishSpan(span) }()

	return c.Client.JobVarsFunc(ctx)
}

func (c *tracingClient) GenerateJWT() (tokenString string, err error) {
	return c.Client.GenerateJWT()
}

func (c *tracingClient) GetInstallations(ctx context.Context) (installations []*BitbucketAppInstallation, err error) {
	return c.Client.GetInstallations(ctx)
}

func (c *tracingClient) AddInstallation(ctx context.Context, installation BitbucketAppInstallation) (err error) {
	return c.Client.AddInstallation(ctx, installation)
}

func (c *tracingClient) RemoveInstallation(ctx context.Context, installation BitbucketAppInstallation) (err error) {
	return c.Client.RemoveInstallation(ctx, installation)
}

func (c *tracingClient) GetWorkspace(ctx context.Context, installation BitbucketAppInstallation) (workspace *Workspace, err error) {
	return c.Client.GetWorkspace(ctx, installation)
}
