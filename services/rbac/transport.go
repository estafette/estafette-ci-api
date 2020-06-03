package rbac

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/estafette/estafette-ci-api/auth"
	"github.com/estafette/estafette-ci-api/config"
	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// NewHandler returns a new rbac.Handler
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

func (h *Handler) GetLoggedInUser(c *gin.Context) {

	user := c.MustGet(gin.AuthUserKey).(auth.User)

	c.JSON(http.StatusOK, user)
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

	ctx := c.Request.Context()

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
			token, err := cfg.Exchange(ctx, code)
			if err != nil {
				log.Error().Err(err).Msg("Exchanging code for token failed")
				c.String(http.StatusInternalServerError, "Exchanging code for token failed")
				return
			}

			identity, err := p.GetUserIdentity(ctx, cfg, token)
			if err != nil || identity == nil {
				log.Error().Err(err).Msg("Retrieving user identity failed")
				c.String(http.StatusInternalServerError, "Retrieving user identity failed")
				return
			}

			user, err := h.service.GetUser(c.Request.Context(), *identity)
			if err != nil && errors.Is(err, ErrUserNotFound) {
				user, err = h.service.CreateUser(c.Request.Context(), *identity)
				if err != nil {
					log.Error().Err(err).Msg("Creating user in db failed")
					c.String(http.StatusInternalServerError, "Creating user in db failed")
					return
				}
			} else if err != nil {
				log.Error().Err(err).Msg("Retrieving user from db failed")
				c.String(http.StatusInternalServerError, "Retrieving user from db failed")
				return
			}

			if user == nil {
				log.Error().Err(err).Msg("User from db is nil")
				c.String(http.StatusInternalServerError, "User from db is nil")
				return
			}

			// update user last visit and identity
			lastVisit := time.Now().UTC()
			user.LastVisit = &lastVisit
			user.Active = true

			// update identity
			hasIdentityForProvider := false
			for _, i := range user.Identities {
				if i.Provider == p.Name {
					hasIdentityForProvider = true

					// update to the fetched identity
					*i = *identity
					break
				}
			}
			if !hasIdentityForProvider {
				// add identity
				user.Identities = append(user.Identities, identity)
			}

			go func(user contracts.User) {
				err = h.service.UpdateUser(c.Request.Context(), user)
				if err != nil {
					log.Warn().Err(err).Msg("Failed updating user in db")
				}
			}(*user)

			c.Redirect(http.StatusTemporaryRedirect, "/preferences")

			return
		}
	}

	c.JSON(http.StatusBadRequest, gin.H{"message": "Provider is not configured"})
}
