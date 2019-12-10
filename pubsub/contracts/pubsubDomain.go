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

// GetTopicProject returns the project id which owns the topic
func (m PubSubPushMessage) GetTopicProject(topicProjectAttributeName string) string {
	var subscriptionProject = m.GetSubscriptionProject()

	attributes := m.GetAttributes()
	if attributes == nil {
		return subscriptionProject
	}

	topicOwnerProject, ok := attributes[topicProjectAttributeName]
	if !ok || len(topicOwnerProject) == 0 {
		return subscriptionProject
	}

	return topicOwnerProject
}

// GetSubscriptionProject returns the project id for the pubsub subscription
func (m PubSubPushMessage) GetSubscriptionProject() string {
	return strings.Split(m.Subscription, "/")[1]
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

// GetAttributes returns the attributes of the message
func (m PubSubPushMessage) GetAttributes() map[string]string {
	if m.Message.Attributes == nil {
		return nil
	}

	return (*m.Message.Attributes)
}
