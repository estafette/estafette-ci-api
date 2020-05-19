package pubsubapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	stdpubsub "cloud.google.com/go/pubsub"
	"github.com/estafette/estafette-ci-api/config"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/rs/zerolog/log"
)

// Client is the interface for communicating with the pubsub apis
type Client interface {
	SubscriptionForTopic(ctx context.Context, message PubSubPushMessage) (event *manifest.EstafettePubSubEvent, err error)
	SubscribeToTopic(ctx context.Context, projectID, topicID string) (err error)
	SubscribeToPubsubTriggers(ctx context.Context, manifestString string) (err error)
}

// NewClient returns a new pubsub.Client
func NewClient(config *config.APIConfig, pubsubClient *stdpubsub.Client) Client {
	return &client{
		config:       config,
		pubsubClient: pubsubClient,
	}
}

type client struct {
	config       *config.APIConfig
	pubsubClient *stdpubsub.Client
}

func (c *client) SubscriptionForTopic(ctx context.Context, message PubSubPushMessage) (*manifest.EstafettePubSubEvent, error) {

	projectID := message.GetProject()
	subscriptionName := message.GetSubscription()

	if strings.HasSuffix(subscriptionName, c.config.Integrations.Pubsub.SubscriptionNameSuffix) {
		return &manifest.EstafettePubSubEvent{
			Project: projectID,
			Topic:   strings.TrimSuffix(subscriptionName, c.config.Integrations.Pubsub.SubscriptionNameSuffix),
		}, nil
	}

	subscription := c.pubsubClient.SubscriptionInProject(subscriptionName, projectID)
	if subscription == nil {
		return nil, fmt.Errorf("Can't find subscription %v in project %v", subscriptionName, projectID)
	}

	subscriptionConfig, err := subscription.Config(context.Background())
	if err != nil {
		return nil, err
	}

	return &manifest.EstafettePubSubEvent{
		Project: projectID,
		Topic:   subscriptionConfig.Topic.ID(),
	}, nil
}

func (c *client) SubscribeToTopic(ctx context.Context, projectID, topicID string) error {

	// check if topic exists
	topic := c.pubsubClient.TopicInProject(topicID, projectID)
	topicExists, err := topic.Exists(context.Background())
	if err != nil {
		return err
	}
	if !topicExists {
		return fmt.Errorf("Pub/Sub topic %v does not exist in project %v, cannot subscribe to it", topicID, projectID)
	}

	// check if subscription already exists
	subscriptionName := c.getSubscriptionName(topicID)
	subscription := c.pubsubClient.SubscriptionInProject(subscriptionName, projectID)
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
	_, err = c.pubsubClient.CreateSubscription(context.Background(), subscriptionName, stdpubsub.SubscriptionConfig{
		Topic: topic,
		PushConfig: stdpubsub.PushConfig{
			Endpoint: c.config.Integrations.Pubsub.Endpoint,
			AuthenticationMethod: &stdpubsub.OIDCToken{
				Audience:            c.config.Integrations.Pubsub.Audience,
				ServiceAccountEmail: c.config.Integrations.Pubsub.ServiceAccountEmail,
			},
		},
		AckDeadline:       20 * time.Second,
		RetentionDuration: 3 * time.Hour,
		ExpirationPolicy:  time.Duration(c.config.Integrations.Pubsub.SubscriptionIdleExpirationDays) * 24 * time.Hour,
	})
	if err != nil {
		return err
	}

	log.Info().Msgf("Created subscription %v in project %v", subscriptionName, projectID)

	return nil
}

func (c *client) getSubscriptionName(topicName string) string {
	// it must start with a letter, and contain only letters ([A-Za-z]), numbers ([0-9]), dashes (-), underscores (_), periods (.), tildes (~), plus (+) or percent signs (%). It must be between 3 and 255 characters in length, and must not start with "goog".
	return topicName + c.config.Integrations.Pubsub.SubscriptionNameSuffix
}

func (c *client) SubscribeToPubsubTriggers(ctx context.Context, manifestString string) error {

	mft, err := manifest.ReadManifest(c.config.ManifestPreferences, manifestString)
	if err != nil {
		return err
	}

	if len(mft.Triggers) > 0 {
		for _, t := range mft.Triggers {
			if t.PubSub != nil {
				err := c.SubscribeToTopic(ctx, t.PubSub.Project, t.PubSub.Topic)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
