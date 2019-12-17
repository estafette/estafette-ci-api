package bitbucketapi

import (
	"context"

	"github.com/estafette/estafette-ci-api/helpers"
	"github.com/opentracing/opentracing-go"
)

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c}
}

type tracingClient struct {
	Client
}

func (c *tracingClient) GetAccessToken(ctx context.Context) (accesstoken AccessToken, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetAccessToken"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetAccessToken(ctx)
}

func (c *tracingClient) GetAuthenticatedRepositoryURL(ctx context.Context, accessToken AccessToken, htmlURL string) (url string, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetAuthenticatedRepositoryURL"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetAuthenticatedRepositoryURL(ctx, accessToken, htmlURL)
}

func (c *tracingClient) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event RepositoryPushEvent) (valid bool, manifest string, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetEstafetteManifest"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetEstafetteManifest(ctx, accesstoken, event)
}

func (c *tracingClient) JobVarsFunc(ctx context.Context) func(context.Context, string, string, string) (string, string, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("JobVarsFunc"))
	defer func() { helpers.FinishSpan(span) }()

	return c.Client.JobVarsFunc(ctx)
}

func (c *tracingClient) getSpanName(funcName string) string {
	return "bitbucketapi:" + funcName
}
