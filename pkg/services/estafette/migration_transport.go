package estafette

import (
	"net/http"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/migration"
)

func (h *Handler) Migrate(c *gin.Context) {
	if !api.RequestTokenIsValid(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusText(http.StatusUnauthorized), "message": "JWT is invalid"})
		return
	}
	var request migration.Request
	err := c.BindJSON(&request)
	if err != nil {
		errorMessage := "Binding Migrate body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}
	requestID := requestid.Get(c)
	err = h.databaseClient.QueueMigration(c.Request.Context(), request.ToTask(requestID))
	if err != nil {
		errorMessage := "Queuing migration failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": requestID})
}
