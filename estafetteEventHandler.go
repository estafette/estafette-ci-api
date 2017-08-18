package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

var (
	// channel for passing push events to worker that cleans up finished jobs
	estafetteCiBuilderEvents = make(chan EstafetteCiBuilderEvent, 100)
)

// EstafetteEventHandler handles events from estafette components
type EstafetteEventHandler interface {
	Handle(http.ResponseWriter, *http.Request)
}

type estafetteEventHandlerImpl struct {
}

func newEstafetteEventHandler() EstafetteEventHandler {
	return &estafetteEventHandlerImpl{}
}

func (h *estafetteEventHandlerImpl) Handle(w http.ResponseWriter, r *http.Request) {

	eventType := r.Header.Get("X-Estafette-Event")
	webhookTotal.With(prometheus.Labels{"event": eventType, "source": "estafette"}).Inc()

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
	var ciBuilderEvent EstafetteCiBuilderEvent
	err = json.Unmarshal(body, &ciBuilderEvent)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to EstafetteCiBuilderEvent failed")
		return
	}

	log.Debug().Interface("buildFinishedEvent", ciBuilderEvent).Msgf("Deserialized Estafette 'build finished' event for job %v", ciBuilderEvent.JobName)

	switch eventType {
	case "builder:nomanifest":
	case "builder:succeeded":
	case "builder:failed":
		// send via channel to worker
		estafetteCiBuilderEvents <- ciBuilderEvent

	default:
		log.Warn().Str("event", eventType).Msgf("Unsupported Estafette event of type '%v'", eventType)
	}

	log.Debug().
		Str("jobName", ciBuilderEvent.JobName).
		Msgf("Received event of type '%v' from estafette-ci-builder for job %v...", eventType, ciBuilderEvent.JobName)

	fmt.Fprintf(w, "Aye aye!")
}
