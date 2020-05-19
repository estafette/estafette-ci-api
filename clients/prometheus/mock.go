package prometheus

import (
	"context"

	"github.com/estafette/estafette-ci-api/config"
)

type MockClient struct {
	AwaitScrapeIntervalFunc   func(ctx context.Context)
	GetMaxMemoryByPodNameFunc func(ctx context.Context, podName string) (max float64, err error)
	GetMaxCPUByPodNameFunc    func(ctx context.Context, podName string) (max float64, err error)
}

func (c MockClient) AwaitScrapeInterval(ctx context.Context) {
	if c.AwaitScrapeIntervalFunc == nil {
		return
	}
	c.AwaitScrapeIntervalFunc(ctx)
}

func (c MockClient) GetMaxMemoryByPodName(ctx context.Context, podName string) (max float64, err error) {
	if c.GetMaxMemoryByPodNameFunc == nil {
		return
	}
	return c.GetMaxMemoryByPodNameFunc(ctx, podName)
}

func (c MockClient) GetMaxCPUByPodName(ctx context.Context, podName string) (max float64, err error) {
	if c.GetMaxCPUByPodNameFunc == nil {
		return
	}
	return c.GetMaxCPUByPodNameFunc(ctx, podName)
}

func (c MockClient) RefreshConfig(config *config.APIConfig) {
}
