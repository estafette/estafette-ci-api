package bitbucket

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	bitbucketclt "github.com/estafette/estafette-ci-api/clients/bitbucket"
	"github.com/estafette/estafette-ci-api/clients/pubsub"
	"github.com/estafette/estafette-ci-api/config"
	bitbucketdom "github.com/estafette/estafette-ci-api/domain/bitbucket"
	"github.com/estafette/estafette-ci-api/services/estafette"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// Service handles http events for Bitbucket integration
type Service interface {
	Handle(*gin.Context)
	CreateJobForBitbucketPush(context.Context, bitbucketdom.RepositoryPushEvent)
	Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error
}

type service struct {
	config                       config.BitbucketConfig
	apiClient                    bitbucketclt.Client
	pubsubAPIClient              pubsub.Client
	buildService                 estafette.Service
	prometheusInboundEventTotals *prometheus.CounterVec
}

// NewService returns a new bitbucket.Service
func NewService(config config.BitbucketConfig, apiClient bitbucketclt.Client, pubsubAPIClient pubsub.Client, buildService estafette.Service, prometheusInboundEventTotals *prometheus.CounterVec) Service {
	return &service{
		config:                       config,
		apiClient:                    apiClient,
		pubsubAPIClient:              pubsubAPIClient,
		buildService:                 buildService,
		prometheusInboundEventTotals: prometheusInboundEventTotals,
	}
}

func (h *service) Handle(c *gin.Context) {

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

	// unmarshal json body to check if installation is whitelisted
	var anyEvent bitbucketdom.AnyEvent
	err = json.Unmarshal(body, &anyEvent)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to BitbucketAnyEvent failed")
		return
	}

	// verify owner is whitelisted
	if !h.IsWhitelistedOwner(anyEvent.Repository) {
		c.Status(http.StatusUnauthorized)
		return
	}

	switch eventType {
	case "repo:push":

		// unmarshal json body
		var pushEvent bitbucketdom.RepositoryPushEvent
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

		// unmarshal json body
		var repoUpdatedEvent bitbucketdom.RepoUpdatedEvent
		err := json.Unmarshal(body, &repoUpdatedEvent)
		if err != nil {
			log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to BitbucketRepoUpdatedEvent failed")
			return
		}

		if repoUpdatedEvent.IsValidRenameEvent() {
			log.Info().Msgf("Renaming repository from %v/%v/%v to %v/%v/%v", repoUpdatedEvent.GetRepoSource(), repoUpdatedEvent.GetOldRepoOwner(), repoUpdatedEvent.GetOldRepoName(), repoUpdatedEvent.GetRepoSource(), repoUpdatedEvent.GetNewRepoOwner(), repoUpdatedEvent.GetNewRepoName())
			err = h.Rename(ctx, repoUpdatedEvent.GetRepoSource(), repoUpdatedEvent.GetOldRepoOwner(), repoUpdatedEvent.GetOldRepoName(), repoUpdatedEvent.GetRepoSource(), repoUpdatedEvent.GetNewRepoOwner(), repoUpdatedEvent.GetNewRepoName())
			if err != nil {
				log.Error().Err(err).Msgf("Failed renaming repository from %v/%v/%v to %v/%v/%v", repoUpdatedEvent.GetRepoSource(), repoUpdatedEvent.GetOldRepoOwner(), repoUpdatedEvent.GetOldRepoName(), repoUpdatedEvent.GetRepoSource(), repoUpdatedEvent.GetNewRepoOwner(), repoUpdatedEvent.GetNewRepoName())
				return
			}
		}

	default:
		log.Warn().Str("event", eventType).Msgf("Unsupported Bitbucket webhook event of type '%v'", eventType)
	}

	c.String(http.StatusOK, "Aye aye!")
}

func (h *service) CreateJobForBitbucketPush(ctx context.Context, pushEvent bitbucketdom.RepositoryPushEvent) {

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

func (h *service) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Bitbucket::Rename")
	defer span.Finish()

	return h.buildService.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (h *service) IsWhitelistedOwner(repository bitbucketdom.Repository) bool {

	if len(h.config.WhitelistedOwners) == 0 {
		return true
	}

	for _, owner := range h.config.WhitelistedOwners {
		if owner == repository.Owner.UserName {
			return true
		}
	}

	return false
}
