package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/githubapi"
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

// Service handles http events for Github integration
//
//go:generate mockgen -package=github -destination ./mock.go -source=service.go
type Service interface {
	CreateJobForGithubPush(ctx context.Context, event githubapi.PushEvent) (err error)
	PublishGithubEvent(ctx context.Context, event manifest.EstafetteGithubEvent) (err error)
	HasValidSignature(ctx context.Context, body []byte, appIDHeader, signatureHeader string) (validSignature bool, err error)
	Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
	Archive(ctx context.Context, repoSource, repoOwner, repoName string) (err error)
	Unarchive(ctx context.Context, repoSource, repoOwner, repoName string) (err error)
	IsAllowedInstallation(ctx context.Context, installationID int) (isAllowed bool, organizations []*contracts.Organization)
}

// NewService returns a github.Service to handle incoming webhook events
func NewService(config *api.APIConfig, githubapiClient githubapi.Client, pubsubapiClient pubsubapi.Client, estafetteService estafette.Service, queueService queue.Service) Service {
	return &service{
		githubapiClient:  githubapiClient,
		pubsubapiClient:  pubsubapiClient,
		estafetteService: estafetteService,
		queueService:     queueService,
		config:           config,
	}
}

type service struct {
	githubapiClient  githubapi.Client
	pubsubapiClient  pubsubapi.Client
	estafetteService estafette.Service
	config           *api.APIConfig
	queueService     queue.Service
}

func (s *service) CreateJobForGithubPush(ctx context.Context, pushEvent githubapi.PushEvent) (err error) {

	// check to see that it's a cloneable event
	if !strings.HasPrefix(pushEvent.Ref, "refs/heads/") {
		return ErrNonCloneableEvent
	}

	if s.isBuildBlocked(pushEvent) {
		return api.ErrBlockedRepository
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

	app, installation, err := s.githubapiClient.GetAppAndInstallationByID(ctx, pushEvent.Installation.ID)
	if err != nil {
		log.Error().Err(err).Msgf("Retrieving app and installation by id %v failed", pushEvent.Installation.ID)
		return
	}
	if app == nil {
		log.Error().Msgf("App for installation id %v is nil", pushEvent.Installation.ID)
		return fmt.Errorf("App for installation id %v is nil", pushEvent.Installation.ID)
	}
	if installation == nil {
		log.Error().Msgf("Installation for installation id %v is nil", pushEvent.Installation.ID)
		return fmt.Errorf("Installation for installation id %v is nil", pushEvent.Installation.ID)
	}

	// get access token
	accessToken, err := s.githubapiClient.GetInstallationToken(ctx, *app, *installation)
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
	_, organizations := s.IsAllowedInstallation(ctx, pushEvent.Installation.ID)

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
		return
	}

	log.Debug().Msgf("Created build for pipeline %v/%v/%v with revision %v", pushEvent.GetRepoSource(), pushEvent.GetRepoOwner(), pushEvent.GetRepoName(), pushEvent.GetRepoRevision())

	go func() {
		// create new context to avoid cancellation impacting execution
		span, _ := opentracing.StartSpanFromContext(ctx, "github:AsyncSubscribeToPubsubTriggers")
		ctx = opentracing.ContextWithSpan(context.Background(), span)
		defer span.Finish()

		err := s.pubsubapiClient.SubscribeToPubsubTriggers(ctx, manifestString)
		if err != nil {
			log.Error().Err(err).Msgf("Failed subscribing to topics for pubsub triggers for build %v/%v/%v revision %v", pushEvent.GetRepoSource(), pushEvent.GetRepoOwner(), pushEvent.GetRepoName(), pushEvent.GetRepoRevision())
		}
	}()

	return nil
}

func (s *service) PublishGithubEvent(ctx context.Context, event manifest.EstafetteGithubEvent) (err error) {
	log.Debug().Msgf("Publishing github event '%v' for repository '%v' to topic...", event.Event, event.Repository)

	return s.queueService.PublishGithubEvent(ctx, event)
}

func (s *service) HasValidSignature(ctx context.Context, body []byte, appIDHeader, signatureHeader string) (bool, error) {

	id, err := strconv.Atoi(appIDHeader)
	if err != nil {
		return false, err
	}

	app, err := s.githubapiClient.GetAppByID(ctx, id)
	if err != nil {
		return false, err
	}

	if app == nil {
		return false, fmt.Errorf("App for id %v is nil", id)
	}

	// https://developer.github.com/webhooks/securing/
	signature := strings.Replace(signatureHeader, "sha1=", "", 1)
	actualMAC, err := hex.DecodeString(signature)
	if err != nil {
		return false, errors.New("Decoding hexadecimal X-Hub-Signature to byte array failed")
	}

	// calculate expected MAC
	mac := hmac.New(sha1.New, []byte(app.WebhookSecret))
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

func (s *service) IsAllowedInstallation(ctx context.Context, installationID int) (isAllowed bool, organizations []*contracts.Organization) {
	_, installation, err := s.githubapiClient.GetAppAndInstallationByID(ctx, installationID)
	if err != nil && errors.Is(err, githubapi.ErrMissingInstallation) {
		return false, []*contracts.Organization{}
	}

	if err != nil {
		log.Error().Err(err).Msgf("Failed getting github installation for id %v", installationID)
		return false, []*contracts.Organization{}
	}

	if installation == nil {
		log.Error().Err(err).Msgf("Github installation for id %v is nil", installationID)
		return false, []*contracts.Organization{}
	}

	return true, installation.Organizations
}
