package prometheus

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/opentracing/opentracing-go"
)

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c, "prometheus"}
}

type tracingClient struct {
	Client Client
	prefix string
}

func (c *tracingClient) AwaitScrapeInterval(ctx context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "AwaitScrapeInterval"))
	defer func() { api.FinishSpan(span) }()

	c.Client.AwaitScrapeInterval(ctx)
}

func (c *tracingClient) GetMaxMemoryByPodName(ctx context.Context, podName string) (max float64, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetMaxMemoryByPodName"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetMaxMemoryByPodName(ctx, podName)
}

func (c *tracingClient) GetMaxCPUByPodName(ctx context.Context, podName string) (max float64, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetMaxCPUByPodName"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetMaxCPUByPodName(ctx, podName)
}
