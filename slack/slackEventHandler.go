package slack

import (
	"fmt"
	"net/http"
	"strings"

	crypt "github.com/estafette/estafette-ci-crypt"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// EventHandler handles http events for Slack integration
type EventHandler interface {
	Handle(*gin.Context)
	HasValidVerificationToken(SlashCommand) bool
}

type eventHandlerImpl struct {
	secretHelper                 crypt.SecretHelper
	slackAppVerificationToken    string
	eventsChannel                chan SlashCommand
	prometheusInboundEventTotals *prometheus.CounterVec
}

// NewSlackEventHandler returns a new slack.EventHandler
func NewSlackEventHandler(secretHelper crypt.SecretHelper, slackAppVerificationToken string, eventsChannel chan SlashCommand, prometheusInboundEventTotals *prometheus.CounterVec) EventHandler {
	return &eventHandlerImpl{
		secretHelper:                 secretHelper,
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

	log.Debug().Interface("slashCommand", slashCommand).Msg("Deserialized slash command")

	hasValidVerificationToken := h.HasValidVerificationToken(slashCommand)
	if !hasValidVerificationToken {
		log.Warn().Str("expectedToken", h.slackAppVerificationToken).Str("actualToken", slashCommand.Token).Msg("Verification token for Slack command is invalid")
		c.String(http.StatusBadRequest, "Verification token for Slack command is invalid")
		return
	}

	if slashCommand.Command == "/estafette" {
		if slashCommand.Text != "" {
			splittedText := strings.Split(slashCommand.Text, " ")
			if splittedText != nil && len(splittedText) > 0 {
				command := splittedText[0]
				arguments := splittedText[1:len(splittedText)]
				switch command {
				case "encrypt":

					encryptedString, err := h.secretHelper.Encrypt(strings.Join(arguments, " "))
					if err != nil {
						log.Error().Err(err).Interface("slashCommand", slashCommand).Msg("Failed to encrypt secret")
						c.String(http.StatusOK, "Incorrect usage of /estafette encrypt!")
						return
					}

					c.String(http.StatusOK, fmt.Sprintf("estafette.secret(%v)", encryptedString))
					return
				}
			}
		}
	}

	// handle command via worker
	//h.eventsChannel <- slashCommand

	c.String(http.StatusOK, "Aye aye!")
}

func (h *eventHandlerImpl) HasValidVerificationToken(slashCommand SlashCommand) bool {
	return slashCommand.Token == h.slackAppVerificationToken
}
