package oauth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/estafette/estafette-ci-api/config"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// NewHandler returns a new estafette.Handler
func NewHandler(config *config.APIConfig, service Service) Handler {

	return Handler{
		config:  config,
		service: service,
	}
}

type Handler struct {
	config  *config.APIConfig
	service Service
}

func (h *Handler) GetProviders(c *gin.Context) {

	providers, err := h.service.GetProviders(c.Request.Context())

	if err != nil {
		log.Error().Err(err).Msg("Retrieving oauth providers failed")
		c.String(http.StatusInternalServerError, "Retrieving oauth providers failed")
		return
	}

	responseItems := make([]interface{}, 0)
	for _, p := range providers {
		responseItems = append(responseItems, map[string]interface{}{
			"id":   p.Name,
			"name": strings.Title(p.Name),
			"path": fmt.Sprintf("/api/auth/login/%v", p.Name),
		})
	}

	c.JSON(http.StatusOK, responseItems)
}

func (h *Handler) LoginProvider(c *gin.Context) {

	provider := c.Param("provider")

	providers, err := h.service.GetProviders(c.Request.Context())

	if err != nil {
		log.Error().Err(err).Msg("Retrieving oauth providers failed")
		c.String(http.StatusInternalServerError, "Retrieving oauth providers failed")
		return
	}

	for _, p := range providers {
		if p.Name == provider {
			redirectURI := fmt.Sprintf("%vapi/auth/handle/%v", h.config.APIServer.BaseURL, p.Name)
			c.Redirect(http.StatusTemporaryRedirect, p.AuthCodeURL(redirectURI))
		}
	}

	c.JSON(http.StatusBadRequest, gin.H{"message": "Provider is not configured"})
}

func (h *Handler) HandleLoginProviderResponse(c *gin.Context) {

	provider := c.Param("provider")
	code := c.Query("code")

	providers, err := h.service.GetProviders(c.Request.Context())

	if err != nil {
		log.Error().Err(err).Msg("Retrieving oauth providers failed")
		c.String(http.StatusInternalServerError, "Retrieving oauth providers failed")
		return
	}

	for _, p := range providers {
		if p.Name == provider {
			c.JSON(http.StatusOK, gin.H{"code": code, "provider": p.Name})
			return
		}
	}

	c.JSON(http.StatusBadRequest, gin.H{"message": "Provider is not configured"})
}
