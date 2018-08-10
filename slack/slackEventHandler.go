package slack

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/estafette/estafette-ci-manifest"

	"github.com/estafette/estafette-ci-contracts"

	"github.com/estafette/estafette-ci-api/cockroach"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/estafette"
	crypt "github.com/estafette/estafette-ci-crypt"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// EventHandler handles http events for Slack integration
type EventHandler interface {
	Handle(*gin.Context)
	HasValidVerificationToken(SlashCommand) bool
}

type eventHandlerImpl struct {
	secretHelper                 crypt.SecretHelper
	config                       config.SlackConfig
	cockroachDBClient            cockroach.DBClient
	apiConfig                    config.APIServerConfig
	ciBuilderClient              estafette.CiBuilderClient
	prometheusInboundEventTotals *prometheus.CounterVec
}

// NewSlackEventHandler returns a new slack.EventHandler
func NewSlackEventHandler(secretHelper crypt.SecretHelper, config config.SlackConfig, cockroachDBClient cockroach.DBClient, apiConfig config.APIServerConfig, ciBuilderClient estafette.CiBuilderClient, prometheusInboundEventTotals *prometheus.CounterVec) EventHandler {
	return &eventHandlerImpl{
		secretHelper:                 secretHelper,
		config:                       config,
		cockroachDBClient:            cockroachDBClient,
		apiConfig:                    apiConfig,
		ciBuilderClient:              ciBuilderClient,
		prometheusInboundEventTotals: prometheusInboundEventTotals,
	}
}

func (h *eventHandlerImpl) Handle(c *gin.Context) {

	// https://api.slack.com/slash-commands

	h.prometheusInboundEventTotals.With(prometheus.Labels{"event": "", "source": "slack"}).Inc()

	var slashCommand SlashCommand
	// This will infer what binder to use depending on the content-type header.
	err := c.Bind(&slashCommand)
	if err != nil {
		log.Error().Err(err).Msg("Binding form data from Slack command webhook failed")
		c.String(http.StatusInternalServerError, "Binding form data from Slack command webhook failed")
		return
	}

	log.Debug().Interface("slashCommand", slashCommand).Msg("Deserialized slash command")

	hasValidVerificationToken := h.HasValidVerificationToken(slashCommand)
	if !hasValidVerificationToken {
		log.Warn().Str("expectedToken", h.config.AppVerificationToken).Str("actualToken", slashCommand.Token).Msg("Verification token for Slack command is invalid")
		c.String(http.StatusBadRequest, "Verification token for Slack command is invalid")
		return
	}

	if slashCommand.Command == "/estafette" {
		if slashCommand.Text != "" {
			splittedText := strings.Split(slashCommand.Text, " ")
			if splittedText != nil && len(splittedText) > 0 {
				command := splittedText[0]
				arguments := splittedText[1:len(splittedText)]
				switch command {
				case "encrypt":

					encryptedString, err := h.secretHelper.Encrypt(strings.Join(arguments, " "))
					if err != nil {
						log.Error().Err(err).Interface("slashCommand", slashCommand).Msg("Failed to encrypt secret")
						c.String(http.StatusOK, "Incorrect usage of /estafette encrypt!")
						return
					}

					c.String(http.StatusOK, fmt.Sprintf("estafette.secret(%v)", encryptedString))
					return

				case "release":

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
						pipelines, err := h.cockroachDBClient.GetPipelinesByRepoName(fullRepoName)
						if err != nil {
							log.Error().Err(err).Msgf("Failed retrieving pipelines for repo name %v by name", fullRepoName)
							c.String(http.StatusOK, fmt.Sprintf("Retrieving the pipeline for repository %v from the database failed: %v", fullRepoName, err))
							return
						}
						log.Debug().Msgf("Retrieved %v pipelines for repo name %v", len(pipelines), fullRepoName)
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
						pipeline, err := h.cockroachDBClient.GetPipeline(fullRepoNameArray[0], fullRepoNameArray[1], fullRepoNameArray[2])
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
					build, err := h.cockroachDBClient.GetPipelineBuildByVersion(pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName, buildVersion)
					if err == nil && build != nil && build.BuildStatus == "succeeded" {
						// check if release target exists
						releaseExists := false
						for _, release := range build.Releases {
							if release.Name == releaseName {
								releaseExists = true
								break
							}
						}

						if !releaseExists {

							log.Debug().Interface("releases", build.Releases).Msgf("Release %v for repo name %v can't be found", releaseName, fullRepoName)

							c.String(http.StatusOK, fmt.Sprintf("The release %v in your command is not defined in the manifest", releaseName))
							return
						}
					}
					if err != nil {
						c.String(http.StatusOK, fmt.Sprintf("Retrieving the build for repository %v and version %v from the database failed: %v", fullRepoName, buildVersion, err))
						return
					}
					if build == nil {
						c.String(http.StatusOK, fmt.Sprintf("The version %v in your command does not exist", buildVersion))
						return
					}
					if build.BuildStatus != "succeeded" {
						c.String(http.StatusOK, fmt.Sprintf("The build for version %v is not successful and cannot be used", buildVersion))
						return
					}

					// create release in database
					release := contracts.Release{
						Name:           releaseName,
						RepoSource:     build.RepoSource,
						RepoOwner:      build.RepoOwner,
						RepoName:       build.RepoName,
						ReleaseVersion: buildVersion,
						ReleaseStatus:  "running",
						TriggeredBy:    slashCommand.UserName,
					}
					insertedRelease, err := h.cockroachDBClient.InsertRelease(release)
					if err != nil {
						c.String(http.StatusOK, fmt.Sprintf("Inserting the release into the database failed: %v", err))
						return
					}

					// get authenticated url
					var authenticatedRepositoryURL string
					// switch build.RepoSource {
					// case "github.com":
					// 	// get access token
					// 	accessToken, err := w.apiClient.GetInstallationToken(pushEvent.Installation.ID)
					// 	if err != nil {
					// 		log.Error().Err(err).
					// 			Msg("Retrieving access token failed")
					// 		return
					// 	}
					// 	// get authenticated url for the repository
					// 	authenticatedRepositoryURL, err := w.apiClient.GetAuthenticatedRepositoryURL(accessToken, pushEvent.Repository.HTMLURL)
					// 	if err != nil {
					// 		log.Error().Err(err).
					// 			Msg("Retrieving authenticated repository failed")
					// 		return
					// 	}

					// case "bitbucket.org":
					// 	// get access token
					// 	accessToken, err := w.apiClient.GetAccessToken()
					// 	if err != nil {
					// 		log.Error().Err(err).
					// 			Msg("Retrieving Estafettte manifest failed")
					// 		return
					// 	}
					// 	// get authenticated url for the repository
					// 	authenticatedRepositoryURL, err := w.apiClient.GetAuthenticatedRepositoryURL(accessToken, fmt.Sprintf("https://bitbucket.org/%v/%v", build.RepoOwner, build.RepoName))
					// 	if err != nil {
					// 		log.Error().Err(err).
					// 			Msg("Retrieving authenticated repository failed")
					// 		return
					// 	}
					// }

					manifest, err := manifest.ReadManifest(build.Manifest)
					if err != nil {

					}

					// start release job
					ciBuilderParams := estafette.CiBuilderParams{
						RepoSource:   build.RepoSource,
						RepoFullName: fmt.Sprintf("%v/%v", build.RepoOwner, build.RepoName),
						RepoURL:      authenticatedRepositoryURL,
						RepoBranch:   build.RepoBranch,
						RepoRevision: build.RepoRevision,
						//EnvironmentVariables: map[string]string{"ESTAFETTE_BITBUCKET_API_TOKEN": accessToken.AccessToken}, // EnvironmentVariables: map[string]string{"ESTAFETTE_GITHUB_API_TOKEN": accessToken.Token},
						Track: manifest.Builder.Track,
						//AutoIncrement:    autoincrement,
						VersionNumber:    buildVersion,
						HasValidManifest: true,
						Manifest:         manifest,
						ReleaseID:        insertedRelease.ID,
						ReleaseName:      releaseName,
					}

					_, err = h.ciBuilderClient.CreateCiBuilderJob(ciBuilderParams)
					if err != nil {
						c.String(http.StatusOK, fmt.Sprintf("Creating the release job failed: %v", err))
						return
					}

					c.String(http.StatusOK, fmt.Sprintf("Started releasing version %v to %v: %vpipelines/%v/%v/%v/releases/%v/logs", buildVersion, releaseName, h.apiConfig.BaseURL, build.RepoSource, build.RepoOwner, build.RepoName, insertedRelease.ID))
					return
				}
			}
		}
	}

	c.String(http.StatusOK, "Aye aye!")
}

func (h *eventHandlerImpl) HasValidVerificationToken(slashCommand SlashCommand) bool {
	return slashCommand.Token == h.config.AppVerificationToken
}
