package prometheus

import (
	"context"

	"github.com/opentracing/opentracing-go"
)

type tracingClient struct {
	Client
}

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c}
}

func (s *tracingClient) AwaitScrapeInterval(ctx context.Context) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("AwaitScrapeInterval"))
	defer span.Finish()

	s.Client.AwaitScrapeInterval(ctx)
}

func (s *tracingClient) GetMaxMemoryByPodName(ctx context.Context, podName string) (float64, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetMaxMemoryByPodName"))
	defer span.Finish()

	return s.Client.GetMaxMemoryByPodName(ctx, podName)
}

func (s *tracingClient) GetMaxCPUByPodName(ctx context.Context, podName string) (float64, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetMaxCPUByPodName"))
	defer span.Finish()

	return s.Client.GetMaxCPUByPodName(ctx, podName)
}

func (s *tracingClient) getSpanName(funcName string) string {
	return "prometheus:" + funcName
}
