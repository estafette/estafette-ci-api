package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/estafette/estafette-ci-api/config"
	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Middleware handles authentication for routes requiring authentication
type Middleware interface {
	APIKeyMiddlewareFunc() gin.HandlerFunc
	IAPJWTMiddlewareFunc() gin.HandlerFunc
	GoogleJWTMiddlewareFunc() gin.HandlerFunc
	GinJWTMiddleware(authenticator func(c *gin.Context) (interface{}, error)) (middleware *jwt.GinJWTMiddleware, err error)
}

// NewAuthMiddleware returns a new auth.AuthMiddleware
func NewAuthMiddleware(config *config.APIConfig) (authMiddleware Middleware) {
	authMiddleware = &authMiddlewareImpl{
		config: config,
	}

	return
}

type authMiddlewareImpl struct {
	config *config.APIConfig
}

func (m *authMiddlewareImpl) APIKeyMiddlewareFunc() gin.HandlerFunc {
	return func(c *gin.Context) {

		authorizationHeader := c.GetHeader("Authorization")
		if authorizationHeader != fmt.Sprintf("Bearer %v", m.config.Auth.APIKey) {
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
		if !m.config.Auth.IAP.Enable {
			c.Status(http.StatusUnauthorized)
			return
		}

		if m.config.Auth.IAP.Enable {

			tokenString := c.Request.Header.Get("x-goog-iap-jwt-assertion")
			user, err := GetUserFromIAPJWT(tokenString, m.config.Auth.IAP.Audience)
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

func (m *authMiddlewareImpl) GinJWTMiddleware(authenticator func(c *gin.Context) (interface{}, error)) (middleware *jwt.GinJWTMiddleware, err error) {
	return jwt.New(&jwt.GinJWTMiddleware{
		Realm:          m.config.Auth.JWT.Domain,
		Key:            []byte(m.config.Auth.JWT.Key),
		TokenLookup:    "header:Authorization, cookie:jwt",
		Authenticator:  authenticator,
		SendCookie:     true,
		SecureCookie:   true,
		CookieHTTPOnly: true,
		CookieDomain:   m.config.Auth.JWT.Domain,
		LoginResponse: func(c *gin.Context, code int, token string, expire time.Time) {

			// see if gin context has a return url
			returnURL, exists := c.Get("returnURL")
			if exists {
				c.Redirect(http.StatusFound, returnURL.(string))
				return
			}

			// cookie is used, so token does not need to be returned via response
			c.Redirect(http.StatusFound, "/")
		},
		TimeFunc: time.Now().UTC,
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			// add user properties as claims
			if user, ok := data.(*contracts.User); ok {
				return jwt.MapClaims{
					jwt.IdentityKey: user.ID,
					"name":          user.GetName(),
					"email":         user.GetEmail(),
					"provider":      user.GetProvider(),
				}
			}
			return jwt.MapClaims{}
		},
	})
}
