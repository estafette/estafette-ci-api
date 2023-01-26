package pubsubapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	stdpubsub "cloud.google.com/go/pubsub"
	"github.com/estafette/estafette-ci-api/pkg/api"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/rs/zerolog/log"
)

// Client is the interface for communicating with the pubsub apis
//
//go:generate mockgen -package=pubsubapi -destination ./mock.go -source=client.go
type Client interface {
	SubscriptionForTopic(ctx context.Context, message PubSubPushMessage) (event *manifest.EstafettePubSubEvent, err error)
	SubscribeToTopic(ctx context.Context, projectID, topicID string) (err error)
	SubscribeToPubsubTriggers(ctx context.Context, manifestString string) (err error)
}

// NewClient returns a new pubsub.Client
func NewClient(config *api.APIConfig, pubsubClient *stdpubsub.Client) Client {
	if config == nil || config.Integrations == nil || config.Integrations.Pubsub == nil || !config.Integrations.Pubsub.Enable {
		return &client{
			enabled: false,
		}
	}

	return &client{
		enabled:      true,
		config:       config,
		pubsubClient: pubsubClient,
	}
}

type client struct {
	enabled      bool
	config       *api.APIConfig
	pubsubClient *stdpubsub.Client
}

func (c *client) SubscriptionForTopic(ctx context.Context, message PubSubPushMessage) (*manifest.EstafettePubSubEvent, error) {

	if !c.enabled {
		return nil, nil
	}

	subscriptionProjectID := message.GetSubcriptionProject()
	subscriptionName := message.GetSubscriptionID()

	subscription := c.pubsubClient.SubscriptionInProject(subscriptionName, subscriptionProjectID)
	if subscription == nil {
		return nil, fmt.Errorf("Can't find subscription %v in project %v", subscriptionName, subscriptionProjectID)
	}

	subscriptionConfig, err := subscription.Config(ctx)
	if err != nil {
		return nil, err
	}

	topicID := subscriptionConfig.Topic.ID()
	topicSlices := strings.Split(subscriptionConfig.Topic.String(), "/")
	var topicProjectID string
	if len(topicSlices) > 1 {
		topicProjectID = topicSlices[1]
	}

	return &manifest.EstafettePubSubEvent{
		Project: topicProjectID,
		Topic:   topicID,
		Message: message.Message,
	}, nil
}

func (c *client) SubscribeToTopic(ctx context.Context, projectID, topicID string) error {

	if !c.enabled {
		return nil
	}

	// check if topic exists
	topic := c.pubsubClient.TopicInProject(topicID, projectID)
	topicExists, err := topic.Exists(ctx)
	if err != nil {
		return err
	}
	if !topicExists {
		return fmt.Errorf("Pub/Sub topic %v does not exist in project %v, cannot subscribe to it", topicID, projectID)
	}

	// check if subscription already exists
	subscriptionName := c.getSubscriptionName(topicID)
	subscription := c.pubsubClient.SubscriptionInProject(subscriptionName, projectID)
	log.Debug().Msgf("Checking if subscription %v for topic %v in project %v exists...", subscriptionName, topicID, projectID)
	subscriptionExists, err := subscription.Exists(ctx)
	if err != nil {
		return err
	}
	if subscriptionExists {
		// already exists, no need to do anything
		return nil
	}

	// create a subscription to the topic
	log.Debug().Msgf("Creating subscription %v for topic %v in project %v...", subscriptionName, topicID, projectID)
	_, err = c.pubsubClient.CreateSubscription(ctx, subscriptionName, stdpubsub.SubscriptionConfig{
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

	log.Debug().Msgf("Created subscription %v in project %v", subscriptionName, projectID)

	return nil
}

func (c *client) getSubscriptionName(topicName string) string {
	// it must start with a letter, and contain only letters ([A-Za-z]), numbers ([0-9]), dashes (-), underscores (_), periods (.), tildes (~), plus (+) or percent signs (%). It must be between 3 and 255 characters in length, and must not start with "goog".
	return topicName + c.config.Integrations.Pubsub.SubscriptionNameSuffix
}

func (c *client) SubscribeToPubsubTriggers(ctx context.Context, manifestString string) error {

	if !c.enabled {
		return nil
	}

	mft, err := manifest.ReadManifest(c.config.ManifestPreferences, manifestString, false)
	if err != nil {
		return err
	}

	for _, t := range mft.Triggers {
		if t.PubSub != nil {
			err := c.SubscribeToTopic(ctx, t.PubSub.Project, t.PubSub.Topic)
			if err != nil {
				return err
			}
		}
	}

	for _, r := range mft.Releases {
		for _, t := range r.Triggers {
			if t.PubSub != nil {
				err := c.SubscribeToTopic(ctx, t.PubSub.Project, t.PubSub.Topic)
				if err != nil {
					return err
				}
			}
		}
	}

	for _, b := range mft.Bots {
		for _, t := range b.Triggers {
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
