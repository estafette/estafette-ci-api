package pubsubapi

import (
	"encoding/base64"
	"strings"

	manifest "github.com/estafette/estafette-ci-manifest"
)

// PubSubPushMessage is a container for a pubsub push message
type PubSubPushMessage struct {
	Message      manifest.PubsubMessage `json:"message,omitempty"`
	Subscription string                 `json:"subscription,omitempty"`
}

// GetProject returns the project id for the pubsub subscription
func (m PubSubPushMessage) GetSubcriptionProject() string {
	return strings.Split(m.Subscription, "/")[1]
}

// GetSubscription returns the subscription name
func (m PubSubPushMessage) GetSubscriptionID() string {
	return strings.Split(m.Subscription, "/")[3]
}

// GetDecodedData returns the base64 decoded data
func (m PubSubPushMessage) GetDecodedData() string {
	data, err := base64.StdEncoding.DecodeString(m.Message.Data)
	if err != nil {
		return m.Message.Data
	}
	return string(data)
}
