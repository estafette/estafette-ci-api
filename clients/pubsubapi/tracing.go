package pubsubapi

import (
	"context"

	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/opentracing/opentracing-go"
)

type tracingClient struct {
	Client
}

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c}
}

func (s *tracingClient) SubscriptionForTopic(ctx context.Context, message PubSubPushMessage) (*manifest.EstafettePubSubEvent, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("SubscriptionForTopic"))
	defer span.Finish()

	return s.Client.SubscriptionForTopic(ctx, message)
}

func (s *tracingClient) SubscribeToTopic(ctx context.Context, projectID, topicID string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("SubscribeToTopic"))
	defer span.Finish()

	return s.Client.SubscribeToTopic(ctx, projectID, topicID)
}

func (s *tracingClient) SubscribeToPubsubTriggers(ctx context.Context, manifestString string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("SubscribeToPubsubTriggers"))
	defer span.Finish()

	return s.Client.SubscribeToPubsubTriggers(ctx, manifestString)
}

func (s *tracingClient) getSpanName(funcName string) string {
	return "pubsubapi:" + funcName
}
