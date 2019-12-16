package bitbucket

import (
	"context"

	"github.com/estafette/estafette-ci-api/clients/bitbucketapi"
	"github.com/estafette/estafette-ci-api/clients/pubsubapi"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/services/estafette"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// Service handles http events for Bitbucket integration
type Service interface {
	CreateJobForBitbucketPush(context.Context, bitbucketapi.RepositoryPushEvent)
	Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error
	IsWhitelistedOwner(repository bitbucketapi.Repository) bool
}

type service struct {
	config                       config.BitbucketConfig
	apiClient                    bitbucketapi.Client
	pubsubAPIClient              pubsubapi.Client
	buildService                 estafette.Service
	prometheusInboundEventTotals *prometheus.CounterVec
}

// NewService returns a new bitbucket.Service
func NewService(config config.BitbucketConfig, apiClient bitbucketapi.Client, pubsubAPIClient pubsubapi.Client, buildService estafette.Service, prometheusInboundEventTotals *prometheus.CounterVec) Service {
	return &service{
		config:                       config,
		apiClient:                    apiClient,
		pubsubAPIClient:              pubsubAPIClient,
		buildService:                 buildService,
		prometheusInboundEventTotals: prometheusInboundEventTotals,
	}
}

func (h *service) CreateJobForBitbucketPush(ctx context.Context, pushEvent bitbucketapi.RepositoryPushEvent) {

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

func (h *service) IsWhitelistedOwner(repository bitbucketapi.Repository) bool {

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
