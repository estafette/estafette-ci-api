package pubsubapi

import (
	"context"

	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/helpers"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/opentracing/opentracing-go"
)

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c, "pubsubapi"}
}

type tracingClient struct {
	Client
	prefix string
}

func (c *tracingClient) SubscriptionForTopic(ctx context.Context, message PubSubPushMessage) (event *manifest.EstafettePubSubEvent, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "SubscriptionForTopic"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.SubscriptionForTopic(ctx, message)
}

func (c *tracingClient) SubscribeToTopic(ctx context.Context, projectID, topicID string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "SubscribeToTopic"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.SubscribeToTopic(ctx, projectID, topicID)
}

func (c *tracingClient) SubscribeToPubsubTriggers(ctx context.Context, manifestString string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "SubscribeToPubsubTriggers"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.SubscribeToPubsubTriggers(ctx, manifestString)
}

func (c *tracingClient) RefreshConfig(config *config.APIConfig, manifestPreferences manifest.EstafetteManifestPreferences) {
	c.Client.RefreshConfig(config, manifestPreferences)
}
