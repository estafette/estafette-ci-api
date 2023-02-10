package estafette

import (
	"fmt"
	"net/http"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/migration"
)

func (h *Handler) Migrate(c *gin.Context) {
	if !api.RequestTokenIsValid(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusText(http.StatusUnauthorized), "message": "JWT is invalid"})
		return
	}
	var request migration.TaskRequest
	err := c.BindJSON(&request)
	if err != nil {
		log.Error().Err(err).Msg("Binding Migrate body failed")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": "invalid request body"})
		return
	}
	task := &migration.Task{}
	if request.Restart != "" {
		// if restart is true, set status to queued to restart/ resume migration
		task.Status = migration.StatusQueued
		if request.Restart != migration.LastStage {
			task.LastStep = migration.FailedStepOf(request.Restart)
		}
	}
	if request.ID == "" {
		request.ID = requestid.Get(c)
	}
	task, err = h.databaseClient.QueueMigration(c.Request.Context(), task)
	if err != nil {
		errorMessage := "Queuing migration failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}
	if task.UpdatedAt.Sub(task.QueuedAt) > 10 {
		c.JSON(http.StatusOK, gin.H{"id": request.ID})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": request.ID})
}

func (h *Handler) GetMigrationStatus(c *gin.Context) {
	taskID := c.Param("taskID")
	if taskID == "" {
		errorMessage := "taskID path parameter is required"
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}
	task, err := h.databaseClient.GetMigrationStatus(c.Request.Context(), taskID)
	if err != nil {
		errorMessage := "Failed to get migration status"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}
	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": fmt.Sprintf("migration with taskID %s not found", taskID)})
		return
	}
	c.JSON(http.StatusFound, task)
}
