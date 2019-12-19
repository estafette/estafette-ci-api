package githubapi

import (
	"context"

	"github.com/estafette/estafette-ci-api/helpers"
	"github.com/opentracing/opentracing-go"
)

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c, "githubapi"}
}

type tracingClient struct {
	Client
	prefix string
}

func (c *tracingClient) GetGithubAppToken(ctx context.Context) (token string, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetGithubAppToken"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetGithubAppToken(ctx)
}

func (c *tracingClient) GetInstallationID(ctx context.Context, repoOwner string) (installationID int, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetInstallationID"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetInstallationID(ctx, repoOwner)
}

func (c *tracingClient) GetInstallationToken(ctx context.Context, installationID int) (token AccessToken, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetInstallationToken"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetInstallationToken(ctx, installationID)
}

func (c *tracingClient) GetAuthenticatedRepositoryURL(ctx context.Context, accesstoken AccessToken, htmlURL string) (url string, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetAuthenticatedRepositoryURL"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetAuthenticatedRepositoryURL(ctx, accesstoken, htmlURL)
}

func (c *tracingClient) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event PushEvent) (valid bool, manifest string, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetEstafetteManifest"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetEstafetteManifest(ctx, accesstoken, event)
}

func (c *tracingClient) JobVarsFunc(ctx context.Context) func(context.Context, string, string, string) (string, string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "JobVarsFunc"))
	defer func() { helpers.FinishSpan(span) }()

	return c.Client.JobVarsFunc(ctx)
}
