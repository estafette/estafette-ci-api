package estafette

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/database"
)

func (h *Handler) Migrate(c *gin.Context) {
	if !api.RequestTokenIsValid(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusText(http.StatusUnauthorized), "message": "JWT is invalid"})
		return
	}
	var migrate database.Migrate
	err := c.BindJSON(&migrate)
	if err != nil {
		errorMessage := "Binding Migrate body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}
	steps := []struct {
		name string
		do   func(context.Context, database.Migrate) (*database.Migrate, error)
	}{
		{"releases", h.databaseClient.MigrateReleases},
		{"releaseLogs", h.databaseClient.MigrateReleaseLogs},
		{"builds", h.databaseClient.MigrateBuilds},
		{"buildLogs", h.databaseClient.MigrateBuildLogs},
		{"buildVersions", h.databaseClient.MigrateBuildVersions},
	}
	results := make(map[string][]database.MigrationHistory)
	for _, step := range steps {
		output, err := step.do(c.Request.Context(), migrate)
		if err != nil {
			errorMessage := fmt.Sprintf("Migrating %s failed", step.name)
			log.Error().Err(err).Msg(errorMessage)
			// TODO details of failed steps/ message
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
			return
		}
		results[step.name] = output.Results
	}
	c.JSON(http.StatusOK, results)
}
