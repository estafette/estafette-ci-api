package estafette

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/database"
	"github.com/estafette/migration"
)

func (h *Handler) QueueMigration(c *gin.Context) {
	if !api.RequestTokenIsValid(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusText(http.StatusUnauthorized), "message": "JWT is invalid"})
		return
	}
	var request migration.Request
	err := c.BindJSON(&request)
	if err != nil {
		log.Error().Err(err).Msg("Binding QueueMigration body failed")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": "invalid request body"})
		return
	}
	task := &migration.Task{
		Request: request,
	}
	if request.Restart != "" {
		// if restart is not empty, set status to queued to restart/ resume migration
		task.Status = migration.StatusQueued
		// if to be restarted from a specific stage, set last step to the failed step of that stage
		if request.Restart != migration.LastStage {
			task.LastStep = request.Restart.FailedStep()
		}
	}
	var savedTask *migration.Task
	savedTask, err = h.databaseClient.QueueMigration(c.Request.Context(), task)
	if err != nil {
		if errors.Is(err, database.ErrPipelineNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": err.Error()})
			return
		}
		if errors.Is(err, database.ErrMigrationExists) {
			c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": err.Error() + ", use 'restart' parameter with 'id` to restart migration"})
			return
		}
		errorMessage := "Failed to queue migration"
		log.Error().Str("fromFQN", task.FromFQN()).Str("toFQN", task.ToFQN()).Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}
	log.Info().Str("taskID", savedTask.ID).Msg("migration task queued")
	if savedTask.UpdatedAt.Sub(savedTask.QueuedAt) > 10 {
		c.JSON(http.StatusOK, savedTask)
		return
	}
	c.JSON(http.StatusCreated, savedTask)
}

func (h *Handler) GetMigrationByID(c *gin.Context) {
	taskID := c.Param("taskID")
	if taskID == "" {
		errorMessage := "taskID path parameter is required"
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}
	task, err := h.databaseClient.GetMigrationByID(c.Request.Context(), taskID)
	if err != nil {
		errorMessage := "Failed to get migration status"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}
	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": fmt.Sprintf("migration with taskID '%s' not found", taskID)})
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *Handler) RollbackMigration(c *gin.Context) {
	taskID := c.Param("taskID")
	if taskID == "" {
		errorMessage := "taskID path parameter is required"
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}
	task, err := h.databaseClient.GetMigrationByID(c.Request.Context(), taskID)
	if err != nil {
		errorMessage := "Failed to get migration for taskID: " + taskID
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}
	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": fmt.Sprintf("migration with taskID '%s' not found", taskID)})
		return
	}
	changes, err := h.databaseClient.RollbackMigration(c.Request.Context(), task)
	if err != nil {
		errorMessage := "Failed rollback migration"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}
	if err = h.rollbackLogObjects(c.Request.Context(), task); err != nil {
		errorMessage := "Failed to rollback log objects"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}
	c.JSON(http.StatusOK, changes)
}

func (h *Handler) GetMigrationByFromRepo(c *gin.Context) {
	source, owner, name, invalid := validatePathParams(c)
	if invalid {
		return
	}
	task, err := h.databaseClient.GetMigrationByFromRepo(c.Request.Context(), source, owner, name)
	if err != nil {
		if errors.Is(err, database.ErrMigrationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": err.Error()})
			return
		}
		errorMessage := "Failed to migration"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}
	c.JSON(http.StatusOK, task)
}

func validatePathParams(c *gin.Context) (string, string, string, bool) {
	source := c.Param("source")
	owner := c.Param("owner")
	name := c.Param("name")
	if source == "" || owner == "" || name == "" {
		errorMessage := ""
		if source == "" {
			errorMessage = "source"
		}
		if owner == "" {
			if errorMessage != "" {
				errorMessage += ", "
			}
			errorMessage += "owner"
		}
		if name == "" {
			if errorMessage != "" {
				errorMessage += ", "
			}
			errorMessage += "name"
		}
		errorMessage += " path parameters are required"
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return "", "", "", true
	}
	return source, owner, name, false
}

func (h *Handler) GetAllMigrations(c *gin.Context) {
	tasks, err := h.databaseClient.GetAllMigrations(c.Request.Context())
	if err != nil {
		errorMessage := "Failed to get all migration"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}
	c.JSON(http.StatusOK, tasks)
}

func (h *Handler) GetMigratedBuild(c *gin.Context) {
	buildID := c.Param("buildID")
	if buildID == "" {
		errorMessage := "buildID path parameter is required"
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}
	build, err := h.databaseClient.GetMigratedBuild(c.Request.Context(), buildID)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get migrated build")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}
	if build == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": fmt.Sprintf("no migrated build with id '%s'", buildID)})
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

func (h *Handler) GetMigratedRelease(c *gin.Context) {
	releaseID := c.Param("releaseID")
	if releaseID == "" {
		errorMessage := "releaseID path parameter is required"
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}
	release, err := h.databaseClient.GetMigratedRelease(c.Request.Context(), releaseID)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get migrated release")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}
	if release == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": fmt.Sprintf("no migrated release with id '%s'", releaseID)})
		return
	}
	c.JSON(http.StatusOK, release)
}

func (h *Handler) SetPipelineArchival(archived bool) func(c *gin.Context) {
	return func(c *gin.Context) {
		source, owner, name, invalid := validatePathParams(c)
		if invalid {
			return
		}
		err := h.databaseClient.SetPipelineArchival(c.Request.Context(), source, owner, name, archived)
		if err != nil {
			errorMessage := "Failed to update archival status"
			log.Error().Err(err).Msg(errorMessage)
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
			return
		}
		c.JSON(http.StatusOK, gin.H{"archived": archived})
	}
}
