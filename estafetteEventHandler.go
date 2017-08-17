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
	estafetteBuildFinishedEvents = make(chan EstafetteBuildFinishedEvent, 100)
)

// EstafetteEventHandler handles events from estafette components
type EstafetteEventHandler interface {
	HandleBuildFinished(http.ResponseWriter, *http.Request)
}

type estafetteEventHandlerImpl struct {
}

func newEstafetteEventHandler() EstafetteEventHandler {
	return &estafetteEventHandlerImpl{}
}

func (h *estafetteEventHandlerImpl) HandleBuildFinished(w http.ResponseWriter, r *http.Request) {

	webhookTotal.With(prometheus.Labels{"event": "buildfinished", "source": "ci-builder"}).Inc()

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
	var buildFinishedEvent EstafetteBuildFinishedEvent
	err = json.Unmarshal(body, &buildFinishedEvent)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to EstafetteBuildFinishedEvent failed")
		return
	}

	log.Debug().Interface("buildFinishedEvent", buildFinishedEvent).Msgf("Deserialized Estafette 'build finished' event for job %v", buildFinishedEvent.JobName)

	// send via channel to worker
	estafetteBuildFinishedEvents <- buildFinishedEvent

	log.Info().
		Str("jobName", buildFinishedEvent.JobName).
		Msgf("Received event of type 'build finished' from estafette-ci-builder for job %v...", buildFinishedEvent.JobName)

	fmt.Fprintf(w, "Aye aye!")
}
