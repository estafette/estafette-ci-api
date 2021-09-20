package estafette

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/builderapi"
	"github.com/estafette/estafette-ci-api/pkg/clients/cloudstorage"
	"github.com/estafette/estafette-ci-api/pkg/clients/database"
	contracts "github.com/estafette/estafette-ci-contracts"
	crypt "github.com/estafette/estafette-ci-crypt"
	"github.com/gin-gonic/gin"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestMarshal(t *testing.T) {
	t.Run("ReturnsUpdatedConfigAfterReload", func(t *testing.T) {

		emailFilter := struct {
			Identities []struct {
				Email string `json:"email"`
			} `json:"identities"`
		}{
			[]struct {
				Email string `json:"email"`
			}{
				{
					Email: "someone@server.com",
				},
			},
		}

		// act
		bytes, err := json.Marshal(emailFilter)

		assert.Nil(t, err)
		assert.Equal(t, "{\"identities\":[{\"email\":\"someone@server.com\"}]}", string(bytes))
	})
}

func TestSql(t *testing.T) {
	t.Run("ReturnsUpdatedConfigAfterReload", func(t *testing.T) {

		emailFilter := struct {
			Identities []struct {
				Email string `json:"email"`
			} `json:"identities"`
		}{
			[]struct {
				Email string `json:"email"`
			}{
				{
					Email: "someone@server.com",
				},
			},
		}

		emailFilterBytes, err := json.Marshal(emailFilter)
		assert.Nil(t, err)

		psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

		query := psql.
			Select("a.id, a.user_data, a.inserted_at").
			From("users a").
			Where("a.user_data @> ?", string(emailFilterBytes)).
			Limit(uint64(1))

		// act
		sql, params, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "{\"identities\":[{\"email\":\"someone@server.com\"}]}", string(emailFilterBytes))
		assert.Equal(t, "SELECT a.id, a.user_data, a.inserted_at FROM users a WHERE a.user_data @> $1 LIMIT 1", sql)
		assert.Equal(t, []interface{}([]interface{}{"{\"identities\":[{\"email\":\"someone@server.com\"}]}"}), params)
	})
}

func TestGetCatalogFilters(t *testing.T) {

	t.Run("ReturnsUpdatedConfigAfterReload", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		templatesPath := "/templates"
		cfg := &api.APIConfig{
			Catalog: &api.CatalogConfig{
				Filters: []string{
					"type",
				},
			},
		}
		encryptedConfig := cfg

		databaseClient := database.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		buildService := NewMockService(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		warningHelper := api.NewWarningHelper(secretHelper)

		handler := NewHandler(templatesPath, cfg, encryptedConfig, databaseClient, cloudStorageClient, builderapiClient, buildService, warningHelper, secretHelper)
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)

		// act
		handler.GetCatalogFilters(c)

		assert.Equal(t, http.StatusOK, recorder.Result().StatusCode)
		body, err := ioutil.ReadAll(recorder.Result().Body)
		assert.Nil(t, err)
		assert.Equal(t, "[\"type\"]", string(body))

		// act
		*cfg = api.APIConfig{
			Catalog: &api.CatalogConfig{
				Filters: []string{
					"type",
					"language",
				},
			},
		}
		recorder = httptest.NewRecorder()
		c, _ = gin.CreateTestContext(recorder)
		handler.GetCatalogFilters(c)

		assert.Equal(t, http.StatusOK, recorder.Result().StatusCode)
		body, err = ioutil.ReadAll(recorder.Result().Body)
		assert.Nil(t, err)
		assert.Equal(t, "[\"type\",\"language\"]", string(body))

	})
}

func TestGetPipeline(t *testing.T) {

	t.Run("ReturnsPipelineFromNewdatabaseClientAfterReload", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		templatesPath := "/templates"
		cfg := &api.APIConfig{}
		encryptedConfig := cfg

		databaseClient := database.NewMockClient(ctrl)
		databaseClient.
			EXPECT().
			GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string, optimized bool) (pipeline *contracts.Pipeline, err error) {
				pipeline = &contracts.Pipeline{
					BuildStatus: contracts.StatusSucceeded,
				}
				return
			})
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)
		buildService := NewMockService(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		warningHelper := api.NewWarningHelper(secretHelper)

		handler := NewHandler(templatesPath, cfg, encryptedConfig, databaseClient, cloudStorageClient, builderapiClient, buildService, warningHelper, secretHelper)
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		bodyReader := strings.NewReader("")
		c.Request = httptest.NewRequest("GET", "https://ci.estafette.io/pipelines/a/b/c", bodyReader)
		if !assert.NotNil(t, c.Request) {
			return
		}

		// act
		handler.GetPipeline(c)

		assert.Equal(t, http.StatusOK, recorder.Result().StatusCode)
		body, err := ioutil.ReadAll(recorder.Result().Body)
		assert.Nil(t, err)
		assert.Equal(t, "{\"id\":\"\",\"repoSource\":\"\",\"repoOwner\":\"\",\"repoName\":\"\",\"repoBranch\":\"\",\"repoRevision\":\"\",\"buildStatus\":\"succeeded\",\"insertedAt\":\"0001-01-01T00:00:00Z\",\"updatedAt\":\"0001-01-01T00:00:00Z\",\"duration\":0,\"lastUpdatedAt\":\"0001-01-01T00:00:00Z\"}", string(body))

		// act
		databaseClient.
			EXPECT().
			GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string, optimized bool) (pipeline *contracts.Pipeline, err error) {
				pipeline = &contracts.Pipeline{
					BuildStatus: contracts.StatusFailed,
				}
				return
			})

		recorder = httptest.NewRecorder()
		c, _ = gin.CreateTestContext(recorder)
		bodyReader = strings.NewReader("")
		c.Request = httptest.NewRequest("GET", "https://ci.estafette.io/pipelines/a/b/c", bodyReader)
		handler.GetPipeline(c)

		assert.Equal(t, http.StatusOK, recorder.Result().StatusCode)
		_, err = ioutil.ReadAll(recorder.Result().Body)
		assert.Nil(t, err)
		// assert.Equal(t, "{\"id\":\"\",\"repoSource\":\"\",\"repoOwner\":\"\",\"repoName\":\"\",\"repoBranch\":\"\",\"repoRevision\":\"\",\"buildStatus\":\"failed\",\"insertedAt\":\"0001-01-01T00:00:00Z\",\"updatedAt\":\"0001-01-01T00:00:00Z\",\"duration\":0,\"lastUpdatedAt\":\"0001-01-01T00:00:00Z\"}", string(body))
	})
}

func TestGetManifestTemplates(t *testing.T) {
	t.Run("ReturnsTemplatesCorrectly", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		content := []byte("track: stable")
		templatesPath, err := ioutil.TempDir("", "templates")
		assert.Nil(t, err)

		defer os.RemoveAll(templatesPath) // clean up

		tmpfn := filepath.Join(templatesPath, "manifest-docker.tmpl")
		err = ioutil.WriteFile(tmpfn, content, 0666)
		assert.Nil(t, err)

		cfg := &api.APIConfig{}
		encryptedConfig := cfg

		databaseClient := database.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)
		buildService := NewMockService(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		warningHelper := api.NewWarningHelper(secretHelper)

		handler := NewHandler(templatesPath, cfg, encryptedConfig, databaseClient, cloudStorageClient, builderapiClient, buildService, warningHelper, secretHelper)
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		bodyReader := strings.NewReader("")
		c.Request = httptest.NewRequest("GET", "https://ci.estafette.io/manifest/templates", bodyReader)
		if !assert.NotNil(t, c.Request) {
			return
		}

		// act
		handler.GetManifestTemplates(c)

		assert.Equal(t, http.StatusOK, recorder.Result().StatusCode)
		body, err := ioutil.ReadAll(recorder.Result().Body)
		assert.Nil(t, err)
		assert.Equal(t, "{\"templates\":[{\"placeholders\":[],\"template\":\"docker\"}]}", string(body))

	})
}

func TestGenerateManifest(t *testing.T) {
	t.Run("ReturnsManifestFromTemplate", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		content := []byte("track: stable\nteam: {{.TeamName}}")
		templatesPath, err := ioutil.TempDir("", "templates")
		assert.Nil(t, err)

		defer os.RemoveAll(templatesPath) // clean up

		tmpfn := filepath.Join(templatesPath, "manifest-docker.tmpl")
		err = ioutil.WriteFile(tmpfn, content, 0666)
		assert.Nil(t, err)

		cfg := &api.APIConfig{}
		encryptedConfig := cfg

		databaseClient := database.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)
		buildService := NewMockService(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		warningHelper := api.NewWarningHelper(secretHelper)

		handler := NewHandler(templatesPath, cfg, encryptedConfig, databaseClient, cloudStorageClient, builderapiClient, buildService, warningHelper, secretHelper)
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		bodyReader := strings.NewReader("{\"template\": \"docker\", \"placeholders\": {\"TeamName\": \"estafette\"}}")
		c.Request = httptest.NewRequest("POST", "https://ci.estafette.io/manifest/generate", bodyReader)
		if !assert.NotNil(t, c.Request) {
			return
		}

		// act
		handler.GenerateManifest(c)

		assert.Equal(t, http.StatusOK, recorder.Result().StatusCode)
		body, err := ioutil.ReadAll(recorder.Result().Body)
		assert.Nil(t, err)
		assert.Equal(t, "{\"manifest\":\"track: stable\\nteam: estafette\"}", string(body))

	})
}
