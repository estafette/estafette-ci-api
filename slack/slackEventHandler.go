package slack

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// EventHandler handles http events for Slack integration
type EventHandler interface {
	Handle(*gin.Context)
	HandleSlashCommand(slashCommand SlashCommand)
}

type eventHandlerImpl struct {
	eventsChannel                chan SlashCommand
	prometheusInboundEventTotals *prometheus.CounterVec
}

// NewSlackEventHandler returns a new slack.EventHandler
func NewSlackEventHandler(eventsChannel chan SlashCommand, prometheusInboundEventTotals *prometheus.CounterVec) EventHandler {
	return &eventHandlerImpl{
		eventsChannel:                eventsChannel,
		prometheusInboundEventTotals: prometheusInboundEventTotals,
	}
}

func (h *eventHandlerImpl) Handle(c *gin.Context) {

	// https://api.slack.com/slash-commands

	h.prometheusInboundEventTotals.With(prometheus.Labels{"event": "", "source": "slack"}).Inc()

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body from Slack webhook failed")
		c.String(http.StatusInternalServerError, "Reading body from Slack webhook failed")
		return
	}

	// unmarshal json body
	var b interface{}
	err = json.Unmarshal(body, &b)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body from Slack webhook failed")
		c.String(http.StatusInternalServerError, "Deserializing body from Slack webhook failed")
		return
	}

	log.Debug().
		Str("method", c.Request.Method).
		Str("url", c.Request.URL.String()).
		Interface("headers", c.Request.Header).
		Interface("body", b).
		Msg("Received webhook event from Slack...")

	// unmarshal json body
	var slashCommand SlashCommand
	err = json.Unmarshal(body, &slashCommand)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to SlashCommand failed")
		return
	}

	h.HandleSlashCommand(slashCommand)

	c.String(http.StatusOK, "Aye aye!")
}

func (h *eventHandlerImpl) HandleSlashCommand(slashCommand SlashCommand) {

	log.Debug().Interface("slashCommand", slashCommand).Msg("Deserialized slash command")

	// handle command via worker
	h.eventsChannel <- slashCommand
}
