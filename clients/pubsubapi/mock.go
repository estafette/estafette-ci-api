package pubsubapi

import (
	"context"

	"github.com/estafette/estafette-ci-api/config"
	manifest "github.com/estafette/estafette-ci-manifest"
)

type MockClient struct {
	SubscriptionForTopicFunc      func(ctx context.Context, message PubSubPushMessage) (event *manifest.EstafettePubSubEvent, err error)
	SubscribeToTopicFunc          func(ctx context.Context, projectID, topicID string) (err error)
	SubscribeToPubsubTriggersFunc func(ctx context.Context, manifestString string) (err error)
}

func (c MockClient) SubscriptionForTopic(ctx context.Context, message PubSubPushMessage) (event *manifest.EstafettePubSubEvent, err error) {
	if c.SubscriptionForTopicFunc == nil {
		return
	}
	return c.SubscriptionForTopicFunc(ctx, message)
}

func (c MockClient) SubscribeToTopic(ctx context.Context, projectID, topicID string) (err error) {
	if c.SubscribeToTopicFunc == nil {
		return
	}
	return c.SubscribeToTopicFunc(ctx, projectID, topicID)
}

func (c MockClient) SubscribeToPubsubTriggers(ctx context.Context, manifestString string) (err error) {
	if c.SubscribeToPubsubTriggersFunc == nil {
		return
	}
	return c.SubscribeToPubsubTriggersFunc(ctx, manifestString)
}

func (c MockClient) RefreshConfig(config *config.APIConfig) {
}
