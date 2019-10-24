package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/estafette"
	ghcontracts "github.com/estafette/estafette-ci-api/github/contracts"
	"github.com/estafette/estafette-ci-api/pubsub"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// EventHandler handles http events for Github integration
type EventHandler interface {
	Handle(*gin.Context)
	CreateJobForGithubPush(context.Context, ghcontracts.PushEvent)
	HasValidSignature([]byte, string) (bool, error)
}

type eventHandlerImpl struct {
	apiClient                    APIClient
	pubsubAPIClient              pubsub.APIClient
	buildService                 estafette.BuildService
	config                       config.GithubConfig
	prometheusInboundEventTotals *prometheus.CounterVec
}

// NewGithubEventHandler returns a github.EventHandler to handle incoming webhook events
func NewGithubEventHandler(apiClient APIClient, pubsubAPIClient pubsub.APIClient, buildService estafette.BuildService, config config.GithubConfig, prometheusInboundEventTotals *prometheus.CounterVec) EventHandler {
	return &eventHandlerImpl{
		apiClient:                    apiClient,
		pubsubAPIClient:              pubsubAPIClient,
		buildService:                 buildService,
		config:                       config,
		prometheusInboundEventTotals: prometheusInboundEventTotals,
	}
}

func (h *eventHandlerImpl) Handle(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Github::Handle")
	defer span.Finish()

	// https://developer.github.com/webhooks/
	eventType := c.GetHeader("X-Github-Event")
	h.prometheusInboundEventTotals.With(prometheus.Labels{"event": eventType, "source": "github"}).Inc()

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body from Github webhook failed")
		c.String(http.StatusInternalServerError, "Reading body from Github webhook failed")
		return
	}

	// verify hmac signature
	hasValidSignature, err := h.HasValidSignature(body, c.GetHeader("X-Hub-Signature"))
	if err != nil {
		log.Error().Err(err).Msg("Verifying signature from Github webhook failed")
		c.String(http.StatusInternalServerError, "Verifying signature from Github webhook failed")
		return
	}
	if !hasValidSignature {
		log.Warn().Msg("Signature from Github webhook is invalid")
		c.String(http.StatusBadRequest, "Signature from Github webhook is invalid")
		return
	}

	switch eventType {
	case "push": // Any Git push to a Repository, including editing tags or branches. Commits via API actions that update references are also counted. This is the default event.

		// unmarshal json body
		var pushEvent ghcontracts.PushEvent
		err := json.Unmarshal(body, &pushEvent)
		if err != nil {
			log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to GithubPushEvent failed")
			return
		}

		h.CreateJobForGithubPush(ctx, pushEvent)

	case
		"commit_comment",                        // Any time a Commit is commented on.
		"create",                                // Any time a Branch or Tag is created.
		"delete",                                // Any time a Branch or Tag is deleted.
		"deployment",                            // Any time a Repository has a new deployment created from the API.
		"deployment_status",                     // Any time a deployment for a Repository has a status update from the API.
		"fork",                                  // Any time a Repository is forked.
		"gollum",                                // Any time a Wiki page is updated.
		"installation",                          // Any time a GitHub App is installed or uninstalled.
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
		var repositoryEvent ghcontracts.RepositoryEvent
		err := json.Unmarshal(body, &repositoryEvent)
		if err != nil {
			log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to GithubRepositoryEvent failed")
			return
		}

		if repositoryEvent.Action == "renamed" && repositoryEvent.Changes.Repository.Name.From != "" && repositoryEvent.Repository.Name != "" {
			log.Info().Msgf("Renaming repository from %v to %v", repositoryEvent.Changes.Repository.Name.From, repositoryEvent.Repository.Name)
		}

	default:
		log.Warn().Str("event", eventType).Msgf("Unsupported Github webhook event of type '%v'", eventType)
	}

	c.String(http.StatusOK, "Aye aye!")
}

func (h *eventHandlerImpl) CreateJobForGithubPush(ctx context.Context, pushEvent ghcontracts.PushEvent) {

	span, ctx := opentracing.StartSpanFromContext(ctx, "Github::CreateJobForGithubPush")
	defer span.Finish()

	// check to see that it's a cloneable event
	if !strings.HasPrefix(pushEvent.Ref, "refs/heads/") {
		return
	}

	span.SetTag("git-repo", pushEvent.GetRepository())
	span.SetTag("git-branch", pushEvent.GetRepoBranch())
	span.SetTag("git-revision", pushEvent.GetRepoRevision())
	span.SetTag("event", "push")

	gitEvent := manifest.EstafetteGitEvent{
		Event:      "push",
		Repository: pushEvent.GetRepository(),
		Branch:     pushEvent.GetRepoBranch(),
	}

	// handle git triggers
	go func() {
		err := h.buildService.FireGitTriggers(ctx, gitEvent)
		if err != nil {
			log.Error().Err(err).
				Interface("gitEvent", gitEvent).
				Msg("Failed firing git triggers")
		}
	}()

	// get access token
	accessToken, err := h.apiClient.GetInstallationToken(ctx, pushEvent.Installation.ID)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving access token failed")
		return
	}

	// get manifest file
	manifestExists, manifestString, err := h.apiClient.GetEstafetteManifest(ctx, accessToken, pushEvent)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving Estafettte manifest failed")
		return
	}

	if !manifestExists {
		return
	}

	var commits []contracts.GitCommit
	for _, c := range pushEvent.Commits {
		commits = append(commits, contracts.GitCommit{
			Author: contracts.GitAuthor{
				Email:    c.Author.Email,
				Name:     c.Author.Name,
				Username: c.Author.UserName,
			},
			Message: c.Message,
		})
	}

	// create build object and hand off to build service
	_, err = h.buildService.CreateBuild(ctx, contracts.Build{
		RepoSource:   pushEvent.GetRepoSource(),
		RepoOwner:    pushEvent.GetRepoOwner(),
		RepoName:     pushEvent.GetRepoName(),
		RepoBranch:   pushEvent.GetRepoBranch(),
		RepoRevision: pushEvent.GetRepoRevision(),
		Manifest:     manifestString,
		Commits:      commits,

		Events: []manifest.EstafetteEvent{
			manifest.EstafetteEvent{
				Git: &gitEvent,
			},
		},
	}, false)

	if err != nil {
		log.Error().Err(err).Msgf("Failed creating build for pipeline %v/%v/%v with revision %v", pushEvent.GetRepoSource(), pushEvent.GetRepoOwner(), pushEvent.GetRepoName(), pushEvent.GetRepoRevision())
		return
	}

	log.Info().Msgf("Created build for pipeline %v/%v/%v with revision %v", pushEvent.GetRepoSource(), pushEvent.GetRepoOwner(), pushEvent.GetRepoName(), pushEvent.GetRepoRevision())

	go func() {
		err := h.pubsubAPIClient.SubscribeToPubsubTriggers(ctx, manifestString)
		if err != nil {
			log.Error().Err(err).Msgf("Failed subscribing to topics for pubsub triggers for build %v/%v/%v revision %v", pushEvent.GetRepoSource(), pushEvent.GetRepoOwner(), pushEvent.GetRepoName(), pushEvent.GetRepoRevision())
		}
	}()
}

func (h *eventHandlerImpl) HasValidSignature(body []byte, signatureHeader string) (bool, error) {

	// https://developer.github.com/webhooks/securing/
	signature := strings.Replace(signatureHeader, "sha1=", "", 1)
	actualMAC, err := hex.DecodeString(signature)
	if err != nil {
		return false, errors.New("Decoding hexadecimal X-Hub-Signature to byte array failed")
	}

	// calculate expected MAC
	mac := hmac.New(sha1.New, []byte(h.config.WebhookSecret))
	mac.Write(body)
	expectedMAC := mac.Sum(nil)

	// compare actual and expected MAC
	if hmac.Equal(actualMAC, expectedMAC) {
		return true, nil
	}

	log.Warn().
		Str("expectedMAC", hex.EncodeToString(expectedMAC)).
		Str("actualMAC", hex.EncodeToString(actualMAC)).
		Msg("Expected and actual MAC do not match")

	return false, nil
}
