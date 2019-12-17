package pubsubapi

import (
	"context"

	"github.com/estafette/estafette-ci-api/helpers"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/opentracing/opentracing-go"
)

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c}
}

type tracingClient struct {
	Client
}

func (c *tracingClient) SubscriptionForTopic(ctx context.Context, message PubSubPushMessage) (event *manifest.EstafettePubSubEvent, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("SubscriptionForTopic"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.SubscriptionForTopic(ctx, message)
}

func (c *tracingClient) SubscribeToTopic(ctx context.Context, projectID, topicID string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("SubscribeToTopic"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.SubscribeToTopic(ctx, projectID, topicID)
}

func (c *tracingClient) SubscribeToPubsubTriggers(ctx context.Context, manifestString string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("SubscribeToPubsubTriggers"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.SubscribeToPubsubTriggers(ctx, manifestString)
}

func (c *tracingClient) getSpanName(funcName string) string {
	return "pubsubapi:" + funcName
}
