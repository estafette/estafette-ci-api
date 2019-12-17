package prometheus

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

func (c *tracingClient) AwaitScrapeInterval(ctx context.Context) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("AwaitScrapeInterval"))
	defer func() { helpers.FinishSpan(span) }()

	c.Client.AwaitScrapeInterval(ctx)
}

func (c *tracingClient) GetMaxMemoryByPodName(ctx context.Context, podName string) (max float64, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetMaxMemoryByPodName"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetMaxMemoryByPodName(ctx, podName)
}

func (c *tracingClient) GetMaxCPUByPodName(ctx context.Context, podName string) (max float64, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetMaxCPUByPodName"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetMaxCPUByPodName(ctx, podName)
}

func (c *tracingClient) getSpanName(funcName string) string {
	return "prometheus:" + funcName
}
