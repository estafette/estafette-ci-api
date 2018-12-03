package estafette

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/estafette/estafette-ci-api/auth"
	"github.com/estafette/estafette-ci-api/cockroach"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/gin-gonic/gin"
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

	GetPipelineStatsBuildsDurations(*gin.Context)
	GetPipelineStatsReleasesDurations(*gin.Context)
	GetPipelineWarnings(*gin.Context)

	GetStatsPipelinesCount(*gin.Context)
	GetStatsBuildsCount(*gin.Context)
	GetStatsReleasesCount(*gin.Context)

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
}

type apiHandlerImpl struct {
	configFilePath       string
	config               config.APIServerConfig
	authConfig           config.AuthConfig
	encryptedConfig      config.APIConfig
	cockroachDBClient    cockroach.DBClient
	ciBuilderClient      CiBuilderClient
	warningHelper        WarningHelper
	githubJobVarsFunc    func(string, string, string) (string, string, error)
	bitbucketJobVarsFunc func(string, string, string) (string, string, error)
}

// NewAPIHandler returns a new estafette.APIHandler
func NewAPIHandler(configFilePath string, config config.APIServerConfig, authConfig config.AuthConfig, encryptedConfig config.APIConfig, cockroachDBClient cockroach.DBClient, ciBuilderClient CiBuilderClient, warningHelper WarningHelper, githubJobVarsFunc func(string, string, string) (string, string, error), bitbucketJobVarsFunc func(string, string, string) (string, string, error)) (apiHandler APIHandler) {

	apiHandler = &apiHandlerImpl{
		configFilePath:       configFilePath,
		config:               config,
		authConfig:           authConfig,
		encryptedConfig:      encryptedConfig,
		cockroachDBClient:    cockroachDBClient,
		ciBuilderClient:      ciBuilderClient,
		warningHelper:        warningHelper,
		githubJobVarsFunc:    githubJobVarsFunc,
		bitbucketJobVarsFunc: bitbucketJobVarsFunc,
	}

	return

}

func (h *apiHandlerImpl) GetPipelines(c *gin.Context) {

	// get page number query string value or default to 1
	pageNumberValue := c.DefaultQuery("page[number]", "1")
	pageNumber, err := strconv.Atoi(pageNumberValue)
	if err != nil {
		pageNumber = 1
	}

	// get page number query string value or default to 20 (maximize at 100)
	pageSizeValue := c.DefaultQuery("page[size]", "20")
	pageSize, err := strconv.Atoi(pageSizeValue)
	if err != nil {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// get filters (?filter[status]=running,succeeded&filter[since]=1w&filter[labels]=team%3Destafette-team)
	filters := map[string][]string{}
	filters["status"] = h.getStatusFilter(c)
	filters["since"] = h.getSinceFilter(c)
	filters["labels"] = h.getLabelsFilter(c)

	pipelines, err := h.cockroachDBClient.GetPipelines(pageNumber, pageSize, filters, true)
	if err != nil {
		log.Error().Err(err).
			Msg("Failed retrieving pipelines from db")
	}

	pipelinesCount, err := h.cockroachDBClient.GetPipelinesCount(filters)
	if err != nil {
		log.Error().Err(err).
			Msg("Failed retrieving pipelines count from db")
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

func (h *apiHandlerImpl) GetPipeline(c *gin.Context) {
	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	pipeline, err := h.cockroachDBClient.GetPipeline(source, owner, repo, true)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving pipeline for %v/%v/%v from db", source, owner, repo)
	}
	if pipeline == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline not found"})
	}

	c.JSON(http.StatusOK, pipeline)
}

func (h *apiHandlerImpl) GetPipelineBuilds(c *gin.Context) {
	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	// get page number query string value or default to 1
	pageNumberValue, pageNumberExists := c.GetQuery("page[number]")
	pageNumber, err := strconv.Atoi(pageNumberValue)
	if !pageNumberExists || err != nil {
		pageNumber = 1
	}

	// get page number query string value or default to 20 (maximize at 100)
	pageSizeValue, pageSizeExists := c.GetQuery("page[size]")
	pageSize, err := strconv.Atoi(pageSizeValue)
	if !pageSizeExists || err != nil {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// get filters (?filter[status]=running,succeeded&filter[since]=1w&filter[labels]=team%3Destafette-team)
	filters := map[string][]string{}
	filters["status"] = h.getStatusFilter(c)
	filters["since"] = h.getSinceFilter(c)
	filters["labels"] = h.getLabelsFilter(c)

	builds, err := h.cockroachDBClient.GetPipelineBuilds(source, owner, repo, pageNumber, pageSize, filters, true)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving builds for %v/%v/%v from db", source, owner, repo)
	}

	buildsCount, err := h.cockroachDBClient.GetPipelineBuildsCount(source, owner, repo, filters)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving builds count for %v/%v/%v from db", source, owner, repo)
	}

	response := contracts.ListResponse{
		Pagination: contracts.Pagination{
			Page:       pageNumber,
			Size:       pageSize,
			TotalItems: buildsCount,
			TotalPages: int(math.Ceil(float64(buildsCount) / float64(pageSize))),
		},
	}

	response.Items = make([]interface{}, len(builds))
	for i := range builds {
		response.Items[i] = builds[i]
	}

	c.JSON(http.StatusOK, response)
}

func (h *apiHandlerImpl) GetPipelineBuild(c *gin.Context) {
	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	revisionOrID := c.Param("revisionOrId")

	if len(revisionOrID) == 40 {
		build, err := h.cockroachDBClient.GetPipelineBuild(source, owner, repo, revisionOrID, false)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, revisionOrID)
		}
		if build == nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline not found"})
		}

		c.JSON(http.StatusOK, build)
		return
	}

	id, err := strconv.Atoi(revisionOrID)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed reading id from path parameter for %v/%v/%v/builds/%v", source, owner, repo, revisionOrID)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": "Path parameter id is not of type integer"})
	}

	build, err := h.cockroachDBClient.GetPipelineBuildByID(source, owner, repo, id, false)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, id)
	}
	if build == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline build not found"})
	}

	c.JSON(http.StatusOK, build)
}

func (h *apiHandlerImpl) CreatePipelineBuild(c *gin.Context) {

	user := c.MustGet(gin.AuthUserKey).(auth.User)

	var buildCommand contracts.Build
	c.BindJSON(&buildCommand)

	// match source, owner, repo with values in binded release
	if buildCommand.RepoSource != c.Param("source") {
		errorMessage := fmt.Sprintf("RepoSource in path and post data do not match for pipeline %v/%v/%v for build command issued by %v", buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, user)
		log.Error().Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
	}
	if buildCommand.RepoOwner != c.Param("owner") {
		errorMessage := fmt.Sprintf("RepoOwner in path and post data do not match for pipeline %v/%v/%v for build command issued by %v", buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, user)
		log.Error().Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
	}
	if buildCommand.RepoName != c.Param("repo") {
		errorMessage := fmt.Sprintf("RepoName in path and post data do not match for pipeline %v/%v/%v for build command issued by %v", buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, user)
		log.Error().Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
	}

	// check if version exists and is valid to re-run
	builds, err := h.cockroachDBClient.GetPipelineBuildsByVersion(buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, buildCommand.BuildVersion, false)

	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving build %v/%v/%v version %v for build command issued by %v", buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, buildCommand.BuildVersion, user)
		log.Error().Err(err).Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
	}

	var failedBuild *contracts.Build
	// ensure there's no succeeded or running builds
	hasNonFailedBuilds := false
	for _, b := range builds {
		if b.BuildStatus == "failed" || b.BuildStatus == "canceled" {
			failedBuild = b
		} else {
			hasNonFailedBuilds = true
		}
	}

	if failedBuild == nil {
		errorMessage := fmt.Sprintf("No failed build %v/%v/%v version %v for build command issued by %v", buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, buildCommand.BuildVersion, user)
		log.Error().Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
	}
	if hasNonFailedBuilds {
		errorMessage := fmt.Sprintf("Version %v of pipeline %v/%v/%v has builds that are succeeded or running ; only if all builds are failed the pipeline can be re-run", buildCommand.BuildVersion, buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName)
		log.Error().Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
	}

	// store build in db
	insertedBuild, err := h.cockroachDBClient.InsertBuild(contracts.Build{
		RepoSource:     failedBuild.RepoSource,
		RepoOwner:      failedBuild.RepoOwner,
		RepoName:       failedBuild.RepoName,
		RepoBranch:     failedBuild.RepoBranch,
		RepoRevision:   failedBuild.RepoRevision,
		BuildVersion:   failedBuild.BuildVersion,
		BuildStatus:    "running",
		Labels:         failedBuild.Labels,
		ReleaseTargets: failedBuild.ReleaseTargets,
		Manifest:       failedBuild.Manifest,
		Commits:        failedBuild.Commits,
	})
	if err != nil {
		errorMessage := fmt.Sprintf("Failed inserting build into db for rebuilding version %v of repository %v/%v/%v for build command issued by %v", buildCommand.BuildVersion, buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, user)
		log.Error().Err(err).Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
	}

	buildID, err := strconv.Atoi(insertedBuild.ID)
	if err != nil {
		log.Warn().Err(err).Msgf("Failed to convert build id %v to int for build command issued by %v", insertedBuild.ID, user)
	}

	// get authenticated url
	var authenticatedRepositoryURL string
	var environmentVariableWithToken map[string]string
	var gitSource string
	switch failedBuild.RepoSource {
	case "github.com":
		var accessToken string
		accessToken, authenticatedRepositoryURL, err = h.githubJobVarsFunc(buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName)
		if err != nil {
			errorMessage := fmt.Sprintf("Retrieving access token and authenticated github url for repository %v/%v/%v failed for build command issued by %v", buildCommand.BuildVersion, buildCommand.RepoSource, buildCommand.RepoOwner, user)
			log.Error().Err(err).Msg(errorMessage)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		}
		environmentVariableWithToken = map[string]string{"ESTAFETTE_GITHUB_API_TOKEN": accessToken}
		gitSource = "github"

	case "bitbucket.org":
		var accessToken string
		accessToken, authenticatedRepositoryURL, err = h.bitbucketJobVarsFunc(buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName)
		if err != nil {
			errorMessage := fmt.Sprintf("Retrieving access token and authenticated bitbucket url for repository %v/%v/%v failed for build command issued by %v", buildCommand.BuildVersion, buildCommand.RepoSource, buildCommand.RepoOwner, user)
			log.Error().Err(err).Msg(errorMessage)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		}
		environmentVariableWithToken = map[string]string{"ESTAFETTE_BITBUCKET_API_TOKEN": accessToken}
		gitSource = "bitbucket"
	}

	manifest, err := manifest.ReadManifest(failedBuild.Manifest)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed reading manifest for rebuilding version %v of repository %v/%v/%v for build command issued by %v", buildCommand.BuildVersion, buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, user)
		log.Error().Err(err).Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
	}

	// inject steps
	manifest, err = InjectSteps(manifest, manifest.Builder.Track, gitSource)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed injecting steps into manifest for rebuilding version %v of repository %v/%v/%v for build command issued by %v", buildCommand.BuildVersion, buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, user)
		log.Error().Err(err).Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
	}

	// get autoincrement from build version
	autoincrement := 0
	if manifest.Version.SemVer != nil {

		re := regexp.MustCompile(`^[0-9]+\.[0-9]+\.([0-9]+)(-[0-9a-zA-Z-/]+)?$`)
		match := re.FindStringSubmatch(buildCommand.BuildVersion)

		if len(match) > 1 {
			autoincrement, err = strconv.Atoi(match[1])
		}
	}

	// define ci builder params
	ciBuilderParams := CiBuilderParams{
		JobType:              "build",
		RepoSource:           failedBuild.RepoSource,
		RepoOwner:            failedBuild.RepoOwner,
		RepoName:             failedBuild.RepoName,
		RepoURL:              authenticatedRepositoryURL,
		RepoBranch:           failedBuild.RepoBranch,
		RepoRevision:         failedBuild.RepoRevision,
		EnvironmentVariables: environmentVariableWithToken,
		Track:                manifest.Builder.Track,
		AutoIncrement:        autoincrement,
		VersionNumber:        failedBuild.BuildVersion,
		Manifest:             manifest,
		BuildID:              buildID,
	}

	// create ci builder job
	go func(ciBuilderParams CiBuilderParams) {
		_, err := h.ciBuilderClient.CreateCiBuilderJob(ciBuilderParams)
		if err != nil {
			log.Error().Err(err).Msgf("Failed creating rebuild job for %v/%v/%v/%v/%v version %v", ciBuilderParams.RepoSource, ciBuilderParams.RepoOwner, ciBuilderParams.RepoName, ciBuilderParams.RepoBranch, ciBuilderParams.RepoRevision, ciBuilderParams.VersionNumber)
		}
	}(ciBuilderParams)

	c.JSON(http.StatusCreated, insertedBuild)
}

func (h *apiHandlerImpl) CancelPipelineBuild(c *gin.Context) {

	user := c.MustGet(gin.AuthUserKey).(auth.User)

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	revisionOrID := c.Param("revisionOrId")

	id, err := strconv.Atoi(revisionOrID)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed reading id from path parameter for %v/%v/%v/builds/%v", source, owner, repo, revisionOrID)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": "Path parameter id is not of type integer"})
	}

	// retrieve build
	build, err := h.cockroachDBClient.GetPipelineBuildByID(source, owner, repo, id, false)
	if err != nil {
		log.Error().Err(err).Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, revisionOrID)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Retrieving pipeline build failed"})
	}
	if build == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline build not found"})
	}
	if build.BuildStatus != "running" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": fmt.Sprintf("Build with status %v cannot be canceled", build.BuildStatus)})

	}

	// this build can be canceled, set status 'canceling' and cancel the build job
	jobName := h.ciBuilderClient.GetJobName("build", build.RepoOwner, build.RepoName, build.ID)
	err = h.ciBuilderClient.CancelCiBuilderJob(jobName)
	buildStatus := "canceling"
	if err != nil {
		// job might not have created a builder yet, so set status to canceled straightaway
		buildStatus = "canceled"
	}
	build.BuildStatus = buildStatus
	err = h.cockroachDBClient.UpdateBuildStatus(build.RepoSource, build.RepoOwner, build.RepoName, id, buildStatus)
	if err != nil {
		log.Error().Err(err).Msgf("Failed updating build status for %v/%v/%v/builds/%v in db", source, owner, repo, revisionOrID)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Failed setting pipeline build status to canceling"})
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Canceled build by user %v", user.Email)})
}

func (h *apiHandlerImpl) GetPipelineBuildLogs(c *gin.Context) {
	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	revisionOrID := c.Param("revisionOrId")

	var build *contracts.Build
	var err error
	if len(revisionOrID) == 40 {
		build, err = h.cockroachDBClient.GetPipelineBuild(source, owner, repo, revisionOrID, false)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, revisionOrID)
		}
	} else {
		id, err := strconv.Atoi(revisionOrID)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed reading id from path parameter for %v/%v/%v/builds/%v", source, owner, repo, revisionOrID)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": "Path parameter id is not of type integer"})
		}

		build, err = h.cockroachDBClient.GetPipelineBuildByID(source, owner, repo, id, false)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, id)
		}
	}

	if build == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline build not found"})
	}

	buildLog, err := h.cockroachDBClient.GetPipelineBuildLogs(source, owner, repo, build.RepoBranch, build.RepoRevision, build.ID)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving build logs for %v/%v/%v/builds/%v/logs from db", source, owner, repo, revisionOrID)
	}
	if buildLog == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline build log not found"})
	}

	c.JSON(http.StatusOK, buildLog)
}

func (h *apiHandlerImpl) TailPipelineBuildLogs(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	id := c.Param("revisionOrId")

	jobName := h.ciBuilderClient.GetJobName("build", owner, repo, id)

	logChannel := make(chan contracts.TailLogLine, 50)

	go h.ciBuilderClient.TailCiBuilderJobLogs(jobName, logChannel)

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

	if c.MustGet(gin.AuthUserKey).(string) != "apiKey" {
		c.AbortWithStatus(http.StatusUnauthorized)
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
	}

	if len(revisionOrID) != 40 {
		_, err := strconv.Atoi(revisionOrID)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed reading id from path parameter for %v/%v/%v/builds/%v", source, owner, repo, revisionOrID)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": "Path parameter id is not of type integer"})
			return
		}

		buildLog.BuildID = revisionOrID
	}

	err = h.cockroachDBClient.InsertBuildLog(buildLog)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed inserting v2 logs for %v/%v/%v/%v", source, owner, repo, revisionOrID)
	}

	c.String(http.StatusOK, "Aye aye!")
}

func (h *apiHandlerImpl) GetPipelineBuildWarnings(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	revisionOrID := c.Param("revisionOrId")

	id, err := strconv.Atoi(revisionOrID)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed reading id from path parameter for %v/%v/%v/builds/%v", source, owner, repo, revisionOrID)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": "Path parameter id is not of type integer"})
	}

	build, err := h.cockroachDBClient.GetPipelineBuildByID(source, owner, repo, id, false)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, id)
	}
	if build == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline build not found"})
	}

	warnings, err := h.warningHelper.GetManifestWarnings(build.ManifestObject)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed getting warnings for %v/%v/%v/builds/%v manifest", source, owner, repo, revisionOrID)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Failed getting warnings for manifest"})
	}

	c.JSON(http.StatusOK, gin.H{"warnings": warnings})
}

func (h *apiHandlerImpl) GetPipelineReleases(c *gin.Context) {
	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	// get page number query string value or default to 1
	pageNumberValue, pageNumberExists := c.GetQuery("page[number]")
	pageNumber, err := strconv.Atoi(pageNumberValue)
	if !pageNumberExists || err != nil {
		pageNumber = 1
	}

	// get page number query string value or default to 20 (maximize at 100)
	pageSizeValue, pageSizeExists := c.GetQuery("page[size]")
	pageSize, err := strconv.Atoi(pageSizeValue)
	if !pageSizeExists || err != nil {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// get filters (?filter[status]=running,succeeded&filter[since]=1w&filter[labels]=team%3Destafette-team)
	filters := map[string][]string{}
	filters["status"] = h.getStatusFilter(c)
	filters["since"] = h.getSinceFilter(c)
	filters["labels"] = h.getLabelsFilter(c)

	releases, err := h.cockroachDBClient.GetPipelineReleases(source, owner, repo, pageNumber, pageSize, filters)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving releases for %v/%v/%v from db", source, owner, repo)
	}

	releasesCount, err := h.cockroachDBClient.GetPipelineReleasesCount(source, owner, repo, filters)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving releases count for %v/%v/%v from db", source, owner, repo)
	}

	response := contracts.ListResponse{
		Pagination: contracts.Pagination{
			Page:       pageNumber,
			Size:       pageSize,
			TotalItems: releasesCount,
			TotalPages: int(math.Ceil(float64(releasesCount) / float64(pageSize))),
		},
	}

	response.Items = make([]interface{}, len(releases))
	for i := range releases {
		response.Items[i] = releases[i]
	}

	c.JSON(http.StatusOK, response)
}

func (h *apiHandlerImpl) CreatePipelineRelease(c *gin.Context) {

	user := c.MustGet(gin.AuthUserKey).(auth.User)

	var releaseCommand contracts.Release
	c.BindJSON(&releaseCommand)

	// match source, owner, repo with values in binded release
	if releaseCommand.RepoSource != c.Param("source") {
		errorMessage := fmt.Sprintf("RepoSource in path and post data do not match for pipeline %v/%v/%v for release command", releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName)
		log.Error().Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
	}
	if releaseCommand.RepoOwner != c.Param("owner") {
		errorMessage := fmt.Sprintf("RepoOwner in path and post data do not match for pipeline %v/%v/%v for release command", releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName)
		log.Error().Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
	}
	if releaseCommand.RepoName != c.Param("repo") {
		errorMessage := fmt.Sprintf("RepoName in path and post data do not match for pipeline %v/%v/%v for release command", releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName)
		log.Error().Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
	}

	pipeline, err := h.cockroachDBClient.GetPipeline(releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, false)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving pipeline %v/%v/%v for release command", releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName)
		log.Error().Err(err).Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
	}
	if pipeline == nil {
		errorMessage := fmt.Sprintf("No pipeline %v/%v/%v for release command", releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName)
		log.Error().Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
	}

	// check if version exists and is valid to release
	builds, err := h.cockroachDBClient.GetPipelineBuildsByVersion(releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, releaseCommand.ReleaseVersion, false)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving build %v/%v/%v version %v for release command", releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, releaseCommand.ReleaseVersion)
		log.Error().Err(err).Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
	}

	var build *contracts.Build
	// get succeeded build
	for _, b := range builds {
		if b.BuildStatus == "succeeded" {
			build = b
			break
		}
	}

	if build == nil {
		errorMessage := fmt.Sprintf("No build %v/%v/%v version %v for release command", releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, releaseCommand.ReleaseVersion)
		log.Error().Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
	}
	if build.BuildStatus != "succeeded" {
		errorMessage := fmt.Sprintf("Build %v for pipeline %v/%v/%v has status %v for release command; only succeeded pipelines are allowed to be released", releaseCommand.ReleaseVersion, releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, build.BuildStatus)
		log.Error().Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
	}

	// check if release target exists
	releaseExists := false
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

			releaseExists = true
			break
		}
	}
	if !releaseExists {
		errorMessage := fmt.Sprintf("Build %v for pipeline %v/%v/%v has no release %v for release command", releaseCommand.ReleaseVersion, releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, releaseCommand.Name)
		log.Error().Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
	}

	// check if action is defined
	if !actionExists {
		errorMessage := fmt.Sprintf("Build %v for pipeline %v/%v/%v has no action %v for release action", releaseCommand.ReleaseVersion, releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, releaseCommand.Action)
		log.Error().Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
	}

	// create release in database
	release := contracts.Release{
		Name:           releaseCommand.Name,
		Action:         releaseCommand.Action,
		RepoSource:     releaseCommand.RepoSource,
		RepoOwner:      releaseCommand.RepoOwner,
		RepoName:       releaseCommand.RepoName,
		ReleaseVersion: releaseCommand.ReleaseVersion,
		ReleaseStatus:  "running",
		TriggeredBy:    user.Email,
	}
	insertedRelease, err := h.cockroachDBClient.InsertRelease(release)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed creating release in database for build %v for pipeline %v/%v/%v and release %v for release command", releaseCommand.ReleaseVersion, releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, releaseCommand.Name)
		log.Error().Err(err).Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
	}

	// get authenticated url
	var authenticatedRepositoryURL string
	var environmentVariableWithToken map[string]string
	var gitSource string
	switch build.RepoSource {
	case "github.com":
		var accessToken string
		accessToken, authenticatedRepositoryURL, err = h.githubJobVarsFunc(releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName)
		if err != nil {
			errorMessage := fmt.Sprintf("Getting access token and authenticated github url for repository %v/%v/%v failed", releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName)
			log.Error().Err(err).Msg(errorMessage)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		}
		environmentVariableWithToken = map[string]string{"ESTAFETTE_GITHUB_API_TOKEN": accessToken}
		gitSource = "github"

	case "bitbucket.org":
		var accessToken string
		accessToken, authenticatedRepositoryURL, err = h.bitbucketJobVarsFunc(releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName)
		if err != nil {
			errorMessage := fmt.Sprintf("Getting access token and authenticated bitbucket url for repository %v/%v/%v failed", releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName)
			log.Error().Err(err).Msg(errorMessage)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		}
		environmentVariableWithToken = map[string]string{"ESTAFETTE_BITBUCKET_API_TOKEN": accessToken}
		gitSource = "bitbucket"
	}

	manifest, err := manifest.ReadManifest(build.Manifest)
	if err != nil {
		errorMessage := fmt.Sprintf("Reading the estafette manifest for repository %v/%v/%v build %v failed", releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, releaseCommand.ReleaseVersion)
		log.Error().Err(err).Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
	}

	// inject steps
	manifest, err = InjectSteps(manifest, manifest.Builder.Track, gitSource)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed injecting steps into manifest for %v/%v/%v version %v", releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, releaseCommand.ReleaseVersion)
		log.Error().Err(err).Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
	}

	insertedReleaseID, err := strconv.Atoi(insertedRelease.ID)
	if err != nil {
		errorMessage := fmt.Sprintf("Converting the release id to a string for repository %v/%v/%v build %v failed", releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, releaseCommand.ReleaseVersion)
		log.Error().Err(err).Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
	}

	// start release job
	ciBuilderParams := CiBuilderParams{
		JobType:              "release",
		RepoSource:           releaseCommand.RepoSource,
		RepoOwner:            releaseCommand.RepoOwner,
		RepoName:             releaseCommand.RepoName,
		RepoURL:              authenticatedRepositoryURL,
		RepoBranch:           build.RepoBranch,
		RepoRevision:         build.RepoRevision,
		EnvironmentVariables: environmentVariableWithToken,
		Track:                manifest.Builder.Track,
		VersionNumber:        releaseCommand.ReleaseVersion,
		Manifest:             manifest,
		ReleaseID:            insertedReleaseID,
		ReleaseName:          releaseCommand.Name,
		ReleaseAction:        releaseCommand.Action,
	}

	go func(ciBuilderParams CiBuilderParams) {
		_, err := h.ciBuilderClient.CreateCiBuilderJob(ciBuilderParams)
		if err != nil {
			log.Error().Err(err).Msgf("Failed creating release job for %v/%v/%v/%v/%v version %v", ciBuilderParams.RepoSource, ciBuilderParams.RepoOwner, ciBuilderParams.RepoName, ciBuilderParams.RepoBranch, ciBuilderParams.RepoRevision, ciBuilderParams.VersionNumber)
		}
	}(ciBuilderParams)

	c.JSON(http.StatusCreated, insertedRelease)
}

func (h *apiHandlerImpl) CancelPipelineRelease(c *gin.Context) {

	user := c.MustGet(gin.AuthUserKey).(auth.User)

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	idValue := c.Param("id")
	id, err := strconv.Atoi(idValue)
	if err != nil {
		log.Error().Err(err).Msgf("Failed reading id from path parameter for %v/%v/%v/%v", source, owner, repo, idValue)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": "Path parameter id is not of type integer"})
	}

	release, err := h.cockroachDBClient.GetPipelineRelease(source, owner, repo, id)
	if err != nil {
		log.Error().Err(err).Msgf("Failed retrieving release for %v/%v/%v/%v from db", source, owner, repo, id)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Retrieving pipeline release failed"})
	}
	if release == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline release not found"})
	}
	if release.ReleaseStatus != "running" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": fmt.Sprintf("Release with status %v cannot be canceled", release.ReleaseStatus)})
	}

	// this release can be canceled, set status 'canceling' and cancel the release job
	jobName := h.ciBuilderClient.GetJobName("release", release.RepoOwner, release.RepoName, release.ID)
	err = h.ciBuilderClient.CancelCiBuilderJob(jobName)
	releaseStatus := "canceling"
	if err != nil {
		// job might not have created a builder yet, so set status to canceled straightaway
		releaseStatus = "canceled"
	}
	err = h.cockroachDBClient.UpdateReleaseStatus(release.RepoSource, release.RepoOwner, release.RepoName, id, releaseStatus)
	if err != nil {
		log.Error().Err(err).Msgf("Failed updating release status for %v/%v/%v/builds/%v in db", source, owner, repo, id)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Failed setting pipeline release status to canceling"})
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Canceled release by user %v", user.Email)})
}

func (h *apiHandlerImpl) GetPipelineRelease(c *gin.Context) {
	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	idValue := c.Param("id")
	id, err := strconv.Atoi(idValue)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed reading id from path parameter for %v/%v/%v/%v", source, owner, repo, idValue)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": "Path parameter id is not of type integer"})
	}

	release, err := h.cockroachDBClient.GetPipelineRelease(source, owner, repo, id)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving release for %v/%v/%v/%v from db", source, owner, repo, id)
	}
	if release == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline release not found"})
	}

	c.JSON(http.StatusOK, release)
}

func (h *apiHandlerImpl) GetPipelineReleaseLogs(c *gin.Context) {
	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	idValue := c.Param("id")
	id, err := strconv.Atoi(idValue)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed reading id from path parameter for %v/%v/%v/%v", source, owner, repo, idValue)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": "Path parameter id is not of type integer"})
	}

	releaseLog, err := h.cockroachDBClient.GetPipelineReleaseLogs(source, owner, repo, id)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving release logs for %v/%v/%v/%v from db", source, owner, repo, id)
	}
	if releaseLog == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline release log not found"})
	}

	c.JSON(http.StatusOK, releaseLog)
}

func (h *apiHandlerImpl) TailPipelineReleaseLogs(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	id := c.Param("id")

	jobName := h.ciBuilderClient.GetJobName("release", owner, repo, id)

	logChannel := make(chan contracts.TailLogLine, 50)

	go h.ciBuilderClient.TailCiBuilderJobLogs(jobName, logChannel)

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

	if c.MustGet(gin.AuthUserKey).(string) != "apiKey" {
		c.AbortWithStatus(http.StatusUnauthorized)
	}

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	idValue := c.Param("id")
	id, err := strconv.Atoi(idValue)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed reading id from path parameter for %v/%v/%v/%v", source, owner, repo, idValue)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": "Path parameter id is not of type integer"})
	}

	var releaseLog contracts.ReleaseLog
	err = c.Bind(&releaseLog)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed binding release logs for %v/%v/%v/%v", source, owner, repo, id)
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_SERVER_ERROR", "message": "Failed binding release logs from body"})
		return
	}

	err = h.cockroachDBClient.InsertReleaseLog(releaseLog)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed inserting release logs for %v/%v/%v/%v", source, owner, repo, id)
	}

	c.String(http.StatusOK, "Aye aye!")
}

func (h *apiHandlerImpl) GetPipelineStatsBuildsDurations(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	// get filters (?filter[last]=100)
	filters := map[string][]string{}
	filters["last"] = h.getLastFilter(c, 100)

	durations, err := h.cockroachDBClient.GetPipelineBuildsDurations(source, owner, repo, filters)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving build durations from db for %v/%v/%v", source, owner, repo)
		log.Error().Err(err).Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
	}

	c.JSON(http.StatusOK, gin.H{
		"durations": durations,
	})
}

func (h *apiHandlerImpl) GetPipelineStatsReleasesDurations(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	// get filters (?filter[last]=100)
	filters := map[string][]string{}
	filters["last"] = h.getLastFilter(c, 100)

	durations, err := h.cockroachDBClient.GetPipelineReleasesDurations(source, owner, repo, filters)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving releases durations from db for %v/%v/%v", source, owner, repo)
		log.Error().Err(err).Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
	}

	c.JSON(http.StatusOK, gin.H{
		"durations": durations,
	})
}

func (h *apiHandlerImpl) GetPipelineWarnings(c *gin.Context) {

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	// get filters (?filter[last]=100)
	filters := map[string][]string{}
	filters["last"] = h.getLastFilter(c, 25)

	pipeline, err := h.cockroachDBClient.GetPipeline(source, owner, repo, false)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving pipeline for %v/%v/%v from db", source, owner, repo)
	}
	if pipeline == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Pipeline not found"})
	}

	warnings := []contracts.Warning{}

	durations, err := h.cockroachDBClient.GetPipelineBuildsDurations(source, owner, repo, filters)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving build durations from db for pipeline %v/%v/%v warnings", source, owner, repo)
		log.Error().Err(err).Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
	}

	if len(durations) > 0 {
		// pick the item at half of the length
		medianIndex := len(durations)/2 - 1
		duration := durations[medianIndex]["duration"].(time.Duration)
		durationInSeconds := duration.Seconds()

		if durationInSeconds > 300.0 {
			warnings = append(warnings, contracts.Warning{
				Status:  "danger",
				Message: fmt.Sprintf("The median build time of this pipeline is **%v**. This is too slow, please optimize your build speed by using smaller images or running less intensive steps to ensure it finishes at least within 5 minutes, but preferably within 2 minutes.", duration),
			})
		} else if durationInSeconds > 120.0 {
			warnings = append(warnings, contracts.Warning{
				Status:  "warning",
				Message: fmt.Sprintf("The median build time of this pipeline is **%v**. This is a bit too slow, please optimize your build speed by using smaller images or running less intensive steps to ensure it finishes within 2 minutes.", duration),
			})
		}
	}

	manifestWarnings, err := h.warningHelper.GetManifestWarnings(pipeline.ManifestObject)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed getting warnings for %v/%v/%v manifest", source, owner, repo)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": "Failed getting warnings for manifest"})
	}
	warnings = append(warnings, manifestWarnings...)

	c.JSON(http.StatusOK, gin.H{"warnings": warnings})
}

func getContainerImageTag(containerImage string) string {
	containerImageArray := strings.Split(containerImage, ":")
	containerImageTag := "latest"
	if len(containerImageArray) > 1 {
		containerImageTag = containerImageArray[1]
	}

	return containerImageTag
}

func (h *apiHandlerImpl) GetStatsPipelinesCount(c *gin.Context) {

	// get filters (?filter[status]=running,succeeded&filter[since]=1w
	filters := map[string][]string{}
	filters["status"] = h.getStatusFilter(c)
	filters["since"] = h.getSinceFilter(c)

	pipelinesCount, err := h.cockroachDBClient.GetPipelinesCount(filters)
	if err != nil {
		log.Error().Err(err).
			Msg("Failed retrieving pipelines count from db")
	}

	c.JSON(http.StatusOK, gin.H{
		"count": pipelinesCount,
	})
}

func (h *apiHandlerImpl) GetStatsReleasesCount(c *gin.Context) {

	// get filters (?filter[status]=running,succeeded&filter[since]=1w
	filters := map[string][]string{}
	filters["status"] = h.getStatusFilter(c)
	filters["since"] = h.getSinceFilter(c)

	releasesCount, err := h.cockroachDBClient.GetReleasesCount(filters)
	if err != nil {
		log.Error().Err(err).
			Msg("Failed retrieving releases count from db")
	}

	c.JSON(http.StatusOK, gin.H{
		"count": releasesCount,
	})
}

func (h *apiHandlerImpl) GetStatsBuildsCount(c *gin.Context) {

	// get filters (?filter[status]=running,succeeded&filter[since]=1w
	filters := map[string][]string{}
	filters["status"] = h.getStatusFilter(c)
	filters["since"] = h.getSinceFilter(c)

	buildsCount, err := h.cockroachDBClient.GetBuildsCount(filters)
	if err != nil {
		log.Error().Err(err).
			Msg("Failed retrieving builds count from db")
	}

	c.JSON(http.StatusOK, gin.H{
		"count": buildsCount,
	})
}

func (h *apiHandlerImpl) GetStatsBuildsDuration(c *gin.Context) {

	// get filters (?filter[status]=running,succeeded&filter[since]=1w
	filters := map[string][]string{}
	filters["status"] = h.getStatusFilter(c)
	filters["since"] = h.getSinceFilter(c)

	buildsDuration, err := h.cockroachDBClient.GetBuildsDuration(filters)
	if err != nil {
		log.Error().Err(err).
			Msg("Failed retrieving builds duration from db")
	}

	c.JSON(http.StatusOK, gin.H{
		"duration": buildsDuration,
	})
}

func (h *apiHandlerImpl) GetStatsBuildsAdoption(c *gin.Context) {

	buildTimes, err := h.cockroachDBClient.GetFirstBuildTimes()
	if err != nil {
		errorMessage := "Failed retrieving first build times from db"
		log.Error().Err(err).Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
	}

	c.JSON(http.StatusOK, gin.H{
		"datetimes": buildTimes,
	})
}

func (h *apiHandlerImpl) GetStatsReleasesAdoption(c *gin.Context) {
	releaseTimes, err := h.cockroachDBClient.GetFirstReleaseTimes()
	if err != nil {
		errorMessage := "Failed retrieving first release times from db"
		log.Error().Err(err).Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
	}

	c.JSON(http.StatusOK, gin.H{
		"datetimes": releaseTimes,
	})
}

func (h *apiHandlerImpl) GetLoggedInUser(c *gin.Context) {

	user := c.MustGet(gin.AuthUserKey).(auth.User)

	c.JSON(http.StatusOK, user)
}

func (h *apiHandlerImpl) UpdateComputedTables(c *gin.Context) {

	user := c.MustGet(gin.AuthUserKey).(auth.User)

	filters := map[string][]string{}
	filters["status"] = h.getStatusFilter(c)
	filters["since"] = h.getSinceFilter(c)
	filters["labels"] = h.getLabelsFilter(c)
	pipelinesCount, err := h.cockroachDBClient.GetPipelinesCount(filters)
	if err != nil {
		log.Error().Err(err).
			Msg("Failed retrieving pipelines count from db")
	}
	pageSize := 20
	totalPages := int(math.Ceil(float64(pipelinesCount) / float64(pageSize)))
	for pageNumber := 1; pageNumber <= totalPages; pageNumber++ {
		pipelines, err := h.cockroachDBClient.GetPipelines(pageNumber, pageSize, filters, false)
		if err != nil {
			log.Error().Err(err).
				Msg("Failed retrieving pipelines from db")
		}
		for _, p := range pipelines {

			h.cockroachDBClient.UpsertComputedPipeline(p.RepoSource, p.RepoOwner, p.RepoName)
			h.cockroachDBClient.UpdateComputedPipelineFirstInsertedAt(p.RepoSource, p.RepoOwner, p.RepoName)
			manifest, err := manifest.ReadManifest(p.Manifest)
			if err == nil {
				for _, r := range manifest.Releases {
					if len(r.Actions) > 0 {
						for _, a := range r.Actions {
							h.cockroachDBClient.UpsertComputedRelease(p.RepoSource, p.RepoOwner, p.RepoName, r.Name, a.Name)
							h.cockroachDBClient.UpdateComputedReleaseFirstInsertedAt(p.RepoSource, p.RepoOwner, p.RepoName, r.Name, a.Name)
						}
					} else {
						h.cockroachDBClient.UpsertComputedRelease(p.RepoSource, p.RepoOwner, p.RepoName, r.Name, "")
						h.cockroachDBClient.UpdateComputedReleaseFirstInsertedAt(p.RepoSource, p.RepoOwner, p.RepoName, r.Name, "")
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, user)
}

func (h *apiHandlerImpl) GetConfig(c *gin.Context) {

	_ = c.MustGet(gin.AuthUserKey).(auth.User)

	configBytes, err := yaml.Marshal(h.encryptedConfig)
	if err != nil {
		log.Error().Err(err).Msgf("Failed marshalling encrypted config")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
	}

	r, err := regexp.Compile(`estafette\.secret\(([a-zA-Z0-9.=_-]+)\)`)
	if err != nil {
		log.Error().Err(err).Msgf("Failed compiling regex")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
	}

	// obfuscate all secrets
	configString := r.ReplaceAllLiteralString(string(configBytes), "***")

	// add extra whitespace after each top-level item
	addWhitespaceRegex := regexp.MustCompile(`\n([a-z])`)
	configString = addWhitespaceRegex.ReplaceAllString(configString, "\n\n$1")

	c.JSON(http.StatusOK, gin.H{"config": configString})
}

func (h *apiHandlerImpl) GetConfigCredentials(c *gin.Context) {

	_ = c.MustGet(gin.AuthUserKey).(auth.User)

	configBytes, err := yaml.Marshal(h.encryptedConfig.Credentials)
	if err != nil {
		log.Error().Err(err).Msgf("Failed marshalling encrypted config")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
	}

	r, err := regexp.Compile(`estafette\.secret\(([a-zA-Z0-9.=_-]+)\)`)
	if err != nil {
		log.Error().Err(err).Msgf("Failed compiling regex")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
	}

	// obfuscate all secrets
	configString := r.ReplaceAllLiteralString(string(configBytes), "***")

	// add extra whitespace after each top-level item
	addWhitespaceRegex := regexp.MustCompile(`\n([a-z])`)
	configString = addWhitespaceRegex.ReplaceAllString(configString, "\n\n$1")

	c.JSON(http.StatusOK, gin.H{"config": configString})
}

func (h *apiHandlerImpl) GetConfigTrustedImages(c *gin.Context) {

	_ = c.MustGet(gin.AuthUserKey).(auth.User)

	configBytes, err := yaml.Marshal(h.encryptedConfig.TrustedImages)
	if err != nil {
		log.Error().Err(err).Msgf("Failed marshalling encrypted config")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
	}

	r, err := regexp.Compile(`estafette\.secret\(([a-zA-Z0-9.=_-]+)\)`)
	if err != nil {
		log.Error().Err(err).Msgf("Failed compiling regex")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
	}

	// obfuscate all secrets
	configString := r.ReplaceAllLiteralString(string(configBytes), "***")

	// add extra whitespace after each top-level item
	addWhitespaceRegex := regexp.MustCompile(`\n([a-z])`)
	configString = addWhitespaceRegex.ReplaceAllString(configString, "\n\n$1")

	c.JSON(http.StatusOK, gin.H{"config": configString})
}

func (h *apiHandlerImpl) getStatusFilter(c *gin.Context) []string {

	filterStatusValues, filterStatusExist := c.GetQueryArray("filter[status]")
	if filterStatusExist && len(filterStatusValues) > 0 && filterStatusValues[0] != "" {
		return filterStatusValues
	}

	return []string{}
}

func (h *apiHandlerImpl) GetManifestTemplates(c *gin.Context) {

	configFiles, err := ioutil.ReadDir(filepath.Dir(h.configFilePath))
	if err != nil {
		log.Error().Err(err).Msgf("Failed listing config files directory")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
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
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
			}

			placeholderRegex := regexp.MustCompile(`{{\.([a-zA-Z0-9]+)}}`)
			placeholderMatches := placeholderRegex.FindAllStringSubmatch(string(data), -1)

			// reduce and deduplicate [["{{.Application}}","Application"],["{{.Team}}","Team"],["{{.ProjectName}}","ProjectName"],["{{.ProjectName}}","ProjectName"]] to ["Application","Team","ProjectName"]
			placeholders := []string{}
			for _, m := range placeholderMatches {
				if len(m) == 2 && !stringArrayContains(placeholders, m[1]) {
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

func stringArrayContains(array []string, value string) bool {
	for _, v := range array {
		if v == value {
			return true
		}
	}
	return false
}

func (h *apiHandlerImpl) GenerateManifest(c *gin.Context) {

	var aux struct {
		Template     string            `json:"template"`
		Placeholders map[string]string `json:"placeholders,omitempty"`
	}

	err := c.BindJSON(&aux)
	if err != nil {
		log.Error().Err(err).Msg("Failed binding json body")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
	}

	templateFilePath := filepath.Join(filepath.Dir(h.configFilePath), fmt.Sprintf("manifest-%v.tmpl", aux.Template))
	data, err := ioutil.ReadFile(templateFilePath)
	if err != nil {
		log.Error().Err(err).Msgf("Failed reading template file %v", templateFilePath)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
	}

	tmpl, err := template.New(".estafette.yaml").Parse(string(data))
	if err != nil {
		log.Error().Err(err).Msgf("Failed parsing template file %v", templateFilePath)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
	}

	var renderedTemplate bytes.Buffer
	err = tmpl.Execute(&renderedTemplate, aux.Placeholders)
	if err != nil {
		log.Error().Err(err).Msgf("Failed rendering template file %v", templateFilePath)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
	}

	c.JSON(http.StatusOK, gin.H{"manifest": renderedTemplate.String()})
}

func (h *apiHandlerImpl) ValidateManifest(c *gin.Context) {

	var aux struct {
		Template string `json:"template"`
	}

	err := c.BindJSON(&aux)
	if err != nil {
		log.Error().Err(err).Msg("Failed binding json body")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
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
