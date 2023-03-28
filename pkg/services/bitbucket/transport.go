package bitbucket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/bitbucketapi"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
)

func NewHandler(service Service, config *api.APIConfig, bitbucketapiClient bitbucketapi.Client) Handler {
	return Handler{
		config:             config,
		service:            service,
		bitbucketapiClient: bitbucketapiClient,
	}
}

type Handler struct {
	config             *api.APIConfig
	service            Service
	bitbucketapiClient bitbucketapi.Client
}

func (h *Handler) Handle(c *gin.Context) {

	// https://confluence.atlassian.com/bitbucket/manage-webhooks-735643732.html

	eventType := c.GetHeader("X-Event-Key")
	authorizationHeader := c.GetHeader("Authorization")

	installation, err := h.bitbucketapiClient.ValidateInstallationJWT(c.Request.Context(), authorizationHeader)
	if err != nil {
		log.Error().Err(err).Str("authorization", authorizationHeader).Msg("Validating authorization header failed")
		c.Status(http.StatusBadRequest)
		return
	}

	if installation == nil {
		log.Error().Err(err).Str("authorization", authorizationHeader).Msg("Retrieving installation for authorization header failed")
		c.Status(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body from Bitbucket webhook failed")
		c.Status(http.StatusInternalServerError)
		return
	}

	// unmarshal json body to check if installation is allowed
	var eventCheck bitbucketapi.EventCheck
	err = json.Unmarshal(body, &eventCheck)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body for EventCheck failed")
		c.Status(http.StatusBadRequest)
		return
	}

	// verify owner is allowed
	isAllowed, _ := h.service.IsAllowedOwner(c.Request.Context(), eventCheck.GetRepository())
	if !isAllowed {
		log.Warn().Interface("event", eventCheck).Str("body", string(body)).Msg("Bitbucket EventCheck owner is not allowed")
		c.Status(http.StatusUnauthorized)
		return
	}

	switch eventType {
	case "repo:push":
		// unmarshal json body
		var pushEvent bitbucketapi.RepositoryPushEvent

		if eventCheck.Data != nil {
			var pushEventEnvelope bitbucketapi.RepositoryPushEventEnvelope
			err := json.Unmarshal(body, &pushEventEnvelope)
			if err != nil {
				log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to Bitbucket RepositoryPushEventEnvelope failed")
				c.Status(http.StatusInternalServerError)
				return
			}
			pushEvent = pushEventEnvelope.Data
		} else {
			err := json.Unmarshal(body, &pushEvent)
			if err != nil {
				log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to Bitbucket RepositoryPushEvent failed")
				c.Status(http.StatusInternalServerError)
				return
			}
		}

		err = h.service.CreateJobForBitbucketPush(c.Request.Context(), *installation, pushEvent)
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
		log.Debug().Str("event", eventType).Str("requestBody", string(body)).Msgf("Bitbucket webhook event of type '%v', logging request body", eventType)

		// unmarshal json body
		var repoUpdatedEvent bitbucketapi.RepoUpdatedEvent
		err := json.Unmarshal(body, &repoUpdatedEvent)
		if err != nil {
			log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to BitbucketRepoUpdatedEvent failed")
			c.Status(http.StatusInternalServerError)
			return
		}

		if repoUpdatedEvent.IsValidRenameEvent() {
			log.Debug().Msgf("Renaming repository from %v/%v/%v to %v/%v/%v", repoUpdatedEvent.GetRepoSource(), repoUpdatedEvent.GetOldRepoOwner(), repoUpdatedEvent.GetOldRepoName(), repoUpdatedEvent.GetRepoSource(), repoUpdatedEvent.GetNewRepoOwner(), repoUpdatedEvent.GetNewRepoName())
			err = h.service.Rename(c.Request.Context(), repoUpdatedEvent.GetRepoSource(), repoUpdatedEvent.GetOldRepoOwner(), repoUpdatedEvent.GetOldRepoName(), repoUpdatedEvent.GetRepoSource(), repoUpdatedEvent.GetNewRepoOwner(), repoUpdatedEvent.GetNewRepoName())
			if err != nil {
				log.Error().Err(err).Msgf("Failed renaming repository from %v/%v/%v to %v/%v/%v", repoUpdatedEvent.GetRepoSource(), repoUpdatedEvent.GetOldRepoOwner(), repoUpdatedEvent.GetOldRepoName(), repoUpdatedEvent.GetRepoSource(), repoUpdatedEvent.GetNewRepoOwner(), repoUpdatedEvent.GetNewRepoName())
				c.Status(http.StatusInternalServerError)
				return
			}
		}

	case "repo:deleted":
		log.Debug().Str("event", eventType).Str("requestBody", string(body)).Msgf("Bitbucket webhook event of type '%v', logging request body", eventType)
		// unmarshal json body
		var repoDeletedEvent bitbucketapi.RepoDeletedEvent
		err := json.Unmarshal(body, &repoDeletedEvent)
		if err != nil {
			log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to BitbucketRepoDeletedEvent failed")
			c.Status(http.StatusInternalServerError)
			return
		}

		log.Debug().Msgf("Archiving repository %v/%v/%v", repoDeletedEvent.GetRepoSource(), repoDeletedEvent.GetRepoOwner(), repoDeletedEvent.GetRepoName())
		err = h.service.Archive(c.Request.Context(), repoDeletedEvent.GetRepoSource(), repoDeletedEvent.GetRepoOwner(), repoDeletedEvent.GetRepoName())
		if err != nil {
			log.Error().Err(err).Msgf("Failed archiving repository %v/%v/%v", repoDeletedEvent.GetRepoSource(), repoDeletedEvent.GetRepoOwner(), repoDeletedEvent.GetRepoName())
			c.Status(http.StatusInternalServerError)
			return
		}

	default:
		log.Warn().Str("event", eventType).Msgf("Unsupported Bitbucket webhook event of type '%v'", eventType)
	}

	// publish event for bots to run
	go func() {
		// create new context to avoid cancellation impacting execution
		span, _ := opentracing.StartSpanFromContext(c.Request.Context(), "bitbucket:AsyncPublishBitbucketEvent")
		ctx := opentracing.ContextWithSpan(context.Background(), span)
		defer span.Finish()

		err = h.service.PublishBitbucketEvent(ctx, manifest.EstafetteBitbucketEvent{
			Event:         eventType,
			Repository:    eventCheck.GetFullRepository(),
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

	key := c.Query("key")

	if key == "" {
		key = "estafette-ci"
	}

	// https://developer.atlassian.com/cloud/bitbucket/app-descriptor/

	descriptor := Descriptor{
		Key:         key,
		Name:        "Estafette",
		Description: "Estafette - The resilient and cloud-native CI/CD platform",
		Vendor: DescriptorVendor{
			Name: "Estafette",
			URL:  "https://estafette.io",
		},
		BaseURL: h.config.APIServer.IntegrationsURL,
		Authentication: &DescriptorAuthentication{
			Type: "jwt",
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

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body for Bitbucket App installed event failed")
		c.Status(http.StatusInternalServerError)
		return
	}

	var installation bitbucketapi.BitbucketAppInstallation
	err = json.Unmarshal(body, &installation)
	if err != nil {
		log.Error().Err(err).Msg("Failed unmarshalling bitbucket app install body")
		c.Status(http.StatusInternalServerError)
		return
	} else {
		_, err := h.bitbucketapiClient.GetAppByKey(c.Request.Context(), installation.Key)
		if err != nil && !errors.Is(err, bitbucketapi.ErrMissingApp) {
			log.Error().Err(err).Msgf("Failed retrieving app for bitbucket app installation key %v", installation.Key)
			c.Status(http.StatusInternalServerError)
			return
		}
		if err != nil && errors.Is(err, bitbucketapi.ErrMissingApp) {
			// create app
			err = h.bitbucketapiClient.AddApp(c.Request.Context(), bitbucketapi.BitbucketApp{
				Key:           installation.Key,
				Installations: make([]*bitbucketapi.BitbucketAppInstallation, 0),
			})
			if err != nil {
				log.Error().Err(err).Msgf("Failed adding app for bitbucket app installation key %v", installation.Key)
				c.Status(http.StatusInternalServerError)
				return
			}
		}

		workspace, err := h.bitbucketapiClient.GetWorkspace(c.Request.Context(), installation)
		if err != nil {
			log.Error().Err(err).Msg("Failed retrieving workspace for bitbucket app installation")
			c.Status(http.StatusInternalServerError)
			return
		}
		installation.Workspace = workspace

		log.Debug().Interface("installation", installation).Msg("Unmarshalled bitbucket app install body")
		err = h.bitbucketapiClient.AddInstallation(c.Request.Context(), installation)
		if err != nil {
			log.Error().Err(err).Msg("Failed adding bitbucket app installation")
			c.Status(http.StatusInternalServerError)
			return
		} else {
			log.Debug().Msg("Added bitbucket app installation")
		}
	}

	c.Status(http.StatusOK)
}

func (h *Handler) Uninstalled(c *gin.Context) {

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body for Bitbucket App uninstalled event failed")
		c.Status(http.StatusInternalServerError)
		return
	}

	var installation bitbucketapi.BitbucketAppInstallation
	err = json.Unmarshal(body, &installation)
	if err != nil {
		log.Error().Err(err).Msg("Failed unmarshalling bitbucket app uninstall body")
		c.Status(http.StatusInternalServerError)
		return
	} else {
		log.Debug().Interface("installation", installation).Msg("Unmarshalled bitbucket app uninstall body")
		err = h.bitbucketapiClient.RemoveInstallation(c.Request.Context(), installation)
		if err != nil {
			log.Error().Err(err).Msg("Failed removing bitbucket app installation")
			c.Status(http.StatusInternalServerError)
			return
		} else {
			log.Debug().Msg("Removed bitbucket app installation")
		}
	}

	c.Status(http.StatusOK)
}

func (h *Handler) Redirect(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%v/admin/integrations", strings.TrimRight(h.config.APIServer.BaseURL, "/")))
}
