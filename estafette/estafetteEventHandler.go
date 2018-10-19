package estafette

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/estafette/estafette-ci-api/config"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// EventHandler handles events from estafette components
type EventHandler interface {
	Handle(*gin.Context)
}

type eventHandlerImpl struct {
	config                       config.APIServerConfig
	ciBuilderEventsChannel       chan CiBuilderEvent
	prometheusInboundEventTotals *prometheus.CounterVec
}

// NewEstafetteEventHandler returns a new estafette.EventHandler
func NewEstafetteEventHandler(config config.APIServerConfig, ciBuilderEventsChannel chan CiBuilderEvent, prometheusInboundEventTotals *prometheus.CounterVec) EventHandler {
	return &eventHandlerImpl{
		config:                       config,
		ciBuilderEventsChannel:       ciBuilderEventsChannel,
		prometheusInboundEventTotals: prometheusInboundEventTotals,
	}
}

func (h *eventHandlerImpl) Handle(c *gin.Context) {

	if c.MustGet(gin.AuthUserKey).(string) != "apiKey" {
		c.AbortWithStatus(http.StatusUnauthorized)
	}

	eventType := c.GetHeader("X-Estafette-Event")
	h.prometheusInboundEventTotals.With(prometheus.Labels{"event": eventType, "source": "estafette"}).Inc()

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body from Estafette 'build finished' event failed")
		c.String(http.StatusInternalServerError, "Reading body from Estafette 'build finished' event failed")
		return
	}

	switch eventType {
	case
		"builder:nomanifest",
		"builder:succeeded",
		"builder:failed":

		// unmarshal json body
		var ciBuilderEvent CiBuilderEvent
		err = json.Unmarshal(body, &ciBuilderEvent)
		if err != nil {
			log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to CiBuilderEvent failed")
			return
		}

		// send via channel to worker
		h.ciBuilderEventsChannel <- ciBuilderEvent

	default:
		log.Warn().Str("event", eventType).Msgf("Unsupported Estafette event of type '%v'", eventType)
	}

	c.String(http.StatusOK, "Aye aye!")
}
