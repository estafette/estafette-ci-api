package cloudsource

import (
	"context"
	"errors"

	"github.com/estafette/estafette-ci-api/clients/cloudsourceapi"
	"github.com/estafette/estafette-ci-api/clients/pubsubapi"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/services/estafette"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/rs/zerolog/log"
)

var (
	ErrNonCloneableEvent = errors.New("The event is not cloneable")
	ErrNoManifest        = errors.New("The repository has no manifest at the pushed commit")
)

// Service handles pubsub events for Cloud Source Repository integration
type Service interface {
	CreateJobForCloudSourcePush(ctx context.Context, notification cloudsourceapi.PubSubNotification) (err error)
	IsWhitelistedProject(notification cloudsourceapi.PubSubNotification) (isWhiteListed bool)
}

// NewService returns a new bitbucket.Service
func NewService(config config.CloudSourceConfig, cloudsourceapiClient cloudsourceapi.Client, pubsubapiClient pubsubapi.Client, estafetteService estafette.Service) Service {
	return &service{
		config:               config,
		cloudsourceapiClient: cloudsourceapiClient,
		pubsubapiClient:      pubsubapiClient,
		estafetteService:     estafetteService,
	}
}

type service struct {
	config               config.CloudSourceConfig
	cloudsourceapiClient cloudsourceapi.Client
	pubsubapiClient      pubsubapi.Client
	estafetteService     estafette.Service
}

func (s *service) CreateJobForCloudSourcePush(ctx context.Context, notification cloudsourceapi.PubSubNotification) (err error) {

	// check to see that it's a cloneable event

	if notification.RefUpdateEvent == nil {
		return ErrNonCloneableEvent
	}

	var commits []contracts.GitCommit
	var repoBranch string
	var repoRevision string
	for _, refUpdate := range notification.RefUpdateEvent.RefUpdates {
		commits = append(commits, contracts.GitCommit{
			Author: contracts.GitAuthor{
				Email:    notification.RefUpdateEvent.Email,
				Name:     notification.RefUpdateEvent.GetAuthorName(),
				Username: notification.RefUpdateEvent.GetAuthorName(),
			},
			Message: refUpdate.NewId,
		})
		repoBranch = refUpdate.GetRepoBranch()
		repoRevision = refUpdate.NewId
	}

	gitEvent := manifest.EstafetteGitEvent{
		Event:      "push",
		Repository: notification.GetRepository(),
		Branch:     repoBranch,
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
	accessToken, err := s.cloudsourceapiClient.GetAccessToken(ctx)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving Access Token failed")
		return err
	}

	// get manifest file
	manifestExists, manifestString, err := s.cloudsourceapiClient.GetEstafetteManifest(ctx, accessToken, notification)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving Estafettte manifest failed")
		return err
	}

	if !manifestExists {
		return ErrNoManifest
	}

	// create build object and hand off to build service
	_, err = s.estafetteService.CreateBuild(ctx, contracts.Build{
		RepoSource:   notification.GetRepoSource(),
		RepoOwner:    notification.GetRepoOwner(),
		RepoName:     notification.GetRepoName(),
		RepoBranch:   repoBranch,
		RepoRevision: repoRevision,
		Manifest:     manifestString,
		Commits:      commits,

		Events: []manifest.EstafetteEvent{
			manifest.EstafetteEvent{
				Git: &gitEvent,
			},
		},
	}, false)
	if err != nil {
		log.Error().Err(err).Msgf("Failed creating build for pipeline %v/%v/%v with revision %v", notification.GetRepoSource(), notification.GetRepoOwner(), notification.GetRepoName(), repoRevision)
		return err
	}

	log.Info().Msgf("Created build for pipeline %v/%v/%v with revision %v", notification.GetRepoSource(), notification.GetRepoOwner(), notification.GetRepoName(), repoRevision)

	go func() {
		err := s.pubsubapiClient.SubscribeToPubsubTriggers(ctx, manifestString)
		if err != nil {
			log.Error().Err(err).Msgf("Failed subscribing to topics for pubsub triggers for build %v/%v/%v revision %v", notification.GetRepoSource(), notification.GetRepoOwner(), notification.GetRepoName(), repoRevision)
		}
	}()

	return nil
}

func (s *service) IsWhitelistedProject(notification cloudsourceapi.PubSubNotification) (isWhiteListed bool) {

	if len(s.config.WhitelistedProjects) == 0 {
		return true
	}

	for _, project := range s.config.WhitelistedProjects {
		if project == notification.GetRepoOwner() {
			return true
		}
	}

	return false
}
