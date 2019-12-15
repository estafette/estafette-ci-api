package estafette

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/estafette/estafette-ci-api/auth"
	"github.com/estafette/estafette-ci-api/clients/cloudstorage"
	"github.com/estafette/estafette-ci-api/clients/cockroach"
	estafetteclt "github.com/estafette/estafette-ci-api/clients/estafette"
	"github.com/estafette/estafette-ci-api/config"
	estafettedom "github.com/estafette/estafette-ci-api/domain/estafette"
	"github.com/estafette/estafette-ci-api/helpers"
	contracts "github.com/estafette/estafette-ci-contracts"
	crypt "github.com/estafette/estafette-ci-crypt"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v2"
)

// APIHandler handles all api calls
type APIHandler interface {
	GetPipelines(*gin.Context)
	GetPipeline(*gin.Context)
	GetPipelineBuilds(*gin.Context)
	GetPipelineBuild(*gin.Context)
	CreatePipelineBuild(*gin.Context)
	CancelPipelineBuild(*gin.Context)
	GetPipelineBuildLogs(*gin.Context)
	TailPipelineBuildLogs(*gin.Context)
	PostPipelineBuildLogs(*gin.Context)
	GetPipelineBuildWarnings(*gin.Context)
	GetPipelineReleases(*gin.Context)
	GetPipelineRelease(*gin.Context)
	CreatePipelineRelease(*gin.Context)
	CancelPipelineRelease(*gin.Context)
	GetPipelineReleaseLogs(*gin.Context)
	TailPipelineReleaseLogs(*gin.Context)
	PostPipelineReleaseLogs(*gin.Context)
	GetFrequentLabels(*gin.Context)

	GetPipelineStatsBuildsDurations(*gin.Context)
	GetPipelineStatsReleasesDurations(*gin.Context)
	GetPipelineStatsBuildsCPUUsageMeasurements(*gin.Context)
	GetPipelineStatsReleasesCPUUsageMeasurements(*gin.Context)
	GetPipelineStatsBuildsMemoryUsageMeasurements(*gin.Context)
	GetPipelineStatsReleasesMemoryUsageMeasurements(*gin.Context)
	GetPipelineWarnings(*gin.Context)

	GetStatsPipelinesCount(*gin.Context)
	GetStatsBuildsCount(*gin.Context)
	GetStatsReleasesCount(*gin.Context)

	GetStatsMostBuilds(*gin.Context)
	GetStatsMostReleases(*gin.Context)

	GetStatsBuildsDuration(*gin.Context)
	GetStatsBuildsAdoption(*gin.Context)
	GetStatsReleasesAdoption(*gin.Context)

	GetLoggedInUser(*gin.Context)
	UpdateComputedTables(*gin.Context)

	GetConfig(*gin.Context)
	GetConfigCredentials(*gin.Context)
	GetConfigTrustedImages(*gin.Context)

	GetManifestTemplates(*gin.Context)
	GenerateManifest(*gin.Context)
	ValidateManifest(*gin.Context)
	EncryptSecret(*gin.Context)

	PostCronEvent(*gin.Context)

	CopyLogsToCloudStorage(c *gin.Context)

	Commands(*gin.Context)
}

type apiHandlerImpl struct {
	configFilePath       string
	config               config.APIServerConfig
	authConfig           config.AuthConfig
	encryptedConfig      config.APIConfig
	cockroachDBClient    cockroach.Client
	cloudStorageClient   cloudstorage.Client
	ciBuilderClient      estafetteclt.Client
	buildService         Service
	warningHelper        helpers.WarningHelper
	secretHelper         crypt.SecretHelper
	githubJobVarsFunc    func(context.Context, string, string, string) (string, string, error)
	bitbucketJobVarsFunc func(context.Context, string, string, string) (string, string, error)
}

// NewAPIHandler returns a new estafette.APIHandler
func NewAPIHandler(configFilePath string, config config.APIServerConfig, authConfig config.AuthConfig, encryptedConfig config.APIConfig, cockroachDBClient cockroach.Client, cloudStorageClient cloudstorage.Client, ciBuilderClient estafetteclt.Client, buildService Service, warningHelper helpers.WarningHelper, secretHelper crypt.SecretHelper, githubJobVarsFunc func(context.Context, string, string, string) (string, string, error), bitbucketJobVarsFunc func(context.Context, string, string, string) (string, string, error)) (apiHandler APIHandler) {

	apiHandler = &apiHandlerImpl{
		configFilePath:       configFilePath,
		config:               config,
		authConfig:           authConfig,
		encryptedConfig:      encryptedConfig,
		cockroachDBClient:    cockroachDBClient,
		cloudStorageClient:   cloudStorageClient,
		ciBuilderClient:      ciBuilderClient,
		buildService:         buildService,
		warningHelper:        warningHelper,
		secretHelper:         secretHelper,
		githubJobVarsFunc:    githubJobVarsFunc,
		bitbucketJobVarsFunc: bitbucketJobVarsFunc,
	}

	return

}

func (h *apiHandlerImpl) GetPipelines(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetPipelines")
	defer span.Finish()

	pageNumber, pageSize, filters := h.getQueryParameters(c)

	span.SetTag("page-number", pageNumber)
	span.SetTag("page-size", pageSize)

	type PipelinesResult struct {
		pipelines []*contracts.Pipeline
		err       error
	}
	type PipelinesCountResult struct {
		pipelinesCount int
		err            error
	}

	// run 2 database queries in parallel and return their result via channels
	pipelinesChannel := make(chan PipelinesResult)
	pipelinesCountChannel := make(chan PipelinesCountResult)

	go func() {
		defer close(pipelinesChannel)
		pipelines, err := h.cockroachDBClient.GetPipelines(ctx, pageNumber, pageSize, filters, true)

		pipelinesChannel <- PipelinesResult{pipelines, err}
	}()

	go func() {
		defer close(pipelinesCountChannel)
		pipelinesCount, err := h.cockroachDBClient.GetPipelinesCount(ctx, filters)

		pipelinesCountChannel <- PipelinesCountResult{pipelinesCount, err}
	}()

	// wait for GetPipelines to finish and check for errors
	pipelinesResult := <-pipelinesChannel
	if pipelinesResult.err != nil {
		log.Error().Err(pipelinesResult.err).Msg("Failed retrieving pipelines from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	// wait for GetPipelinesCount to finish and check for errors
	pipelinesCountResult := <-pipelinesCountChannel
	if pipelinesCountResult.err != nil {
		log.Error().Err(pipelinesCountResult.err).Msg("Failed retrieving pipelines count from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	response := contracts.ListResponse{
		Pagination: contracts.Pagination{
			Page:       pageNumber,
			Size:       pageSize,
			TotalItems: pipelinesCountResult.pipelinesCount,
			TotalPages: int(math.Ceil(float64(pipelinesCountResult.pipelinesCount) / float64(pageSize))),
		},
	}

	response.Items = make([]interface{}, len(pipelinesResult.pipelines))
	for i := range pipelinesResult.pipelines {
		response.Items[i] = pipelinesResult.pipelines[i]
	}

	c.JSON(http.StatusOK, response)
}

func (h *apiHandlerImpl) GetPipeline(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetPipeline")
	defer span.Finish()

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	span.SetTag("git-repo", fmt.Sprintf("%v/%v/%v", source, owner, repo))

	pipeline, err := h.cockroachDBClient.GetPipeline(ctx, source, owner, repo, true)
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

func (h *apiHandlerImpl) GetPipelineBuilds(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetPipelineBuilds")
	defer span.Finish()

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	span.SetTag("git-repo", fmt.Sprintf("%v/%v/%v", source, owner, repo))

	pageNumber, pageSize, filters := h.getQueryParameters(c)

	span.SetTag("page-number", pageNumber)
	span.SetTag("page-size", pageSize)

	// set tracing span tags
	span.SetTag("source", source)
	span.SetTag("owner", owner)
	span.SetTag("repo", repo)
	span.SetTag("page-number", pageNumber)
	span.SetTag("page-size", pageSize)

	type BuildsResult struct {
		builds []*contracts.Build
		err    error
	}
	type BuildsCountResult struct {
		buildsCount int
		err         error
	}

	// run 2 database queries in parallel and return their result via channels
	buildsChannel := make(chan BuildsResult)
	buildsCountChannel := make(chan BuildsCountResult)

	go func() {
		defer close(buildsChannel)
		builds, err := h.cockroachDBClient.GetPipelineBuilds(ctx, source, owner, repo, pageNumber, pageSize, filters, true)

		buildsChannel <- BuildsResult{builds, err}
	}()

	go func() {
		defer close(buildsCountChannel)
		buildsCount, err := h.cockroachDBClient.GetPipelineBuildsCount(ctx, source, owner, repo, filters)

		buildsCountChannel <- BuildsCountResult{buildsCount, err}
	}()

	// wait for GetPipelineBuilds to finish and check for errors
	buildsResult := <-buildsChannel
	if buildsResult.err != nil {
		log.Error().Err(buildsResult.err).Msgf("Failed retrieving builds for %v/%v/%v from db", source, owner, repo)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	// wait for GetPipelineBuildsCount to finish and check for errors
	buildsCountResult := <-buildsCountChannel
	if buildsCountResult.err != nil {
		log.Error().Err(buildsCountResult.err).Msgf("Failed retrieving builds count for %v/%v/%v from db", source, owner, repo)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	response := contracts.ListResponse{
		Pagination: contracts.Pagination{
			Page:       pageNumber,
			Size:       pageSize,
			TotalItems: buildsCountResult.buildsCount,
			TotalPages: int(math.Ceil(float64(buildsCountResult.buildsCount) / float64(pageSize))),
		},
	}

	response.Items = make([]interface{}, len(buildsResult.builds))
	for i := range buildsResult.builds {
		response.Items[i] = buildsResult.builds[i]
	}

	c.JSON(http.StatusOK, response)
}

func (h *apiHandlerImpl) GetPipelineBuild(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetPipelineBuild")
	defer span.Finish()

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	revisionOrID := c.Param("revisionOrId")

	span.SetTag("git-repo", fmt.Sprintf("%v/%v/%v", source, owner, repo))

	if len(revisionOrID) == 40 {

		span.SetTag("git-revision", revisionOrID)

		build, err := h.cockroachDBClient.GetPipelineBuild(ctx, source, owner, repo, revisionOrID, false)
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

	span.SetTag("build-id", revisionOrID)

	id, err := strconv.Atoi(revisionOrID)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed reading id from path parameter for %v/%v/%v/builds/%v", source, owner, repo, revisionOrID)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": "Path parameter id is not of type integer"})
		return
	}

	build, err := h.cockroachDBClient.GetPipelineBuildByID(ctx, source, owner, repo, id, false)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, id)
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

func (h *apiHandlerImpl) CreatePipelineBuild(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::CreatePipelineBuild")
	defer span.Finish()

	user := c.MustGet(gin.AuthUserKey).(auth.User)

	var buildCommand contracts.Build
	c.BindJSON(&buildCommand)

	// match source, owner, repo with values in binded release
	if buildCommand.RepoSource != c.Param("source") {
		errorMessage := fmt.Sprintf("RepoSource in path and post data do not match for pipeline %v/%v/%v for build command issued by %v", buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, user)
		log.Error().Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}
	if buildCommand.RepoOwner != c.Param("owner") {
		errorMessage := fmt.Sprintf("RepoOwner in path and post data do not match for pipeline %v/%v/%v for build command issued by %v", buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, user)
		log.Error().Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}
	if buildCommand.RepoName != c.Param("repo") {
		errorMessage := fmt.Sprintf("RepoName in path and post data do not match for pipeline %v/%v/%v for build command issued by %v", buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, user)
		log.Error().Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	// check if version exists and is valid to re-run
	failedBuilds, err := h.cockroachDBClient.GetPipelineBuildsByVersion(ctx, buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, buildCommand.BuildVersion, []string{"failed", "canceled"}, 1, false)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving build %v/%v/%v version %v for build command issued by %v", buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, buildCommand.BuildVersion, user)
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	nonFailedBuilds, err := h.cockroachDBClient.GetPipelineBuildsByVersion(ctx, buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, buildCommand.BuildVersion, []string{"succeeded", "running", "pending", "canceling"}, 1, false)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving build %v/%v/%v version %v for build command issued by %v", buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, buildCommand.BuildVersion, user)
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
		errorMessage := fmt.Sprintf("No failed build %v/%v/%v version %v for build command issued by %v", buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, buildCommand.BuildVersion, user)
		log.Error().Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}
	if hasNonFailedBuilds {
		errorMessage := fmt.Sprintf("Version %v of pipeline %v/%v/%v has builds that are succeeded or running ; only if all builds are failed the pipeline can be re-run", buildCommand.BuildVersion, buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName)
		log.Error().Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	// set trigger event to manual
	failedBuild.Events = []manifest.EstafetteEvent{
		manifest.EstafetteEvent{
			Manual: &manifest.EstafetteManualEvent{
				UserID: user.Email,
			},
		},
	}

	// hand off to build service
	createdBuild, err := h.buildService.CreateBuild(ctx, *failedBuild, false)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed creating build %v/%v/%v version %v for build command issued by %v", buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, buildCommand.BuildVersion, user)
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	c.JSON(http.StatusCreated, createdBuild)
}

func (h *apiHandlerImpl) CancelPipelineBuild(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::CancelPipelineBuild")
	defer span.Finish()

	user := c.MustGet(gin.AuthUserKey).(auth.User)

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	revisionOrID := c.Param("revisionOrId")

	span.SetTag("git-repo", fmt.Sprintf("%v/%v/%v", source, owner, repo))
	span.SetTag("build-id", revisionOrID)

	id, err := strconv.Atoi(revisionOrID)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed reading id from path parameter for %v/%v/%v/builds/%v", source, owner, repo, revisionOrID)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": "Path parameter id is not of type integer"})
		return
	}

	// retrieve build
	build, err := h.cockroachDBClient.GetPipelineBuildByID(ctx, source, owner, repo, id, false)
	if err != nil {
		log.Error().Err(err).Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, revisionOrID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Retrieving pipeline build failed"})
		return
	}
	if build == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline build not found"})
		return
	}
	if build.BuildStatus == "canceling" {
		// apparently cancel was already clicked, but somehow the job didn't update the status to canceled
		jobName := h.ciBuilderClient.GetJobName("build", build.RepoOwner, build.RepoName, build.ID)
		h.ciBuilderClient.CancelCiBuilderJob(ctx, jobName)
		h.cockroachDBClient.UpdateBuildStatus(ctx, build.RepoSource, build.RepoOwner, build.RepoName, id, "canceled")
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Canceled build by user %v", user.Email)})
		return
	}

	if build.BuildStatus != "pending" && build.BuildStatus != "running" {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": fmt.Sprintf("Build with status %v cannot be canceled", build.BuildStatus)})
		return
	}

	// this build can be canceled, set status 'canceling' and cancel the build job
	jobName := h.ciBuilderClient.GetJobName("build", build.RepoOwner, build.RepoName, build.ID)
	cancelErr := h.ciBuilderClient.CancelCiBuilderJob(ctx, jobName)
	buildStatus := "canceling"
	if build.BuildStatus == "pending" {
		// job might not have created a builder yet, so set status to canceled straightaway
		buildStatus = "canceled"
	}
	err = h.cockroachDBClient.UpdateBuildStatus(ctx, build.RepoSource, build.RepoOwner, build.RepoName, id, buildStatus)
	if err != nil {
		log.Error().Err(err).Msgf("Failed updating build status for %v/%v/%v/builds/%v in db", source, owner, repo, revisionOrID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Failed setting pipeline build status to canceling"})
		return
	}

	// canceling the job failed because it no longer existed we should set canceled status right after having set it to canceling
	if cancelErr != nil && build.BuildStatus == "running" {
		buildStatus = "canceled"
		err = h.cockroachDBClient.UpdateBuildStatus(ctx, build.RepoSource, build.RepoOwner, build.RepoName, id, buildStatus)
		if err != nil {
			log.Error().Err(err).Msgf("Failed updating build status to canceled after setting it to canceling for %v/%v/%v/builds/%v in db", source, owner, repo, revisionOrID)
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Failed setting pipeline build status to canceled"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Canceled build by user %v", user.Email)})
}

func (h *apiHandlerImpl) GetPipelineBuildLogs(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetPipelineBuildLogs")
	defer span.Finish()

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	revisionOrID := c.Param("revisionOrId")

	span.SetTag("git-repo", fmt.Sprintf("%v/%v/%v", source, owner, repo))

	var build *contracts.Build
	var err error
	if len(revisionOrID) == 40 {
		span.SetTag("git-revision", revisionOrID)

		build, err = h.cockroachDBClient.GetPipelineBuild(ctx, source, owner, repo, revisionOrID, false)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, revisionOrID)
		}
	} else {
		span.SetTag("build-id", revisionOrID)

		id, err := strconv.Atoi(revisionOrID)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed reading id from path parameter for %v/%v/%v/builds/%v", source, owner, repo, revisionOrID)
			c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": "Path parameter id is not of type integer"})
			return
		}

		build, err = h.cockroachDBClient.GetPipelineBuildByID(ctx, source, owner, repo, id, false)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, id)
		}
	}

	if build == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline build not found"})
		return
	}

	buildLog, err := h.cockroachDBClient.GetPipelineBuildLogs(ctx, source, owner, repo, build.RepoBranch, build.RepoRevision, build.ID, h.config.ReadLogFromDatabase())
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving build logs for %v/%v/%v/builds/%v/logs from db", source, owner, repo, revisionOrID)
	}
	if buildLog == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline build log not found"})
		return
	}

	if h.config.ReadLogFromCloudStorage() {
		err := h.cloudStorageClient.GetPipelineBuildLogs(ctx, *buildLog, strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip"), c.Writer)
		if err != nil {
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

func (h *apiHandlerImpl) TailPipelineBuildLogs(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::TailPipelineBuildLogs")
	defer span.Finish()

	owner := c.Param("owner")
	repo := c.Param("repo")
	id := c.Param("revisionOrId")

	jobName := h.ciBuilderClient.GetJobName("build", owner, repo, id)

	logChannel := make(chan contracts.TailLogLine, 50)

	go h.ciBuilderClient.TailCiBuilderJobLogs(ctx, jobName, logChannel)

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

func (h *apiHandlerImpl) PostPipelineBuildLogs(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::PostPipelineBuildLogs")
	defer span.Finish()

	if c.MustGet(gin.AuthUserKey).(string) != "apiKey" {
		c.Status(http.StatusUnauthorized)
		return
	}

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	revisionOrID := c.Param("revisionOrId")

	span.SetTag("git-repo", fmt.Sprintf("%v/%v/%v", source, owner, repo))

	var buildLog contracts.BuildLog
	err := c.Bind(&buildLog)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed binding v2 logs for %v/%v/%v/%v", source, owner, repo, revisionOrID)
		c.String(http.StatusInternalServerError, "Oops, something went wrong")
		return
	}

	if len(revisionOrID) != 40 {
		span.SetTag("build-id", revisionOrID)

		_, err := strconv.Atoi(revisionOrID)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed reading id from path parameter for %v/%v/%v/builds/%v", source, owner, repo, revisionOrID)
			c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": "Path parameter id is not of type integer"})
			return
		}

		buildLog.BuildID = revisionOrID
	}

	insertedBuildLog, err := h.cockroachDBClient.InsertBuildLog(ctx, buildLog, h.config.WriteLogToDatabase())
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed inserting logs for %v/%v/%v/%v", source, owner, repo, revisionOrID)
		c.String(http.StatusInternalServerError, "Oops, something went wrong")
		return
	}

	if h.config.WriteLogToCloudStorage() {
		err = h.cloudStorageClient.InsertBuildLog(ctx, insertedBuildLog)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed inserting logs into cloudstorage for %v/%v/%v/%v", source, owner, repo, revisionOrID)
		}
	}

	c.String(http.StatusOK, "Aye aye!")
}

func (h *apiHandlerImpl) GetPipelineBuildWarnings(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetPipelineBuildWarnings")
	defer span.Finish()

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	revisionOrID := c.Param("revisionOrId")

	span.SetTag("git-repo", fmt.Sprintf("%v/%v/%v", source, owner, repo))
	span.SetTag("build-id", revisionOrID)

	id, err := strconv.Atoi(revisionOrID)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed reading id from path parameter for %v/%v/%v/builds/%v", source, owner, repo, revisionOrID)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": "Path parameter id is not of type integer"})
		return
	}

	build, err := h.cockroachDBClient.GetPipelineBuildByID(ctx, source, owner, repo, id, false)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, id)
	}
	if build == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline build not found"})
		return
	}

	warnings, err := h.warningHelper.GetManifestWarnings(build.ManifestObject, build.RepoOwner)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed getting warnings for %v/%v/%v/builds/%v manifest", source, owner, repo, revisionOrID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Failed getting warnings for manifest"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"warnings": warnings})
}

func (h *apiHandlerImpl) GetPipelineReleases(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetPipelineReleases")
	defer span.Finish()

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	span.SetTag("git-repo", fmt.Sprintf("%v/%v/%v", source, owner, repo))

	pageNumber, pageSize, filters := h.getQueryParameters(c)

	span.SetTag("page-number", pageNumber)
	span.SetTag("page-size", pageSize)

	type ReleasesResult struct {
		releases []*contracts.Release
		err      error
	}
	type ReleasesCountResult struct {
		releasesCount int
		err           error
	}

	// run 2 database queries in parallel and return their result via channels
	releasesChannel := make(chan ReleasesResult)
	releasesCountChannel := make(chan ReleasesCountResult)

	go func() {
		defer close(releasesChannel)
		releases, err := h.cockroachDBClient.GetPipelineReleases(ctx, source, owner, repo, pageNumber, pageSize, filters)

		releasesChannel <- ReleasesResult{releases, err}
	}()

	go func() {
		defer close(releasesCountChannel)
		releasesCount, err := h.cockroachDBClient.GetPipelineReleasesCount(ctx, source, owner, repo, filters)

		releasesCountChannel <- ReleasesCountResult{releasesCount, err}
	}()

	// wait for GetPipelineReleases to finish and check for errors
	releasesResult := <-releasesChannel
	if releasesResult.err != nil {
		log.Error().Err(releasesResult.err).Msgf("Failed retrieving releases for %v/%v/%v from db", source, owner, repo)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	// wait for GetPipelineReleasesCount to finish and check for errors
	releasesCountResult := <-releasesCountChannel
	if releasesCountResult.err != nil {
		log.Error().Err(releasesCountResult.err).Msgf("Failed retrieving releases count for %v/%v/%v from db", source, owner, repo)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	response := contracts.ListResponse{
		Pagination: contracts.Pagination{
			Page:       pageNumber,
			Size:       pageSize,
			TotalItems: releasesCountResult.releasesCount,
			TotalPages: int(math.Ceil(float64(releasesCountResult.releasesCount) / float64(pageSize))),
		},
	}

	response.Items = make([]interface{}, len(releasesResult.releases))
	for i := range releasesResult.releases {
		response.Items[i] = releasesResult.releases[i]
	}

	c.JSON(http.StatusOK, response)
}

func (h *apiHandlerImpl) CreatePipelineRelease(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::CreatePipelineRelease")
	defer span.Finish()

	user := c.MustGet(gin.AuthUserKey).(auth.User)

	var releaseCommand contracts.Release
	c.BindJSON(&releaseCommand)

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

	pipeline, err := h.cockroachDBClient.GetPipeline(ctx, releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, false)
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
	builds, err := h.cockroachDBClient.GetPipelineBuildsByVersion(ctx, releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, releaseCommand.ReleaseVersion, []string{"succeeded"}, 1, false)
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
	if build.BuildStatus != "succeeded" {
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
	createdRelease, err := h.buildService.CreateRelease(ctx, contracts.Release{
		Name:           releaseCommand.Name,
		Action:         releaseCommand.Action,
		RepoSource:     releaseCommand.RepoSource,
		RepoOwner:      releaseCommand.RepoOwner,
		RepoName:       releaseCommand.RepoName,
		ReleaseVersion: releaseCommand.ReleaseVersion,

		// set trigger event to manual
		Events: []manifest.EstafetteEvent{
			manifest.EstafetteEvent{
				Manual: &manifest.EstafetteManualEvent{
					UserID: user.Email,
				},
			},
		},
	}, *build.ManifestObject, build.RepoBranch, build.RepoRevision, true)

	if err != nil {
		errorMessage := fmt.Sprintf("Failed creating release %v for pipeline %v/%v/%v version %v for release command issued by %v", releaseCommand.Name, releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, releaseCommand.ReleaseVersion, user.Email)
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}

	c.JSON(http.StatusCreated, createdRelease)
}

func (h *apiHandlerImpl) CancelPipelineRelease(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::CancelPipelineRelease")
	defer span.Finish()

	user := c.MustGet(gin.AuthUserKey).(auth.User)

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	idValue := c.Param("id")

	span.SetTag("git-repo", fmt.Sprintf("%v/%v/%v", source, owner, repo))
	span.SetTag("release-id", idValue)

	id, err := strconv.Atoi(idValue)
	if err != nil {
		log.Error().Err(err).Msgf("Failed reading id from path parameter for %v/%v/%v/%v", source, owner, repo, idValue)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": "Path parameter id is not of type integer"})
		return
	}

	release, err := h.cockroachDBClient.GetPipelineRelease(ctx, source, owner, repo, id)
	if err != nil {
		log.Error().Err(err).Msgf("Failed retrieving release for %v/%v/%v/%v from db", source, owner, repo, id)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Retrieving pipeline release failed"})
		return
	}
	if release == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline release not found"})
		return
	}
	if release.ReleaseStatus == "canceling" {
		jobName := h.ciBuilderClient.GetJobName("release", release.RepoOwner, release.RepoName, release.ID)
		h.ciBuilderClient.CancelCiBuilderJob(ctx, jobName)
		h.cockroachDBClient.UpdateReleaseStatus(ctx, release.RepoSource, release.RepoOwner, release.RepoName, id, "canceled")
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Canceled release by user %v", user.Email)})
		return
	}
	if release.ReleaseStatus != "pending" && release.ReleaseStatus != "running" {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": fmt.Sprintf("Release with status %v cannot be canceled", release.ReleaseStatus)})
		return
	}

	// this release can be canceled, set status 'canceling' and cancel the release job
	jobName := h.ciBuilderClient.GetJobName("release", release.RepoOwner, release.RepoName, release.ID)
	cancelErr := h.ciBuilderClient.CancelCiBuilderJob(ctx, jobName)
	releaseStatus := "canceling"
	if release.ReleaseStatus == "pending" {
		// job might not have created a builder yet, so set status to canceled straightaway
		releaseStatus = "canceled"
	}
	err = h.cockroachDBClient.UpdateReleaseStatus(ctx, release.RepoSource, release.RepoOwner, release.RepoName, id, releaseStatus)
	if err != nil {
		log.Error().Err(err).Msgf("Failed updating release status for %v/%v/%v/builds/%v in db", source, owner, repo, id)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Failed setting pipeline release status to canceling"})
		return
	}

	// canceling the job failed because it no longer existed we should set canceled status right after having set it to canceling
	if cancelErr != nil && release.ReleaseStatus == "running" {
		releaseStatus = "canceled"
		err = h.cockroachDBClient.UpdateReleaseStatus(ctx, release.RepoSource, release.RepoOwner, release.RepoName, id, releaseStatus)
		if err != nil {
			log.Error().Err(err).Msgf("Failed updating release status to canceled after setting it to canceling for %v/%v/%v/builds/%v in db", source, owner, repo, id)
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Failed setting pipeline release status to canceled"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Canceled release by user %v", user.Email)})
}

func (h *apiHandlerImpl) GetPipelineRelease(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetPipelineRelease")
	defer span.Finish()

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	idValue := c.Param("id")

	span.SetTag("git-repo", fmt.Sprintf("%v/%v/%v", source, owner, repo))
	span.SetTag("release-id", idValue)

	id, err := strconv.Atoi(idValue)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed reading id from path parameter for %v/%v/%v/%v", source, owner, repo, idValue)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": "Path parameter id is not of type integer"})
		return
	}

	release, err := h.cockroachDBClient.GetPipelineRelease(ctx, source, owner, repo, id)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving release for %v/%v/%v/%v from db", source, owner, repo, id)
	}
	if release == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline release not found"})
		return
	}

	c.JSON(http.StatusOK, release)
}

func (h *apiHandlerImpl) GetPipelineReleaseLogs(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetPipelineReleaseLogs")
	defer span.Finish()

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	idValue := c.Param("id")

	span.SetTag("git-repo", fmt.Sprintf("%v/%v/%v", source, owner, repo))
	span.SetTag("release-id", idValue)

	id, err := strconv.Atoi(idValue)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed reading id from path parameter for %v/%v/%v/%v", source, owner, repo, idValue)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": "Path parameter id is not of type integer"})
		return
	}

	releaseLog, err := h.cockroachDBClient.GetPipelineReleaseLogs(ctx, source, owner, repo, id, h.config.ReadLogFromDatabase())
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving release logs for %v/%v/%v/%v from db", source, owner, repo, id)
	}
	if releaseLog == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline release log not found"})
		return
	}

	if h.config.ReadLogFromCloudStorage() {
		err := h.cloudStorageClient.GetPipelineReleaseLogs(ctx, *releaseLog, strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip"), c.Writer)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed retrieving release logs for %v/%v/%v/%v from cloud storage", source, owner, repo, id)
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
			return
		}
		c.Writer.Flush()
		return
	}

	c.JSON(http.StatusOK, releaseLog.Steps)
}

func (h *apiHandlerImpl) TailPipelineReleaseLogs(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::TailPipelineReleaseLogs")
	defer span.Finish()

	owner := c.Param("owner")
	repo := c.Param("repo")
	id := c.Param("id")

	jobName := h.ciBuilderClient.GetJobName("release", owner, repo, id)

	logChannel := make(chan contracts.TailLogLine, 50)

	go h.ciBuilderClient.TailCiBuilderJobLogs(ctx, jobName, logChannel)

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

func (h *apiHandlerImpl) PostPipelineReleaseLogs(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::PostPipelineReleaseLogs")
	defer span.Finish()

	if c.MustGet(gin.AuthUserKey).(string) != "apiKey" {
		c.Status(http.StatusUnauthorized)
		return
	}

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	idValue := c.Param("id")

	span.SetTag("git-repo", fmt.Sprintf("%v/%v/%v", source, owner, repo))
	span.SetTag("release-id", idValue)

	id, err := strconv.Atoi(idValue)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed reading id from path parameter for %v/%v/%v/%v", source, owner, repo, idValue)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": "Path parameter id is not of type integer"})
		return
	}

	var releaseLog contracts.ReleaseLog
	err = c.Bind(&releaseLog)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed binding release logs for %v/%v/%v/%v", source, owner, repo, id)
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_SERVER_ERROR", "message": "Failed binding release logs from body"})
		return
	}

	insertedReleaseLog, err := h.cockroachDBClient.InsertReleaseLog(ctx, releaseLog, h.config.WriteLogToDatabase())
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed inserting release logs for %v/%v/%v/%v", source, owner, repo, id)
		c.String(http.StatusInternalServerError, "Oops, something went wrong")
		return
	}

	if h.config.WriteLogToCloudStorage() {
		err = h.cloudStorageClient.InsertReleaseLog(ctx, insertedReleaseLog)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed inserting release logs into cloud storage for %v/%v/%v/%v", source, owner, repo, id)
		}
	}

	c.String(http.StatusOK, "Aye aye!")
}

func (h *apiHandlerImpl) GetFrequentLabels(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetFrequentLabels")
	defer span.Finish()

	pageNumber, pageSize, filters := h.getQueryParameters(c)

	span.SetTag("page-number", pageNumber)
	span.SetTag("page-size", pageSize)

	type LabelsResult struct {
		labels []map[string]interface{}
		err    error
	}
	type LabelsCountResult struct {
		labelsCount int
		err         error
	}

	// run 2 database queries in parallel and return their result via channels
	labelsChannel := make(chan LabelsResult)
	labelsCountChannel := make(chan LabelsCountResult)

	go func() {
		defer close(labelsChannel)
		labels, err := h.cockroachDBClient.GetFrequentLabels(ctx, pageNumber, pageSize, filters)

		labelsChannel <- LabelsResult{labels, err}
	}()

	go func() {
		defer close(labelsCountChannel)
		labelsCount, err := h.cockroachDBClient.GetFrequentLabelsCount(ctx, filters)

		labelsCountChannel <- LabelsCountResult{labelsCount, err}
	}()

	// wait for GetPipelines to finish and check for errors
	labelsResult := <-labelsChannel
	if labelsResult.err != nil {
		log.Error().Err(labelsResult.err).Msg("Failed retrieving frequent labels from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	// wait for GetPipelinesCount to finish and check for errors
	labelsCountResult := <-labelsCountChannel
	if labelsCountResult.err != nil {
		log.Error().Err(labelsCountResult.err).Msg("Failed retrieving frequent labels count from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	response := contracts.ListResponse{
		Items: make([]interface{}, len(labelsResult.labels)),
		Pagination: contracts.Pagination{
			Page:       pageNumber,
			Size:       pageSize,
			TotalItems: labelsCountResult.labelsCount,
			TotalPages: int(math.Ceil(float64(labelsCountResult.labelsCount) / float64(pageSize))),
		},
	}

	for i := range labelsResult.labels {
		response.Items[i] = labelsResult.labels[i]
	}

	c.JSON(http.StatusOK, response)
}

func (h *apiHandlerImpl) GetPipelineStatsBuildsDurations(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetPipelineStatsBuildsDurations")
	defer span.Finish()

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	span.SetTag("git-repo", fmt.Sprintf("%v/%v/%v", source, owner, repo))

	// get filters (?filter[last]=100)
	filters := map[string][]string{}
	filters["status"] = h.getStatusFilterWithDefault(c, []string{"succeeded"})
	filters["last"] = h.getLastFilter(c, 100)

	durations, err := h.cockroachDBClient.GetPipelineBuildsDurations(ctx, source, owner, repo, filters)
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

func (h *apiHandlerImpl) GetPipelineStatsReleasesDurations(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetPipelineStatsReleasesDurations")
	defer span.Finish()

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	span.SetTag("git-repo", fmt.Sprintf("%v/%v/%v", source, owner, repo))

	// get filters (?filter[last]=100)
	filters := map[string][]string{}
	filters["status"] = h.getStatusFilterWithDefault(c, []string{"succeeded"})
	filters["last"] = h.getLastFilter(c, 100)

	durations, err := h.cockroachDBClient.GetPipelineReleasesDurations(ctx, source, owner, repo, filters)
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

func (h *apiHandlerImpl) GetPipelineStatsBuildsCPUUsageMeasurements(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetPipelineStatsBuildsCPUUsageMeasurements")
	defer span.Finish()

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	span.SetTag("git-repo", fmt.Sprintf("%v/%v/%v", source, owner, repo))

	// get filters (?filter[last]=100)
	filters := map[string][]string{}
	filters["status"] = h.getStatusFilterWithDefault(c, []string{"succeeded"})
	filters["last"] = h.getLastFilter(c, 100)

	measurements, err := h.cockroachDBClient.GetPipelineBuildsCPUUsageMeasurements(ctx, source, owner, repo, filters)
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

func (h *apiHandlerImpl) GetPipelineStatsReleasesCPUUsageMeasurements(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetPipelineStatsReleasesCPUUsageMeasurements")
	defer span.Finish()

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	span.SetTag("git-repo", fmt.Sprintf("%v/%v/%v", source, owner, repo))

	// get filters (?filter[last]=100)
	filters := map[string][]string{}
	filters["status"] = h.getStatusFilterWithDefault(c, []string{"succeeded"})
	filters["last"] = h.getLastFilter(c, 100)

	measurements, err := h.cockroachDBClient.GetPipelineReleasesCPUUsageMeasurements(ctx, source, owner, repo, filters)
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

func (h *apiHandlerImpl) GetPipelineStatsBuildsMemoryUsageMeasurements(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetPipelineStatsBuildsMemoryUsageMeasurements")
	defer span.Finish()

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	span.SetTag("git-repo", fmt.Sprintf("%v/%v/%v", source, owner, repo))

	// get filters (?filter[last]=100)
	filters := map[string][]string{}
	filters["status"] = h.getStatusFilterWithDefault(c, []string{"succeeded"})
	filters["last"] = h.getLastFilter(c, 100)

	measurements, err := h.cockroachDBClient.GetPipelineBuildsMemoryUsageMeasurements(ctx, source, owner, repo, filters)
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

func (h *apiHandlerImpl) GetPipelineStatsReleasesMemoryUsageMeasurements(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetPipelineStatsReleasesMemoryUsageMeasurements")
	defer span.Finish()

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	span.SetTag("git-repo", fmt.Sprintf("%v/%v/%v", source, owner, repo))

	// get filters (?filter[last]=100)
	filters := map[string][]string{}
	filters["status"] = h.getStatusFilterWithDefault(c, []string{"succeeded"})
	filters["last"] = h.getLastFilter(c, 100)

	measurements, err := h.cockroachDBClient.GetPipelineReleasesMemoryUsageMeasurements(ctx, source, owner, repo, filters)
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

func (h *apiHandlerImpl) GetPipelineWarnings(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetPipelineWarnings")
	defer span.Finish()

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	span.SetTag("git-repo", fmt.Sprintf("%v/%v/%v", source, owner, repo))

	// get filters (?filter[last]=100)
	filters := map[string][]string{}
	filters["status"] = h.getStatusFilterWithDefault(c, []string{"succeeded"})
	filters["last"] = h.getLastFilter(c, 25)

	pipeline, err := h.cockroachDBClient.GetPipeline(ctx, source, owner, repo, false)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving pipeline for %v/%v/%v from db", source, owner, repo)
	}
	if pipeline == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline not found"})
		return
	}

	warnings := []contracts.Warning{}

	durations, err := h.cockroachDBClient.GetPipelineBuildsDurations(ctx, source, owner, repo, filters)
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

	manifestWarnings, err := h.warningHelper.GetManifestWarnings(pipeline.ManifestObject, pipeline.RepoOwner)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed getting warnings for %v/%v/%v manifest", source, owner, repo)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Failed getting warnings for manifest"})
		return
	}
	warnings = append(warnings, manifestWarnings...)

	c.JSON(http.StatusOK, gin.H{"warnings": warnings})
}

func (h *apiHandlerImpl) GetStatsPipelinesCount(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetStatsPipelinesCount")
	defer span.Finish()

	// get filters (?filter[status]=running,succeeded&filter[since]=1w
	filters := map[string][]string{}
	filters["status"] = h.getStatusFilter(c)
	filters["since"] = h.getSinceFilter(c)

	pipelinesCount, err := h.cockroachDBClient.GetPipelinesCount(ctx, filters)
	if err != nil {
		log.Error().Err(err).
			Msg("Failed retrieving pipelines count from db")
	}

	c.JSON(http.StatusOK, gin.H{
		"count": pipelinesCount,
	})
}

func (h *apiHandlerImpl) GetStatsReleasesCount(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetStatsReleasesCount")
	defer span.Finish()

	// get filters (?filter[status]=running,succeeded&filter[since]=1w
	filters := map[string][]string{}
	filters["status"] = h.getStatusFilter(c)
	filters["since"] = h.getSinceFilter(c)

	releasesCount, err := h.cockroachDBClient.GetReleasesCount(ctx, filters)
	if err != nil {
		log.Error().Err(err).
			Msg("Failed retrieving releases count from db")
	}

	c.JSON(http.StatusOK, gin.H{
		"count": releasesCount,
	})
}

func (h *apiHandlerImpl) GetStatsBuildsCount(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetStatsBuildsCount")
	defer span.Finish()

	// get filters (?filter[status]=running,succeeded&filter[since]=1w
	filters := map[string][]string{}
	filters["status"] = h.getStatusFilter(c)
	filters["since"] = h.getSinceFilter(c)

	buildsCount, err := h.cockroachDBClient.GetBuildsCount(ctx, filters)
	if err != nil {
		log.Error().Err(err).
			Msg("Failed retrieving builds count from db")
	}

	c.JSON(http.StatusOK, gin.H{
		"count": buildsCount,
	})
}

func (h *apiHandlerImpl) GetStatsMostBuilds(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetStatsMostBuilds")
	defer span.Finish()

	pageNumber, pageSize, filters := h.getQueryParameters(c)

	span.SetTag("page-number", pageNumber)
	span.SetTag("page-size", pageSize)

	pipelines, err := h.cockroachDBClient.GetPipelinesWithMostBuilds(ctx, pageNumber, pageSize, filters)
	if err != nil {
		errorMessage := "Failed retrieving pipelines with most builds from db"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}
	pipelinesCount, err := h.cockroachDBClient.GetPipelinesWithMostBuildsCount(ctx, filters)
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

func (h *apiHandlerImpl) GetStatsMostReleases(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetStatsMostReleases")
	defer span.Finish()

	pageNumber, pageSize, filters := h.getQueryParameters(c)

	span.SetTag("page-number", pageNumber)
	span.SetTag("page-size", pageSize)

	pipelines, err := h.cockroachDBClient.GetPipelinesWithMostReleases(ctx, pageNumber, pageSize, filters)
	if err != nil {
		errorMessage := "Failed retrieving pipelines with most builds from db"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}
	pipelinesCount, err := h.cockroachDBClient.GetPipelinesWithMostReleasesCount(ctx, filters)
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

func (h *apiHandlerImpl) GetStatsBuildsDuration(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetStatsBuildsDuration")
	defer span.Finish()

	// get filters (?filter[status]=running,succeeded&filter[since]=1w
	filters := map[string][]string{}
	filters["status"] = h.getStatusFilter(c)
	filters["since"] = h.getSinceFilter(c)

	buildsDuration, err := h.cockroachDBClient.GetBuildsDuration(ctx, filters)
	if err != nil {
		log.Error().Err(err).
			Msg("Failed retrieving builds duration from db")
	}

	c.JSON(http.StatusOK, gin.H{
		"duration": buildsDuration,
	})
}

func (h *apiHandlerImpl) GetStatsBuildsAdoption(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetStatsBuildsAdoption")
	defer span.Finish()

	buildTimes, err := h.cockroachDBClient.GetFirstBuildTimes(ctx)
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

func (h *apiHandlerImpl) GetStatsReleasesAdoption(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetStatsReleasesAdoption")
	defer span.Finish()

	releaseTimes, err := h.cockroachDBClient.GetFirstReleaseTimes(ctx)
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

func (h *apiHandlerImpl) GetLoggedInUser(c *gin.Context) {

	span, _ := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetLoggedInUser")
	defer span.Finish()

	user := c.MustGet(gin.AuthUserKey).(auth.User)

	c.JSON(http.StatusOK, user)
}

func (h *apiHandlerImpl) UpdateComputedTables(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::UpdateComputedTables")
	defer span.Finish()

	user := c.MustGet(gin.AuthUserKey).(auth.User)

	filters := map[string][]string{}
	filters["status"] = h.getStatusFilter(c)
	filters["since"] = h.getSinceFilter(c)
	filters["labels"] = h.getLabelsFilter(c)
	pipelinesCount, err := h.cockroachDBClient.GetPipelinesCount(ctx, filters)
	if err != nil {
		log.Error().Err(err).
			Msg("Failed retrieving pipelines count from db")
	}
	pageSize := 20
	totalPages := int(math.Ceil(float64(pipelinesCount) / float64(pageSize)))
	for pageNumber := 1; pageNumber <= totalPages; pageNumber++ {
		pipelines, err := h.cockroachDBClient.GetPipelines(ctx, pageNumber, pageSize, filters, false)
		if err != nil {
			log.Error().Err(err).
				Msg("Failed retrieving pipelines from db")
		}
		for _, p := range pipelines {

			h.cockroachDBClient.UpsertComputedPipeline(ctx, p.RepoSource, p.RepoOwner, p.RepoName)
			h.cockroachDBClient.UpdateComputedPipelineFirstInsertedAt(ctx, p.RepoSource, p.RepoOwner, p.RepoName)
			manifest, err := manifest.ReadManifest(p.Manifest)
			if err == nil {
				for _, r := range manifest.Releases {
					if len(r.Actions) > 0 {
						for _, a := range r.Actions {
							h.cockroachDBClient.UpsertComputedRelease(ctx, p.RepoSource, p.RepoOwner, p.RepoName, r.Name, a.Name)
							h.cockroachDBClient.UpdateComputedReleaseFirstInsertedAt(ctx, p.RepoSource, p.RepoOwner, p.RepoName, r.Name, a.Name)
						}
					} else {
						h.cockroachDBClient.UpsertComputedRelease(ctx, p.RepoSource, p.RepoOwner, p.RepoName, r.Name, "")
						h.cockroachDBClient.UpdateComputedReleaseFirstInsertedAt(ctx, p.RepoSource, p.RepoOwner, p.RepoName, r.Name, "")
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, user)
}

func (h *apiHandlerImpl) GetConfig(c *gin.Context) {

	span, _ := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetConfig")
	defer span.Finish()

	_ = c.MustGet(gin.AuthUserKey).(auth.User)

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

func (h *apiHandlerImpl) GetConfigCredentials(c *gin.Context) {

	span, _ := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetConfigCredentials")
	defer span.Finish()

	_ = c.MustGet(gin.AuthUserKey).(auth.User)

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

func (h *apiHandlerImpl) GetConfigTrustedImages(c *gin.Context) {

	span, _ := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetConfigTrustedImages")
	defer span.Finish()

	_ = c.MustGet(gin.AuthUserKey).(auth.User)

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

func (h *apiHandlerImpl) getStatusFilter(c *gin.Context) []string {
	return h.getStatusFilterWithDefault(c, []string{})
}

func (h *apiHandlerImpl) getStatusFilterWithDefault(c *gin.Context, defaultStatuses []string) []string {

	filterStatusValues, filterStatusExist := c.GetQueryArray("filter[status]")
	if filterStatusExist && len(filterStatusValues) > 0 && filterStatusValues[0] != "" {
		return filterStatusValues
	}

	return defaultStatuses
}

func (h *apiHandlerImpl) GetManifestTemplates(c *gin.Context) {

	span, _ := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GetManifestTemplates")
	defer span.Finish()

	configFiles, err := ioutil.ReadDir(filepath.Dir(h.configFilePath))
	if err != nil {
		log.Error().Err(err).Msgf("Failed listing config files directory")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	templates := []interface{}{}
	for _, f := range configFiles {

		configfileName := f.Name()

		// check if it's a manifest template
		re := regexp.MustCompile(`^manifest-(.+)\.tmpl`)
		match := re.FindStringSubmatch(configfileName)

		if len(match) == 2 {

			// read template file
			templateFilePath := filepath.Join(filepath.Dir(h.configFilePath), configfileName)
			data, err := ioutil.ReadFile(templateFilePath)
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
				if len(m) == 2 && !helpers.StringArrayContains(placeholders, m[1]) {
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

func (h *apiHandlerImpl) GenerateManifest(c *gin.Context) {

	span, _ := opentracing.StartSpanFromContext(c.Request.Context(), "Api::GenerateManifest")
	defer span.Finish()

	var aux struct {
		Template     string            `json:"template"`
		Placeholders map[string]string `json:"placeholders,omitempty"`
	}

	err := c.BindJSON(&aux)
	if err != nil {
		log.Error().Err(err).Msg("Failed binding json body")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	templateFilePath := filepath.Join(filepath.Dir(h.configFilePath), fmt.Sprintf("manifest-%v.tmpl", aux.Template))
	data, err := ioutil.ReadFile(templateFilePath)
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

func (h *apiHandlerImpl) ValidateManifest(c *gin.Context) {

	span, _ := opentracing.StartSpanFromContext(c.Request.Context(), "Api::ValidateManifest")
	defer span.Finish()

	var aux struct {
		Template string `json:"template"`
	}

	err := c.BindJSON(&aux)
	if err != nil {
		log.Error().Err(err).Msg("Failed binding json body")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	_, err = manifest.ReadManifest(aux.Template)
	status := "succeeded"
	errorString := ""
	if err != nil {
		status = "failed"
		errorString = err.Error()
	}

	c.JSON(http.StatusOK, gin.H{"status": status, "errors": errorString})
}

func (h *apiHandlerImpl) EncryptSecret(c *gin.Context) {

	span, _ := opentracing.StartSpanFromContext(c.Request.Context(), "Api::EncryptSecret")
	defer span.Finish()

	var aux struct {
		Base64Encode  bool   `json:"base64"`
		DoubleEncrypt bool   `json:"double"`
		Value         string `json:"value"`
	}

	err := c.BindJSON(&aux)
	if err != nil {
		log.Error().Err(err).Msg("Failed binding json body")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	value := aux.Value
	if aux.Base64Encode {
		value = base64.URLEncoding.EncodeToString([]byte(value))
	}

	encryptedString, err := h.secretHelper.EncryptEnvelope(value)
	if err != nil {
		log.Error().Err(err).Msg("Failed encrypting secret")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	if aux.DoubleEncrypt {
		encryptedString, err = h.secretHelper.EncryptEnvelope(encryptedString)
		if err != nil {
			log.Error().Err(err).Msg("Failed encrypting secret")
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"secret": encryptedString})
}

func (h *apiHandlerImpl) PostCronEvent(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::PostCronEvent")
	defer span.Finish()

	if c.MustGet(gin.AuthUserKey).(string) != "apiKey" {
		c.Status(http.StatusUnauthorized)
		return
	}

	err := h.buildService.FireCronTriggers(ctx)

	if err != nil {
		log.Error().Err(err).Msg("Failed firing cron triggers")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Hey Cron, here's a tock for your tick"})
}

func (h *apiHandlerImpl) CopyLogsToCloudStorage(c *gin.Context) {

	span, ctx := opentracing.StartSpanFromContext(c.Request.Context(), "Api::CopyLogsToCloudStorage")
	defer span.Finish()

	if c.MustGet(gin.AuthUserKey).(string) != "apiKey" {
		c.Status(http.StatusUnauthorized)
		return
	}

	pageNumber, pageSize, filters := h.getQueryParameters(c)

	searchValue := "builds"
	if search, ok := filters["search"]; ok && len(search) > 0 && search[0] != "" {
		searchValue = search[0]
	}

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	span.SetTag("git-repo", fmt.Sprintf("%v/%v/%v", source, owner, repo))

	if searchValue == "builds" {
		buildLogs, err := h.cockroachDBClient.GetPipelineBuildLogsPerPage(ctx, source, owner, repo, pageNumber, pageSize)
		if err != nil {
			log.Error().Err(err).Int("pageNumber", pageNumber).Int("pageSize", pageSize).Msgf("Failed retrieving build logs for %v/%v/%v", source, owner, repo)
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "error": err})
			return
		}

		errors := make(chan error, len(buildLogs))

		var wg sync.WaitGroup
		wg.Add(len(buildLogs))

		for _, bl := range buildLogs {
			go func(ctx context.Context, bl contracts.BuildLog) {
				defer wg.Done()

				err = h.cloudStorageClient.InsertBuildLog(ctx, bl)
				if err != nil {
					errors <- err
				}
			}(ctx, *bl)
		}

		// wait for all parallel runs to finish
		wg.Wait()

		// return error if any of them have been generated
		close(errors)
		for e := range errors {
			log.Error().Err(err).Int("pageNumber", pageNumber).Int("pageSize", pageSize).Msgf("Failed inserting build logs for %v/%v/%v into cloud storage", source, owner, repo)
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "error": e})
			return
		}

		c.String(http.StatusOK, strconv.Itoa(len(buildLogs)))
		return

	} else if searchValue == "releases" {
		releaseLogs, err := h.cockroachDBClient.GetPipelineReleaseLogsPerPage(ctx, source, owner, repo, pageNumber, pageSize)
		if err != nil {
			log.Error().Err(err).Int("pageNumber", pageNumber).Int("pageSize", pageSize).Msgf("Failed retrieving release logs for %v/%v/%v", source, owner, repo)
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "error": err})
			return
		}

		errors := make(chan error, len(releaseLogs))

		var wg sync.WaitGroup
		wg.Add(len(releaseLogs))

		for _, rl := range releaseLogs {
			go func(ctx context.Context, rl contracts.ReleaseLog) {
				defer wg.Done()

				err = h.cloudStorageClient.InsertReleaseLog(ctx, rl)
				if err != nil {
					errors <- err
				}
			}(ctx, *rl)
		}

		// wait for all parallel runs to finish
		wg.Wait()

		// return error if any of them have been generated
		close(errors)
		for e := range errors {
			log.Error().Err(err).Int("pageNumber", pageNumber).Int("pageSize", pageSize).Msgf("Failed inserting release logs for %v/%v/%v into cloud storage", source, owner, repo)
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "error": e})
			return
		}

		c.String(http.StatusOK, strconv.Itoa(len(releaseLogs)))
		return
	}

	c.String(http.StatusOK, "Aye aye!")
}

func (h *apiHandlerImpl) getSinceFilter(c *gin.Context) []string {

	filterSinceValues, filterSinceExist := c.GetQueryArray("filter[since]")
	if filterSinceExist {
		return filterSinceValues
	}

	return []string{"eternity"}
}

func (h *apiHandlerImpl) getLastFilter(c *gin.Context, defaultValue int) []string {

	filterLastValues, filterLastExist := c.GetQueryArray("filter[last]")
	if filterLastExist {
		return filterLastValues
	}

	return []string{strconv.Itoa(defaultValue)}
}

func (h *apiHandlerImpl) getLabelsFilter(c *gin.Context) []string {
	filterLabelsValues, filterLabelsExist := c.GetQueryArray("filter[labels]")
	if filterLabelsExist {
		return filterLabelsValues
	}

	return []string{}
}

func (h *apiHandlerImpl) getSearchFilter(c *gin.Context) []string {
	filterSearchValues, filterSearchExist := c.GetQueryArray("filter[search]")
	if filterSearchExist {
		return filterSearchValues
	}

	return []string{}
}

func (h *apiHandlerImpl) getQueryParameters(c *gin.Context) (int, int, map[string][]string) {
	return h.getPageNumber(c), h.getPageSize(c), h.getFilters(c)
}

func (h *apiHandlerImpl) getPageNumber(c *gin.Context) int {
	// get page number query string value or default to 1
	pageNumberValue := c.DefaultQuery("page[number]", "1")
	pageNumber, err := strconv.Atoi(pageNumberValue)
	if err != nil {
		pageNumber = 1
	}

	return pageNumber
}

func (h *apiHandlerImpl) getPageSize(c *gin.Context) int {
	// get page number query string value or default to 20 (maximize at 100)
	pageSizeValue := c.DefaultQuery("page[size]", "20")
	pageSize, err := strconv.Atoi(pageSizeValue)
	if err != nil {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return pageSize
}

func (h *apiHandlerImpl) getFilters(c *gin.Context) map[string][]string {
	// get filters (?filter[status]=running,succeeded&filter[since]=1w&filter[labels]=team%3Destafette-team)
	filters := map[string][]string{}
	filters["status"] = h.getStatusFilter(c)
	filters["since"] = h.getSinceFilter(c)
	filters["labels"] = h.getLabelsFilter(c)
	filters["search"] = h.getSearchFilter(c)

	return filters
}

func (h *apiHandlerImpl) obfuscateSecrets(input string) (string, error) {

	r, err := regexp.Compile(`estafette\.secret\(([a-zA-Z0-9.=_-]+)\)`)
	if err != nil {
		return "", err
	}

	// obfuscate all secrets
	return r.ReplaceAllLiteralString(input, "***"), nil
}

func (h *apiHandlerImpl) Commands(c *gin.Context) {

	ctx := c.Request.Context()

	if c.MustGet(gin.AuthUserKey).(string) != "apiKey" {
		log.Error().Msgf("Authentication for /api/commands failed")
		c.Status(http.StatusUnauthorized)
		return
	}

	eventType := c.GetHeader("X-Estafette-Event")
	log.Debug().Msgf("X-Estafette-Event is set to %v", eventType)
	// h.prometheusInboundEventTotals.With(prometheus.Labels{"event": eventType, "source": "estafette"}).Inc()

	eventJobname := c.GetHeader("X-Estafette-Event-Job-Name")
	log.Debug().Msgf("X-Estafette-Event-Job-Name is set to %v", eventJobname)

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body from Estafette 'build finished' event failed")
		c.String(http.StatusInternalServerError, "Reading body from Estafette 'build finished' event failed")
		return
	}

	log.Debug().Msgf("Read body for /api/commands for job %v", eventJobname)

	switch eventType {
	case
		"builder:nomanifest",
		"builder:running",
		"builder:succeeded",
		"builder:failed",
		"builder:canceled":

		// unmarshal json body
		var ciBuilderEvent estafettedom.CiBuilderEvent
		err = json.Unmarshal(body, &ciBuilderEvent)
		if err != nil {
			log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to CiBuilderEvent failed")
			return
		}

		log.Debug().Interface("ciBuilderEvent", ciBuilderEvent).Msgf("Unmarshaled body of /api/commands event %v for job %v", eventType, eventJobname)

		err := h.buildService.UpdateBuildStatus(c.Request.Context(), ciBuilderEvent)
		if err != nil {
			errorMessage := fmt.Sprintf("Failed updating build status for job %v to %v, not removing the job", eventJobname, ciBuilderEvent.BuildStatus)
			log.Error().Err(err).Interface("ciBuilderEvent", ciBuilderEvent).Msg(errorMessage)
			c.String(http.StatusInternalServerError, errorMessage)
			return
		}

	case "builder:clean":

		// unmarshal json body
		var ciBuilderEvent estafettedom.CiBuilderEvent
		err = json.Unmarshal(body, &ciBuilderEvent)
		if err != nil {
			log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to CiBuilderEvent failed")
			return
		}

		log.Debug().Interface("ciBuilderEvent", ciBuilderEvent).Msgf("Unmarshaled body of /api/commands event %v for job %v", eventType, eventJobname)

		if ciBuilderEvent.BuildStatus != "canceled" {
			go func(eventJobname string) {
				err = h.ciBuilderClient.RemoveCiBuilderJob(ctx, eventJobname)
				if err != nil {
					errorMessage := fmt.Sprintf("Failed removing job %v for event %v", eventJobname, eventType)
					log.Error().Err(err).Interface("ciBuilderEvent", ciBuilderEvent).Msg(errorMessage)
				}
			}(eventJobname)
		} else {
			log.Info().Msgf("Job %v is already removed by cancellation, no need to remove for event %v", eventJobname, eventType)
		}

		go func(ctx context.Context, ciBuilderEvent estafettedom.CiBuilderEvent) {
			err := h.buildService.UpdateJobResources(ctx, ciBuilderEvent)
			if err != nil {
				log.Error().Err(err).Msgf("Failed updating max cpu and memory from prometheus for pod %v", ciBuilderEvent.PodName)
			}
		}(c.Request.Context(), ciBuilderEvent)

	default:
		log.Warn().Str("event", eventType).Msgf("Unsupported Estafette event of type '%v'", eventType)
	}

	c.String(http.StatusOK, "Aye aye!")
}
