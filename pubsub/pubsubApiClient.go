package pubsub

import (
	"context"
	"fmt"
	"strings"
	"time"

	ps "cloud.google.com/go/pubsub"
	"github.com/estafette/estafette-ci-api/config"
	pscontracts "github.com/estafette/estafette-ci-api/pubsub/contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
)

// APIClient is the interface for communicating with the pubsub apis
type APIClient interface {
	SubscriptionForTopic(ctx context.Context, message pscontracts.PubSubPushMessage) (*manifest.EstafettePubSubEvent, error)
	SubscribeToTopic(ctx context.Context, projectID, topicID string) error
	SubscribeToPubsubTriggers(ctx context.Context, manifestString string) error
}

type apiClient struct {
	config       config.PubsubConfig
	pubsubClient *ps.Client
}

// NewPubSubAPIClient returns a new pubsub.APIClient
func NewPubSubAPIClient(config config.PubsubConfig) (APIClient, error) {

	ctx := context.Background()
	pubsubClient, err := ps.NewClient(ctx, config.DefaultProject)
	if err != nil {
		return nil, err
	}

	return &apiClient{
		config:       config,
		pubsubClient: pubsubClient,
	}, nil
}

func (ac *apiClient) SubscriptionForTopic(ctx context.Context, message pscontracts.PubSubPushMessage) (*manifest.EstafettePubSubEvent, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, "PubSubApi::SubscriptionForTopic")
	defer span.Finish()

	projectID := message.GetProject()
	subscriptionName := message.GetSubscription()

	span.SetTag("project", projectID)
	span.SetTag("subscription", subscriptionName)

	if strings.HasSuffix(subscriptionName, ac.config.SubscriptionNameSuffix) {
		return &manifest.EstafettePubSubEvent{
			Project: projectID,
			Topic:   strings.TrimSuffix(subscriptionName, ac.config.SubscriptionNameSuffix),
		}, nil
	}

	subscription := ac.pubsubClient.SubscriptionInProject(subscriptionName, projectID)
	if subscription == nil {
		return nil, fmt.Errorf("Can't find subscription %v in project %v", subscriptionName, projectID)
	}

	subscriptionConfig, err := subscription.Config(context.Background())
	if err != nil {
		return nil, err
	}

	span.SetTag("topic", subscriptionConfig.Topic.ID())

	return &manifest.EstafettePubSubEvent{
		Project: projectID,
		Topic:   subscriptionConfig.Topic.ID(),
	}, nil
}

func (ac *apiClient) SubscribeToTopic(ctx context.Context, projectID, topicID string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, "PubSubApi::SubscribeToTopic")
	defer span.Finish()

	span.SetTag("project", projectID)
	span.SetTag("topic", topicID)

	// check if topic exists
	topic := ac.pubsubClient.TopicInProject(topicID, projectID)
	topicExists, err := topic.Exists(context.Background())
	if err != nil {
		return err
	}
	if !topicExists {
		return fmt.Errorf("Pub/Sub topic %v does not exist in project %v, cannot subscribe to it", topicID, projectID)
	}

	// check if subscription already exists
	subscriptionName := ac.getSubscriptionName(topicID)
	subscription := ac.pubsubClient.SubscriptionInProject(subscriptionName, projectID)
	log.Info().Msgf("Checking if subscription %v for topic %v in project %v exists...", subscriptionName, topicID, projectID)
	subscriptionExists, err := subscription.Exists(context.Background())
	if err != nil {
		return err
	}
	if subscriptionExists {
		// already exists, no need to do anything
		return nil
	}

	// create a subscription to the topic
	log.Info().Msgf("Creating subscription %v for topic %v in project %v...", subscriptionName, topicID, projectID)
	_, err = ac.pubsubClient.CreateSubscription(context.Background(), subscriptionName, ps.SubscriptionConfig{
		Topic: topic,
		PushConfig: ps.PushConfig{
			Endpoint: ac.config.Endpoint,
			AuthenticationMethod: &ps.OIDCToken{
				Audience:            ac.config.Audience,
				ServiceAccountEmail: ac.config.ServiceAccountEmail,
			},
		},
		AckDeadline:       10 * time.Second,
		RetentionDuration: 3 * time.Hour,
		ExpirationPolicy:  31 * 24 * time.Hour,
	})
	if err != nil {
		return err
	}

	log.Info().Msgf("Created subscription %v in project %v", subscriptionName, projectID)

	return nil
}

func (ac *apiClient) getSubscriptionName(topicName string) string {
	// it must start with a letter, and contain only letters ([A-Za-z]), numbers ([0-9]), dashes (-), underscores (_), periods (.), tildes (~), plus (+) or percent signs (%). It must be between 3 and 255 characters in length, and must not start with "goog".
	return topicName + ac.config.SubscriptionNameSuffix
}

func (ac *apiClient) SubscribeToPubsubTriggers(ctx context.Context, manifestString string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, "PubSubApi::SubscribeToPubsubTriggers")
	defer span.Finish()

	mft, err := manifest.ReadManifest(manifestString)
	if err != nil {
		return err
	}

	if len(mft.Triggers) > 0 {
		for _, t := range mft.Triggers {
			if t.PubSub != nil {
				err := ac.SubscribeToTopic(ctx, t.PubSub.Project, t.PubSub.Topic)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
