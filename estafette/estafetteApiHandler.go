package estafette

import (
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/estafette/estafette-ci-api/cockroach"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-contracts"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// APIHandler handles all api calls
type APIHandler interface {
	GetPipelines(*gin.Context)
	GetPipeline(*gin.Context)
	GetPipelineBuilds(*gin.Context)
	GetPipelineBuild(*gin.Context)
	GetPipelineBuildLogs(*gin.Context)
	PostPipelineBuildLogs(*gin.Context)
}

type apiHandlerImpl struct {
	config            config.APIServerConfig
	cockroachDBClient cockroach.DBClient
}

// NewAPIHandler returns a new estafette.APIHandler
func NewAPIHandler(config config.APIServerConfig, cockroachDBClient cockroach.DBClient) (apiHandler APIHandler) {

	apiHandler = &apiHandlerImpl{
		config:            config,
		cockroachDBClient: cockroachDBClient,
	}

	return

}

func (h *apiHandlerImpl) GetPipelines(c *gin.Context) {

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

	// get filters (?filter[post]=1,2&filter[author]=12)
	filters := map[string][]string{}
	filterStatusValues, filterStatusExist := c.GetQueryArray("filter[status]")
	if filterStatusExist && len(filterStatusValues) > 0 && filterStatusValues[0] != "" {
		filters["status"] = filterStatusValues
	}
	filterSinceValues, filterSinceExist := c.GetQueryArray("filter[since]")
	if filterSinceExist {
		filters["since"] = filterSinceValues
	} else {
		filters["since"] = []string{"eternity"}
	}
	filterLabelsValues, filterLabelsExist := c.GetQueryArray("filter[labels]")
	if filterLabelsExist {
		filters["labels"] = filterLabelsValues
	}

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
	revision := c.Param("revision")

	build, err := h.cockroachDBClient.GetPipelineBuild(source, owner, repo, revision)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving build for %v/%v/%v/%v from db", source, owner, repo, revision)
	}
	if build == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": "PAGE_NOT_FOUND", "message": "Pipeline build not found"})
		return
	}
	log.Info().Msgf("Retrieved builds for %v/%v/%v/%v", source, owner, repo, revision)

	c.JSON(http.StatusOK, build)
}

func (h *apiHandlerImpl) GetPipelineBuildLogs(c *gin.Context) {
	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	revision := c.Param("revision")

	buildLog, err := h.cockroachDBClient.GetPipelineBuildLogs(source, owner, repo, revision)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed retrieving build logs for %v/%v/%v/%v from db", source, owner, repo, revision)
	}
	if buildLog == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": "PAGE_NOT_FOUND", "message": "Pipeline build log not found"})
		return
	}
	log.Info().Msgf("Retrieved build logs for %v/%v/%v/%v", source, owner, repo, revision)

	c.JSON(http.StatusOK, buildLog)

}

func (h *apiHandlerImpl) PostPipelineBuildLogs(c *gin.Context) {

	authorizationHeader := c.GetHeader("Authorization")
	if authorizationHeader != fmt.Sprintf("Bearer %v", h.config.APIKey) {
		log.Error().
			Str("authorizationHeader", authorizationHeader).
			Msg("Authorization header for Estafette v2 logs is incorrect")
		c.String(http.StatusUnauthorized, "Authorization failed")
		return
	}

	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")
	revision := c.Param("revision")

	var buildLog contracts.BuildLog
	err := c.Bind(&buildLog)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed binding v2 logs for %v/%v/%v/%v", source, owner, repo, revision)
	}

	log.Info().Interface("buildLog", buildLog).Msgf("Binded v2 logs for for %v/%v/%v/%v", source, owner, repo, revision)

	err = h.cockroachDBClient.InsertBuildLog(buildLog)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed inserting v2 logs for %v/%v/%v/%v", source, owner, repo, revision)
	}
	log.Info().Msgf("Inserted v2 logs for %v/%v/%v/%v", source, owner, repo, revision)

	c.String(http.StatusOK, "Aye aye!")
}
