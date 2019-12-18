package bitbucket

import (
	"context"

	"github.com/estafette/estafette-ci-api/clients/bitbucketapi"
	"github.com/estafette/estafette-ci-api/clients/pubsubapi"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/services/estafette"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/rs/zerolog/log"
)

// Service handles http events for Bitbucket integration
type Service interface {
	CreateJobForBitbucketPush(ctx context.Context, event bitbucketapi.RepositoryPushEvent)
	Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
	IsWhitelistedOwner(repository bitbucketapi.Repository) (isWhiteListed bool)
}

// NewService returns a new bitbucket.Service
func NewService(config config.BitbucketConfig, bitbucketapiClient bitbucketapi.Client, pubsubapiClient pubsubapi.Client, estafetteService estafette.Service) Service {
	return &service{
		config:             config,
		bitbucketapiClient: bitbucketapiClient,
		pubsubapiClient:    pubsubapiClient,
		estafetteService:   estafetteService,
	}
}

type service struct {
	config             config.BitbucketConfig
	bitbucketapiClient bitbucketapi.Client
	pubsubapiClient    pubsubapi.Client
	estafetteService   estafette.Service
}

func (s *service) CreateJobForBitbucketPush(ctx context.Context, pushEvent bitbucketapi.RepositoryPushEvent) {

	// check to see that it's a cloneable event
	if len(pushEvent.Push.Changes) == 0 || pushEvent.Push.Changes[0].New == nil || pushEvent.Push.Changes[0].New.Type != "branch" || len(pushEvent.Push.Changes[0].New.Target.Hash) == 0 {
		return
	}

	gitEvent := manifest.EstafetteGitEvent{
		Event:      "push",
		Repository: pushEvent.GetRepository(),
		Branch:     pushEvent.GetRepoBranch(),
	}

	// handle git triggers
	go func() {
		err := s.estafetteService.FireGitTriggers(ctx, gitEvent)
		if err != nil {
			log.Error().Err(err).
				Interface("gitEvent", gitEvent).
				Msg("Failed firing git triggers")
		}
	}()

	// get access token
	accessToken, err := s.bitbucketapiClient.GetAccessToken(ctx)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving Estafettte manifest failed")
		return
	}

	// get manifest file
	manifestExists, manifestString, err := s.bitbucketapiClient.GetEstafetteManifest(ctx, accessToken, pushEvent)
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
	_, err = s.estafetteService.CreateBuild(ctx, contracts.Build{
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
		err := s.pubsubapiClient.SubscribeToPubsubTriggers(ctx, manifestString)
		if err != nil {
			log.Error().Err(err).Msgf("Failed subscribing to topics for pubsub triggers for build %v/%v/%v revision %v", pushEvent.GetRepoSource(), pushEvent.GetRepoOwner(), pushEvent.GetRepoName(), pushEvent.GetRepoRevision())
		}
	}()
}

func (s *service) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {
	return s.estafetteService.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *service) IsWhitelistedOwner(repository bitbucketapi.Repository) bool {

	if len(s.config.WhitelistedOwners) == 0 {
		return true
	}

	for _, owner := range s.config.WhitelistedOwners {
		if owner == repository.Owner.UserName {
			return true
		}
	}

	return false
}
