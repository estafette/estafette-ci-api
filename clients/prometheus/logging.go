package prometheus

import (
	"context"

	"github.com/estafette/estafette-ci-api/helpers"
)

// NewLoggingClient returns a new instance of a logging Client.
func NewLoggingClient(c Client) Client {
	return &loggingClient{c, "prometheus"}
}

type loggingClient struct {
	Client
	prefix string
}

func (c *loggingClient) AwaitScrapeInterval(ctx context.Context) {
	c.Client.AwaitScrapeInterval(ctx)
}

func (c *loggingClient) GetMaxMemoryByPodName(ctx context.Context, podName string) (max float64, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetMaxMemoryByPodName", err) }()

	return c.Client.GetMaxMemoryByPodName(ctx, podName)
}

func (c *loggingClient) GetMaxCPUByPodName(ctx context.Context, podName string) (max float64, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetMaxCPUByPodName", err) }()

	return c.Client.GetMaxCPUByPodName(ctx, podName)
}
