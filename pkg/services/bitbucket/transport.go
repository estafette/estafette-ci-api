package bitbucket

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/bitbucketapi"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
)

func NewHandler(service Service, config *api.APIConfig) Handler {
	return Handler{
		config:  config,
		service: service,
	}
}

type Handler struct {
	config  *api.APIConfig
	service Service
}

func (h *Handler) Handle(c *gin.Context) {

	// https://confluence.atlassian.com/bitbucket/manage-webhooks-735643732.html

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body from Bitbucket webhook failed")
		c.Status(http.StatusInternalServerError)
		return
	}

	// unmarshal json body to check if installation is allowed
	var anyEvent bitbucketapi.AnyEvent
	err = json.Unmarshal(body, &anyEvent)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to BitbucketAnyEvent failed")
		c.Status(http.StatusBadRequest)
		return
	}

	// verify owner is allowed
	isAllowed, _ := h.service.IsAllowedOwner(anyEvent.Data.Repository)
	if !isAllowed {
		log.Warn().Interface("event", anyEvent).Str("body", string(body)).Msg("BitbucketAnyEvent owner is not allowed")
		c.Status(http.StatusUnauthorized)
		return
	}

	switch anyEvent.Event {
	case "repo:push":
		// unmarshal json body
		var pushEvent bitbucketapi.RepositoryPushEvent
		err := json.Unmarshal(body, &pushEvent)
		if err != nil {
			log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to BitbucketRepositoryPushEvent failed")
			c.Status(http.StatusInternalServerError)
			return
		}

		err = h.service.CreateJobForBitbucketPush(c.Request.Context(), pushEvent)
		if err != nil && !errors.Is(err, ErrNonCloneableEvent) && !errors.Is(err, ErrNoManifest) {
			c.Status(http.StatusInternalServerError)
			return
		}

	case
		"repo:fork",
		"repo:transfer",
		"repo:created",
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

	case "repo:updated":
		log.Debug().Str("event", anyEvent.Event).Str("requestBody", string(body)).Msgf("Bitbucket webhook event of type '%v', logging request body", anyEvent.Event)

		// unmarshal json body
		var repoUpdatedEvent bitbucketapi.RepoUpdatedEvent
		err := json.Unmarshal(body, &repoUpdatedEvent)
		if err != nil {
			log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to BitbucketRepoUpdatedEvent failed")
			c.Status(http.StatusInternalServerError)
			return
		}

		if repoUpdatedEvent.IsValidRenameEvent() {
			log.Info().Msgf("Renaming repository from %v/%v/%v to %v/%v/%v", repoUpdatedEvent.GetRepoSource(), repoUpdatedEvent.GetOldRepoOwner(), repoUpdatedEvent.GetOldRepoName(), repoUpdatedEvent.GetRepoSource(), repoUpdatedEvent.GetNewRepoOwner(), repoUpdatedEvent.GetNewRepoName())
			err = h.service.Rename(c.Request.Context(), repoUpdatedEvent.GetRepoSource(), repoUpdatedEvent.GetOldRepoOwner(), repoUpdatedEvent.GetOldRepoName(), repoUpdatedEvent.GetRepoSource(), repoUpdatedEvent.GetNewRepoOwner(), repoUpdatedEvent.GetNewRepoName())
			if err != nil {
				log.Error().Err(err).Msgf("Failed renaming repository from %v/%v/%v to %v/%v/%v", repoUpdatedEvent.GetRepoSource(), repoUpdatedEvent.GetOldRepoOwner(), repoUpdatedEvent.GetOldRepoName(), repoUpdatedEvent.GetRepoSource(), repoUpdatedEvent.GetNewRepoOwner(), repoUpdatedEvent.GetNewRepoName())
				c.Status(http.StatusInternalServerError)
				return
			}
		}

	case "repo:deleted":
		log.Debug().Str("event", anyEvent.Event).Str("requestBody", string(body)).Msgf("Bitbucket webhook event of type '%v', logging request body", anyEvent.Event)
		// unmarshal json body
		var repoDeletedEvent bitbucketapi.RepoDeletedEvent
		err := json.Unmarshal(body, &repoDeletedEvent)
		if err != nil {
			log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to BitbucketRepoDeletedEvent failed")
			c.Status(http.StatusInternalServerError)
			return
		}

		log.Info().Msgf("Archiving repository %v/%v/%v", repoDeletedEvent.GetRepoSource(), repoDeletedEvent.GetRepoOwner(), repoDeletedEvent.GetRepoName())
		err = h.service.Archive(c.Request.Context(), repoDeletedEvent.GetRepoSource(), repoDeletedEvent.GetRepoOwner(), repoDeletedEvent.GetRepoName())
		if err != nil {
			log.Error().Err(err).Msgf("Failed archiving repository %v/%v/%v", repoDeletedEvent.GetRepoSource(), repoDeletedEvent.GetRepoOwner(), repoDeletedEvent.GetRepoName())
			c.Status(http.StatusInternalServerError)
			return
		}

	default:
		log.Warn().Str("event", anyEvent.Event).Msgf("Unsupported Bitbucket webhook event of type '%v'", anyEvent.Event)
	}

	// publish event for bots to run
	go func() {
		// create new context to avoid cancellation impacting execution
		span, _ := opentracing.StartSpanFromContext(c.Request.Context(), "bitbucket:AsyncPublishBitbucketEvent")
		ctx := opentracing.ContextWithSpan(context.Background(), span)
		defer span.Finish()

		err = h.service.PublishBitbucketEvent(ctx, manifest.EstafetteBitbucketEvent{
			Event:         anyEvent.Event,
			Repository:    anyEvent.GetRepository(),
			HookUUID:      c.GetHeader("X-Hook-UUID"),
			RequestUUID:   c.GetHeader("X-Request-UUID"),
			AttemptNumber: c.GetHeader("X-Attempt-Number"),
			Payload:       string(body),
		})
		if err != nil {
			log.Error().Err(err).Msgf("Failed PublishBitbucketEvent")
		}
	}()

	c.Status(http.StatusOK)
}

func (h *Handler) Descriptor(c *gin.Context) {

	// https://developer.atlassian.com/cloud/bitbucket/app-descriptor/

	descriptor := Descriptor{
		Key:         h.config.Integrations.Bitbucket.Key,
		Name:        h.config.Integrations.Bitbucket.Name,
		Description: "Estafette - The The resilient and cloud-native CI/CD platform",
		BaseURL:     h.config.APIServer.IntegrationsURL,
		Authentication: &DescriptorAuthentication{
			Type: "none",
		},
		Lifecycle: &DescriptorLifecycle{
			Installed:   "/api/integrations/bitbucket/installed",
			Uninstalled: "/api/integrations/bitbucket/uninstalled",
		},
		Scopes:   []string{"repository:write", "pullrequest:write"},
		Contexts: []string{"account"},
		Modules: &DescriptorModules{
			Webhooks: []DescriptorWebhook{
				{
					Event: "repo:push",
					URL:   "/api/integrations/bitbucket/events",
				},
				{
					Event: "repo:updated",
					URL:   "/api/integrations/bitbucket/events",
				},
				{
					Event: "repo:deleted",
					URL:   "/api/integrations/bitbucket/events",
				},
			},
		},
	}

	c.JSON(http.StatusOK, descriptor)
}

func (h *Handler) Installed(c *gin.Context) {

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body for Bitbucket App installed event failed")
		c.Status(http.StatusInternalServerError)
		return
	}

	log.Info().Str("body", string(body)).Msg("Bitbucket App installed")

	c.Status(http.StatusOK)
}

func (h *Handler) Uninstalled(c *gin.Context) {

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body for Bitbucket App uninstalled event failed")
		c.Status(http.StatusInternalServerError)
		return
	}

	log.Info().Str("body", string(body)).Msg("Bitbucket App uninstalled")

	c.Status(http.StatusOK)
}
