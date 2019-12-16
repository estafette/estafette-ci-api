package slack

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/estafette/estafette-ci-api/clients/cockroachdb"
	"github.com/estafette/estafette-ci-api/clients/slackapi"
	"github.com/estafette/estafette-ci-api/config"
	slackdom "github.com/estafette/estafette-ci-api/domain/slack"
	"github.com/estafette/estafette-ci-api/services/estafette"
	contracts "github.com/estafette/estafette-ci-contracts"
	crypt "github.com/estafette/estafette-ci-crypt"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// Service handles http events for Slack integration
type Service interface {
	Handle(*gin.Context)
	HasValidVerificationToken(slackdom.SlashCommand) bool
}

type service struct {
	secretHelper                 crypt.SecretHelper
	config                       config.SlackConfig
	slackAPIClient               slackapi.Client
	cockroachDBClient            cockroachdb.Client
	apiConfig                    config.APIServerConfig
	buildService                 estafette.Service
	githubJobVarsFunc            func(context.Context, string, string, string) (string, string, error)
	bitbucketJobVarsFunc         func(context.Context, string, string, string) (string, string, error)
	prometheusInboundEventTotals *prometheus.CounterVec
}

// NewService returns a new slack.Service
func NewService(secretHelper crypt.SecretHelper, config config.SlackConfig, slackAPIClient slackapi.Client, cockroachDBClient cockroachdb.Client, apiConfig config.APIServerConfig, buildService estafette.Service, githubJobVarsFunc func(context.Context, string, string, string) (string, string, error), bitbucketJobVarsFunc func(context.Context, string, string, string) (string, string, error), prometheusInboundEventTotals *prometheus.CounterVec) Service {
	return &service{
		secretHelper:                 secretHelper,
		config:                       config,
		slackAPIClient:               slackAPIClient,
		cockroachDBClient:            cockroachDBClient,
		apiConfig:                    apiConfig,
		buildService:                 buildService,
		githubJobVarsFunc:            githubJobVarsFunc,
		bitbucketJobVarsFunc:         bitbucketJobVarsFunc,
		prometheusInboundEventTotals: prometheusInboundEventTotals,
	}
}

func (h *service) Handle(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Slack::Handle")
	defer span.Finish()

	// https://api.slack.com/slash-commands

	h.prometheusInboundEventTotals.With(prometheus.Labels{"event": "", "source": "slack"}).Inc()

	var slashCommand slackdom.SlashCommand
	// This will infer what binder to use depending on the content-type header.
	err := c.Bind(&slashCommand)
	if err != nil {
		log.Error().Err(err).Msg("Binding form data from Slack command webhook failed")
		c.String(http.StatusInternalServerError, "Binding form data from Slack command webhook failed")
		return
	}

	hasValidVerificationToken := h.HasValidVerificationToken(slashCommand)
	if !hasValidVerificationToken {
		log.Warn().Str("expectedToken", h.config.AppVerificationToken).Str("actualToken", slashCommand.Token).Msg("Verification token for Slack command is invalid")
		c.String(http.StatusBadRequest, "Verification token for Slack command is invalid")
		return
	}

	span.SetTag("slash-command", slashCommand.Command)

	if slashCommand.Command == "/estafette" {
		if slashCommand.Text != "" {
			splittedText := strings.Split(slashCommand.Text, " ")
			if splittedText != nil && len(splittedText) > 0 {
				command := splittedText[0]
				arguments := splittedText[1:len(splittedText)]
				switch command {
				case "encrypt":

					log.Debug().Msg("Handling slash command /estafette encrypt")

					encryptedString, err := h.secretHelper.Encrypt(strings.Join(arguments, " "))
					if err != nil {
						log.Error().Err(err).Interface("slashCommand", slashCommand).Msg("Failed to encrypt secret")
						c.String(http.StatusOK, "Incorrect usage of /estafette encrypt")
						return
					}

					c.String(http.StatusOK, fmt.Sprintf("estafette.secret(%v)", encryptedString))
					return

				case "release":

					log.Debug().Msg("Handling slash command /estafette release")

					// # release latest green version (for release branch?)
					// /estafette release github.com/estafette/estafette-ci-builder beta
					// /estafette release github.com/estafette/estafette-ci-api tooling

					// # release latest green version for particular branch
					// /estafette release github.com/estafette/estafette-ci-builder beta master

					// # release specific version
					// /estafette release github.com/estafette/estafette-ci-builder beta 0.0.47
					// /estafette release github.com/estafette/estafette-ci-api beta 0.0.130

					if len(arguments) < 3 {
						c.String(http.StatusOK, "You have to few arguments, the command has to be of type /estafette release <repo> <release> <version>")
						return
					}

					// retrieve pipeline with first argument
					fullRepoName := arguments[0]
					releaseName := arguments[1]
					buildVersion := arguments[2]

					fullRepoNameArray := strings.Split(fullRepoName, "/")
					if len(fullRepoNameArray) != 1 && len(fullRepoNameArray) != 3 {
						c.String(http.StatusOK, "Your repository needs to be of the form <repo name> or <repo source>/<repo owner>/<repo name>")
						return
					}

					var pipeline *contracts.Pipeline
					if len(fullRepoNameArray) == 1 {
						pipelines, err := h.cockroachDBClient.GetPipelinesByRepoName(ctx, fullRepoName, false)
						if err != nil {
							log.Error().Err(err).Msgf("Failed retrieving pipelines for repo name %v by name", fullRepoName)
							c.String(http.StatusOK, fmt.Sprintf("Retrieving the pipeline for repository %v from the database failed: %v", fullRepoName, err))
							return
						}
						if len(pipelines) <= 0 {
							c.String(http.StatusOK, fmt.Sprintf("The repo %v in your command does not have any estafette builds", fullRepoName))
							return
						}
						if len(pipelines) > 1 {
							commandsExample := ""
							for _, p := range pipelines {
								commandsExample += fmt.Sprintf("/estafette release %v/%v/%v %v %v\n", p.RepoSource, p.RepoOwner, p.RepoName, releaseName, buildVersion)
							}
							c.String(http.StatusOK, fmt.Sprintf("There are multiple pipelines with name %v, use the full name instead:\n%v", fullRepoName, commandsExample))
							return
						}
						pipeline = pipelines[0]
					} else {
						pipeline, err := h.cockroachDBClient.GetPipeline(ctx, fullRepoNameArray[0], fullRepoNameArray[1], fullRepoNameArray[2], false)
						if err != nil {
							c.String(http.StatusOK, fmt.Sprintf("Retrieving the pipeline for repository %v from the database failed: %v", fullRepoName, err))
							return
						}
						if pipeline == nil {
							c.String(http.StatusOK, fmt.Sprintf("The repo %v in your command does not have any estafette builds", fullRepoName))
							return
						}
					}

					// check if version exists
					builds, err := h.cockroachDBClient.GetPipelineBuildsByVersion(ctx, pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName, buildVersion, []string{"succeeded"}, 1, false)

					if err != nil {
						c.String(http.StatusOK, fmt.Sprintf("Retrieving the build for repository %v and version %v from the database failed: %v", fullRepoName, buildVersion, err))
						return
					}

					var build *contracts.Build
					// get first build
					if len(builds) > 0 {
						build = builds[0]
					}
					if build == nil {
						c.String(http.StatusOK, fmt.Sprintf("The version %v in your command does not exist", buildVersion))
						return
					}
					if build.BuildStatus != "succeeded" {
						c.String(http.StatusOK, fmt.Sprintf("The build for version %v is not successful and cannot be used", buildVersion))
						return
					}

					// check if release target exists
					releaseExists := false
					for _, releaseTarget := range build.ReleaseTargets {
						if releaseTarget.Name == releaseName {
							// todo support release action

							releaseExists = true
							break
						}
					}

					if !releaseExists {
						c.String(http.StatusOK, fmt.Sprintf("The release %v in your command is not defined in the manifest", releaseName))
						return
					}

					// get user profile from api to set email address for TriggeredBy
					profile, err := h.slackAPIClient.GetUserProfile(ctx, slashCommand.UserID)
					if err != nil {
						c.String(http.StatusOK, fmt.Sprintf("Failed retrieving Slack user profile for user id %v: %v", slashCommand.UserID, err))
						return
					}

					// create release object and hand off to build service
					createdRelease, err := h.buildService.CreateRelease(ctx, contracts.Release{
						Name:           releaseName,
						Action:         "", // no support for releas action yet
						RepoSource:     build.RepoSource,
						RepoOwner:      build.RepoOwner,
						RepoName:       build.RepoName,
						ReleaseVersion: buildVersion,

						Events: []manifest.EstafetteEvent{
							manifest.EstafetteEvent{
								Manual: &manifest.EstafetteManualEvent{
									UserID: profile.Email,
								},
							},
						},
					}, *build.ManifestObject, build.RepoBranch, build.RepoRevision, false)

					if err != nil {
						errorMessage := fmt.Sprintf("Failed creating release %v for pipeline %v/%v/%v version %v for release command issued by %v", releaseName, build.RepoSource, build.RepoOwner, build.RepoName, buildVersion, profile.Email)
						log.Error().Err(err).Msg(errorMessage)
						c.String(http.StatusOK, fmt.Sprintf("Inserting starting the release: %v", err))
						return
					}

					c.String(http.StatusOK, fmt.Sprintf("Started releasing version %v to %v: %vpipelines/%v/%v/%v/releases/%v/logs", buildVersion, releaseName, h.apiConfig.BaseURL, build.RepoSource, build.RepoOwner, build.RepoName, createdRelease.ID))
					return
				}
			}
		}
	}

	c.String(http.StatusOK, "Aye aye!")
}

func (h *service) HasValidVerificationToken(slashCommand slackdom.SlashCommand) bool {
	return slashCommand.Token == h.config.AppVerificationToken
}
