package cloudsource

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/estafette/estafette-ci-api/clients/cloudsourceapi"
	"github.com/estafette/estafette-ci-api/clients/pubsubapi"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func NewHandler(pubsubapiClient pubsubapi.Client, service Service) Handler {
	return Handler{
		pubsubapiClient: pubsubapiClient,
		service:         service,
	}
}

type Handler struct {
	pubsubapiClient pubsubapi.Client
	service         Service
}

func (h *Handler) PostPubsubEvent(c *gin.Context) {

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

	var notification cloudsourceapi.PubSubNotification
	byteData := []byte(message.GetDecodedData())
	if err := json.Unmarshal(byteData, &notification); err != nil {
		log.Error().Err(err).Msg("Failed unmarshalling pubsub notification")
	}
	log.Info().
		Interface("msg", message).
		Str("data", message.GetDecodedData()).
		Str("project", message.GetProject()).
		Str("subscription", message.GetSubscription()).
		Msg("Successfully binded pubsub push event")

	err = h.service.CreateJobForCloudSourcePush(c.Request.Context(), notification)
	if err != nil && !errors.Is(err, ErrNonCloneableEvent) && !errors.Is(err, ErrNoManifest) {
		c.String(http.StatusInternalServerError, "Oops, something went wrong!")
		return
	}

	c.String(http.StatusOK, "Aye aye!")
}
