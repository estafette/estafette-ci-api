package bitbucketapi

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

func (c *tracingClient) GetAccessToken(ctx context.Context) (AccessToken, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetAccessToken"))
	defer span.Finish()

	accessToken, err := c.Client.GetAccessToken(ctx)

	c.handleError(span, err)

	return accessToken, err
}

func (c *tracingClient) GetAuthenticatedRepositoryURL(ctx context.Context, accessToken AccessToken, htmlURL string) (string, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetAuthenticatedRepositoryURL"))
	defer span.Finish()

	url, err := c.Client.GetAuthenticatedRepositoryURL(ctx, accessToken, htmlURL)

	c.handleError(span, err)

	return url, err
}

func (c *tracingClient) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event RepositoryPushEvent) (bool, string, error) {

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
	return "bitbucketapi:" + funcName
}

func (c *tracingClient) handleError(span opentracing.Span, err error) error {
	if err != nil {
		ext.Error.Set(span, true)
		span.LogFields(log.Error(err))
	}
	return err
}
