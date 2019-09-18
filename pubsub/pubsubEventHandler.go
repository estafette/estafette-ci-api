package pubsub

import (
	"net/http"

	pscontracts "github.com/estafette/estafette-ci-api/pubsub/contracts"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
)

// EventHandler handles http events for Pubsub integration
type EventHandler interface {
	PostPubsubEvent(*gin.Context)
}

type eventHandler struct {
	apiClient APIClient
}

// NewPubSubEventHandler returns a pubsub.EventHandler
func NewPubSubEventHandler(apiClient APIClient) EventHandler {
	return &eventHandler{
		apiClient: apiClient,
	}
}

func (eh *eventHandler) PostPubsubEvent(c *gin.Context) {

	span, _ := opentracing.StartSpanFromContext(c.Request.Context(), "Api::PostPubsubEvent")
	defer span.Finish()

	if c.MustGet(gin.AuthUserKey).(string) != "google-jwt" {
		c.Status(http.StatusUnauthorized)
		return
	}

	var message pscontracts.PubSubPushMessage
	err := c.BindJSON(&message)
	if err != nil {
		log.Error().Err(err).Msg("Failed binding pubsub push event")
		c.String(http.StatusInternalServerError, "Oop, something's wrong!")
		return
	}

	pubsubEvent, err := eh.apiClient.SubscriptionToTopic(message)
	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving topic for pubsub subscription")
		c.String(http.StatusInternalServerError, "Oop, something's wrong!")
		return
	}

	log.Info().
		Interface("msg", message).
		Str("data", message.GetDecodedData()).
		Str("project", message.GetProject()).
		Str("subscription", message.GetSubscription()).
		Str("topic", pubsubEvent.Topic).
		Msg("Successfully binded pubsub push event")

	c.String(http.StatusOK, "Aye aye!")
	return
}
