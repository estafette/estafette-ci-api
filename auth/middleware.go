package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/estafette/estafette-ci-api/config"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Middleware handles authentication for routes requiring authentication
type Middleware interface {
	APIKeyMiddlewareFunc() gin.HandlerFunc
	IAPJWTMiddlewareFunc() gin.HandlerFunc
	GoogleJWTMiddlewareFunc() gin.HandlerFunc
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

func (m *authMiddlewareImpl) IAPJWTMiddlewareFunc() gin.HandlerFunc {
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

func (m *authMiddlewareImpl) GoogleJWTMiddlewareFunc() gin.HandlerFunc {
	return func(c *gin.Context) {

		authorizationHeader := c.Request.Header.Get("Authorization")

		if !strings.HasPrefix(authorizationHeader, "Bearer ") {
			return
		}

		bearerTokenString := strings.TrimPrefix(authorizationHeader, "Bearer ")
		valid, err := isValidGoogleJWT(bearerTokenString)

		if err != nil {
			log.Warn().Err(err).Str("bearer", bearerTokenString).Msgf("Error when validating Google JWT")
			return
		}

		if !valid {
			log.Warn().Err(err).Str("bearer", bearerTokenString).Msgf("Google JWT is not valid")
			return
		}

		// set 'user' to enforce a handler method to require api key auth with `user := c.MustGet(gin.AuthUserKey).(string)` and ensuring the user equals 'apiKey'
		c.Set(gin.AuthUserKey, "google-jwt")
	}
}
