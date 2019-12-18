package pubsubapi

import (
	"context"

	"github.com/estafette/estafette-ci-api/helpers"
	manifest "github.com/estafette/estafette-ci-manifest"
)

// NewLoggingClient returns a new instance of a logging Client.
func NewLoggingClient(c Client) Client {
	return &loggingClient{c, "pubsubapi"}
}

type loggingClient struct {
	Client
	prefix string
}

func (c *loggingClient) SubscriptionForTopic(ctx context.Context, message PubSubPushMessage) (event *manifest.EstafettePubSubEvent, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "SubscriptionForTopic", err) }()

	return c.Client.SubscriptionForTopic(ctx, message)
}

func (c *loggingClient) SubscribeToTopic(ctx context.Context, projectID, topicID string) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "SubscribeToTopic", err) }()

	return c.Client.SubscribeToTopic(ctx, projectID, topicID)
}

func (c *loggingClient) SubscribeToPubsubTriggers(ctx context.Context, manifestString string) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "SubscribeToPubsubTriggers", err) }()

	return c.Client.SubscribeToPubsubTriggers(ctx, manifestString)
}
