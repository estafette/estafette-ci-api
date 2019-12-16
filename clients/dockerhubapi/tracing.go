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

func (c *tracingClient) GetToken(ctx context.Context, repository string) (DockerHubToken, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetToken"))
	defer span.Finish()

	token, err := c.Client.GetToken(ctx, repository)
	c.handleError(span, err)

	return token, err
}

func (c *tracingClient) GetDigest(ctx context.Context, token DockerHubToken, repository string, tag string) (DockerImageDigest, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetDigest"))
	defer span.Finish()

	digest, err := c.Client.GetDigest(ctx, token, repository, tag)
	c.handleError(span, err)

	return digest, err
}

func (c *tracingClient) GetDigestCached(ctx context.Context, repository string, tag string) (DockerImageDigest, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetDigestCached"))
	defer span.Finish()

	digest, err := c.Client.GetDigestCached(ctx, repository, tag)
	c.handleError(span, err)

	return digest, err
}

func (c *tracingClient) getSpanName(funcName string) string {
	return "dockerhubapi:" + funcName
}

func (c *tracingClient) handleError(span opentracing.Span, err error) error {
	if err != nil {
		ext.Error.Set(span, true)
		span.LogFields(log.Error(err))
	}
	return err
}
