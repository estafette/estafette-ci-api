package estafette

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"

	"github.com/estafette/estafette-ci-api/auth"
	"github.com/estafette/estafette-ci-api/cockroach"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// APIHandler handles all api calls
type APIHandler interface {
	GetPipelines(*gin.Context)
	GetPipeline(*gin.Context)
	GetPipelineBuilds(*gin.Context)
	GetPipelineBuild(*gin.Context)
	CreatePipelineBuild(*gin.Context)
	GetPipelineBuildLogs(*gin.Context)
	TailPipelineBuildLogs(*gin.Context)
	PostPipelineBuildLogs(*gin.Context)
	GetPipelineReleases(*gin.Context)
	GetPipelineRelease(*gin.Context)
	CreatePipelineRelease(*gin.Context)
	GetPipelineReleaseLogs(*gin.Context)
	TailPipelineReleaseLogs(*gin.Context)
	PostPipelineReleaseLogs(*gin.Context)

	GetStatsPipelinesCount(c *gin.Context)
	GetStatsBuildsCount(c *gin.Context)
	GetStatsReleasesCount(c *gin.Context)

	GetStatsBuildsDuration(c *gin.Context)

	GetLoggedInUser(*gin.Context)
}

type apiHandlerImpl struct {
	config            config.APIServerConfig
	authConfig        config.AuthConfig
	cockroachDBClient cockroach.DBClient

	ciBuilderClient      CiBuilderClient
	githubJobVarsFunc    func(string, string, string) (string, string, error)
	bitbucketJobVarsFunc func(string, string, string) (string, string, error)
}

// NewAPIHandler returns a new estafette.APIHandler
func NewAPIHandler(config config.APIServerConfig, authConfig config.AuthConfig, cockroachDBClient cockroach.DBClient, ciBuilderClient CiBuilderClient, githubJobVarsFunc func(string, string, string) (string, string, error), bitbucketJobVarsFunc func(string, string, string) (string, string, error)) (apiHandler APIHandler) {

	apiHandler = &apiHandlerImpl{
		config:               config,
		authConfig:           authConfig,
		cockroachDBClient:    cockroachDBClient,
		ciBuilderClient:      ciBuilderClient,
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

	pipelines, err := h.cockroachDBClient.GetPipelines(pageNumber, pageSize, filters)
	if err != nil {
		log.Error().Err(err).
			Msg("Failed retrieving pipelines from db")
	}
	log.Info().Msgf("Retrieved %v pipelines", len(pipelines))

	pipelinesCount, err := h.cockroachDBClient.GetPipelinesCount(filters)
	if err != nil {
		log.Error().Err(err).
			Msg("Failed retrieving pipelines count from db")
	}
	log.Info().Msgf("Retrieved pipelines count %v", pipelinesCount)

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

	pipeline, err := h.cockroachDBClient.GetPipeline(source, owner, repo)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving pipeline for %v/%v/%v from db", source, owner, repo)
	}
	if pipeline == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": "PAGE_NOT_FOUND", "message": "Pipeline not found"})
		return
	}

	log.Info().Msgf("Retrieved pipeline for %v/%v/%v", source, owner, repo)

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

	builds, err := h.cockroachDBClient.GetPipelineBuilds(source, owner, repo, pageNumber, pageSize)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving builds for %v/%v/%v from db", source, owner, repo)
	}
	log.Info().Msgf("Retrieved %v builds for %v/%v/%v", len(builds), source, owner, repo)

	buildsCount, err := h.cockroachDBClient.GetPipelineBuildsCount(source, owner, repo)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving builds count for %v/%v/%v from db", source, owner, repo)
	}
	log.Info().Msgf("Retrieved builds count %v for %v/%v/%v", buildsCount, source, owner, repo)

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
		build, err := h.cockroachDBClient.GetPipelineBuild(source, owner, repo, revisionOrID)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, revisionOrID)
		}
		if build == nil {
			c.JSON(http.StatusNotFound, gin.H{"code": "PAGE_NOT_FOUND", "message": "Pipeline build not found"})
			return
		}
		log.Info().Msgf("Retrieved builds for %v/%v/%v/builds/%v", source, owner, repo, revisionOrID)

		c.JSON(http.StatusOK, build)
		return
	}

	id, err := strconv.Atoi(revisionOrID)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed reading id from path parameter for %v/%v/%v/builds/%v", source, owner, repo, revisionOrID)
		c.JSON(http.StatusBadRequest, gin.H{"code": "BAD_REQUEST", "message": "Path parameter id is not of type integer"})
		return
	}

	build, err := h.cockroachDBClient.GetPipelineBuildByID(source, owner, repo, id)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, id)
	}
	if build == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": "PAGE_NOT_FOUND", "message": "Pipeline build not found"})
		return
	}
	log.Info().Msgf("Retrieved builds for %v/%v/%v/builds/%v", source, owner, repo, id)

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
	builds, err := h.cockroachDBClient.GetPipelineBuildsByVersion(buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, buildCommand.BuildVersion)

	if err != nil {
		errorMessage := fmt.Sprintf("Failed retrieving build %v/%v/%v version %v for build command issued by %v", buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, buildCommand.BuildVersion, user)
		log.Error().Err(err).Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
	}

	var failedBuild *contracts.Build
	// ensure there's no succeeded or running builds
	hasNonFailedBuilds := false
	for _, b := range builds {
		if b.BuildStatus == "failed" {
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
		RepoSource:   failedBuild.RepoSource,
		RepoOwner:    failedBuild.RepoOwner,
		RepoName:     failedBuild.RepoName,
		RepoBranch:   failedBuild.RepoBranch,
		RepoRevision: failedBuild.RepoRevision,
		BuildVersion: failedBuild.BuildVersion,
		BuildStatus:  "running",
		Labels:       failedBuild.Labels,
		Releases:     failedBuild.Releases,
		Manifest:     failedBuild.Manifest,
		Commits:      failedBuild.Commits,
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
	switch failedBuild.RepoSource {
	case "github.com":
		var accessToken string
		accessToken, authenticatedRepositoryURL, err = h.githubJobVarsFunc(buildCommand.BuildVersion, buildCommand.RepoSource, buildCommand.RepoOwner)
		if err != nil {
			errorMessage := fmt.Sprintf("Retrieving access token and authenticated github url for repository %v/%v/%v failed for build command issued by %v", buildCommand.BuildVersion, buildCommand.RepoSource, buildCommand.RepoOwner, user)
			log.Error().Err(err).Msg(errorMessage)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		}
		environmentVariableWithToken = map[string]string{"ESTAFETTE_GITHUB_API_TOKEN": accessToken}

	case "bitbucket.org":
		var accessToken string
		accessToken, authenticatedRepositoryURL, err = h.bitbucketJobVarsFunc(buildCommand.BuildVersion, buildCommand.RepoSource, buildCommand.RepoOwner)
		if err != nil {
			errorMessage := fmt.Sprintf("Retrieving access token and authenticated bitbucket url for repository %v/%v/%v failed for build command issued by %v", buildCommand.BuildVersion, buildCommand.RepoSource, buildCommand.RepoOwner, user)
			log.Error().Err(err).Msg(errorMessage)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		}
		environmentVariableWithToken = map[string]string{"ESTAFETTE_BITBUCKET_API_TOKEN": accessToken}
	}

	manifest, err := manifest.ReadManifest(failedBuild.Manifest)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed reading manifest for rebuilding version %v of repository %v/%v/%v for build command issued by %v", buildCommand.BuildVersion, buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, user)
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
	go h.ciBuilderClient.CreateCiBuilderJob(ciBuilderParams)

	c.JSON(http.StatusCreated, insertedBuild)
}

func (h *apiHandlerImpl) GetPipelineBuildLogs(c *gin.Context) {
	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	revisionOrID := c.Param("revisionOrId")

	var build *contracts.Build
	var err error
	if len(revisionOrID) == 40 {
		build, err = h.cockroachDBClient.GetPipelineBuild(source, owner, repo, revisionOrID)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, revisionOrID)
		}
	} else {
		id, err := strconv.Atoi(revisionOrID)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed reading id from path parameter for %v/%v/%v/builds/%v", source, owner, repo, revisionOrID)
			c.JSON(http.StatusBadRequest, gin.H{"code": "BAD_REQUEST", "message": "Path parameter id is not of type integer"})
			return
		}

		build, err = h.cockroachDBClient.GetPipelineBuildByID(source, owner, repo, id)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed retrieving build for %v/%v/%v/builds/%v from db", source, owner, repo, id)
		}
	}

	if build == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": "PAGE_NOT_FOUND", "message": "Pipeline build not found"})
		return
	}

	buildLog, err := h.cockroachDBClient.GetPipelineBuildLogs(source, owner, repo, build.RepoBranch, build.RepoRevision, build.ID)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving build logs for %v/%v/%v/builds/%v/logs from db", source, owner, repo, revisionOrID)
	}
	if buildLog == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": "PAGE_NOT_FOUND", "message": "Pipeline build log not found"})
		return
	}
	log.Info().Msgf("Retrieved build logs for %v/%v/%v/%v", source, owner, repo, revisionOrID)

	c.JSON(http.StatusOK, buildLog)
}

func (h *apiHandlerImpl) TailPipelineBuildLogs(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	id := c.Param("revisionOrId")

	jobName := h.ciBuilderClient.GetJobName("build", owner, repo, id)

	logChannel := make(chan contracts.TailLogLine, 50)

	go h.ciBuilderClient.TailCiBuilderJobLogs(jobName, logChannel)

	c.Stream(func(w io.Writer) bool {
		if ll, ok := <-logChannel; ok {
			c.SSEvent("logLine", ll)
			return true
		}
		return false
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
			c.JSON(http.StatusBadRequest, gin.H{"code": "BAD_REQUEST", "message": "Path parameter id is not of type integer"})
			return
		}

		buildLog.BuildID = revisionOrID
	}

	log.Info().Interface("buildLog", buildLog).Msgf("Binded v2 logs for for %v/%v/%v/%v", source, owner, repo, revisionOrID)

	err = h.cockroachDBClient.InsertBuildLog(buildLog)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed inserting v2 logs for %v/%v/%v/%v", source, owner, repo, revisionOrID)
	}
	log.Info().Msgf("Inserted v2 logs for %v/%v/%v/%v", source, owner, repo, revisionOrID)

	c.String(http.StatusOK, "Aye aye!")
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

	releases, err := h.cockroachDBClient.GetPipelineReleases(source, owner, repo, pageNumber, pageSize)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving releases for %v/%v/%v from db", source, owner, repo)
	}
	log.Info().Msgf("Retrieved %v releases for %v/%v/%v", len(releases), source, owner, repo)

	releasesCount, err := h.cockroachDBClient.GetPipelineReleasesCount(source, owner, repo)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving releases count for %v/%v/%v from db", source, owner, repo)
	}
	log.Info().Msgf("Retrieved releases count %v for %v/%v/%v", releasesCount, source, owner, repo)

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

	pipeline, err := h.cockroachDBClient.GetPipeline(releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName)
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
	builds, err := h.cockroachDBClient.GetPipelineBuildsByVersion(releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, releaseCommand.ReleaseVersion)
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
	for _, release := range build.Releases {
		if release.Name == releaseCommand.Name {
			releaseExists = true
			break
		}
	}
	if !releaseExists {
		errorMessage := fmt.Sprintf("Build %v for pipeline %v/%v/%v has no release %v for release command", releaseCommand.ReleaseVersion, releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, releaseCommand.Name)
		log.Error().Msg(errorMessage)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
	}

	// create release in database
	release := contracts.Release{
		Name:           releaseCommand.Name,
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

	case "bitbucket.org":
		var accessToken string
		accessToken, authenticatedRepositoryURL, err = h.bitbucketJobVarsFunc(releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName)
		if err != nil {
			errorMessage := fmt.Sprintf("Getting access token and authenticated bitbucket url for repository %v/%v/%v failed", releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName)
			log.Error().Err(err).Msg(errorMessage)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		}
		environmentVariableWithToken = map[string]string{"ESTAFETTE_BITBUCKET_API_TOKEN": accessToken}
	}

	manifest, err := manifest.ReadManifest(build.Manifest)
	if err != nil {
		errorMessage := fmt.Sprintf("Reading the estafette manifest for repository %v/%v/%v build %v failed", releaseCommand.RepoSource, releaseCommand.RepoOwner, releaseCommand.RepoName, releaseCommand.ReleaseVersion)
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
	}

	go h.ciBuilderClient.CreateCiBuilderJob(ciBuilderParams)

	c.JSON(http.StatusCreated, insertedRelease)
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
		c.JSON(http.StatusBadRequest, gin.H{"code": "BAD_REQUEST", "message": "Path parameter id is not of type integer"})
		return
	}

	release, err := h.cockroachDBClient.GetPipelineRelease(source, owner, repo, id)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving release for %v/%v/%v/%v from db", source, owner, repo, id)
	}
	if release == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": "PAGE_NOT_FOUND", "message": "Pipeline release not found"})
		return
	}
	log.Info().Msgf("Retrieved release for %v/%v/%v/%v", source, owner, repo, id)

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
		c.JSON(http.StatusBadRequest, gin.H{"code": "BAD_REQUEST", "message": "Path parameter id is not of type integer"})
		return
	}

	releaseLog, err := h.cockroachDBClient.GetPipelineReleaseLogs(source, owner, repo, id)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving release logs for %v/%v/%v/%v from db", source, owner, repo, id)
	}
	if releaseLog == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": "PAGE_NOT_FOUND", "message": "Pipeline release log not found"})
		return
	}
	log.Info().Msgf("Retrieved release logs for %v/%v/%v/%v", source, owner, repo, id)

	c.JSON(http.StatusOK, releaseLog)
}

func (h *apiHandlerImpl) TailPipelineReleaseLogs(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	id := c.Param("id")

	jobName := h.ciBuilderClient.GetJobName("release", owner, repo, id)

	logChannel := make(chan contracts.TailLogLine, 50)

	go h.ciBuilderClient.TailCiBuilderJobLogs(jobName, logChannel)

	c.Stream(func(w io.Writer) bool {
		if ll, ok := <-logChannel; ok {
			c.SSEvent("logLine", ll)
			return true
		}
		return false
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
		c.JSON(http.StatusBadRequest, gin.H{"code": "BAD_REQUEST", "message": "Path parameter id is not of type integer"})
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

	log.Info().Interface("releaseLog", releaseLog).Msgf("Binded release logs for for %v/%v/%v/%v", source, owner, repo, id)

	err = h.cockroachDBClient.InsertReleaseLog(releaseLog)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed inserting release logs for %v/%v/%v/%v", source, owner, repo, id)
	}
	log.Info().Msgf("Inserted release logs for %v/%v/%v/%v", source, owner, repo, id)

	c.String(http.StatusOK, "Aye aye!")
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
	log.Info().Msgf("Retrieved pipelines count %v", pipelinesCount)

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
	log.Info().Msgf("Retrieved releases count %v", releasesCount)

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
	log.Info().Msgf("Retrieved builds count %v", buildsCount)

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
	log.Info().Msgf("Retrieved builds duration %v", buildsDuration)

	c.JSON(http.StatusOK, gin.H{
		"duration": buildsDuration,
	})
}

func (h *apiHandlerImpl) GetLoggedInUser(c *gin.Context) {

	user := c.MustGet(gin.AuthUserKey).(auth.User)

	c.JSON(http.StatusOK, user)
}

func (h *apiHandlerImpl) getStatusFilter(c *gin.Context) []string {

	filterStatusValues, filterStatusExist := c.GetQueryArray("filter[status]")
	if filterStatusExist && len(filterStatusValues) > 0 && filterStatusValues[0] != "" {
		return filterStatusValues
	}

	return []string{}
}

func (h *apiHandlerImpl) getSinceFilter(c *gin.Context) []string {

	filterSinceValues, filterSinceExist := c.GetQueryArray("filter[since]")
	if filterSinceExist {
		return filterSinceValues
	}

	return []string{"eternity"}
}

func (h *apiHandlerImpl) getLabelsFilter(c *gin.Context) []string {
	filterLabelsValues, filterLabelsExist := c.GetQueryArray("filter[labels]")
	if filterLabelsExist {
		return filterLabelsValues
	}

	return []string{}
}
