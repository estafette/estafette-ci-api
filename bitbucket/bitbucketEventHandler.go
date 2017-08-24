package bitbucket

import (
	"bytes"
	"encoding/json"
	"io"
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

	// unmarshal json body
	var b interface{}
	err := json.NewDecoder(io.TeeReader(c.Request.Body, bytes.NewBuffer(make([]byte, 0)))).Decode(&b)
	if err != nil {
		body, _ := ioutil.ReadAll(io.TeeReader(c.Request.Body, bytes.NewBuffer(make([]byte, 0))))
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body from Bitbucket webhook failed")
		c.String(http.StatusInternalServerError, "Deserializing body from Github webhook failed")
		return
	}

	log.Debug().
		Str("method", c.Request.Method).
		Str("url", c.Request.URL.String()).
		Interface("headers", c.Request.Header).
		Interface("body", b).
		Msgf("Received webhook event of type '%v' from Bitbucket...", eventType)

	switch eventType {
	case "repo:push":

		// unmarshal json body
		var pushEvent RepositoryPushEvent
		err := json.NewDecoder(io.TeeReader(c.Request.Body, bytes.NewBuffer(make([]byte, 0)))).Decode(&pushEvent)
		if err != nil {
			body, _ := ioutil.ReadAll(io.TeeReader(c.Request.Body, bytes.NewBuffer(make([]byte, 0))))
			log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to BitbucketRepositoryPushEvent failed")
			return
		}

		h.HandlePushEvent(pushEvent)

	case
		"repo:fork",
		"repo:updated",
		"repo:transfer",
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
