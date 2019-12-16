package pubsub

import (
	"net/http"

	"github.com/estafette/estafette-ci-api/clients/pubsubapi"
	"github.com/estafette/estafette-ci-api/services/estafette"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
)

// Service handles http events for Pubsub integration
type Service interface {
	PostPubsubEvent(*gin.Context)
}

type service struct {
	apiClient    pubsubapi.Client
	buildService estafette.Service
}

// NewService returns a pubsub.Service
func NewService(apiClient pubsubapi.Client, buildService estafette.Service) Service {
	return &service{
		apiClient:    apiClient,
		buildService: buildService,
	}
}

func (eh *service) PostPubsubEvent(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "PubSub::PostPubsubEvent")
	defer span.Finish()

	if c.MustGet(gin.AuthUserKey).(string) != "google-jwt" {
		c.Status(http.StatusUnauthorized)
		return
	}

	var message pubsubapi.PubSubPushMessage
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
		Str("project", message.GetProject()).
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
