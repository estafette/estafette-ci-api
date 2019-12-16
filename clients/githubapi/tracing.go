package githubapi

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
)

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c}
}

type tracingClient struct {
	Client
}

func (c *tracingClient) GetGithubAppToken(ctx context.Context) (token string, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetGithubAppToken"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.GetGithubAppToken(ctx)
}

func (c *tracingClient) GetInstallationID(ctx context.Context, repoOwner string) (installationID int, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetInstallationID"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.GetInstallationID(ctx, repoOwner)
}

func (c *tracingClient) GetInstallationToken(ctx context.Context, installationID int) (token AccessToken, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetInstallationToken"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.GetInstallationToken(ctx, installationID)
}

func (c *tracingClient) GetAuthenticatedRepositoryURL(ctx context.Context, accesstoken AccessToken, htmlURL string) (url string, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetAuthenticatedRepositoryURL"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.GetAuthenticatedRepositoryURL(ctx, accesstoken, htmlURL)
}

func (c *tracingClient) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event PushEvent) (valid bool, manifest string, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetEstafetteManifest"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.GetEstafetteManifest(ctx, accesstoken, event)
}

func (c *tracingClient) JobVarsFunc(ctx context.Context) func(context.Context, string, string, string) (string, string, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("JobVarsFunc"))
	defer span.Finish()

	return c.Client.JobVarsFunc(ctx)
}

func (c *tracingClient) getSpanName(funcName string) string {
	return "githubapi:" + funcName
}

func (c *tracingClient) handleError(span opentracing.Span, err error) {
	if err != nil {
		ext.Error.Set(span, true)
		span.LogFields(log.Error(err))
	}
}
