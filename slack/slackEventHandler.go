package slack

import (
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
	slackAppVerificationToken    string
	eventsChannel                chan SlashCommand
	prometheusInboundEventTotals *prometheus.CounterVec
}

// NewSlackEventHandler returns a new slack.EventHandler
func NewSlackEventHandler(slackAppVerificationToken string, eventsChannel chan SlashCommand, prometheusInboundEventTotals *prometheus.CounterVec) EventHandler {
	return &eventHandlerImpl{
		slackAppVerificationToken:    slackAppVerificationToken,
		eventsChannel:                eventsChannel,
		prometheusInboundEventTotals: prometheusInboundEventTotals,
	}
}

func (h *eventHandlerImpl) Handle(c *gin.Context) {

	// https://api.slack.com/slash-commands

	h.prometheusInboundEventTotals.With(prometheus.Labels{"event": "", "source": "slack"}).Inc()

	var slashCommand SlashCommand
	// This will infer what binder to use depending on the content-type header.
	err := c.Bind(&slashCommand)
	if err != nil {
		log.Error().Err(err).Msg("Binding form data from Slack command webhook failed")
		c.String(http.StatusInternalServerError, "Binding form data from Slack command webhook failed")
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
