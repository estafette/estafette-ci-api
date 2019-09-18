package pubsub

import (
	"context"
	"fmt"

	ps "cloud.google.com/go/pubsub"
	pscontracts "github.com/estafette/estafette-ci-api/pubsub/contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
)

// APIClient is the interface for communicating with the pubsub apis
type APIClient interface {
	SubscriptionToTopic(message pscontracts.PubSubPushMessage) (*manifest.EstafettePubSubEvent, error)
}

type apiClient struct {
}

// NewPubSubAPIClient returns a new pubsub.APIClient
func NewPubSubAPIClient() APIClient {
	return &apiClient{}
}

func (ac *apiClient) SubscriptionToTopic(message pscontracts.PubSubPushMessage) (*manifest.EstafettePubSubEvent, error) {

	projectID := message.GetProject()

	ctx := context.Background()
	pubsubClient, err := ps.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	subscriptionName := message.GetSubscription()

	subscription := pubsubClient.SubscriptionInProject(subscriptionName, projectID)
	if subscription == nil {
		return nil, fmt.Errorf("Can't find subscription %v in project %v", subscriptionName, projectID)
	}

	subscriptionConfig, err := subscription.Config(ctx)
	if err != nil {
		return nil, err
	}

	return &manifest.EstafettePubSubEvent{
		Project: projectID,
		Topic:   subscriptionConfig.Topic.ID(),
	}, nil
}
