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

// GetProject returns the project id for the pubsub subscription
func (m PubSubPushMessage) GetProject(topicProjectAttributeName string) string {
	var projectNameFromSubscription = strings.Split(m.Subscription, "/")[1]

	if m.Message.Attributes == nil {
		return projectNameFromSubscription
	}

	projectNameFromAttribute, ok := (*m.Message.Attributes)[topicProjectAttributeName]
	if !ok || len(projectNameFromAttribute) == 0 {
		return projectNameFromSubscription
	}

	return projectNameFromAttribute
}

// GetSubscription returns the subscription name
func (m PubSubPushMessage) GetSubscription() string {
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
