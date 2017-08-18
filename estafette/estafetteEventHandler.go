package estafette

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// EventHandler handles events from estafette components
type EventHandler interface {
	Handle(http.ResponseWriter, *http.Request)
}

type eventHandlerImpl struct {
	CiAPIKey      string
	EventsChannel chan CiBuilderEvent
}

// NewEstafetteEventHandler returns a new estafette.EventHandler
func NewEstafetteEventHandler(ciAPIKey string, eventsChannel chan CiBuilderEvent) EventHandler {
	return &eventHandlerImpl{
		CiAPIKey:      ciAPIKey,
		EventsChannel: eventsChannel,
	}
}

func (h *eventHandlerImpl) Handle(w http.ResponseWriter, r *http.Request) {

	authorizationHeader := r.Header.Get("Authorization")
	if authorizationHeader != fmt.Sprintf("Bearer %v", h.CiAPIKey) {
		log.Error().
			Str("authorizationHeader", authorizationHeader).
			Str("apiKey", h.CiAPIKey).
			Msg("Authorization header for Estafette event is incorrect")
		http.Error(w, "authorization failed", http.StatusUnauthorized)
		return
	}

	eventType := r.Header.Get("X-Estafette-Event")
	WebhookTotal.With(prometheus.Labels{"event": eventType, "source": "estafette"}).Inc()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body from Estafette 'build finished' event failed")
		http.Error(w, "Reading body from Estafette build 'finished event' failed", 500)
		return
	}

	// unmarshal json body
	var b interface{}
	err = json.Unmarshal(body, &b)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body from Estafette 'build finished' event failed")
		http.Error(w, "Deserializing body from Estafette 'build finished' event failed", 500)
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
		h.EventsChannel <- ciBuilderEvent

	default:
		log.Warn().Str("event", eventType).Msgf("Unsupported Estafette event of type '%v'", eventType)
	}

	log.Debug().
		Str("jobName", ciBuilderEvent.JobName).
		Msgf("Received event of type '%v' from estafette-ci-builder for job %v...", eventType, ciBuilderEvent.JobName)

	fmt.Fprintf(w, "Aye aye!")
}
