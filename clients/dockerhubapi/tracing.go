package dockerhubapi

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

func (c *tracingClient) GetToken(ctx context.Context, repository string) (token DockerHubToken, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetToken"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.GetToken(ctx, repository)
}

func (c *tracingClient) GetDigest(ctx context.Context, token DockerHubToken, repository string, tag string) (digest DockerImageDigest, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetDigest"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.GetDigest(ctx, token, repository, tag)
}

func (c *tracingClient) GetDigestCached(ctx context.Context, repository string, tag string) (digest DockerImageDigest, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetDigestCached"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.GetDigestCached(ctx, repository, tag)
}

func (c *tracingClient) getSpanName(funcName string) string {
	return "dockerhubapi:" + funcName
}

func (c *tracingClient) handleError(span opentracing.Span, err error) {
	if err != nil {
		ext.Error.Set(span, true)
		span.LogFields(log.Error(err))
	}
}
