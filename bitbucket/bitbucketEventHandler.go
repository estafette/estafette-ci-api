package bitbucket

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// EventHandler handles http events for Bitbucket integration
type EventHandler interface {
	Handle(*gin.Context)
	HandlePushEvent(pushEvent RepositoryPushEvent)
	logRequest(string, *http.Request, []byte)
}

type eventHandlerImpl struct {
	eventsChannel                chan RepositoryPushEvent
	prometheusInboundEventTotals *prometheus.CounterVec
}

// NewBitbucketEventHandler returns a new bitbucket.EventHandler
func NewBitbucketEventHandler(eventsChannel chan RepositoryPushEvent, prometheusInboundEventTotals *prometheus.CounterVec) EventHandler {
	return &eventHandlerImpl{
		eventsChannel:                eventsChannel,
		prometheusInboundEventTotals: prometheusInboundEventTotals,
	}
}

func (h *eventHandlerImpl) Handle(c *gin.Context) {

	// https://confluence.atlassian.com/bitbucket/manage-webhooks-735643732.html

	eventType := c.GetHeader("X-Event-Key")
	h.prometheusInboundEventTotals.With(prometheus.Labels{"event": eventType, "source": "bitbucket"}).Inc()

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body from Bitbucket webhook failed")
		c.String(http.StatusInternalServerError, "Reading body from Bitbucket webhook failed")
		return
	}

	// unmarshal json body in background
	go h.logRequest(eventType, c.Request, body)

	switch eventType {
	case "repo:push":

		// unmarshal json body
		var pushEvent RepositoryPushEvent
		err := json.Unmarshal(body, &pushEvent)
		if err != nil {
			log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to BitbucketRepositoryPushEvent failed")
			return
		}

		h.HandlePushEvent(pushEvent)

	case
		"repo:fork",
		"repo:updated",
		"repo:transfer",
		"repo:created",
		"repo:deleted",
		"repo:commit_comment_created",
		"repo:commit_status_created",
		"repo:commit_status_updated",
		"issue:created",
		"issue:updated",
		"issue:comment_created",
		"pullrequest:created",
		"pullrequest:updated",
		"pullrequest:approved",
		"pullrequest:unapproved",
		"pullrequest:fulfilled",
		"pullrequest:rejected",
		"pullrequest:comment_created",
		"pullrequest:comment_updated",
		"pullrequest:comment_deleted":
		log.Debug().Str("event", eventType).Msgf("Not implemented Bitbucket webhook event of type '%v'", eventType)

	default:
		log.Warn().Str("event", eventType).Msgf("Unsupported Bitbucket webhook event of type '%v'", eventType)
	}

	c.String(http.StatusOK, "Aye aye!")
}

func (h *eventHandlerImpl) HandlePushEvent(pushEvent RepositoryPushEvent) {

	log.Debug().Interface("pushEvent", pushEvent).Msgf("Deserialized Bitbucket push event for repository %v", pushEvent.Repository.FullName)

	// test making api calls for bitbucket app in the background
	h.eventsChannel <- pushEvent
}

func (h *eventHandlerImpl) logRequest(eventType string, request *http.Request, body []byte) {
	var b interface{}
	err := json.Unmarshal(body, &b)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body from Bitbucket webhook failed")
		return
	}

	log.Debug().
		Str("method", request.Method).
		Str("url", request.URL.String()).
		Interface("headers", request.Header).
		Interface("body", b).
		Msgf("Received webhook event of type '%v' from Bitbucket...", eventType)
}
