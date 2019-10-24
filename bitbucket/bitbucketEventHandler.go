package bitbucket

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	bbcontracts "github.com/estafette/estafette-ci-api/bitbucket/contracts"
	"github.com/estafette/estafette-ci-api/estafette"
	"github.com/estafette/estafette-ci-api/pubsub"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// EventHandler handles http events for Bitbucket integration
type EventHandler interface {
	Handle(*gin.Context)
	CreateJobForBitbucketPush(context.Context, bbcontracts.RepositoryPushEvent)
}

type eventHandlerImpl struct {
	apiClient                    APIClient
	pubsubAPIClient              pubsub.APIClient
	buildService                 estafette.BuildService
	prometheusInboundEventTotals *prometheus.CounterVec
}

// NewBitbucketEventHandler returns a new bitbucket.EventHandler
func NewBitbucketEventHandler(apiClient APIClient, pubsubAPIClient pubsub.APIClient, buildService estafette.BuildService, prometheusInboundEventTotals *prometheus.CounterVec) EventHandler {
	return &eventHandlerImpl{
		apiClient:                    apiClient,
		pubsubAPIClient:              pubsubAPIClient,
		buildService:                 buildService,
		prometheusInboundEventTotals: prometheusInboundEventTotals,
	}
}

func (h *eventHandlerImpl) Handle(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Bitbucket::Handle")
	defer span.Finish()

	// https://confluence.atlassian.com/bitbucket/manage-webhooks-735643732.html

	eventType := c.GetHeader("X-Event-Key")
	h.prometheusInboundEventTotals.With(prometheus.Labels{"event": eventType, "source": "bitbucket"}).Inc()

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body from Bitbucket webhook failed")
		c.String(http.StatusInternalServerError, "Reading body from Bitbucket webhook failed")
		return
	}

	switch eventType {
	case "repo:push":

		// unmarshal json body
		var pushEvent bbcontracts.RepositoryPushEvent
		err := json.Unmarshal(body, &pushEvent)
		if err != nil {
			log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to BitbucketRepositoryPushEvent failed")
			return
		}

		h.CreateJobForBitbucketPush(ctx, pushEvent)

	case
		"repo:fork",
		"repo:transfer",
		"repo:created",
		"repo:deleted",
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

	default:
		log.Warn().Str("event", eventType).Msgf("Unsupported Bitbucket webhook event of type '%v'", eventType)
	}

	c.String(http.StatusOK, "Aye aye!")
}

func (h *eventHandlerImpl) CreateJobForBitbucketPush(ctx context.Context, pushEvent bbcontracts.RepositoryPushEvent) {

	span, ctx := opentracing.StartSpanFromContext(ctx, "Bitbucket::CreateJobForBitbucketPush")
	defer span.Finish()

	// check to see that it's a cloneable event
	if len(pushEvent.Push.Changes) == 0 || pushEvent.Push.Changes[0].New == nil || pushEvent.Push.Changes[0].New.Type != "branch" || len(pushEvent.Push.Changes[0].New.Target.Hash) == 0 {
		return
	}

	span.SetTag("git-repo", pushEvent.GetRepository())
	span.SetTag("git-branch", pushEvent.GetRepoBranch())
	span.SetTag("git-revision", pushEvent.GetRepoRevision())
	span.SetTag("event", "repo:push")

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
	accessToken, err := h.apiClient.GetAccessToken(ctx)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving Estafettte manifest failed")
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
	for _, c := range pushEvent.Push.Changes {
		if len(c.Commits) > 0 {
			commits = append(commits, contracts.GitCommit{
				Author: contracts.GitAuthor{
					Email:    c.Commits[0].Author.GetEmailAddress(),
					Name:     c.Commits[0].Author.GetName(),
					Username: c.Commits[0].Author.Username,
				},
				Message: c.Commits[0].GetCommitMessage(),
			})
		}
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
