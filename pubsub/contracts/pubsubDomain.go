package contracts

import (
	"encoding/base64"
	"strings"
)

// PubSubPushMessage is a container for a pubsub push message
type PubSubPushMessage struct {
	Message struct {
		Attributes  *map[string]string `json:"attributes,omitempty"`
		Data        string             `json:"data,omitempty"`
		MessageID   string             `json:"messageId,omitempty"`
		PublishTime string             `json:"publishTime,omitempty"`
	} `json:"message,omitempty"`
	Subscription string `json:"subscription,omitempty"`
}

// GetSubscriptionProject returns the project id for the subscription
func (m PubSubPushMessage) GetSubscriptionProject() string {
	return strings.Split(m.Subscription, "/")[1]
}

// GetSubscriptionID returns the unique identifier of the subscription within its project.
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
