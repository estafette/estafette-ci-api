package pubsubapi

import (
	"context"
	"time"

	"github.com/estafette/estafette-ci-api/pkg/api"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/go-kit/kit/metrics"
)

// NewMetricsClient returns a new instance of a metrics Client.
func NewMetricsClient(c Client, requestCount metrics.Counter, requestLatency metrics.Histogram) Client {
	return &metricsClient{c, requestCount, requestLatency}
}

type metricsClient struct {
	Client         Client
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
}

func (c *metricsClient) SubscriptionForTopic(ctx context.Context, message PubSubPushMessage) (event *manifest.EstafettePubSubEvent, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "SubscriptionForTopic", begin)
	}(time.Now())

	return c.Client.SubscriptionForTopic(ctx, message)
}

func (c *metricsClient) SubscribeToTopic(ctx context.Context, projectID, topicID string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "SubscribeToTopic", begin)
	}(time.Now())

	return c.Client.SubscribeToTopic(ctx, projectID, topicID)
}

func (c *metricsClient) SubscribeToPubsubTriggers(ctx context.Context, manifestString string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "SubscribeToPubsubTriggers", begin)
	}(time.Now())

	return c.Client.SubscribeToPubsubTriggers(ctx, manifestString)
}
