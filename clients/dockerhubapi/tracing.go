package dockerhubapi

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

func (c *tracingClient) GetToken(ctx context.Context, repository string) (token DockerHubToken, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetToken"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetToken(ctx, repository)
}

func (c *tracingClient) GetDigest(ctx context.Context, token DockerHubToken, repository string, tag string) (digest DockerImageDigest, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetDigest"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetDigest(ctx, token, repository, tag)
}

func (c *tracingClient) GetDigestCached(ctx context.Context, repository string, tag string) (digest DockerImageDigest, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetDigestCached"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetDigestCached(ctx, repository, tag)
}

func (c *tracingClient) getSpanName(funcName string) string {
	return "dockerhubapi:" + funcName
}
