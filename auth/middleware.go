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
	GinJWTMiddleware() (middleware *jwt.GinJWTMiddleware, err error)
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

// JWTIdentityKey is the identifying field for the logged in user
var JWTIdentityKey = "id"

func (m *authMiddlewareImpl) GinJWTMiddleware() (middleware *jwt.GinJWTMiddleware, err error) {

	return jwt.New(&jwt.GinJWTMiddleware{
		Realm:       m.config.Auth.JWT.Realm,
		Key:         []byte(m.config.Auth.JWT.Key),
		Timeout:     time.Hour,
		MaxRefresh:  time.Hour,
		IdentityKey: JWTIdentityKey,
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			if v, ok := data.(*contracts.User); ok {
				return jwt.MapClaims{
					JWTIdentityKey: v.ID,
				}
			}
			return jwt.MapClaims{}
		},
		IdentityHandler: func(c *gin.Context) interface{} {
			claims := jwt.ExtractClaims(c)
			return &contracts.User{
				ID: claims[JWTIdentityKey].(string),
			}
		},
		Authenticator: func(c *gin.Context) (interface{}, error) {
			// var loginVals login
			// if err := c.ShouldBind(&loginVals); err != nil {
			// 	return "", jwt.ErrMissingLoginValues
			// }
			// userID := loginVals.Username
			// password := loginVals.Password

			// if (userID == "admin" && password == "admin") || (userID == "test" && password == "test") {
			// 	return &User{
			// 		UserName:  userID,
			// 		LastName:  "Bo-Yi",
			// 		FirstName: "Wu",
			// 	}, nil
			// }

			return nil, jwt.ErrFailedAuthentication
		},
		Authorizator: func(data interface{}, c *gin.Context) bool {
			if v, ok := data.(*contracts.User); ok && v.Active {
				return true
			}

			return false
		},
		Unauthorized: func(c *gin.Context, code int, message string) {
			c.JSON(code, gin.H{
				"code":    code,
				"message": message,
			})
		},
		// TokenLookup is a string in the form of "<source>:<name>" that is used
		// to extract token from the request.
		// Optional. Default value "header:Authorization".
		// Possible values:
		// - "header:<name>"
		// - "query:<name>"
		// - "cookie:<name>"
		// - "param:<name>"
		TokenLookup: "header: Authorization, query: token, cookie: jwt",
		// TokenLookup: "query:token",
		// TokenLookup: "cookie:token",

		// TokenHeadName is a string in the header. Default value is "Bearer"
		TokenHeadName: "Bearer",

		// TimeFunc provides the current time. You can override it to use another time value. This is useful for testing or if your server uses a different time zone than your tokens.
		TimeFunc: time.Now,
	})
}
