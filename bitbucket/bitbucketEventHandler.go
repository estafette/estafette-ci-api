package bitbucket

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// EventHandler handles http events for Bitbucket integration
type EventHandler interface {
	Handle(http.ResponseWriter, *http.Request)
	HandlePushEvent([]byte)
}

type eventHandlerImpl struct {
	EventsChannel                chan RepositoryPushEvent
	PrometheusInboundEventTotals *prometheus.CounterVec
}

// NewBitbucketEventHandler returns a new bitbucket.EventHandler
func NewBitbucketEventHandler(eventsChannel chan RepositoryPushEvent, prometheusInboundEventTotals *prometheus.CounterVec) EventHandler {
	return &eventHandlerImpl{
		EventsChannel:                eventsChannel,
		PrometheusInboundEventTotals: prometheusInboundEventTotals,
	}
}

func (h *eventHandlerImpl) Handle(w http.ResponseWriter, r *http.Request) {

	// https://confluence.atlassian.com/bitbucket/manage-webhooks-735643732.html

	eventType := r.Header.Get("X-Event-Key")
	h.PrometheusInboundEventTotals.With(prometheus.Labels{"event": eventType, "source": "bitbucket"}).Inc()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body from Bitbucket webhook failed")
		http.Error(w, "Reading body from Bitbucket webhook failed", 500)
		return
	}

	// unmarshal json body
	var b interface{}
	err = json.Unmarshal(body, &b)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body from Bitbucket webhook failed")
		http.Error(w, "Deserializing body from Bitbucket webhook failed", 500)
		return
	}

	log.Debug().
		Str("method", r.Method).
		Str("url", r.URL.String()).
		Interface("headers", r.Header).
		Interface("body", b).
		Msgf("Received webhook event of type '%v' from Bitbucket...", eventType)

	switch eventType {
	case "repo:push":
		h.HandlePushEvent(body)

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

	fmt.Fprintf(w, "Aye aye!")
}

func (h *eventHandlerImpl) HandlePushEvent(body []byte) {

	// unmarshal json body
	var pushEvent RepositoryPushEvent
	err := json.Unmarshal(body, &pushEvent)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to BitbucketRepositoryPushEvent failed")
		return
	}

	log.Debug().Interface("pushEvent", pushEvent).Msgf("Deserialized Bitbucket push event for repository %v", pushEvent.Repository.FullName)

	// test making api calls for bitbucket app in the background
	h.EventsChannel <- pushEvent
}
