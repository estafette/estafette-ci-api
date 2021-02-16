package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/estafette/estafette-ci-api/api"
	"github.com/estafette/estafette-ci-api/clients/githubapi"
	"github.com/estafette/estafette-ci-api/clients/pubsubapi"
	"github.com/estafette/estafette-ci-api/services/estafette"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/rs/zerolog/log"
)

var (
	ErrNonCloneableEvent = errors.New("The event is not cloneable")
	ErrNoManifest        = errors.New("The repository has no manifest at the pushed commit")
)

// Service handles http events for Github integration
//go:generate mockgen -package=github -destination ./mock.go -source=service.go
type Service interface {
	CreateJobForGithubPush(ctx context.Context, event githubapi.PushEvent) (err error)
	HasValidSignature(ctx context.Context, body []byte, signatureHeader string) (validSignature bool, err error)
	Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
	Archive(ctx context.Context, repoSource, repoOwner, repoName string) (err error)
	Unarchive(ctx context.Context, repoSource, repoOwner, repoName string) (err error)
	IsAllowedInstallation(ctx context.Context, installation githubapi.Installation) (isAllowed bool, organizations []*contracts.Organization)
}

// NewService returns a github.Service to handle incoming webhook events
func NewService(config *api.APIConfig, githubapiClient githubapi.Client, pubsubapiClient pubsubapi.Client, estafetteService estafette.Service, gitEventTopic *api.GitEventTopic) Service {
	return &service{
		githubapiClient:  githubapiClient,
		pubsubapiClient:  pubsubapiClient,
		estafetteService: estafetteService,
		config:           config,
		gitEventTopic:    gitEventTopic,
	}
}

type service struct {
	githubapiClient  githubapi.Client
	pubsubapiClient  pubsubapi.Client
	estafetteService estafette.Service
	config           *api.APIConfig
	gitEventTopic    *api.GitEventTopic
}

func (s *service) CreateJobForGithubPush(ctx context.Context, pushEvent githubapi.PushEvent) (err error) {

	// check to see that it's a cloneable event
	if !strings.HasPrefix(pushEvent.Ref, "refs/heads/") {
		return ErrNonCloneableEvent
	}

	gitEvent := manifest.EstafetteGitEvent{
		Event:      "push",
		Repository: pushEvent.GetRepository(),
		Branch:     pushEvent.GetRepoBranch(),
	}

	// handle git triggers
	s.gitEventTopic.Publish("github.Service", api.GitEventTopicMessage{Ctx: ctx, Event: gitEvent})

	// get access token
	accessToken, err := s.githubapiClient.GetInstallationToken(ctx, pushEvent.Installation.ID)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving access token failed")
		return
	}

	// get manifest file
	manifestExists, manifestString, err := s.githubapiClient.GetEstafetteManifest(ctx, accessToken, pushEvent)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving Estafettte manifest failed")
		return
	}

	if !manifestExists {
		return ErrNoManifest
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

	// get organizations linked to integration
	_, organizations := s.IsAllowedInstallation(ctx, pushEvent.Installation)

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

	return nil
}

func (s *service) HasValidSignature(ctx context.Context, body []byte, signatureHeader string) (bool, error) {

	// https://developer.github.com/webhooks/securing/
	signature := strings.Replace(signatureHeader, "sha1=", "", 1)
	actualMAC, err := hex.DecodeString(signature)
	if err != nil {
		return false, errors.New("Decoding hexadecimal X-Hub-Signature to byte array failed")
	}

	// calculate expected MAC
	mac := hmac.New(sha1.New, []byte(s.config.Integrations.Github.WebhookSecret))
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

func (s *service) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {
	return s.estafetteService.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *service) Archive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	return s.estafetteService.Archive(ctx, repoSource, repoOwner, repoName)
}

func (s *service) Unarchive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	return s.estafetteService.Unarchive(ctx, repoSource, repoOwner, repoName)
}

func (s *service) IsAllowedInstallation(ctx context.Context, installation githubapi.Installation) (isAllowed bool, organizations []*contracts.Organization) {

	if len(s.config.Integrations.Github.InstallationOrganizations) == 0 {
		return true, []*contracts.Organization{}
	}

	for _, io := range s.config.Integrations.Github.InstallationOrganizations {
		if io.Installation == installation.ID {
			return true, io.Organizations
		}
	}

	return false, []*contracts.Organization{}
}
