package auth

import (
	"fmt"
	"net/http"

	"github.com/estafette/estafette-ci-api/config"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Middleware handles authentication for routes requiring authentication
type Middleware interface {
	MiddlewareFunc() gin.HandlerFunc
	APIKeyMiddlewareFunc() gin.HandlerFunc
}

type authMiddlewareImpl struct {
	config config.AuthConfig
}

// NewAuthMiddleware returns a new auth.AuthMiddleware
func NewAuthMiddleware(config config.AuthConfig) (authMiddleware Middleware) {

	authMiddleware = &authMiddlewareImpl{
		config: config,
	}

	return
}

func (m *authMiddlewareImpl) MiddlewareFunc() gin.HandlerFunc {
	return func(c *gin.Context) {

		// if no form of authentication is enabled return 401
		if !m.config.IAP.Enable {
			c.Status(http.StatusUnauthorized)
			return
		}

		if m.config.IAP.Enable {

			tokenString := c.Request.Header.Get("x-goog-iap-jwt-assertion")
			user, err := GetUserFromIAPJWT(tokenString, m.config.IAP.Audience)
			if err != nil {
				log.Warn().Str("jwt", tokenString).Err(err).Msg("Checking iap jwt failed")
				c.Status(http.StatusUnauthorized)
				return
			}

			// set user to access from request handlers; retrieve with `user := c.MustGet(gin.AuthUserKey).(auth.User)`
			c.Set(gin.AuthUserKey, user)
		}
	}
}

func (m *authMiddlewareImpl) APIKeyMiddlewareFunc() gin.HandlerFunc {
	return func(c *gin.Context) {

		authorizationHeader := c.GetHeader("Authorization")
		if authorizationHeader != fmt.Sprintf("Bearer %v", m.config.APIKey) {
			log.Error().
				Str("authorizationHeader", authorizationHeader).
				Msg("Authorization header bearer token is incorrect")
			c.Status(http.StatusUnauthorized)
			return
		}

		// set 'user' to enforce a handler method to require api key auth with `user := c.MustGet(gin.AuthUserKey).(string)` and ensuring the user equals 'apiKey'
		c.Set(gin.AuthUserKey, "apiKey")
	}
}
