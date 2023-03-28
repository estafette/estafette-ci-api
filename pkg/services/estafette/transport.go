package estafette

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/estafette/estafette-ci-api/pkg/migrationpb"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/builderapi"
	"github.com/estafette/estafette-ci-api/pkg/clients/cloudstorage"
	"github.com/estafette/estafette-ci-api/pkg/clients/database"
	contracts "github.com/estafette/estafette-ci-contracts"
	crypt "github.com/estafette/estafette-ci-crypt"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v2"
)

// NewHandler returns a new estafette.Handler
func NewHandler(templatesPath string, config *api.APIConfig, encryptedConfig *api.APIConfig, databaseClient database.Client, cloudStorageClient cloudstorage.Client, ciBuilderClient builderapi.Client, buildService Service, warningHelper api.WarningHelper, secretHelper crypt.SecretHelper, gcsMigratorClient migrationpb.ServiceClient) Handler {
	h := Handler{
		templatesPath:      templatesPath,
		config:             config,
		encryptedConfig:    encryptedConfig,
		databaseClient:     databaseClient,
		cloudStorageClient: cloudStorageClient,
		ciBuilderClient:    ciBuilderClient,
		buildService:       buildService,
		warningHelper:      warningHelper,
		secretHelper:       secretHelper,
		// !! Migration changes !!
		gcsMigratorClient: gcsMigratorClient,
	}
	return h
}

type Handler struct {
	templatesPath      string
	config             *api.APIConfig
	encryptedConfig    *api.APIConfig
	databaseClient     database.Client
	cloudStorageClient cloudstorage.Client
	ciBuilderClient    builderapi.Client
	buildService       Service
	warningHelper      api.WarningHelper
	secretHelper       crypt.SecretHelper
	// !! Migration changes !!
	gcsMigratorClient migrationpb.ServiceClient
}

func (h *Handler) GetPipelines(c *gin.Context) {

	pageNumber, pageSize, filters, sortings := api.GetQueryParameters(c)

	// filter on organizations / groups
	filters = api.SetPermissionsFilters(c, filters)

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			pipelines, err := h.databaseClient.GetPipelines(c.Request.Context(), pageNumber, pageSize, filters, sortings, true)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(pipelines))
			for i := range pipelines {
				items[i] = pipelines[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetPipelinesCount(c.Request.Context(), filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving pipelines or count from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetPipeline(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	filters := api.GetPipelineFilters(c)

	pipeline, err := h.databaseClient.GetPipeline(c.Request.Context(), source, owner, repo, filters, true)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving pipeline for %v/%v/%v from db", source, owner, repo)
	}
	if pipeline == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline not found"})
		return
	}

	c.JSON(http.StatusOK, pipeline)
}

func (h *Handler) GetPipelineRecentBuilds(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	builds, err := h.databaseClient.GetPipelineRecentBuilds(c.Request.Context(), source, owner, repo, true)
	if err != nil {
		log.Error().Err(err).Msgf("Failed retrieving recent builds for %v/%v/%v from db", source, owner, repo)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, builds)
}

func (h *Handler) GetPipelineBuildBranches(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	pageNumber, pageSize, filters, _ := api.GetQueryParameters(c)

	// filter on organizations / groups
	filters = api.SetPermissionsFilters(c, filters)

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			buildBranches, err := h.databaseClient.GetPipelineBuildBranches(c.Request.Context(), source, owner, repo, pageNumber, pageSize, filters)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(buildBranches))
			for i := range buildBranches {
				items[i] = buildBranches[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetPipelineBuildBranchesCount(c.Request.Context(), source, owner, repo, filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving pipeline build branches from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetPipelineBuilds(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	pageNumber, pageSize, filters, sortings := api.GetQueryParameters(c)

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			builds, err := h.databaseClient.GetPipelineBuilds(c.Request.Context(), source, owner, repo, pageNumber, pageSize, filters, sortings, true)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(builds))
			for i := range builds {
				items[i] = builds[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetPipelineBuildsCount(c.Request.Context(), source, owner, repo, filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msgf("Failed retrieving builds for %v/%v/%v from db", source, owner, repo)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetPipelineBuild(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	revisionOrID := c.Param("revisionOrId")

	if len(revisionOrID) == 40 {

		build, err := h.databaseClient.GetPipelineBuild(c.Request.Context(), source, owner, repo, revisionOrID, false)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, revisionOrID)
		}
		if build == nil {
			c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline not found"})
			return
		}

		c.JSON(http.StatusOK, build)
		return
	}

	build, err := h.databaseClient.GetPipelineBuildByID(c.Request.Context(), source, owner, repo, revisionOrID, false)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, revisionOrID)
	}
	if build == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline build not found"})
		return
	}

	// obfuscate all secrets
	build.Manifest, err = h.obfuscateSecrets(build.Manifest)
	if err != nil {
		log.Error().Err(err).Msgf("Failed obfuscating secrets")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}
	build.ManifestWithDefaults, err = h.obfuscateSecrets(build.ManifestWithDefaults)
	if err != nil {
		log.Error().Err(err).Msgf("Failed obfuscating secrets")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, build)
}

func (h *Handler) CreatePipelineBuild(c *gin.Context) {

	if !api.RequestTokenIsValid(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusText(http.StatusUnauthorized), "message": "JWT is invalid"})
		return
	}

	claims := jwt.ExtractClaims(c)
	email := claims["email"].(string)

	var buildCommand contracts.Build
	err := c.BindJSON(&buildCommand)
	if err != nil {
		errorMessage := "Binding CreatePipelineBuild body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	// match source, owner, repo with values in binded release
	if buildCommand.RepoSource != c.Param("source") {
		errorMessage := fmt.Sprintf("RepoSource in path and post data do not match for pipeline %v/%v/%v for build command issued by %v", buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, email)
		log.Error().Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}
	if buildCommand.RepoOwner != c.Param("owner") {
		errorMessage := fmt.Sprintf("RepoOwner in path and post data do not match for pipeline %v/%v/%v for build command issued by %v", buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, email)
		log.Error().Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}
	if buildCommand.RepoName != c.Param("repo") {
		errorMessage := fmt.Sprintf("RepoName in path and post data do not match for pipeline %v/%v/%v for build command issued by %v", buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, email)
		log.Error().Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	// check if version exists and is valid to re-run
	failedBuilds, err := h.databaseClient.GetPipelineBuildsByVersion(c.Request.Context(), buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, buildCommand.BuildVersion, []contracts.Status{contracts.StatusFailed, contracts.StatusCanceled}, 1, false)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving build %v/%v/%v version %v for build command issued by %v", buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, buildCommand.BuildVersion, email)
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	nonFailedBuilds, err := h.databaseClient.GetPipelineBuildsByVersion(c.Request.Context(), buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, buildCommand.BuildVersion, []contracts.Status{contracts.StatusSucceeded, contracts.StatusRunning, contracts.StatusPending, contracts.StatusCanceling}, 1, false)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving build %v/%v/%v version %v for build command issued by %v", buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, buildCommand.BuildVersion, email)
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	var failedBuild *contracts.Build
	// get first failed build
	if len(failedBuilds) > 0 {
		failedBuild = failedBuilds[0]
	}

	// ensure there's no succeeded or running builds
	hasNonFailedBuilds := len(nonFailedBuilds) > 0

	if failedBuild == nil {
		errorMessage := fmt.Sprintf("No failed build %v/%v/%v version %v for build command issued by %v", buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, buildCommand.BuildVersion, email)
		log.Error().Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}
	if hasNonFailedBuilds {
		errorMessage := fmt.Sprintf("Version %v of pipeline %v/%v/%v has builds that are succeeded or running; only if all builds are failed the pipeline can be re-run; build command issued by %v", buildCommand.BuildVersion, buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, email)
		log.Error().Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	// set trigger event to manual
	failedBuild.Events = []manifest.EstafetteEvent{
		{
			Fired: true,
			Manual: &manifest.EstafetteManualEvent{
				UserID: email,
			},
		},
	}

	// hand off to build service
	createdBuild, err := h.buildService.CreateBuild(c.Request.Context(), *failedBuild)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed creating build %v/%v/%v version %v for build command issued by %v", buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, buildCommand.BuildVersion, email)
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	c.JSON(http.StatusCreated, createdBuild)
}

func (h *Handler) CancelPipelineBuild(c *gin.Context) {

	if !api.RequestTokenIsValid(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusText(http.StatusUnauthorized), "message": "JWT is invalid"})
		return
	}

	claims := jwt.ExtractClaims(c)
	email := claims["email"].(string)

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	revisionOrID := c.Param("revisionOrId")

	log.Debug().Msgf("Canceling pipeline build %v/%v/%v with id %v...", source, owner, repo, revisionOrID)

	// retrieve build
	build, err := h.databaseClient.GetPipelineBuildByID(c.Request.Context(), source, owner, repo, revisionOrID, false)
	if err != nil {
		log.Error().Err(err).Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, revisionOrID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Retrieving pipeline build failed"})
		return
	}
	if build == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline build not found"})
		return
	}
	if build.BuildStatus == contracts.StatusCanceling {
		// apparently cancel was already clicked, but somehow the job didn't update the status to canceled
		jobName := h.ciBuilderClient.GetJobName(c.Request.Context(), contracts.JobTypeBuild, build.RepoOwner, build.RepoName, build.ID)
		_ = h.ciBuilderClient.CancelCiBuilderJob(c.Request.Context(), jobName)
		_ = h.databaseClient.UpdateBuildStatus(c.Request.Context(), build.RepoSource, build.RepoOwner, build.RepoName, build.ID, contracts.StatusCanceled)
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Canceled build by user %v", email)})
		return
	}
	if build.BuildStatus != contracts.StatusPending && build.BuildStatus != contracts.StatusRunning {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": fmt.Sprintf("Build with status %v cannot be canceled", build.BuildStatus)})
		return
	}

	// this build can be canceled, set status 'canceling' and cancel the build job
	jobName := h.ciBuilderClient.GetJobName(c.Request.Context(), contracts.JobTypeBuild, build.RepoOwner, build.RepoName, build.ID)
	cancelErr := h.ciBuilderClient.CancelCiBuilderJob(c.Request.Context(), jobName)
	buildStatus := contracts.StatusCanceling
	if build.BuildStatus == contracts.StatusPending {
		// job might not have created a builder yet, so set status to canceled straightaway
		buildStatus = contracts.StatusCanceled
	}
	err = h.databaseClient.UpdateBuildStatus(c.Request.Context(), build.RepoSource, build.RepoOwner, build.RepoName, build.ID, buildStatus)
	if err != nil {
		log.Error().Err(err).Msgf("Failed updating build status for %v/%v/%v/builds/%v in db", source, owner, repo, revisionOrID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Failed setting pipeline build status to canceling"})
		return
	}

	log.Debug().Msgf("Updated build status for canceling pipeline build %v/%v/%v with id %v...", source, owner, repo, revisionOrID)

	// canceling the job failed because it no longer existed we should set canceled status right after having set it to canceling
	if errors.Is(cancelErr, builderapi.ErrJobNotFound) && build.BuildStatus == contracts.StatusRunning {
		buildStatus = contracts.StatusCanceled
		err = h.databaseClient.UpdateBuildStatus(c.Request.Context(), build.RepoSource, build.RepoOwner, build.RepoName, build.ID, buildStatus)
		if err != nil {
			log.Error().Err(err).Msgf("Failed updating build status to canceled after setting it to canceling for %v/%v/%v/builds/%v in db", source, owner, repo, revisionOrID)
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Failed setting pipeline build status to canceled"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Canceled build by user %v", email)})
}

func (h *Handler) CreatePipelineBot(c *gin.Context) {

	if !api.RequestTokenIsValid(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusText(http.StatusUnauthorized), "message": "JWT is invalid"})
		return
	}

	claims := jwt.ExtractClaims(c)
	email := claims["email"].(string)

	var botCommand contracts.Bot
	err := c.BindJSON(&botCommand)
	if err != nil {
		errorMessage := "Binding CreatePipelineBot body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	// match source, owner, repo with values in binded release
	if botCommand.RepoSource != c.Param("source") {
		errorMessage := fmt.Sprintf("RepoSource in path and post data do not match for pipeline %v/%v/%v for bot command issued by %v", botCommand.RepoSource, botCommand.RepoOwner, botCommand.RepoName, email)
		log.Error().Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}
	if botCommand.RepoOwner != c.Param("owner") {
		errorMessage := fmt.Sprintf("RepoOwner in path and post data do not match for pipeline %v/%v/%v for bot command issued by %v", botCommand.RepoSource, botCommand.RepoOwner, botCommand.RepoName, email)
		log.Error().Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}
	if botCommand.RepoName != c.Param("repo") {
		errorMessage := fmt.Sprintf("RepoName in path and post data do not match for pipeline %v/%v/%v for bot command issued by %v", botCommand.RepoSource, botCommand.RepoOwner, botCommand.RepoName, email)
		log.Error().Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	filters := api.GetPipelineFilters(c)

	pipeline, err := h.databaseClient.GetPipeline(c.Request.Context(), botCommand.RepoSource, botCommand.RepoOwner, botCommand.RepoName, filters, false)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving pipeline %v/%v/%v for bot command", botCommand.RepoSource, botCommand.RepoOwner, botCommand.RepoName)
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}
	if pipeline == nil {
		errorMessage := fmt.Sprintf("No pipeline %v/%v/%v for bot command", botCommand.RepoSource, botCommand.RepoOwner, botCommand.RepoName)
		log.Error().Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	// hand off to build service
	// todo check which branch to pass
	createdBot, err := h.buildService.CreateBot(c.Request.Context(), botCommand, *pipeline.ManifestObject, pipeline.RepoBranch)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed creating bot %v/%v/%v name %v for bot command issued by %v", botCommand.RepoSource, botCommand.RepoOwner, botCommand.RepoName, botCommand.Name, email)
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	c.JSON(http.StatusCreated, createdBot)
}

func (h *Handler) CreateNotification(c *gin.Context) {

	if !api.RequestTokenIsValid(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusText(http.StatusUnauthorized), "message": "JWT is invalid"})
		return
	}

	var notification contracts.NotificationRecord
	err := c.BindJSON(&notification)
	if err != nil {
		errorMessage := "Binding CreateNotification body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	createdNotification, err := h.databaseClient.InsertNotification(c.Request.Context(), notification)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed creating notification %v of type %v", createdNotification.LinkID, createdNotification.LinkType)
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	c.JSON(http.StatusCreated, createdNotification)
}

func (h *Handler) GetPipelineBuildLogs(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	revisionOrID := c.Param("revisionOrId")

	var build *contracts.Build
	var err error
	if len(revisionOrID) == 40 {
		build, err = h.databaseClient.GetPipelineBuild(c.Request.Context(), source, owner, repo, revisionOrID, false)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, revisionOrID)
		}
	} else {
		build, err = h.databaseClient.GetPipelineBuildByID(c.Request.Context(), source, owner, repo, revisionOrID, false)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, revisionOrID)
		}
	}

	if build == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline build not found"})
		return
	}

	buildLog, err := h.databaseClient.GetPipelineBuildLogs(c.Request.Context(), source, owner, repo, build.RepoBranch, build.RepoRevision, build.ID)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving build logs for %v/%v/%v/builds/%v/logs from db", source, owner, repo, revisionOrID)
	}
	if buildLog == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline build log not found"})
		return
	}

	if h.config.APIServer.ReadLogFromCloudStorage() {
		err := h.cloudStorageClient.GetPipelineBuildLogs(c.Request.Context(), *buildLog, strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip"), c.Writer)
		if err != nil {

			if errors.Is(err, cloudstorage.ErrLogNotExist) {
				log.Warn().Err(err).
					Msgf("Failed retrieving build logs for %v/%v/%v/builds/%v/logs from cloud storage", source, owner, repo, revisionOrID)
				c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound)})
			}

			log.Error().Err(err).
				Msgf("Failed retrieving build logs for %v/%v/%v/builds/%v/logs from cloud storage", source, owner, repo, revisionOrID)
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
			return
		}
		c.Writer.Flush()
		return
	}

	c.JSON(http.StatusOK, buildLog.Steps)
}

func (h *Handler) GetPipelineBuildLogsByID(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	revisionOrID := c.Param("revisionOrId")
	id := c.Param("id")

	var build *contracts.Build
	var err error
	if len(revisionOrID) == 40 {
		build, err = h.databaseClient.GetPipelineBuild(c.Request.Context(), source, owner, repo, revisionOrID, false)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, revisionOrID)
		}
	} else {
		build, err = h.databaseClient.GetPipelineBuildByID(c.Request.Context(), source, owner, repo, revisionOrID, false)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, id)
		}
	}

	if build == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline build not found"})
		return
	}

	buildLog, err := h.databaseClient.GetPipelineBuildLogsByID(c.Request.Context(), source, owner, repo, build.ID, id)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving build logs for %v/%v/%v/builds/%v/logs from db", source, owner, repo, revisionOrID)
	}
	if buildLog == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline build log not found"})
		return
	}

	if h.config.APIServer.ReadLogFromCloudStorage() {
		err := h.cloudStorageClient.GetPipelineBuildLogs(c.Request.Context(), *buildLog, strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip"), c.Writer)
		if err != nil {

			if errors.Is(err, cloudstorage.ErrLogNotExist) {
				log.Warn().Err(err).
					Msgf("Failed retrieving build logs for %v/%v/%v/builds/%v/logs from cloud storage", source, owner, repo, revisionOrID)
				c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound)})
			}

			log.Error().Err(err).
				Msgf("Failed retrieving build logs for %v/%v/%v/builds/%v/logs from cloud storage", source, owner, repo, revisionOrID)
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
			return
		}
		c.Writer.Flush()
		return
	}

	c.JSON(http.StatusOK, buildLog.Steps)
}

func (h *Handler) GetPipelineBuildLogsPerPage(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	revisionOrID := c.Param("revisionOrId")

	var build *contracts.Build
	var err error
	if len(revisionOrID) == 40 {
		build, err = h.databaseClient.GetPipelineBuild(c.Request.Context(), source, owner, repo, revisionOrID, false)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, revisionOrID)
		}
	} else {
		build, err = h.databaseClient.GetPipelineBuildByID(c.Request.Context(), source, owner, repo, revisionOrID, false)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, revisionOrID)
		}
	}

	if build == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline build not found"})
		return
	}

	pageNumber, pageSize, _, _ := api.GetQueryParameters(c)

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			logs, err := h.databaseClient.GetPipelineBuildLogsPerPage(c.Request.Context(), source, owner, repo, build.RepoBranch, build.RepoRevision, build.ID, pageNumber, pageSize)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(logs))
			for i := range logs {
				items[i] = logs[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetPipelineBuildLogsCount(c.Request.Context(), source, owner, repo, build.RepoBranch, build.RepoRevision, build.ID)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msgf("Failed retrieving build logs for %v/%v/%v with id %v from db", source, owner, repo, build.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) TailPipelineBuildLogs(c *gin.Context) {

	owner := c.Param("owner")
	repo := c.Param("repo")
	id := c.Param("revisionOrId")

	jobName := h.ciBuilderClient.GetJobName(c.Request.Context(), contracts.JobTypeBuild, owner, repo, id)

	logChannel := make(chan contracts.TailLogLine, 50)

	go func() {
		err := h.ciBuilderClient.TailCiBuilderJobLogs(c.Request.Context(), jobName, logChannel)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Error().Err(err).Msgf("Failed tailing build job %v", jobName)
		}
	}()

	ticker := time.NewTicker(5 * time.Second)

	// ensure openresty doesn't buffer this response but sends the chunks rightaway
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	c.Stream(func(w io.Writer) bool {
		select {
		case ll, ok := <-logChannel:
			if !ok {
				c.SSEvent("close", true)
				return false
			}
			c.SSEvent("log", ll)
		case <-ticker.C:
			c.SSEvent("ping", true)
		}
		return true
	})
}

func (h *Handler) PostPipelineBuildLogs(c *gin.Context) {

	// ensure the request has the correct claims
	claims := jwt.ExtractClaims(c)
	job := claims["job"].(string)
	if job == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusText(http.StatusUnauthorized), "message": "JWT is invalid or has invalid claim"})
		return
	}

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	revisionOrID := c.Param("revisionOrId")

	var buildLog contracts.BuildLog
	err := c.Bind(&buildLog)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed binding v2 logs for %v/%v/%v/%v", source, owner, repo, revisionOrID)
		c.String(http.StatusInternalServerError, "Oops, something went wrong")
		return
	}

	if len(revisionOrID) != 40 {

		_, err := strconv.Atoi(revisionOrID)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed reading id from path parameter for %v/%v/%v/builds/%v", source, owner, repo, revisionOrID)
			c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": "Path parameter id is not of type integer"})
			return
		}

		buildLog.BuildID = revisionOrID
	}

	insertedBuildLog, err := h.databaseClient.InsertBuildLog(c.Request.Context(), buildLog)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed inserting logs for %v/%v/%v/%v", source, owner, repo, revisionOrID)
		c.String(http.StatusInternalServerError, "Oops, something went wrong")
		return
	}

	if h.config.APIServer.WriteLogToCloudStorage() {
		err = h.cloudStorageClient.InsertBuildLog(c.Request.Context(), insertedBuildLog)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed inserting logs into cloudstorage for %v/%v/%v/%v", source, owner, repo, revisionOrID)
			c.String(http.StatusInternalServerError, "Oops, something went wrong")
			return
		}
	}

	c.String(http.StatusOK, "Aye aye!")
}

func (h *Handler) GetPipelineBuildWarnings(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	revisionOrID := c.Param("revisionOrId")

	build, err := h.databaseClient.GetPipelineBuildByID(c.Request.Context(), source, owner, repo, revisionOrID, false)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, revisionOrID)
	}
	if build == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline build not found"})
		return
	}

	warnings, err := h.warningHelper.GetManifestWarnings(build.ManifestObject, build.GetFullRepoPath())
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed getting warnings for %v/%v/%v/builds/%v manifest", source, owner, repo, revisionOrID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Failed getting warnings for manifest"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"warnings": warnings})
}

func (h *Handler) GetPipelineReleases(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	pageNumber, pageSize, filters, sortings := api.GetQueryParameters(c)

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			releases, err := h.databaseClient.GetPipelineReleases(c.Request.Context(), source, owner, repo, pageNumber, pageSize, filters, sortings)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(releases))
			for i := range releases {
				items[i] = releases[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetPipelineReleasesCount(c.Request.Context(), source, owner, repo, filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msgf("Failed retrieving releases for %v/%v/%v from db", source, owner, repo)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) CreatePipelineRelease(c *gin.Context) {

	if !api.RequestTokenIsValid(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusText(http.StatusUnauthorized), "message": "JWT is invalid"})
		return
	}

	claims := jwt.ExtractClaims(c)
	email := claims["email"].(string)

	var releaseCommand contracts.Release
	err := c.BindJSON(&releaseCommand)
	if err != nil {
		errorMessage := "Binding CreatePipelineRelease body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	// match source, owner, repo with values in binded release
	if releaseCommand.RepoSource != c.Param("source") {
		errorMessage := fmt.Sprintf("RepoSource in path and post data do not match for pipeline %v/%v/%v for release command", releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName)
		log.Error().Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}
	if releaseCommand.RepoOwner != c.Param("owner") {
		errorMessage := fmt.Sprintf("RepoOwner in path and post data do not match for pipeline %v/%v/%v for release command", releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName)
		log.Error().Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}
	if releaseCommand.RepoName != c.Param("repo") {
		errorMessage := fmt.Sprintf("RepoName in path and post data do not match for pipeline %v/%v/%v for release command", releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName)
		log.Error().Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	filters := api.GetPipelineFilters(c)

	pipeline, err := h.databaseClient.GetPipeline(c.Request.Context(), releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, filters, false)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving pipeline %v/%v/%v for release command", releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName)
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}
	if pipeline == nil {
		errorMessage := fmt.Sprintf("No pipeline %v/%v/%v for release command", releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName)
		log.Error().Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	// check if version exists and is valid to release
	builds, err := h.databaseClient.GetPipelineBuildsByVersion(c.Request.Context(), releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, releaseCommand.ReleaseVersion, []contracts.Status{contracts.StatusSucceeded}, 1, false)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving build %v/%v/%v version %v for release command", releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, releaseCommand.ReleaseVersion)
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	var build *contracts.Build
	// get first build
	if len(builds) > 0 {
		build = builds[0]
	}

	if build == nil {
		errorMessage := fmt.Sprintf("No build %v/%v/%v version %v for release command", releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, releaseCommand.ReleaseVersion)
		log.Error().Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}
	if build.BuildStatus != contracts.StatusSucceeded {
		errorMessage := fmt.Sprintf("Build %v for pipeline %v/%v/%v has status %v for release command; only succeeded pipelines are allowed to be released", releaseCommand.ReleaseVersion, releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, build.BuildStatus)
		log.Error().Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	// check if release target exists
	releaseTargetExists := false
	actionExists := false
	for _, releaseTarget := range build.ReleaseTargets {
		if releaseTarget.Name == releaseCommand.Name {
			if len(releaseTarget.Actions) == 0 && releaseCommand.Action == "" {
				actionExists = true
			} else if len(releaseTarget.Actions) > 0 {
				for _, a := range releaseTarget.Actions {
					if a.Name == releaseCommand.Action {
						actionExists = true
						break
					}
				}
			}

			releaseTargetExists = true
			break
		}
	}
	if !releaseTargetExists {
		errorMessage := fmt.Sprintf("Build %v for pipeline %v/%v/%v has no release %v for release command", releaseCommand.ReleaseVersion, releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, releaseCommand.Name)
		log.Error().Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	// check if action is defined
	if !actionExists {
		errorMessage := fmt.Sprintf("Build %v for pipeline %v/%v/%v has no action %v for release action", releaseCommand.ReleaseVersion, releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, releaseCommand.Action)
		log.Error().Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	// create release object and hand off to build service
	createdRelease, err := h.buildService.CreateRelease(c.Request.Context(), contracts.Release{
		Name:           releaseCommand.Name,
		Action:         releaseCommand.Action,
		RepoSource:     releaseCommand.RepoSource,
		RepoOwner:      releaseCommand.RepoOwner,
		RepoName:       releaseCommand.RepoName,
		ReleaseVersion: releaseCommand.ReleaseVersion,
		Groups:         build.Groups,
		Organizations:  build.Organizations,

		// set trigger event to manual
		Events: []manifest.EstafetteEvent{
			{
				Fired: true,
				Manual: &manifest.EstafetteManualEvent{
					UserID: email,
				},
			},
		},
	}, *build.ManifestObject, build.RepoBranch, build.RepoRevision)

	if err != nil {
		if errors.Is(err, ErrReleaseNotAllowed) {
			c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "error": err})
		} else {
			errorMessage := fmt.Sprintf("Failed creating release %v for pipeline %v/%v/%v version %v for release command issued by %v", releaseCommand.Name, releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, releaseCommand.ReleaseVersion, email)
			log.Error().Err(err).Msg(errorMessage)
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		}
		return
	}

	c.JSON(http.StatusCreated, createdRelease)
}

func (h *Handler) CancelPipelineRelease(c *gin.Context) {

	if !api.RequestTokenIsValid(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusText(http.StatusUnauthorized), "message": "JWT is invalid"})
		return
	}

	claims := jwt.ExtractClaims(c)
	email := claims["email"].(string)

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	idValue := c.Param("id")

	log.Debug().Msgf("Canceling pipeline release %v/%v/%v with id %v...", source, owner, repo, idValue)

	release, err := h.databaseClient.GetPipelineRelease(c.Request.Context(), source, owner, repo, idValue)
	if err != nil {
		log.Error().Err(err).Msgf("Failed retrieving release for %v/%v/%v/%v from db in CancelPipelineRelease", source, owner, repo, idValue)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Retrieving pipeline release failed"})
		return
	}
	if release == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline release not found"})
		return
	}
	if release.ReleaseStatus == contracts.StatusCanceling {
		jobName := h.ciBuilderClient.GetJobName(c.Request.Context(), contracts.JobTypeRelease, release.RepoOwner, release.RepoName, release.ID)
		_ = h.ciBuilderClient.CancelCiBuilderJob(c.Request.Context(), jobName)
		_ = h.databaseClient.UpdateReleaseStatus(c.Request.Context(), release.RepoSource, release.RepoOwner, release.RepoName, release.ID, contracts.StatusCanceled)
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Canceled release by user %v", email)})
		return
	}
	if release.ReleaseStatus != contracts.StatusPending && release.ReleaseStatus != contracts.StatusRunning {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": fmt.Sprintf("Release with status %v cannot be canceled", release.ReleaseStatus)})
		return
	}

	// this release can be canceled, set status 'canceling' and cancel the release job
	jobName := h.ciBuilderClient.GetJobName(c.Request.Context(), contracts.JobTypeRelease, release.RepoOwner, release.RepoName, release.ID)
	cancelErr := h.ciBuilderClient.CancelCiBuilderJob(c.Request.Context(), jobName)
	releaseStatus := contracts.StatusCanceling
	if release.ReleaseStatus == contracts.StatusPending {
		// job might not have created a builder yet, so set status to canceled straightaway
		releaseStatus = contracts.StatusCanceled
	}
	err = h.databaseClient.UpdateReleaseStatus(c.Request.Context(), release.RepoSource, release.RepoOwner, release.RepoName, release.ID, releaseStatus)
	if err != nil {
		log.Error().Err(err).Msgf("Failed updating release status for %v/%v/%v/builds/%v in db in CancelPipelineRelease", source, owner, repo, release.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Failed setting pipeline release status to canceling"})
		return
	}

	log.Debug().Msgf("Updated build status for canceling pipeline release %v/%v/%v with id %v...", source, owner, repo, idValue)

	// canceling the job failed because it no longer existed we should set canceled status right after having set it to canceling
	if errors.Is(cancelErr, builderapi.ErrJobNotFound) && release.ReleaseStatus == contracts.StatusRunning {
		releaseStatus = contracts.StatusCanceled
		err = h.databaseClient.UpdateReleaseStatus(c.Request.Context(), release.RepoSource, release.RepoOwner, release.RepoName, release.ID, releaseStatus)
		if err != nil {
			log.Error().Err(err).Msgf("Failed updating release status to canceled after setting it to canceling for %v/%v/%v/builds/%v in db", source, owner, repo, release.ID)
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Failed setting pipeline release status to canceled"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Canceled release by user %v", email)})
}

func (h *Handler) GetPipelineRelease(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	releaseID := c.Param("releaseId")

	release, err := h.databaseClient.GetPipelineRelease(c.Request.Context(), source, owner, repo, releaseID)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving release for %v/%v/%v/%v from db in GetPipelineRelease", source, owner, repo, releaseID)
	}
	if release == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline release not found"})
		return
	}

	c.JSON(http.StatusOK, release)
}

func (h *Handler) GetPipelineReleaseLogs(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	releaseID := c.Param("releaseId")

	releaseLog, err := h.databaseClient.GetPipelineReleaseLogs(c.Request.Context(), source, owner, repo, releaseID)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving release logs for %v/%v/%v/%v from db", source, owner, repo, releaseID)
	}
	if releaseLog == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline release log not found"})
		return
	}

	if h.config.APIServer.ReadLogFromCloudStorage() {
		err := h.cloudStorageClient.GetPipelineReleaseLogs(c.Request.Context(), *releaseLog, strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip"), c.Writer)
		if err != nil {

			if errors.Is(err, cloudstorage.ErrLogNotExist) {
				log.Warn().Err(err).
					Msgf("Failed retrieving release logs for %v/%v/%v/%v from cloud storage", source, owner, repo, releaseID)
				c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound)})
			}

			log.Error().Err(err).
				Msgf("Failed retrieving release logs for %v/%v/%v/%v from cloud storage", source, owner, repo, releaseID)
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
			return
		}
		c.Writer.Flush()
		return
	}

	c.JSON(http.StatusOK, releaseLog.Steps)
}

func (h *Handler) GetPipelineReleaseLogsByID(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	releaseID := c.Param("releaseId")
	id := c.Param("id")

	releaseLog, err := h.databaseClient.GetPipelineReleaseLogsByID(c.Request.Context(), source, owner, repo, releaseID, id)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving release logs for %v/%v/%v/%v from db", source, owner, repo, releaseID)
	}
	if releaseLog == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline release log not found"})
		return
	}

	if h.config.APIServer.ReadLogFromCloudStorage() {
		err := h.cloudStorageClient.GetPipelineReleaseLogs(c.Request.Context(), *releaseLog, strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip"), c.Writer)
		if err != nil {

			if errors.Is(err, cloudstorage.ErrLogNotExist) {
				log.Warn().Err(err).
					Msgf("Failed retrieving release logs for %v/%v/%v/%v from cloud storage", source, owner, repo, releaseID)
				c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound)})
			}

			log.Error().Err(err).
				Msgf("Failed retrieving release logs for %v/%v/%v/%v from cloud storage", source, owner, repo, releaseID)
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
			return
		}
		c.Writer.Flush()
		return
	}

	c.JSON(http.StatusOK, releaseLog.Steps)
}

func (h *Handler) GetPipelineReleaseLogsPerPage(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	releaseID := c.Param("releaseId")

	pageNumber, pageSize, _, _ := api.GetQueryParameters(c)

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			logs, err := h.databaseClient.GetPipelineReleaseLogsPerPage(c.Request.Context(), source, owner, repo, releaseID, pageNumber, pageSize)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(logs))
			for i := range logs {
				items[i] = logs[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetPipelineReleaseLogsCount(c.Request.Context(), source, owner, repo, releaseID)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msgf("Failed retrieving release logs for %v/%v/%v with id %v from db", source, owner, repo, releaseID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)

}

func (h *Handler) TailPipelineReleaseLogs(c *gin.Context) {

	owner := c.Param("owner")
	repo := c.Param("repo")
	releaseID := c.Param("releaseId")

	jobName := h.ciBuilderClient.GetJobName(c.Request.Context(), contracts.JobTypeRelease, owner, repo, releaseID)

	logChannel := make(chan contracts.TailLogLine, 50)

	go func() {
		err := h.ciBuilderClient.TailCiBuilderJobLogs(c.Request.Context(), jobName, logChannel)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Error().Err(err).Msgf("Failed tailing release job %v", jobName)
		}
	}()

	ticker := time.NewTicker(5 * time.Second)

	// ensure openresty doesn't buffer this response but sends the chunks rightaway
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	c.Stream(func(w io.Writer) bool {
		select {
		case ll, ok := <-logChannel:
			if !ok {
				c.SSEvent("close", true)
				return false
			}
			c.SSEvent("log", ll)
		case <-ticker.C:
			c.SSEvent("ping", true)
		}
		return true
	})
}

func (h *Handler) PostPipelineReleaseLogs(c *gin.Context) {

	// ensure the request has the correct claims
	claims := jwt.ExtractClaims(c)
	job := claims["job"].(string)
	if job == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusText(http.StatusUnauthorized), "message": "JWT is invalid or has invalid claim"})
		return
	}

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	releaseID := c.Param("releaseId")

	var releaseLog contracts.ReleaseLog
	err := c.Bind(&releaseLog)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed binding release logs for %v/%v/%v/%v", source, owner, repo, releaseID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_SERVER_ERROR", "message": "Failed binding release logs from body"})
		return
	}

	insertedReleaseLog, err := h.databaseClient.InsertReleaseLog(c.Request.Context(), releaseLog)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed inserting release logs for %v/%v/%v/%v", source, owner, repo, releaseID)
		c.String(http.StatusInternalServerError, "Oops, something went wrong")
		return
	}

	if h.config.APIServer.WriteLogToCloudStorage() {
		err = h.cloudStorageClient.InsertReleaseLog(c.Request.Context(), insertedReleaseLog)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed inserting release logs into cloud storage for %v/%v/%v/%v", source, owner, repo, releaseID)
			c.String(http.StatusInternalServerError, "Oops, something went wrong")
			return
		}
	}

	c.String(http.StatusOK, "Aye aye!")
}

func (h *Handler) GetPipelineBotNames(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	pageNumber, pageSize, filters, _ := api.GetQueryParameters(c)

	// filter on organizations / groups
	filters = api.SetPermissionsFilters(c, filters)

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			botNames, err := h.databaseClient.GetPipelineBotNames(c.Request.Context(), source, owner, repo, pageNumber, pageSize, filters)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(botNames))
			for i := range botNames {
				items[i] = botNames[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetPipelineBotNamesCount(c.Request.Context(), source, owner, repo, filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving pipeline build branches from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetPipelineBots(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	pageNumber, pageSize, filters, sortings := api.GetQueryParameters(c)

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			bots, err := h.databaseClient.GetPipelineBots(c.Request.Context(), source, owner, repo, pageNumber, pageSize, filters, sortings)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(bots))
			for i := range bots {
				items[i] = bots[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetPipelineBotsCount(c.Request.Context(), source, owner, repo, filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msgf("Failed retrieving bots for %v/%v/%v from db", source, owner, repo)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) CancelPipelineBot(c *gin.Context) {

	if !api.RequestTokenIsValid(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusText(http.StatusUnauthorized), "message": "JWT is invalid"})
		return
	}

	claims := jwt.ExtractClaims(c)
	email := claims["email"].(string)

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	botID := c.Param("id")

	log.Debug().Msgf("Canceling pipeline bot %v/%v/%v with id %v...", source, owner, repo, botID)

	bot, err := h.databaseClient.GetPipelineBot(c.Request.Context(), source, owner, repo, botID)
	if err != nil {
		log.Error().Err(err).Msgf("Failed retrieving bot for %v/%v/%v/%v from db", source, owner, repo, botID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Retrieving pipeline bot failed"})
		return
	}
	if bot == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline bot not found"})
		return
	}
	if bot.BotStatus == contracts.StatusCanceling {
		jobName := h.ciBuilderClient.GetJobName(c.Request.Context(), contracts.JobTypeBot, bot.RepoOwner, bot.RepoName, bot.ID)
		_ = h.ciBuilderClient.CancelCiBuilderJob(c.Request.Context(), jobName)
		_ = h.databaseClient.UpdateBotStatus(c.Request.Context(), bot.RepoSource, bot.RepoOwner, bot.RepoName, botID, contracts.StatusCanceled)
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Canceled bot by user %v", email)})
		return
	}
	if bot.BotStatus != contracts.StatusPending && bot.BotStatus != contracts.StatusRunning {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": fmt.Sprintf("Bot with status %v cannot be canceled", bot.BotStatus)})
		return
	}

	// this bot can be canceled, set status 'canceling' and cancel the bot job
	jobName := h.ciBuilderClient.GetJobName(c.Request.Context(), contracts.JobTypeBot, bot.RepoOwner, bot.RepoName, bot.ID)
	cancelErr := h.ciBuilderClient.CancelCiBuilderJob(c.Request.Context(), jobName)
	botStatus := contracts.StatusCanceling
	if bot.BotStatus == contracts.StatusPending {
		// job might not have created a builder yet, so set status to canceled straightaway
		botStatus = contracts.StatusCanceled
	}
	err = h.databaseClient.UpdateBotStatus(c.Request.Context(), bot.RepoSource, bot.RepoOwner, bot.RepoName, botID, botStatus)
	if err != nil {
		log.Error().Err(err).Msgf("Failed updating bot status for %v/%v/%v/bots/%v in db", source, owner, repo, botID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Failed setting pipeline bot status to canceling"})
		return
	}

	log.Debug().Msgf("Updated build status for canceling pipeline bot %v/%v/%v with id %v...", source, owner, repo, botID)

	// canceling the job failed because it no longer existed we should set canceled status right after having set it to canceling
	if errors.Is(cancelErr, builderapi.ErrJobNotFound) && bot.BotStatus == contracts.StatusRunning {
		botStatus = contracts.StatusCanceled
		err = h.databaseClient.UpdateBotStatus(c.Request.Context(), bot.RepoSource, bot.RepoOwner, bot.RepoName, botID, botStatus)
		if err != nil {
			log.Error().Err(err).Msgf("Failed updating bot status to canceled after setting it to canceling for %v/%v/%v/bots/%v in db", source, owner, repo, botID)
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Failed setting pipeline bot status to canceled"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Canceled bot by user %v", email)})
}

func (h *Handler) GetPipelineBot(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	botID := c.Param("botId")

	bot, err := h.databaseClient.GetPipelineBot(c.Request.Context(), source, owner, repo, botID)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving bot for %v/%v/%v/%v from db", source, owner, repo, botID)
	}
	if bot == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline bot not found"})
		return
	}

	c.JSON(http.StatusOK, bot)
}

func (h *Handler) GetPipelineBotLogs(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	botID := c.Param("botId")

	botLog, err := h.databaseClient.GetPipelineBotLogs(c.Request.Context(), source, owner, repo, botID)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving bot logs for %v/%v/%v/%v from db", source, owner, repo, botID)
	}
	if botLog == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline bot log not found"})
		return
	}

	if h.config.APIServer.ReadLogFromCloudStorage() {
		err := h.cloudStorageClient.GetPipelineBotLogs(c.Request.Context(), *botLog, strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip"), c.Writer)
		if err != nil {

			if errors.Is(err, cloudstorage.ErrLogNotExist) {
				log.Warn().Err(err).
					Msgf("Failed retrieving bot logs for %v/%v/%v/%v from cloud storage", source, owner, repo, botID)
				c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound)})
			}

			log.Error().Err(err).
				Msgf("Failed retrieving bot logs for %v/%v/%v/%v from cloud storage", source, owner, repo, botID)
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
			return
		}
		c.Writer.Flush()
		return
	}

	c.JSON(http.StatusOK, botLog.Steps)
}

func (h *Handler) GetPipelineBotLogsByID(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	botID := c.Param("botId")
	id := c.Param("id")

	botLog, err := h.databaseClient.GetPipelineBotLogsByID(c.Request.Context(), source, owner, repo, botID, id)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving bot logs for %v/%v/%v/%v from db", source, owner, repo, botID)
	}
	if botLog == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline bot log not found"})
		return
	}

	if h.config.APIServer.ReadLogFromCloudStorage() {
		err := h.cloudStorageClient.GetPipelineBotLogs(c.Request.Context(), *botLog, strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip"), c.Writer)
		if err != nil {

			if errors.Is(err, cloudstorage.ErrLogNotExist) {
				log.Warn().Err(err).
					Msgf("Failed retrieving bot logs for %v/%v/%v/%v from cloud storage", source, owner, repo, botID)
				c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound)})
			}

			log.Error().Err(err).
				Msgf("Failed retrieving bot logs for %v/%v/%v/%v from cloud storage", source, owner, repo, botID)
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
			return
		}
		c.Writer.Flush()
		return
	}

	c.JSON(http.StatusOK, botLog.Steps)
}

func (h *Handler) GetPipelineBotLogsPerPage(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	botID := c.Param("botId")

	pageNumber, pageSize, _, _ := api.GetQueryParameters(c)

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			logs, err := h.databaseClient.GetPipelineBotLogsPerPage(c.Request.Context(), source, owner, repo, botID, pageNumber, pageSize)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(logs))
			for i := range logs {
				items[i] = logs[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetPipelineBotLogsCount(c.Request.Context(), source, owner, repo, botID)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msgf("Failed retrieving bot logs for %v/%v/%v with id %v from db", source, owner, repo, botID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)

}

func (h *Handler) TailPipelineBotLogs(c *gin.Context) {

	owner := c.Param("owner")
	repo := c.Param("repo")
	botID := c.Param("botId")

	jobName := h.ciBuilderClient.GetJobName(c.Request.Context(), contracts.JobTypeBot, owner, repo, botID)

	logChannel := make(chan contracts.TailLogLine, 50)

	go func() {
		err := h.ciBuilderClient.TailCiBuilderJobLogs(c.Request.Context(), jobName, logChannel)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Error().Err(err).Msgf("Failed tailing bot job %v", jobName)
		}
	}()

	ticker := time.NewTicker(5 * time.Second)

	// ensure openresty doesn't buffer this response but sends the chunks rightaway
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	c.Stream(func(w io.Writer) bool {
		select {
		case ll, ok := <-logChannel:
			if !ok {
				c.SSEvent("close", true)
				return false
			}
			c.SSEvent("log", ll)
		case <-ticker.C:
			c.SSEvent("ping", true)
		}
		return true
	})
}

func (h *Handler) PostPipelineBotLogs(c *gin.Context) {

	// ensure the request has the correct claims
	claims := jwt.ExtractClaims(c)
	job := claims["job"].(string)
	if job == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusText(http.StatusUnauthorized), "message": "JWT is invalid or has invalid claim"})
		return
	}

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	botIDValue := c.Param("botId")

	botID, err := strconv.Atoi(botIDValue)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed reading id from path parameter for %v/%v/%v/%v", source, owner, repo, botIDValue)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": "Path parameter id is not of type integer"})
		return
	}

	var botLog contracts.BotLog
	err = c.Bind(&botLog)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed binding bot logs for %v/%v/%v/%v", source, owner, repo, botID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_SERVER_ERROR", "message": "Failed binding bot logs from body"})
		return
	}

	insertedBotLog, err := h.databaseClient.InsertBotLog(c.Request.Context(), botLog)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed inserting bot logs for %v/%v/%v/%v", source, owner, repo, botID)
		c.String(http.StatusInternalServerError, "Oops, something went wrong")
		return
	}

	if h.config.APIServer.WriteLogToCloudStorage() {
		err = h.cloudStorageClient.InsertBotLog(c.Request.Context(), insertedBotLog)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed inserting bot logs into cloud storage for %v/%v/%v/%v", source, owner, repo, botID)
			c.String(http.StatusInternalServerError, "Oops, something went wrong")
			return
		}
	}

	c.String(http.StatusOK, "Aye aye!")
}

func (h *Handler) GetAllPipelineBuilds(c *gin.Context) {

	pageNumber, pageSize, filters, sortings := api.GetQueryParameters(c)

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			builds, err := h.databaseClient.GetAllPipelineBuilds(c.Request.Context(), pageNumber, pageSize, filters, sortings, true)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(builds))
			for i := range builds {
				items[i] = builds[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetAllPipelineBuildsCount(c.Request.Context(), filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving all builds from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetAllPipelineReleases(c *gin.Context) {

	pageNumber, pageSize, filters, sortings := api.GetQueryParameters(c)

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			releases, err := h.databaseClient.GetAllPipelineReleases(c.Request.Context(), pageNumber, pageSize, filters, sortings)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(releases))
			for i := range releases {
				items[i] = releases[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetAllPipelineReleasesCount(c.Request.Context(), filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving all releases from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetAllPipelineBots(c *gin.Context) {

	pageNumber, pageSize, filters, sortings := api.GetQueryParameters(c)

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			bots, err := h.databaseClient.GetAllPipelineBots(c.Request.Context(), pageNumber, pageSize, filters, sortings)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(bots))
			for i := range bots {
				items[i] = bots[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetAllPipelineBotsCount(c.Request.Context(), filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving all bots from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetAllNotifications(c *gin.Context) {

	pageNumber, pageSize, filters, sortings := api.GetQueryParameters(c)

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			notifications, err := h.databaseClient.GetAllNotifications(c.Request.Context(), pageNumber, pageSize, filters, sortings)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(notifications))
			for i := range notifications {
				items[i] = notifications[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetAllNotificationsCount(c.Request.Context(), filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving all notifications from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetReleaseTargets(c *gin.Context) {

	pageNumber, pageSize, filters, _ := api.GetQueryParameters(c)

	// filter on organizations / groups
	filters = api.SetPermissionsFilters(c, filters)

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			releaseTargets, err := h.databaseClient.GetReleaseTargets(c.Request.Context(), pageNumber, pageSize, filters)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(releaseTargets))
			for i := range releaseTargets {
				items[i] = releaseTargets[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetReleaseTargetsCount(c.Request.Context(), filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving pipeline release targets from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetAllPipelinesReleaseTargets(c *gin.Context) {

	pageNumber, pageSize, filters, _ := api.GetQueryParameters(c)

	// filter on organizations / groups
	filters = api.SetPermissionsFilters(c, filters)

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			releaseTargets, err := h.databaseClient.GetAllPipelinesReleaseTargets(c.Request.Context(), pageNumber, pageSize, filters)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(releaseTargets))
			for i := range releaseTargets {
				items[i] = releaseTargets[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetAllPipelinesReleaseTargetsCount(c.Request.Context(), filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving pipeline release targets from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetAllReleasesReleaseTargets(c *gin.Context) {

	pageNumber, pageSize, filters, _ := api.GetQueryParameters(c)

	// filter on organizations / groups
	filters = api.SetPermissionsFilters(c, filters)

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			releaseTargets, err := h.databaseClient.GetAllReleasesReleaseTargets(c.Request.Context(), pageNumber, pageSize, filters)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(releaseTargets))
			for i := range releaseTargets {
				items[i] = releaseTargets[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetAllReleasesReleaseTargetsCount(c.Request.Context(), filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving release release targets from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetFrequentLabels(c *gin.Context) {

	pageNumber, pageSize, filters, _ := api.GetQueryParameters(c)

	// filter on organizations / groups
	filters = api.SetPermissionsFilters(c, filters)

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			labels, err := h.databaseClient.GetFrequentLabels(c.Request.Context(), pageNumber, pageSize, filters)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(labels))
			for i := range labels {
				items[i] = labels[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetFrequentLabelsCount(c.Request.Context(), filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving frequent labels from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetPipelineStatsBuildsDurations(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	// get filters (?filter[last]=100)
	filters := map[api.FilterType][]string{}
	filters[api.FilterStatus] = api.GetStatusFilter(c, contracts.StatusSucceeded)
	filters[api.FilterLast] = api.GetLastFilter(c, 100)

	durations, err := h.databaseClient.GetPipelineBuildsDurations(c.Request.Context(), source, owner, repo, filters)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving build durations from db for %v/%v/%v", source, owner, repo)
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"durations": durations,
	})
}

func (h *Handler) GetPipelineStatsReleasesDurations(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	// get filters (?filter[last]=100)
	filters := map[api.FilterType][]string{}
	filters[api.FilterStatus] = api.GetStatusFilter(c, contracts.StatusSucceeded)
	filters[api.FilterLast] = api.GetLastFilter(c, 100)

	durations, err := h.databaseClient.GetPipelineReleasesDurations(c.Request.Context(), source, owner, repo, filters)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving releases durations from db for %v/%v/%v", source, owner, repo)
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"durations": durations,
	})
}

func (h *Handler) GetPipelineStatsBotsDurations(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	// get filters (?filter[last]=100)
	filters := map[api.FilterType][]string{}
	filters[api.FilterStatus] = api.GetStatusFilter(c, contracts.StatusSucceeded)
	filters[api.FilterLast] = api.GetLastFilter(c, 100)

	durations, err := h.databaseClient.GetPipelineBotsDurations(c.Request.Context(), source, owner, repo, filters)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving bots durations from db for %v/%v/%v", source, owner, repo)
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"durations": durations,
	})
}

func (h *Handler) GetPipelineStatsBuildsCPUUsageMeasurements(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	// get filters (?filter[last]=100)
	filters := map[api.FilterType][]string{}
	filters[api.FilterStatus] = api.GetStatusFilter(c, contracts.StatusSucceeded)
	filters[api.FilterLast] = api.GetLastFilter(c, 100)

	measurements, err := h.databaseClient.GetPipelineBuildsCPUUsageMeasurements(c.Request.Context(), source, owner, repo, filters)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving build cpu usage measurements from db for %v/%v/%v", source, owner, repo)
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"measurements": measurements,
	})
}

func (h *Handler) GetPipelineStatsReleasesCPUUsageMeasurements(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	// get filters (?filter[last]=100)
	filters := map[api.FilterType][]string{}
	filters[api.FilterStatus] = api.GetStatusFilter(c, contracts.StatusSucceeded)
	filters[api.FilterLast] = api.GetLastFilter(c, 100)

	measurements, err := h.databaseClient.GetPipelineReleasesCPUUsageMeasurements(c.Request.Context(), source, owner, repo, filters)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving release cpu usage measurements from db for %v/%v/%v", source, owner, repo)
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"measurements": measurements,
	})
}

func (h *Handler) GetPipelineStatsBotsCPUUsageMeasurements(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	// get filters (?filter[last]=100)
	filters := map[api.FilterType][]string{}
	filters[api.FilterStatus] = api.GetStatusFilter(c, contracts.StatusSucceeded)
	filters[api.FilterLast] = api.GetLastFilter(c, 100)

	measurements, err := h.databaseClient.GetPipelineBotsCPUUsageMeasurements(c.Request.Context(), source, owner, repo, filters)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving bots cpu usage measurements from db for %v/%v/%v", source, owner, repo)
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"measurements": measurements,
	})
}

func (h *Handler) GetPipelineStatsBuildsMemoryUsageMeasurements(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	// get filters (?filter[last]=100)
	filters := map[api.FilterType][]string{}
	filters[api.FilterStatus] = api.GetStatusFilter(c, contracts.StatusSucceeded)
	filters[api.FilterLast] = api.GetLastFilter(c, 100)

	measurements, err := h.databaseClient.GetPipelineBuildsMemoryUsageMeasurements(c.Request.Context(), source, owner, repo, filters)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving build memory usage measurements from db for %v/%v/%v", source, owner, repo)
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"measurements": measurements,
	})
}

func (h *Handler) GetPipelineStatsReleasesMemoryUsageMeasurements(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	// get filters (?filter[last]=100)
	filters := map[api.FilterType][]string{}
	filters[api.FilterStatus] = api.GetStatusFilter(c, contracts.StatusSucceeded)
	filters[api.FilterLast] = api.GetLastFilter(c, 100)

	measurements, err := h.databaseClient.GetPipelineReleasesMemoryUsageMeasurements(c.Request.Context(), source, owner, repo, filters)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving release memory usage measurements from db for %v/%v/%v", source, owner, repo)
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"measurements": measurements,
	})
}

func (h *Handler) GetPipelineStatsBotsMemoryUsageMeasurements(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	// get filters (?filter[last]=100)
	filters := map[api.FilterType][]string{}
	filters[api.FilterStatus] = api.GetStatusFilter(c, contracts.StatusSucceeded)
	filters[api.FilterLast] = api.GetLastFilter(c, 100)

	measurements, err := h.databaseClient.GetPipelineBotsMemoryUsageMeasurements(c.Request.Context(), source, owner, repo, filters)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving bots memory usage measurements from db for %v/%v/%v", source, owner, repo)
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"measurements": measurements,
	})
}

func (h *Handler) GetPipelineWarnings(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	filters := api.GetPipelineFilters(c)

	pipeline, err := h.databaseClient.GetPipeline(c.Request.Context(), source, owner, repo, filters, false)
	if err != nil {
		log.Error().Err(err).Msgf("Failed retrieving pipeline for %v/%v/%v from db", source, owner, repo)
	}
	if pipeline == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline not found"})
		return
	}

	warnings := make([]contracts.Warning, 0)

	// get filters (?filter[last]=100)
	buildsFilters := map[api.FilterType][]string{}

	// only use durations of successful builds
	buildsFilters[api.FilterStatus] = api.GetStatusFilter(c, contracts.StatusSucceeded)

	// get last 25 builds
	buildsFilters[api.FilterLast] = api.GetLastFilter(c, 25)

	durations, err := h.databaseClient.GetPipelineBuildsDurations(c.Request.Context(), source, owner, repo, buildsFilters)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving build durations from db for pipeline %v/%v/%v warnings", source, owner, repo)
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	if len(durations) > 0 {
		// pick the item at half of the length
		medianIndex := len(durations)/2 - 1
		if medianIndex < 0 {
			medianIndex = 0
		}
		duration := durations[medianIndex]["duration"].(time.Duration)
		durationInSeconds := duration.Seconds()

		if durationInSeconds > 300.0 {
			warnings = append(warnings, contracts.Warning{
				Status:  "danger",
				Message: fmt.Sprintf("The [median build time](/pipelines/%v/%v/%v/statistics?last=25) of this pipeline is **%v**. This is too slow, please optimize your build speed by using smaller images or running less intensive steps to ensure it finishes at least within 5 minutes, but preferably within 2 minutes.", source, owner, repo, duration),
			})
		} else if durationInSeconds > 120.0 {
			warnings = append(warnings, contracts.Warning{
				Status:  "warning",
				Message: fmt.Sprintf("The [median build time](/pipelines/%v/%v/%v/statistics?last=25) of this pipeline is **%v**. This is a bit too slow, please optimize your build speed by using smaller images or running less intensive steps to ensure it finishes within 2 minutes.", source, owner, repo, duration),
			})
		}
	}

	manifestWarnings, err := h.warningHelper.GetManifestWarnings(pipeline.ManifestObject, pipeline.GetFullRepoPath())
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed getting warnings for %v/%v/%v manifest", source, owner, repo)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Failed getting warnings for manifest"})
		return
	}
	warnings = append(warnings, manifestWarnings...)

	c.JSON(http.StatusOK, gin.H{"warnings": warnings})
}

func (h *Handler) GetCatalogFilters(c *gin.Context) {

	if h.config == nil || h.config.Catalog == nil {
		c.JSON(http.StatusOK, []string{})
		return
	}

	c.JSON(http.StatusOK, h.config.Catalog.Filters)
}

func (h *Handler) GetCatalogFilterValues(c *gin.Context) {

	labelKey := c.DefaultQuery("filter[labels]", "type")

	labels, err := h.databaseClient.GetLabelValues(c.Request.Context(), labelKey)
	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving label values from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, labels)
}

func (h *Handler) GetStatsPipelinesCount(c *gin.Context) {

	// get filters (?filter[status]=running,succeeded&filter[since]=1w
	filters := map[api.FilterType][]string{}
	filters[api.FilterStatus] = api.GetStatusFilter(c)
	filters[api.FilterSince] = api.GetSinceFilter(c)

	pipelinesCount, err := h.databaseClient.GetPipelinesCount(c.Request.Context(), filters)
	if err != nil {
		log.Error().Err(err).
			Msg("Failed retrieving pipelines count from db")
	}

	c.JSON(http.StatusOK, gin.H{
		"count": pipelinesCount,
	})
}

func (h *Handler) GetStatsReleasesCount(c *gin.Context) {

	// get filters (?filter[status]=running,succeeded&filter[since]=1w
	filters := map[api.FilterType][]string{}
	filters[api.FilterStatus] = api.GetStatusFilter(c)
	filters[api.FilterSince] = api.GetSinceFilter(c)

	releasesCount, err := h.databaseClient.GetReleasesCount(c.Request.Context(), filters)
	if err != nil {
		log.Error().Err(err).
			Msg("Failed retrieving releases count from db")
	}

	c.JSON(http.StatusOK, gin.H{
		"count": releasesCount,
	})
}

func (h *Handler) GetStatsBotsCount(c *gin.Context) {

	// get filters (?filter[status]=running,succeeded&filter[since]=1w
	filters := map[api.FilterType][]string{}
	filters[api.FilterStatus] = api.GetStatusFilter(c)
	filters[api.FilterSince] = api.GetSinceFilter(c)

	botsCount, err := h.databaseClient.GetBotsCount(c.Request.Context(), filters)
	if err != nil {
		log.Error().Err(err).
			Msg("Failed retrieving bots count from db")
	}

	c.JSON(http.StatusOK, gin.H{
		"count": botsCount,
	})
}

func (h *Handler) GetStatsBuildsCount(c *gin.Context) {

	// get filters (?filter[status]=running,succeeded&filter[since]=1w
	filters := map[api.FilterType][]string{}
	filters[api.FilterStatus] = api.GetStatusFilter(c)
	filters[api.FilterSince] = api.GetSinceFilter(c)

	buildsCount, err := h.databaseClient.GetBuildsCount(c.Request.Context(), filters)
	if err != nil {
		log.Error().Err(err).
			Msg("Failed retrieving builds count from db")
	}

	c.JSON(http.StatusOK, gin.H{
		"count": buildsCount,
	})
}

func (h *Handler) GetStatsMostBuilds(c *gin.Context) {

	pageNumber, pageSize, filters, _ := api.GetQueryParameters(c)

	pipelines, err := h.databaseClient.GetPipelinesWithMostBuilds(c.Request.Context(), pageNumber, pageSize, filters)
	if err != nil {
		errorMessage := "Failed retrieving pipelines with most builds from db"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}
	pipelinesCount, err := h.databaseClient.GetPipelinesWithMostBuildsCount(c.Request.Context(), filters)
	if err != nil {
		errorMessage := "Failed retrieving pipelines count from db"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	response := contracts.ListResponse{
		Pagination: contracts.Pagination{
			Page:       pageNumber,
			Size:       pageSize,
			TotalItems: pipelinesCount,
			TotalPages: int(math.Ceil(float64(pipelinesCount) / float64(pageSize))),
		},
	}

	response.Items = make([]interface{}, len(pipelines))
	for i := range pipelines {
		response.Items[i] = pipelines[i]
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetStatsMostReleases(c *gin.Context) {

	pageNumber, pageSize, filters, _ := api.GetQueryParameters(c)

	pipelines, err := h.databaseClient.GetPipelinesWithMostReleases(c.Request.Context(), pageNumber, pageSize, filters)
	if err != nil {
		errorMessage := "Failed retrieving pipelines with most builds from db"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}
	pipelinesCount, err := h.databaseClient.GetPipelinesWithMostReleasesCount(c.Request.Context(), filters)
	if err != nil {
		errorMessage := "Failed retrieving pipelines count from db"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	response := contracts.ListResponse{
		Pagination: contracts.Pagination{
			Page:       pageNumber,
			Size:       pageSize,
			TotalItems: pipelinesCount,
			TotalPages: int(math.Ceil(float64(pipelinesCount) / float64(pageSize))),
		},
	}

	response.Items = make([]interface{}, len(pipelines))
	for i := range pipelines {
		response.Items[i] = pipelines[i]
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetStatsMostBots(c *gin.Context) {

	pageNumber, pageSize, filters, _ := api.GetQueryParameters(c)

	pipelines, err := h.databaseClient.GetPipelinesWithMostBots(c.Request.Context(), pageNumber, pageSize, filters)
	if err != nil {
		errorMessage := "Failed retrieving pipelines with most bots from db"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}
	pipelinesCount, err := h.databaseClient.GetPipelinesWithMostBotsCount(c.Request.Context(), filters)
	if err != nil {
		errorMessage := "Failed retrieving pipelines count from db"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	response := contracts.ListResponse{
		Pagination: contracts.Pagination{
			Page:       pageNumber,
			Size:       pageSize,
			TotalItems: pipelinesCount,
			TotalPages: int(math.Ceil(float64(pipelinesCount) / float64(pageSize))),
		},
	}

	response.Items = make([]interface{}, len(pipelines))
	for i := range pipelines {
		response.Items[i] = pipelines[i]
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetStatsBuildsDuration(c *gin.Context) {

	// get filters (?filter[status]=running,succeeded&filter[since]=1w
	filters := map[api.FilterType][]string{}
	filters[api.FilterStatus] = api.GetStatusFilter(c)
	filters[api.FilterSince] = api.GetSinceFilter(c)

	buildsDuration, err := h.databaseClient.GetBuildsDuration(c.Request.Context(), filters)
	if err != nil {
		log.Error().Err(err).
			Msg("Failed retrieving builds duration from db")
	}

	c.JSON(http.StatusOK, gin.H{
		"duration": buildsDuration,
	})
}

func (h *Handler) GetStatsBuildsAdoption(c *gin.Context) {

	buildTimes, err := h.databaseClient.GetFirstBuildTimes(c.Request.Context())
	if err != nil {
		errorMessage := "Failed retrieving first build times from db"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"datetimes": buildTimes,
	})
}

func (h *Handler) GetStatsReleasesAdoption(c *gin.Context) {

	releaseTimes, err := h.databaseClient.GetFirstReleaseTimes(c.Request.Context())
	if err != nil {
		errorMessage := "Failed retrieving first release times from db"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"datetimes": releaseTimes,
	})
}

func (h *Handler) GetStatsBotsAdoption(c *gin.Context) {

	releaseTimes, err := h.databaseClient.GetFirstBotTimes(c.Request.Context())
	if err != nil {
		errorMessage := "Failed retrieving first bot times from db"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"datetimes": releaseTimes,
	})
}

func (h *Handler) GetConfig(c *gin.Context) {

	configBytes, err := yaml.Marshal(h.encryptedConfig)
	if err != nil {
		log.Error().Err(err).Msgf("Failed marshalling encrypted config")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	// obfuscate all secrets
	configString, err := h.obfuscateSecrets(string(configBytes))
	if err != nil {
		log.Error().Err(err).Msgf("Failed obfuscating secrets")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	// add extra whitespace after each top-level item
	addWhitespaceRegex := regexp.MustCompile(`\n([a-z])`)
	configString = addWhitespaceRegex.ReplaceAllString(configString, "\n\n$1")

	c.JSON(http.StatusOK, gin.H{"config": configString})
}

func (h *Handler) GetConfigCredentials(c *gin.Context) {

	configBytes, err := yaml.Marshal(h.encryptedConfig.Credentials)
	if err != nil {
		log.Error().Err(err).Msgf("Failed marshalling encrypted config")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	// obfuscate all secrets
	configString, err := h.obfuscateSecrets(string(configBytes))
	if err != nil {
		log.Error().Err(err).Msgf("Failed obfuscating secrets")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	// add extra whitespace after each top-level item
	addWhitespaceRegex := regexp.MustCompile(`\n([a-z])`)
	configString = addWhitespaceRegex.ReplaceAllString(configString, "\n\n$1")

	c.JSON(http.StatusOK, gin.H{"config": configString})
}

func (h *Handler) GetConfigTrustedImages(c *gin.Context) {

	configBytes, err := yaml.Marshal(h.encryptedConfig.TrustedImages)
	if err != nil {
		log.Error().Err(err).Msgf("Failed marshalling encrypted config")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	// obfuscate all secrets
	configString, err := h.obfuscateSecrets(string(configBytes))
	if err != nil {
		log.Error().Err(err).Msgf("Failed obfuscating secrets")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	// add extra whitespace after each top-level item
	addWhitespaceRegex := regexp.MustCompile(`\n([a-z])`)
	configString = addWhitespaceRegex.ReplaceAllString(configString, "\n\n$1")

	c.JSON(http.StatusOK, gin.H{"config": configString})
}

func (h *Handler) GetConfigBuildControl(c *gin.Context) {

	configBytes, err := yaml.Marshal(h.config.BuildControl)
	if err != nil {
		log.Error().Err(err).Msgf("Failed marshalling buildControl")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"config": string(configBytes)})
}

func (h *Handler) GetManifestTemplates(c *gin.Context) {

	templateFiles, err := os.ReadDir(h.templatesPath)
	if err != nil {
		log.Error().Err(err).Msgf("Failed listing template files directory")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	templates := []interface{}{}
	for _, f := range templateFiles {

		templateFileName := f.Name()

		// check if it's a manifest template
		re := regexp.MustCompile(`^manifest-(.+)\.tmpl`)
		match := re.FindStringSubmatch(templateFileName)

		if len(match) == 2 {

			// read template file
			templateFilePath := filepath.Join(h.templatesPath, templateFileName)
			data, err := os.ReadFile(templateFilePath)
			if err != nil {
				log.Error().Err(err).Msgf("Failed reading template file %v", templateFilePath)
				c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
				return
			}

			placeholderRegex := regexp.MustCompile(`{{\.([a-zA-Z0-9]+)}}`)
			placeholderMatches := placeholderRegex.FindAllStringSubmatch(string(data), -1)

			// reduce and deduplicate [["{{.Application}}","Application"],["{{.Team}}","Team"],["{{.ProjectName}}","ProjectName"],["{{.ProjectName}}","ProjectName"]] to ["Application","Team","ProjectName"]
			placeholders := []string{}
			for _, m := range placeholderMatches {
				if len(m) == 2 && !api.StringArrayContains(placeholders, m[1]) {
					placeholders = append(placeholders, m[1])
				}
			}

			templateData := map[string]interface{}{
				"template":     match[1],
				"placeholders": placeholders,
			}

			templates = append(templates, templateData)
		}
	}

	c.JSON(http.StatusOK, gin.H{"templates": templates})
}

func (h *Handler) GenerateManifest(c *gin.Context) {

	var aux struct {
		Template     string            `json:"template"`
		Placeholders map[string]string `json:"placeholders,omitempty"`
	}

	err := c.BindJSON(&aux)
	if err != nil {
		errorMessage := "Binding GenerateManifest body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	templateFilePath := filepath.Join(h.templatesPath, fmt.Sprintf("manifest-%v.tmpl", aux.Template))
	data, err := os.ReadFile(templateFilePath)
	if err != nil {
		log.Error().Err(err).Msgf("Failed reading template file %v", templateFilePath)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	tmpl, err := template.New(".estafette.yaml").Parse(string(data))
	if err != nil {
		log.Error().Err(err).Msgf("Failed parsing template file %v", templateFilePath)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	var renderedTemplate bytes.Buffer
	err = tmpl.Execute(&renderedTemplate, aux.Placeholders)
	if err != nil {
		log.Error().Err(err).Msgf("Failed rendering template file %v", templateFilePath)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"manifest": renderedTemplate.String()})
}

func (h *Handler) ValidateManifest(c *gin.Context) {

	var aux struct {
		Manifest string `json:"manifest"`
	}

	err := c.BindJSON(&aux)
	if err != nil {
		errorMessage := "Binding ValidateManifest body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	_, err = manifest.ReadManifest(h.config.ManifestPreferences, aux.Manifest, true)
	status := contracts.StatusSucceeded
	errorString := ""
	if err != nil {
		status = contracts.StatusFailed
		errorString = err.Error()
	}

	c.JSON(http.StatusOK, gin.H{"status": status, "errors": errorString})
}

func (h *Handler) EncryptSecret(c *gin.Context) {

	var aux struct {
		Base64Encode      bool   `json:"base64"`
		DoubleEncrypt     bool   `json:"double"`
		PipelineAllowList string `json:"pipelineAllowList"`
		Value             string `json:"value"`
	}

	err := c.BindJSON(&aux)
	if err != nil {
		errorMessage := "Binding EncryptSecret body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	value := aux.Value

	// trim any whitespace and newlines at beginning and end of string
	value = strings.TrimSpace(value)

	if aux.Base64Encode {
		// base64 encode for use in a kubernetes secret
		value = base64.StdEncoding.EncodeToString([]byte(value))
	}

	encryptedString, err := h.secretHelper.EncryptEnvelope(value, aux.PipelineAllowList)
	if err != nil {
		log.Error().Err(err).Msg("Failed encrypting secret")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	if aux.DoubleEncrypt {
		encryptedString, err = h.secretHelper.EncryptEnvelope(encryptedString, crypt.DefaultPipelineAllowList)
		if err != nil {
			log.Error().Err(err).Msg("Failed encrypting secret")
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"secret": encryptedString})
}

func (h *Handler) obfuscateSecrets(input string) (string, error) {

	r, err := regexp.Compile(`estafette\.secret\(([a-zA-Z0-9.=_-]+)\)`)
	if err != nil {
		return "", err
	}

	// obfuscate all secrets
	return r.ReplaceAllLiteralString(input, "***"), nil
}

func (h *Handler) Commands(c *gin.Context) {

	// ensure the request has the correct claims
	claims := jwt.ExtractClaims(c)
	job := claims["job"].(string)
	if job == "" {
		log.Error().Msg("JWT is invalid or has invalid claim")
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusText(http.StatusUnauthorized), "message": "JWT is invalid or has invalid claim"})
		return
	}

	log.Debug().Msgf("X-Estafette-Event-Job-Name is set to %v", c.GetHeader("X-Estafette-Event-Job-Name"))

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body from Estafette 'build finished' event failed")
		c.String(http.StatusInternalServerError, "Reading body from Estafette 'build finished' event failed")
		return
	}

	// unmarshal json body
	var ciBuilderEvent contracts.EstafetteCiBuilderEvent
	err = json.Unmarshal(body, &ciBuilderEvent)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to CiBuilderEvent failed")
		return
	}

	err = ciBuilderEvent.Validate()
	if err != nil {
		log.Error().Err(err).Interface("ciBuilderEvent", ciBuilderEvent).Msg("CiBuilderEvent is not valid")
		return
	}

	switch ciBuilderEvent.BuildEventType {
	case contracts.BuildEventTypeUpdateStatus:

		log.Debug().Msgf("Updating status for job %v and pod %v", ciBuilderEvent.JobName, ciBuilderEvent.PodName)

		err := h.buildService.UpdateBuildStatus(c.Request.Context(), ciBuilderEvent)
		if err != nil {
			errorMessage := fmt.Sprintf("Failed updating build status for job %v and pod %v to %v, not removing the job", ciBuilderEvent.JobName, ciBuilderEvent.PodName, ciBuilderEvent.Build.BuildStatus)
			log.Error().Err(err).Interface("ciBuilderEvent", ciBuilderEvent).Msg(errorMessage)
			c.String(http.StatusInternalServerError, errorMessage)
			return
		}

	case contracts.BuildEventTypeClean:

		log.Debug().Msgf("Cleaning resources for job %v and pod %v", ciBuilderEvent.JobName, ciBuilderEvent.PodName)

		if ciBuilderEvent.GetStatus() != contracts.StatusCanceled {
			go func(eventJobname string) {
				// create new context to avoid cancellation impacting execution
				span, _ := opentracing.StartSpanFromContext(c.Request.Context(), "estafette:AsyncRemoveCiBuilderJob")
				ctx := opentracing.ContextWithSpan(context.Background(), span)
				defer span.Finish()

				err = h.ciBuilderClient.RemoveCiBuilderJob(ctx, eventJobname)
				if err != nil {
					log.Error().Err(err).Interface("ciBuilderEvent", ciBuilderEvent).Msgf("Failed removing job %v and pod %v for event %v", ciBuilderEvent.JobName, ciBuilderEvent.PodName, ciBuilderEvent.BuildEventType)
				}
			}(ciBuilderEvent.JobName)
		} else {
			log.Debug().Msgf("Job is already removed by cancellation, no need to remove job %v and pod %v for event %v", ciBuilderEvent.JobName, ciBuilderEvent.PodName, ciBuilderEvent.BuildEventType)
		}

		go func(ciBuilderEvent contracts.EstafetteCiBuilderEvent) {
			// create new context to avoid cancellation impacting execution
			span, _ := opentracing.StartSpanFromContext(c.Request.Context(), "estafette:AsyncUpdateJobResources")
			ctx := opentracing.ContextWithSpan(context.Background(), span)
			defer span.Finish()

			err = h.buildService.UpdateJobResources(ctx, ciBuilderEvent)
			if err != nil {
				log.Error().Err(err).Msgf("Failed updating max cpu and memory from prometheus for pod %v", ciBuilderEvent.PodName)
			}
		}(ciBuilderEvent)

	default:
		log.Warn().Msgf("Unsupported Estafette build event type '%v'", ciBuilderEvent.BuildEventType)
	}

	c.String(http.StatusOK, "Aye aye!")
}
