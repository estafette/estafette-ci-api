package estafette

import "strings"

// PubSubPushMessage is a container for a pubsub push message
type PubSubPushMessage struct {
	Message struct {
		Attributes  map[string]string `json:"attributes"`
		Data        string            `json:"data"`
		MessageID   string            `json:"messageId"`
		PublishTime string            `json:"publishTime"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}

// GetProject returns the project id for the pubsub subscription
func (m PubSubPushMessage) GetProject() string {
	return strings.Split(m.Subscription, "/")[1]
}

// GetSubscription returns the subscription name
func (m PubSubPushMessage) GetSubscription() string {
	return strings.Split(m.Subscription, "/")[3]
}
