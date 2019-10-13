package estafette

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/estafette/estafette-ci-api/config"
	prom "github.com/estafette/estafette-ci-api/prometheus"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// EventHandler handles events from estafette components
type EventHandler interface {
	Handle(*gin.Context)
	UpdateBuildStatus(context.Context, CiBuilderEvent) error
	UpdateJobResources(CiBuilderEvent) error
}

type eventHandlerImpl struct {
	config                       config.APIServerConfig
	ciBuilderClient              CiBuilderClient
	prometheusClient             prom.PrometheusClient
	buildService                 BuildService
	prometheusInboundEventTotals *prometheus.CounterVec
}

// NewEstafetteEventHandler returns a new estafette.EventHandler
func NewEstafetteEventHandler(config config.APIServerConfig, ciBuilderClient CiBuilderClient, prometheusClient prom.PrometheusClient, buildService BuildService, prometheusInboundEventTotals *prometheus.CounterVec) EventHandler {
	return &eventHandlerImpl{
		config:                       config,
		ciBuilderClient:              ciBuilderClient,
		prometheusClient:             prometheusClient,
		buildService:                 buildService,
		prometheusInboundEventTotals: prometheusInboundEventTotals,
	}
}

func (h *eventHandlerImpl) Handle(c *gin.Context) {

	ctx := c.Request.Context()

	if c.MustGet(gin.AuthUserKey).(string) != "apiKey" {
		log.Error().Msgf("Authentication for /api/commands failed")
		c.Status(http.StatusUnauthorized)
		return
	}

	eventType := c.GetHeader("X-Estafette-Event")
	log.Debug().Msgf("X-Estafette-Event is set to %v", eventType)
	h.prometheusInboundEventTotals.With(prometheus.Labels{"event": eventType, "source": "estafette"}).Inc()

	eventJobname := c.GetHeader("X-Estafette-Event-Job-Name")
	log.Debug().Msgf("X-Estafette-Event-Job-Name is set to %v", eventJobname)

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body from Estafette 'build finished' event failed")
		c.String(http.StatusInternalServerError, "Reading body from Estafette 'build finished' event failed")
		return
	}

	log.Debug().Msgf("Read body for /api/commands for job %v", eventJobname)

	switch eventType {
	case
		"builder:nomanifest",
		"builder:running",
		"builder:succeeded",
		"builder:failed",
		"builder:canceled":

		// unmarshal json body
		var ciBuilderEvent CiBuilderEvent
		err = json.Unmarshal(body, &ciBuilderEvent)
		if err != nil {
			log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to CiBuilderEvent failed")
			return
		}

		log.Debug().Interface("ciBuilderEvent", ciBuilderEvent).Msgf("Unmarshaled body of /api/commands event %v for job %v", eventType, eventJobname)

		err := h.UpdateBuildStatus(c.Request.Context(), ciBuilderEvent)
		if err != nil {
			errorMessage := fmt.Sprintf("Failed updating build status for job %v to %v, not removing the job", eventJobname, ciBuilderEvent.BuildStatus)
			log.Error().Err(err).Interface("ciBuilderEvent", ciBuilderEvent).Msg(errorMessage)
			c.String(http.StatusInternalServerError, errorMessage)
			return
		}

	case "builder:clean":

		// unmarshal json body
		var ciBuilderEvent CiBuilderEvent
		err = json.Unmarshal(body, &ciBuilderEvent)
		if err != nil {
			log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to CiBuilderEvent failed")
			return
		}

		log.Debug().Interface("ciBuilderEvent", ciBuilderEvent).Msgf("Unmarshaled body of /api/commands event %v for job %v", eventType, eventJobname)

		if ciBuilderEvent.BuildStatus != "canceled" {
			go func(eventJobname string) {
				err = h.ciBuilderClient.RemoveCiBuilderJob(ctx, eventJobname)
				if err != nil {
					errorMessage := fmt.Sprintf("Failed removing job %v for event %v", eventJobname, eventType)
					log.Error().Err(err).Interface("ciBuilderEvent", ciBuilderEvent).Msg(errorMessage)
				}
			}(eventJobname)
		} else {
			log.Info().Msgf("Job %v is already removed by cancellation, no need to remove for event %v", eventJobname, eventType)
		}

		go func(ciBuilderEvent CiBuilderEvent) {
			err := h.UpdateJobResources(ciBuilderEvent)
			if err != nil {
				log.Error().Err(err).Msgf("Failed retrieving max cpu and memory from prometheus for pod %v", ciBuilderEvent.PodName)
			}
		}(ciBuilderEvent)

	default:
		log.Warn().Str("event", eventType).Msgf("Unsupported Estafette event of type '%v'", eventType)
	}

	c.String(http.StatusOK, "Aye aye!")
}

func (h *eventHandlerImpl) UpdateBuildStatus(ctx context.Context, ciBuilderEvent CiBuilderEvent) (err error) {

	log.Debug().Interface("ciBuilderEvent", ciBuilderEvent).Msgf("UpdateBuildStatus executing...")

	if ciBuilderEvent.BuildStatus != "" && ciBuilderEvent.ReleaseID != "" {

		releaseID, err := strconv.Atoi(ciBuilderEvent.ReleaseID)
		if err != nil {
			return err
		}

		log.Debug().Msgf("Converted release id %v", releaseID)

		err = h.buildService.FinishRelease(ctx, ciBuilderEvent.RepoSource, ciBuilderEvent.RepoOwner, ciBuilderEvent.RepoName, releaseID, ciBuilderEvent.BuildStatus)
		if err != nil {
			return err
		}

		log.Debug().Msgf("Updated release status for job %v to %v", ciBuilderEvent.JobName, ciBuilderEvent.BuildStatus)

		return err

	} else if ciBuilderEvent.BuildStatus != "" && ciBuilderEvent.BuildID != "" {

		buildID, err := strconv.Atoi(ciBuilderEvent.BuildID)
		if err != nil {
			return err
		}

		log.Debug().Msgf("Converted build id %v", buildID)

		err = h.buildService.FinishBuild(ctx, ciBuilderEvent.RepoSource, ciBuilderEvent.RepoOwner, ciBuilderEvent.RepoName, buildID, ciBuilderEvent.BuildStatus)
		if err != nil {
			return err
		}

		log.Debug().Msgf("Updated build status for job %v to %v", ciBuilderEvent.JobName, ciBuilderEvent.BuildStatus)

		return err
	}

	return fmt.Errorf("CiBuilderEvent has invalid state, not updating build status")
}

func (h *eventHandlerImpl) UpdateJobResources(ciBuilderEvent CiBuilderEvent) (err error) {

	log.Info().Msgf("Updating job resources for pod %v", ciBuilderEvent.PodName)

	if ciBuilderEvent.PodName != "" {

		h.prometheusClient.AwaitScrapeInterval()

		maxCPU, err := h.prometheusClient.GetMaxCPUByPodName(ciBuilderEvent.PodName)
		if err != nil {
			return err
		}

		log.Info().Msgf("Max cpu usage for pod %v is %v", ciBuilderEvent.PodName, maxCPU)

		maxMemory, err := h.prometheusClient.GetMaxMemoryByPodName(ciBuilderEvent.PodName)
		if err != nil {
			return err
		}

		log.Info().Msgf("Max memory usage for pod %v is %v", ciBuilderEvent.PodName, maxMemory)

	}

	return nil
}
