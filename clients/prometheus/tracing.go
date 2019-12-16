package prometheus

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

func (c *tracingClient) AwaitScrapeInterval(ctx context.Context) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("AwaitScrapeInterval"))
	defer span.Finish()

	c.Client.AwaitScrapeInterval(ctx)
}

func (c *tracingClient) GetMaxMemoryByPodName(ctx context.Context, podName string) (max float64, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetMaxMemoryByPodName"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.GetMaxMemoryByPodName(ctx, podName)
}

func (c *tracingClient) GetMaxCPUByPodName(ctx context.Context, podName string) (max float64, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetMaxCPUByPodName"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.GetMaxCPUByPodName(ctx, podName)
}

func (c *tracingClient) getSpanName(funcName string) string {
	return "prometheus:" + funcName
}

func (c *tracingClient) handleError(span opentracing.Span, err error) {
	if err != nil {
		ext.Error.Set(span, true)
		span.LogFields(log.Error(err))
	}
}
