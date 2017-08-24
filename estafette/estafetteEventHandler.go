package estafette

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// EventHandler handles events from estafette components
type EventHandler interface {
	Handle(*gin.Context)
}

type eventHandlerImpl struct {
	ciAPIKey                     string
	eventsChannel                chan CiBuilderEvent
	prometheusInboundEventTotals *prometheus.CounterVec
}

// NewEstafetteEventHandler returns a new estafette.EventHandler
func NewEstafetteEventHandler(ciAPIKey string, eventsChannel chan CiBuilderEvent, prometheusInboundEventTotals *prometheus.CounterVec) EventHandler {
	return &eventHandlerImpl{
		ciAPIKey:                     ciAPIKey,
		eventsChannel:                eventsChannel,
		prometheusInboundEventTotals: prometheusInboundEventTotals,
	}
}

func (h *eventHandlerImpl) Handle(c *gin.Context) {

	authorizationHeader := c.GetHeader("Authorization")
	if authorizationHeader != fmt.Sprintf("Bearer %v", h.ciAPIKey) {
		log.Error().
			Str("authorizationHeader", authorizationHeader).
			Str("apiKey", h.ciAPIKey).
			Msg("Authorization header for Estafette event is incorrect")
		c.String(http.StatusUnauthorized, "Authorization failed")
		return
	}

	eventType := c.GetHeader("X-Estafette-Event")
	h.prometheusInboundEventTotals.With(prometheus.Labels{"event": eventType, "source": "estafette"}).Inc()

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body from Estafette 'build finished' event failed")
		c.String(http.StatusInternalServerError, "Reading body from Estafette 'build finished' event failed")
		return
	}

	// unmarshal json body
	var b interface{}
	err = c.BindJSON(&b)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body from Estafette 'build finished' event failed")
		c.String(http.StatusInternalServerError, "Deserializing body from Estafette 'build finished' event failed")
		return
	}

	// unmarshal json body
	var ciBuilderEvent CiBuilderEvent
	err = json.Unmarshal(body, &ciBuilderEvent)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to EstafetteCiBuilderEvent failed")
		return
	}

	log.Debug().Interface("event", ciBuilderEvent).Msgf("Deserialized Estafette event for job %v", ciBuilderEvent.JobName)

	switch eventType {
	case
		"builder:nomanifest",
		"builder:succeeded",
		"builder:failed":
		// send via channel to worker
		h.eventsChannel <- ciBuilderEvent

	default:
		log.Warn().Str("event", eventType).Msgf("Unsupported Estafette event of type '%v'", eventType)
	}

	log.Debug().
		Str("jobName", ciBuilderEvent.JobName).
		Msgf("Received event of type '%v' from estafette-ci-builder for job %v...", eventType, ciBuilderEvent.JobName)

	c.String(http.StatusOK, "Aye aye!")
}
