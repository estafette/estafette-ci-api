package slack

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/estafette/estafette-ci-contracts"

	"github.com/estafette/estafette-ci-api/cockroach"
	"github.com/estafette/estafette-ci-api/config"
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
	prometheusInboundEventTotals *prometheus.CounterVec
}

// NewSlackEventHandler returns a new slack.EventHandler
func NewSlackEventHandler(secretHelper crypt.SecretHelper, config config.SlackConfig, cockroachDBClient cockroach.DBClient, apiConfig config.APIServerConfig, prometheusInboundEventTotals *prometheus.CounterVec) EventHandler {
	return &eventHandlerImpl{
		secretHelper:                 secretHelper,
		config:                       config,
		cockroachDBClient:            cockroachDBClient,
		apiConfig:                    apiConfig,
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

					pipeline, err := h.cockroachDBClient.GetPipeline(fullRepoNameArray[0], fullRepoNameArray[1], fullRepoNameArray[2])
					if err != nil {
						c.String(http.StatusOK, fmt.Sprintf("The repo %v in your command does not have any estafette builds", fullRepoName))
						return
					}

					// check if release target exists
					releaseExists := false
					for _, release := range pipeline.Releases {
						if release.Name == releaseName {
							releaseExists = false
							break
						}
					}

					if !releaseExists {
						c.String(http.StatusOK, fmt.Sprintf("The release %v in your command is not defined in the manifest", releaseName))
						return
					}

					// check if version exists
					build, err := h.cockroachDBClient.GetPipelineBuildByVersion(fullRepoNameArray[0], fullRepoNameArray[1], fullRepoNameArray[2], buildVersion)
					if err != nil {
						c.String(http.StatusOK, fmt.Sprintf("The version %v in your command does not exist", buildVersion))
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

					// start release job

					c.String(http.StatusOK, fmt.Sprintf("Started releasing version %v to %v: %vpipelines/%v/releases/%v", buildVersion, releaseName, h.apiConfig.BaseURL, fullRepoName, insertedRelease.ID))
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
