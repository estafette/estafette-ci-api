package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/estafette/estafette-ci-api/pkg/clients/database"
	"github.com/estafette/migration"
	"io"
	"net/http"
	"strings"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/githubapi"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
)

func NewHandler(service Service, config *api.APIConfig, githubapiClient githubapi.Client, databaseClient database.Client) Handler {
	return Handler{
		config:          config,
		service:         service,
		githubapiClient: githubapiClient,
		databaseClient:  databaseClient,
	}
}

type Handler struct {
	config          *api.APIConfig
	service         Service
	githubapiClient githubapi.Client
	databaseClient  database.Client
}

func (h *Handler) Handle(c *gin.Context) {

	// https://developer.github.com/webhooks/
	eventType := c.GetHeader("X-Github-Event")
	// h.prometheusInboundEventTotals.With(prometheus.Labels{"event": eventType, "source": "github"}).Inc()

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body from Github webhook failed")
		c.Status(http.StatusInternalServerError)
		return
	}

	// verify hmac signature
	hasValidSignature, err := h.service.HasValidSignature(c.Request.Context(), body, c.GetHeader("X-GitHub-Hook-Installation-Target-ID"), c.GetHeader("X-Hub-Signature"))
	if err != nil {
		log.Error().Err(err).Msg("Verifying signature from Github webhook failed")
		c.Status(http.StatusInternalServerError)
		return
	}
	if !hasValidSignature {
		log.Error().Msg("Signature from Github webhook is invalid")
		c.Status(http.StatusBadRequest)
		return
	}

	// unmarshal json body to check if installation is allowed
	var anyEvent githubapi.AnyEvent
	err = json.Unmarshal(body, &anyEvent)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to GithubAnyEvent failed")
		c.Status(http.StatusBadRequest)
		return
	}

	switch eventType {
	case "push": // Any Git push to a Repository, including editing tags or branches. Commits via API actions that update references are also counted. This is the default event.

		// verify installation id is allowed
		isAllowed, _ := h.service.IsAllowedInstallation(c.Request.Context(), anyEvent.Installation.ID)
		if !isAllowed {
			log.Warn().Interface("event", anyEvent).Str("body", string(body)).Msg("GithubAnyEvent installation is not allowed")
			c.Status(http.StatusUnauthorized)
			return
		}

		// unmarshal json body
		var pushEvent githubapi.PushEvent
		err := json.Unmarshal(body, &pushEvent)
		if err != nil {
			log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to GithubPushEvent failed")
			c.Status(http.StatusBadRequest)
			return
		}

		task, err := h.databaseClient.GetMigrationByToRepo(c.Request.Context(), "github.com", pushEvent.GetRepoOwner(), pushEvent.GetRepoName())
		if err != nil {
			log.Error().Err(err).Msg("Failed getting migration for github push event")
			c.Status(http.StatusInternalServerError)
			return
		}
		if task != nil && task.Status != migration.StatusCompleted {
			log.Warn().Str("taskID", task.ID).Str("repoName", pushEvent.GetRepoFullName()).Str("status", task.Status.String()).Msg("ignoring github push event for repo that is being migrated")
			c.Status(http.StatusConflict)
			return
		}

		err = h.service.CreateJobForGithubPush(c.Request.Context(), pushEvent)
		if err != nil && !errors.Is(err, ErrNonCloneableEvent) && !errors.Is(err, ErrNoManifest) {
			log.Error().Err(err).Msg("Creating build job for github push failed")
			c.Status(http.StatusInternalServerError)
			return
		}

	case "installation": // Any time a GitHub App is installed or uninstalled.
		log.Debug().Str("event", eventType).Str("requestBody", string(body)).Msgf("Github webhook event of type '%v', logging request body", eventType)

		// unmarshal json body
		var repositoryEvent githubapi.RepositoryEvent
		err = json.Unmarshal(body, &repositoryEvent)
		if err != nil {
			log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to GithubRepositoryEvent failed")
			c.Status(http.StatusBadRequest)
			return
		}

		switch repositoryEvent.Action {
		case "created":
			err = h.githubapiClient.AddInstallation(c.Request.Context(), anyEvent.Installation)
			if err != nil {
				log.Error().Err(err).Str("body", string(body)).Msg("Adding installation failed")
				c.Status(http.StatusBadRequest)
				return
			}

		case "deleted":
			err = h.githubapiClient.RemoveInstallation(c.Request.Context(), anyEvent.Installation)
			if err != nil {
				log.Error().Err(err).Str("body", string(body)).Msg("Removing installation failed")
				c.Status(http.StatusBadRequest)
				return
			}

		case "suspend":
		case "unsuspend":
		default:
			log.Warn().Str("event", eventType).Str("action", repositoryEvent.Action).Msgf("Unsupported Github webhook installation event action of type '%v'", repositoryEvent.Action)
		}

	case
		"commit_comment",    // Any time a Commit is commented on.
		"create",            // Any time a Branch or Tag is created.
		"delete",            // Any time a Branch or Tag is deleted.
		"deployment",        // Any time a Repository has a new deployment created from the API.
		"deployment_status", // Any time a deployment for a Repository has a status update from the API.
		"fork",              // Any time a Repository is forked.
		"gollum",            // Any time a Wiki page is updated.

		"installation_repositories",             // Any time a repository is added or removed from an installation.
		"issue_comment",                         // Any time a comment on an issue is created, edited, or deleted.
		"issues",                                // Any time an Issue is assigned, unassigned, labeled, unlabeled, opened, edited, milestoned, demilestoned, closed, or reopened.
		"label",                                 // Any time a Label is created, edited, or deleted.
		"marketplace_purchase",                  // Any time a user purchases, cancels, or changes their GitHub Marketplace plan.
		"member",                                // Any time a User is added or removed as a collaborator to a Repository, or has their permissions modified.
		"membership",                            // Any time a User is added or removed from a team. Organization hooks only.
		"milestone",                             // Any time a Milestone is created, closed, opened, edited, or deleted.
		"organization",                          // Any time a user is added, removed, or invited to an Organization. Organization hooks only.
		"org_block",                             // Any time an organization blocks or unblocks a user. Organization hooks only.
		"page_build",                            // Any time a Pages site is built or results in a failed build.
		"project_card",                          // Any time a Project Card is created, edited, moved, converted to an issue, or deleted.
		"project_column",                        // Any time a Project Column is created, edited, moved, or deleted.
		"project",                               // Any time a Project is created, edited, closed, reopened, or deleted.
		"public",                                // Any time a Repository changes from private to public.
		"pull_request_review_comment",           // Any time a comment on a pull request's unified diff is created, edited, or deleted (in the Files Changed tab).
		"pull_request_review",                   // Any time a pull request review is submitted, edited, or dismissed.
		"pull_request",                          // Any time a pull request is assigned, unassigned, labeled, unlabeled, opened, edited, closed, reopened, or synchronized (updated due to a new push in the branch that the pull request is tracking). Also any time a pull request review is requested, or a review request is removed.
		"release",                               // Any time a Release is published in a Repository.
		"status",                                // Any time a Repository has a status update from the API
		"team",                                  // Any time a team is created, deleted, modified, or added to or removed from a repository. Organization hooks only
		"team_add",                              // Any time a team is added or modified on a Repository.
		"watch",                                 // Any time a User stars a Repository.
		"integration_installation_repositories": // ?

	case "repository": // Any time a Repository is created, deleted (organization hooks only), made public, or made private.
		log.Debug().Str("event", eventType).Str("requestBody", string(body)).Msgf("Github webhook event of type '%v', logging request body", eventType)

		// unmarshal json body
		var repositoryEvent githubapi.RepositoryEvent
		err := json.Unmarshal(body, &repositoryEvent)
		if err != nil {
			log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to GithubRepositoryEvent failed")
			c.Status(http.StatusBadRequest)
			return
		}

		switch repositoryEvent.Action {
		case "renamed":
			if repositoryEvent.IsValidRenameEvent() {
				log.Debug().Msgf("Renaming repository from %v/%v/%v to %v/%v/%v", repositoryEvent.GetRepoSource(), repositoryEvent.GetOldRepoOwner(), repositoryEvent.GetOldRepoName(), repositoryEvent.GetRepoSource(), repositoryEvent.GetNewRepoOwner(), repositoryEvent.GetNewRepoName())
				err = h.service.Rename(c.Request.Context(), repositoryEvent.GetRepoSource(), repositoryEvent.GetOldRepoOwner(), repositoryEvent.GetOldRepoName(), repositoryEvent.GetRepoSource(), repositoryEvent.GetNewRepoOwner(), repositoryEvent.GetNewRepoName())
				if err != nil {
					log.Error().Err(err).Msgf("Failed renaming repository from %v/%v/%v to %v/%v/%v", repositoryEvent.GetRepoSource(), repositoryEvent.GetOldRepoOwner(), repositoryEvent.GetOldRepoName(), repositoryEvent.GetRepoSource(), repositoryEvent.GetNewRepoOwner(), repositoryEvent.GetNewRepoName())
					c.Status(http.StatusInternalServerError)
					return
				}
			}

		case "deleted",
			"archived":
			log.Debug().Msgf("Archiving repository from %v/%v/%v", repositoryEvent.GetRepoSource(), repositoryEvent.GetRepoOwner(), repositoryEvent.GetRepoName())
			err = h.service.Archive(c.Request.Context(), repositoryEvent.GetRepoSource(), repositoryEvent.GetRepoOwner(), repositoryEvent.GetRepoName())
			if err != nil {
				log.Error().Err(err).Msgf("Failed archiving repository %v/%v/%v", repositoryEvent.GetRepoSource(), repositoryEvent.GetRepoOwner(), repositoryEvent.GetRepoName())
				c.Status(http.StatusInternalServerError)
				return
			}

		case "unarchived":
			log.Debug().Msgf("Unarchiving repository from %v/%v/%v", repositoryEvent.GetRepoSource(), repositoryEvent.GetRepoOwner(), repositoryEvent.GetRepoName())
			err = h.service.Unarchive(c.Request.Context(), repositoryEvent.GetRepoSource(), repositoryEvent.GetRepoOwner(), repositoryEvent.GetRepoName())
			if err != nil {
				log.Error().Err(err).Msgf("Failed unarchiving repository %v/%v/%v", repositoryEvent.GetRepoSource(), repositoryEvent.GetRepoOwner(), repositoryEvent.GetRepoName())
				c.Status(http.StatusInternalServerError)
				return
			}

		case
			"created",
			"edited",
			"transferred",
			"publicized",
			"privatized":
		}

	default:
		log.Warn().Str("event", eventType).Msgf("Unsupported Github webhook event of type '%v'", eventType)
	}

	// publish event for bots to run
	go func() {
		// create new context to avoid cancellation impacting execution
		span, _ := opentracing.StartSpanFromContext(c.Request.Context(), "github:AsyncPublishGithubEvent")
		ctx := opentracing.ContextWithSpan(context.Background(), span)
		defer span.Finish()

		err = h.service.PublishGithubEvent(ctx, manifest.EstafetteGithubEvent{
			Event:      eventType,
			Repository: anyEvent.GetRepository(),
			Delivery:   c.GetHeader("X-GitHub-Delivery"),
			Payload:    string(body),
		})
		if err != nil {
			log.Error().Err(err).Msgf("Failed PublishGithubEvent")
		}
	}()

	c.Status(http.StatusOK)
}

func (h *Handler) Redirect(c *gin.Context) {
	// https://docs.github.com/en/developers/apps/building-github-apps/creating-a-github-app-from-a-manifest

	code := c.Query("code")

	err := h.githubapiClient.ConvertAppManifestCode(c.Request.Context(), code)
	if err != nil {
		log.Error().Err(err).Msg("Converting Github app install code failed")
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%v/admin/integrations", strings.TrimRight(h.config.APIServer.BaseURL, "/")))
}
