package oauth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/estafette/estafette-ci-api/config"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
	oauth2v1 "google.golang.org/api/oauth2/v1"
	"google.golang.org/api/option"
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
			c.Redirect(http.StatusTemporaryRedirect, p.AuthCodeURL(h.config.APIServer.BaseURL, "state"))
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

			cfg := p.GetConfig(h.config.APIServer.BaseURL)
			token, err := cfg.Exchange(oauth2.NoContext, code)
			if err != nil {
				log.Error().Err(err).Msg("Exchanging code for token failed")
				c.String(http.StatusInternalServerError, "Exchanging code for token failed")
				return
			}

			oauth2Service, err := oauth2v1.NewService(c.Request.Context(), option.WithTokenSource(cfg.TokenSource(c.Request.Context(), token)))
			userInfoPlus, err := oauth2Service.Userinfo.Get().Do()

			c.JSON(http.StatusOK, gin.H{"userInfo": userInfoPlus, "provider": p.Name})
			return
		}
	}

	c.JSON(http.StatusBadRequest, gin.H{"message": "Provider is not configured"})
}
