package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/estafette/estafette-ci-api/clients/githubapi"
	"github.com/estafette/estafette-ci-api/clients/pubsubapi"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/services/estafette"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// Service handles http events for Github integration
type Service interface {
	CreateJobForGithubPush(ctx context.Context, event githubapi.PushEvent)
	HasValidSignature(ctx context.Context, body []byte, signatureHeader string) (bool, error)
	Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error
	IsWhitelistedInstallation(ctx context.Context, installation githubapi.Installation) bool
}

type service struct {
	apiClient                    githubapi.Client
	pubsubAPIClient              pubsubapi.Client
	buildService                 estafette.Service
	config                       config.GithubConfig
	prometheusInboundEventTotals *prometheus.CounterVec
}

// NewService returns a github.Service to handle incoming webhook events
func NewService(apiClient githubapi.Client, pubsubAPIClient pubsubapi.Client, buildService estafette.Service, config config.GithubConfig, prometheusInboundEventTotals *prometheus.CounterVec) Service {
	return &service{
		apiClient:                    apiClient,
		pubsubAPIClient:              pubsubAPIClient,
		buildService:                 buildService,
		config:                       config,
		prometheusInboundEventTotals: prometheusInboundEventTotals,
	}
}

func (h *service) CreateJobForGithubPush(ctx context.Context, pushEvent githubapi.PushEvent) {

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

func (h *service) HasValidSignature(ctx context.Context, body []byte, signatureHeader string) (bool, error) {

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

func (h *service) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Github::Rename")
	defer span.Finish()

	return h.buildService.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (h *service) IsWhitelistedInstallation(ctx context.Context, installation githubapi.Installation) bool {

	if len(h.config.WhitelistedInstallations) == 0 {
		return true
	}

	for _, id := range h.config.WhitelistedInstallations {
		if id == installation.ID {
			return true
		}
	}

	return false
}
