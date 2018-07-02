package estafette

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/estafette/estafette-ci-api/cockroach"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// EventHandler handles events from estafette components
type EventHandler interface {
	Handle(*gin.Context)
	logRequest(string, *http.Request, []byte)
}

type eventHandlerImpl struct {
	config                       config.APIServerConfig
	ciBuilderEventsChannel       chan CiBuilderEvent
	buildJobLogsChannel          chan cockroach.BuildJobLogs
	prometheusInboundEventTotals *prometheus.CounterVec
}

// NewEstafetteEventHandler returns a new estafette.EventHandler
func NewEstafetteEventHandler(config config.APIServerConfig, ciBuilderEventsChannel chan CiBuilderEvent, buildJobLogsChannel chan cockroach.BuildJobLogs, prometheusInboundEventTotals *prometheus.CounterVec) EventHandler {
	return &eventHandlerImpl{
		config:                       config,
		ciBuilderEventsChannel:       ciBuilderEventsChannel,
		buildJobLogsChannel:          buildJobLogsChannel,
		prometheusInboundEventTotals: prometheusInboundEventTotals,
	}
}

func (h *eventHandlerImpl) Handle(c *gin.Context) {

	authorizationHeader := c.GetHeader("Authorization")
	if authorizationHeader != fmt.Sprintf("Bearer %v", h.config.APIKey) {
		log.Error().
			Str("authorizationHeader", authorizationHeader).
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

	go h.logRequest(eventType, c.Request, body)

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

		log.Debug().Interface("ciBuilderEvent", ciBuilderEvent).Msgf("Deserialized CiBuilderEvent event for job %v", ciBuilderEvent.JobName)

		// send via channel to worker
		h.ciBuilderEventsChannel <- ciBuilderEvent

		log.Debug().
			Str("jobName", ciBuilderEvent.JobName).
			Msgf("Received event of type '%v' from estafette-ci-builder for job %v...", eventType, ciBuilderEvent.JobName)

	case "builder:logs":

		// unmarshal json body
		var buildJobLogs cockroach.BuildJobLogs
		err = json.Unmarshal(body, &buildJobLogs)
		if err != nil {
			log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to BuildJobLogs failed")
			return
		}

		log.Debug().Interface("buildJobLogs", buildJobLogs).Msgf("Deserialized BuildJobLogs event for job %v", buildJobLogs.RepoFullName)

		// send via channel to worker
		h.buildJobLogsChannel <- buildJobLogs

	default:
		log.Warn().Str("event", eventType).Msgf("Unsupported Estafette event of type '%v'", eventType)
	}

	c.String(http.StatusOK, "Aye aye!")
}

func (h *eventHandlerImpl) logRequest(eventType string, request *http.Request, body []byte) {

	// unmarshal json body
	var b interface{}
	err := json.Unmarshal(body, &b)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body from Estafette 'build finished' event failed")
		return
	}
}
