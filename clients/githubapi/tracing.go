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

func (c *tracingClient) GetGithubAppToken(ctx context.Context) (string, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetGithubAppToken"))
	defer span.Finish()

	token, err := c.Client.GetGithubAppToken(ctx)
	c.handleError(span, err)

	return token, err
}

func (c *tracingClient) GetInstallationID(ctx context.Context, repoOwner string) (int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetInstallationID"))
	defer span.Finish()

	installationid, err := c.Client.GetInstallationID(ctx, repoOwner)
	c.handleError(span, err)

	return installationid, err
}

func (c *tracingClient) GetInstallationToken(ctx context.Context, installationID int) (AccessToken, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetInstallationToken"))
	defer span.Finish()

	token, err := c.Client.GetInstallationToken(ctx, installationID)
	c.handleError(span, err)

	return token, err
}

func (c *tracingClient) GetAuthenticatedRepositoryURL(ctx context.Context, accesstoken AccessToken, htmlURL string) (string, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetAuthenticatedRepositoryURL"))
	defer span.Finish()

	url, err := c.Client.GetAuthenticatedRepositoryURL(ctx, accesstoken, htmlURL)
	c.handleError(span, err)

	return url, err
}

func (c *tracingClient) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event PushEvent) (bool, string, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetEstafetteManifest"))
	defer span.Finish()

	valid, manifest, err := c.Client.GetEstafetteManifest(ctx, accesstoken, event)
	c.handleError(span, err)

	return valid, manifest, err
}

func (c *tracingClient) JobVarsFunc(ctx context.Context) func(context.Context, string, string, string) (string, string, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("JobVarsFunc"))
	defer span.Finish()

	return c.Client.JobVarsFunc(ctx)
}

func (c *tracingClient) getSpanName(funcName string) string {
	return "githubapi:" + funcName
}

func (c *tracingClient) handleError(span opentracing.Span, err error) error {
	if err != nil {
		ext.Error.Set(span, true)
		span.LogFields(log.Error(err))
	}
	return err
}
