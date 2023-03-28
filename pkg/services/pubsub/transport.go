package pubsub

import (
	"net/http"

	"github.com/estafette/estafette-ci-api/pkg/clients/pubsubapi"
	"github.com/estafette/estafette-ci-api/pkg/services/estafette"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// NewHandler returns a pubsub.Handler
func NewHandler(pubsubapiClient pubsubapi.Client, estafetteService estafette.Service) Handler {
	return Handler{
		pubsubapiClient:  pubsubapiClient,
		estafetteService: estafetteService,
	}
}

type Handler struct {
	pubsubapiClient  pubsubapi.Client
	estafetteService estafette.Service
}

func (eh *Handler) PostPubsubEvent(c *gin.Context) {

	if c.MustGet(gin.AuthUserKey).(string) != "google-jwt" {
		c.Status(http.StatusUnauthorized)
		return
	}

	var message pubsubapi.PubSubPushMessage
	err := c.BindJSON(&message)
	if err != nil {
		errorMessage := "Binding PostPubsubEvent body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	pubsubEvent, err := eh.pubsubapiClient.SubscriptionForTopic(c.Request.Context(), message)
	if err != nil {
		log.Error().Err(err).Msgf("Failed retrieving topic for pubsub subscription %v", message.Subscription)
		c.String(http.StatusInternalServerError, "Oop, something's wrong!")
		return
	}
	if pubsubEvent == nil {
		log.Error().Msgf("Failed retrieving pubsubEvent for pubsub subscription %v", message.Subscription)
		c.String(http.StatusInternalServerError, "Oop, something's wrong!")
		return
	}

	log.Debug().
		Interface("msg", message).
		Str("data", message.GetDecodedData()).
		Str("subscriptionProject", message.GetSubcriptionProject()).
		Str("subscriptionID", message.GetSubscriptionID()).
		Str("topicProject", pubsubEvent.Project).
		Str("topicID", pubsubEvent.Topic).
		Msg("Successfully binded pubsub push event")

	err = eh.estafetteService.FirePubSubTriggers(c.Request.Context(), *pubsubEvent)
	if err != nil {
		log.Error().Err(err).Msgf("Failed firing pubsub triggers for topic %v in project %v", pubsubEvent.Topic, pubsubEvent.Project)
		c.String(http.StatusInternalServerError, "Oop, something's wrong!")
		return
	}

	c.String(http.StatusOK, "Aye aye!")
}
