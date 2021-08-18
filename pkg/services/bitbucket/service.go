package bitbucket

import (
	"context"
	"errors"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/bitbucketapi"
	"github.com/estafette/estafette-ci-api/pkg/clients/pubsubapi"
	"github.com/estafette/estafette-ci-api/pkg/services/estafette"
	"github.com/estafette/estafette-ci-api/pkg/services/queue"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
)

var (
	ErrNonCloneableEvent = errors.New("The event is not cloneable")
	ErrNoManifest        = errors.New("The repository has no manifest at the pushed commit")
)

// Service handles http events for Bitbucket integration
//go:generate mockgen -package=bitbucket -destination ./mock.go -source=service.go
type Service interface {
	CreateJobForBitbucketPush(ctx context.Context, event bitbucketapi.RepositoryPushEvent) (err error)
	PublishBitbucketEvent(ctx context.Context, event manifest.EstafetteBitbucketEvent) (err error)
	Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
	Archive(ctx context.Context, repoSource, repoOwner, repoName string) (err error)
	Unarchive(ctx context.Context, repoSource, repoOwner, repoName string) (err error)
	IsAllowedOwner(repository *bitbucketapi.Repository) (isAllowed bool, organizations []*contracts.Organization)
}

// NewService returns a new bitbucket.Service
func NewService(config *api.APIConfig, bitbucketapiClient bitbucketapi.Client, pubsubapiClient pubsubapi.Client, estafetteService estafette.Service, queueService queue.Service) Service {
	return &service{
		config:             config,
		bitbucketapiClient: bitbucketapiClient,
		pubsubapiClient:    pubsubapiClient,
		estafetteService:   estafetteService,
		queueService:       queueService,
	}
}

type service struct {
	config             *api.APIConfig
	bitbucketapiClient bitbucketapi.Client
	pubsubapiClient    pubsubapi.Client
	estafetteService   estafette.Service
	queueService       queue.Service
}

func (s *service) CreateJobForBitbucketPush(ctx context.Context, pushEvent bitbucketapi.RepositoryPushEvent) (err error) {

	// check to see that it's a cloneable event
	if len(pushEvent.Data.Push.Changes) == 0 || pushEvent.Data.Push.Changes[0].New == nil || pushEvent.Data.Push.Changes[0].New.Type != "branch" || len(pushEvent.Data.Push.Changes[0].New.Target.Hash) == 0 {
		return ErrNonCloneableEvent
	}

	gitEvent := manifest.EstafetteGitEvent{
		Event:      "push",
		Repository: pushEvent.GetRepository(),
		Branch:     pushEvent.GetRepoBranch(),
	}

	// handle git triggers
	err = s.queueService.PublishGitEvent(ctx, gitEvent)
	if err != nil {
		return
	}

	// get access token
	accessToken, err := s.bitbucketapiClient.GetAccessToken(ctx)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving Estafettte manifest failed")
		return err
	}

	// get manifest file
	manifestExists, manifestString, err := s.bitbucketapiClient.GetEstafetteManifest(ctx, accessToken, pushEvent)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving Estafettte manifest failed")
		return err
	}

	if !manifestExists {
		return ErrNoManifest
	}

	var commits []contracts.GitCommit
	for _, c := range pushEvent.Data.Push.Changes {
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

	// get organizations linked to integration
	_, organizations := s.IsAllowedOwner(&pushEvent.Data.Repository)

	// create build object and hand off to build service
	_, err = s.estafetteService.CreateBuild(ctx, contracts.Build{
		RepoSource:    pushEvent.GetRepoSource(),
		RepoOwner:     pushEvent.GetRepoOwner(),
		RepoName:      pushEvent.GetRepoName(),
		RepoBranch:    pushEvent.GetRepoBranch(),
		RepoRevision:  pushEvent.GetRepoRevision(),
		Manifest:      manifestString,
		Commits:       commits,
		Organizations: organizations,
		Events: []manifest.EstafetteEvent{
			{
				Fired: true,
				Git:   &gitEvent,
			},
		},
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed creating build for pipeline %v/%v/%v with revision %v", pushEvent.GetRepoSource(), pushEvent.GetRepoOwner(), pushEvent.GetRepoName(), pushEvent.GetRepoRevision())
		return err
	}

	log.Info().Msgf("Created build for pipeline %v/%v/%v with revision %v", pushEvent.GetRepoSource(), pushEvent.GetRepoOwner(), pushEvent.GetRepoName(), pushEvent.GetRepoRevision())

	go func() {
		// create new context to avoid cancellation impacting execution
		span, _ := opentracing.StartSpanFromContext(ctx, "bitucket:AsyncSubscribeToPubsubTriggers")
		ctx = opentracing.ContextWithSpan(context.Background(), span)
		defer span.Finish()

		err := s.pubsubapiClient.SubscribeToPubsubTriggers(ctx, manifestString)
		if err != nil {
			log.Error().Err(err).Msgf("Failed subscribing to topics for pubsub triggers for build %v/%v/%v revision %v", pushEvent.GetRepoSource(), pushEvent.GetRepoOwner(), pushEvent.GetRepoName(), pushEvent.GetRepoRevision())
		}
	}()

	return nil
}

func (s *service) PublishBitbucketEvent(ctx context.Context, event manifest.EstafetteBitbucketEvent) (err error) {
	log.Info().Msgf("Publishing bitbucket event '%v' for repository '%v' to topic...", event.Event, event.Repository)

	return s.queueService.PublishBitbucketEvent(ctx, event)
}

func (s *service) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	return s.estafetteService.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *service) Archive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	return s.estafetteService.Archive(ctx, repoSource, repoOwner, repoName)
}

func (s *service) Unarchive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	return s.estafetteService.Unarchive(ctx, repoSource, repoOwner, repoName)
}

func (s *service) IsAllowedOwner(repository *bitbucketapi.Repository) (isAllowed bool, organizations []*contracts.Organization) {
	if len(s.config.Integrations.Bitbucket.OwnerOrganizations) == 0 {
		return true, []*contracts.Organization{}
	}

	if repository == nil {
		return false, []*contracts.Organization{}
	}

	for _, oo := range s.config.Integrations.Bitbucket.OwnerOrganizations {
		if oo.Owner == repository.Owner.UserName {
			return true, oo.Organizations
		}
	}

	return false, []*contracts.Organization{}
}
