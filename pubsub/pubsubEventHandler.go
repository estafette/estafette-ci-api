package pubsub

import (
	"fmt"
	"net/http"

	"github.com/estafette/estafette-ci-api/estafette"
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
	apiClient    APIClient
	buildService estafette.BuildService
}

// NewPubSubEventHandler returns a pubsub.EventHandler
func NewPubSubEventHandler(apiClient APIClient, buildService estafette.BuildService) EventHandler {
	return &eventHandler{
		apiClient:    apiClient,
		buildService: buildService,
	}
}

func (eh *eventHandler) PostPubsubEvent(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "PubSub::PostPubsubEvent")
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

	pubsubEvent, err := eh.apiClient.SubscriptionForTopic(ctx, message)
	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving topic for pubsub subscription")
		c.String(http.StatusInternalServerError, "Oop, something's wrong!")
		return
	}
	if pubsubEvent == nil {
		log.Error().Msg("Failed retrieving pubsubEvent for pubsub subscription")
		c.String(http.StatusInternalServerError, "Oop, something's wrong!")
		return
	}

	log.Info().
		Interface("msg", message).
		Str("data", message.GetDecodedData()).
		Str("subscriptionProject", message.GetSubscriptionProject()).
		Str("attributes", fmt.Sprintf("%v", message.GetAttributes())).
		Str("subscription", message.GetSubscription()).
		Str("topic", pubsubEvent.Topic).
		Msg("Successfully binded pubsub push event")

	err = eh.buildService.FirePubSubTriggers(ctx, *pubsubEvent)
	if err != nil {
		log.Error().Err(err).Msgf("Failed firing pubsub triggers for topic %v in project %v", pubsubEvent.Topic, pubsubEvent.Project)
		c.String(http.StatusInternalServerError, "Oop, something's wrong!")
		return
	}

	c.String(http.StatusOK, "Aye aye!")
	return
}
