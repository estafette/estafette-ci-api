package pubsubapi

import (
	"context"

	manifest "github.com/estafette/estafette-ci-manifest"
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

func (c *tracingClient) SubscriptionForTopic(ctx context.Context, message PubSubPushMessage) (event *manifest.EstafettePubSubEvent, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("SubscriptionForTopic"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.SubscriptionForTopic(ctx, message)
}

func (c *tracingClient) SubscribeToTopic(ctx context.Context, projectID, topicID string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("SubscribeToTopic"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.SubscribeToTopic(ctx, projectID, topicID)
}

func (c *tracingClient) SubscribeToPubsubTriggers(ctx context.Context, manifestString string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("SubscribeToPubsubTriggers"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.SubscribeToPubsubTriggers(ctx, manifestString)
}

func (c *tracingClient) getSpanName(funcName string) string {
	return "pubsubapi:" + funcName
}

func (c *tracingClient) handleError(span opentracing.Span, err error) {
	if err != nil {
		ext.Error.Set(span, true)
		span.LogFields(log.Error(err))
	}
}
