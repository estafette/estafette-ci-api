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
	"golang.org/x/oauth2"
	oauth2v1 "google.golang.org/api/oauth2/v1"
	"google.golang.org/api/option"
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

	dbUser, err := h.service.GetUser(c.Request.Context(), user)
	if err == nil {
		user.User = dbUser
	} else if errors.Is(err, ErrUserNotFound) {
		dbUser, err = h.service.CreateUser(c.Request.Context(), user)
		if err == nil {
			user.User = dbUser
		}
	}

	if user.User != nil {
		lastVisit := time.Now().UTC()
		user.User.LastVisit = &lastVisit
		user.User.Active = true

		go func(user auth.User) {
			err = h.service.UpdateUser(c.Request.Context(), user)
			if err != nil {
				log.Warn().Err(err).Msg("Failed updating user in db")
			}
		}(user)
	}

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
			if err != nil {
				log.Error().Err(err).Msg("Creating oauth2 service failed")
				c.String(http.StatusInternalServerError, "Creating oauth2 service failed")
				return
			}
			userInfoPlus, err := oauth2Service.Userinfo.Get().Do()
			if err != nil {
				log.Error().Err(err).Msg("Retrieving user info failed")
				c.String(http.StatusInternalServerError, "Retrieving user info failed")
				return
			}

			user := auth.User{
				Email: userInfoPlus.Email,
				User: &contracts.User{
					Name: userInfoPlus.Name,
				},
			}

			dbUser, err := h.service.GetUser(c.Request.Context(), user)
			if err == nil {
				user.User = dbUser
			} else if errors.Is(err, ErrUserNotFound) {
				dbUser, err = h.service.CreateUser(c.Request.Context(), user)
				if err == nil {
					user.User = dbUser
				}
			}

			if user.User != nil {
				lastVisit := time.Now().UTC()
				user.User.LastVisit = &lastVisit
				user.User.Active = true

				// update identity
				hasIdentityForProvider := false
				for _, i := range user.User.Identities {
					if i.Provider == p.Name {
						hasIdentityForProvider = true
						i.Avatar = userInfoPlus.Picture
						i.Name = userInfoPlus.Name
						i.ID = userInfoPlus.Id
						break
					}
				}
				if !hasIdentityForProvider {
					// add identity
					user.User.Identities = append(user.User.Identities, &contracts.UserIdentity{
						Email:  userInfoPlus.Email,
						Name:   userInfoPlus.Name,
						ID:     userInfoPlus.Id,
						Avatar: userInfoPlus.Picture,
					})
				} else {
					log.Debug().Interface("user", user.User).Msg("Updated existing user identity")
				}

				go func(user auth.User) {
					err = h.service.UpdateUser(c.Request.Context(), user)
					if err != nil {
						log.Warn().Err(err).Msg("Failed updating user in db")
					}
				}(user)
			}

			c.Redirect(http.StatusTemporaryRedirect, "/preferences")

			// c.JSON(http.StatusOK, gin.H{"userInfo": userInfoPlus, "provider": p.Name})

			return
		}
	}

	c.JSON(http.StatusBadRequest, gin.H{"message": "Provider is not configured"})
}
