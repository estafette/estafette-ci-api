package prometheus

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
)

type tracingClient struct {
	Client
}

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c}
}

func (c *tracingClient) AwaitScrapeInterval(ctx context.Context) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("AwaitScrapeInterval"))
	defer span.Finish()

	c.Client.AwaitScrapeInterval(ctx)
}

func (c *tracingClient) GetMaxMemoryByPodName(ctx context.Context, podName string) (float64, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetMaxMemoryByPodName"))
	defer span.Finish()

	max, err := c.Client.GetMaxMemoryByPodName(ctx, podName)
	c.handleError(span, err)

	return max, err
}

func (c *tracingClient) GetMaxCPUByPodName(ctx context.Context, podName string) (float64, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetMaxCPUByPodName"))
	defer span.Finish()

	max, err := c.Client.GetMaxCPUByPodName(ctx, podName)
	c.handleError(span, err)

	return max, err
}

func (c *tracingClient) getSpanName(funcName string) string {
	return "prometheus:" + funcName
}

func (c *tracingClient) handleError(span opentracing.Span, err error) error {
	if err != nil {
		ext.Error.Set(span, true)
		span.LogFields(log.Error(err))
	}
	return err
}
