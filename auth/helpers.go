package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/helpers"
	"github.com/gin-gonic/gin"
	"github.com/sethgrid/pester"
)

var (
	// ErrInvalidSigningAlgorithm indicates signing algorithm is invalid, needs to be HS256, HS384, HS512, RS256, RS384 or RS512
	ErrInvalidSigningAlgorithm = errors.New("invalid signing algorithm")
)

func GenerateJWT(config *config.APIConfig, validDuration time.Duration, optionalClaims jwtgo.MapClaims) (tokenString string, err error) {

	// Create the token
	token := jwtgo.New(jwtgo.GetSigningMethod("HS256"))
	claims := token.Claims.(jwtgo.MapClaims)

	// set required claims
	now := time.Now()
	expire := now.Add(time.Hour)
	claims["exp"] = expire.Unix()
	claims["orig_iat"] = now.Unix()

	if optionalClaims != nil {
		for key, value := range optionalClaims {
			claims[key] = value
		}
	}

	// sign the token
	return token.SignedString([]byte(config.Auth.JWT.Key))
}

func ValidateJWT(config *config.APIConfig, tokenString string) (token *jwtgo.Token, err error) {
	return jwtgo.Parse(tokenString, func(t *jwtgo.Token) (interface{}, error) {
		if jwtgo.GetSigningMethod("HS256") != t.Method {
			return nil, ErrInvalidSigningAlgorithm
		}
		return []byte(config.Auth.JWT.Key), nil
	})
}

func GetClaimsFromJWT(config *config.APIConfig, tokenString string) (claims jwtgo.MapClaims, err error) {
	token, err := ValidateJWT(config, tokenString)
	if err != nil {
		return nil, err
	}

	claims = jwtgo.MapClaims{}
	for key, value := range token.Claims.(jwtgo.MapClaims) {
		claims[key] = value
	}

	return claims, nil
}

// getGoogleJWKs returns the list of JWKs used by google's apis from https://www.googleapis.com/oauth2/v3/certs
func getGoogleJWKs() (keysResponse *GoogleJWKResponse, err error) {

	response, err := pester.Get("https://www.googleapis.com/oauth2/v3/certs")
	if err != nil {
		return
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return keysResponse, fmt.Errorf("https://www.googleapis.com/oauth2/v3/certs responded with status code %v", response.StatusCode)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}

	// unmarshal json body
	err = json.Unmarshal(body, &keysResponse)
	if err != nil {
		return
	}

	return
}

var googleJWKs map[string]*rsa.PublicKey
var googleJWKLastFetched time.Time

// GetCachedGoogleJWK returns google's json web keys from cache or fetches them from source
func GetCachedGoogleJWK(kid string) (jwk *rsa.PublicKey, err error) {

	if googleJWKs == nil || googleJWKLastFetched.Add(time.Hour*24).Before(time.Now().UTC()) {

		jwks, err := getGoogleJWKs()
		if err != nil {
			return nil, err
		}

		// turn array into map and converto to *rsa.PublicKey
		googleJWKs = make(map[string]*rsa.PublicKey)
		for _, key := range jwks.Keys {

			n, err := base64.RawURLEncoding.DecodeString(key.N)
			if err != nil {
				return nil, err
			}

			e := 0
			// the default exponent is usually 65537, so just compare the base64 for [1,0,1] or [0,1,0,1]
			if key.E == "AQAB" || key.E == "AAEAAQ" {
				e = 65537
			} else {
				return nil, fmt.Errorf("JWK key exponent %v can't be converted to int", key.E)
			}

			publicKey := &rsa.PublicKey{
				N: new(big.Int).SetBytes(n),
				E: e,
			}

			googleJWKs[key.KeyID] = publicKey
		}

		googleJWKLastFetched = time.Now().UTC()
	}

	if val, ok := googleJWKs[kid]; ok {
		return val, err
	}

	return nil, fmt.Errorf("Key with kid %v does not exist at https://www.googleapis.com/oauth2/v3/certs", kid)
}

func isValidGoogleJWT(tokenString string) (valid bool, err error) {

	// ensure this uses UTC even though google's servers all run in UTC
	jwtgo.TimeFunc = time.Now().UTC

	// Parse takes the token string and a function for looking up the key. The latter is especially
	// useful if you use multiple keys for your application.  The standard is to use 'kid' in the
	// head of the token to identify which key to use, but the parsed token (head and claims) is provided
	// to the callback, providing flexibility.
	token, err := jwtgo.Parse(tokenString, func(token *jwtgo.Token) (interface{}, error) {

		// check algorithm is correct
		if _, ok := token.Method.(*jwtgo.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		// get public key for kid
		publicKey, err := GetCachedGoogleJWK(token.Header["kid"].(string))
		if err != nil {
			return nil, err
		}

		return publicKey, nil
	})

	if err != nil {
		return
	}

	if claims, ok := token.Claims.(jwtgo.MapClaims); ok && token.Valid {

		// "aud": "https://ci.estafette/io/api/integrations/pubsub/events",
		// "azp": "118094230988819892802",
		// "email": "estafette@estafette.iam.gserviceaccount.com",
		// "email_verified": true,
		// "exp": 1568134507,
		// "iat": 1568130907,
		// "iss": "https://accounts.google.com",
		// "sub": "118094230988819892802"

		// verify issuer
		expectedIssuer := "https://accounts.google.com"
		actualIssuer := claims["iss"].(string)
		if actualIssuer != expectedIssuer {
			return false, fmt.Errorf("Actual issuer %v is not equal to expected issuer %v", actualIssuer, expectedIssuer)
		}

		emailVerified := claims["email_verified"].(bool)
		if !emailVerified {
			return false, fmt.Errorf("Email claim is not verified")
		}

		return true, nil
	}

	return false, fmt.Errorf("Token is not valid")
}

func RequestTokenIsValid(c *gin.Context) bool {

	// ensure email claim is set
	claims := jwt.ExtractClaims(c)
	val, ok := claims[jwt.IdentityKey]
	if !ok {
		return false
	}
	identity, ok := val.(string)
	if !ok {
		return false
	}
	if identity == "" {
		return false
	}

	return true
}

func RequestTokenHasRole(c *gin.Context, role Role) bool {

	if !RequestTokenIsValid(c) {
		return false
	}

	// ensure role is present
	claims := jwt.ExtractClaims(c)
	val, ok := claims["roles"]
	if !ok {
		return false
	}

	roles, ok := val.([]interface{})
	if !ok {
		return false
	}

	for _, r := range roles {
		if rval, ok := r.(string); ok && rval == role.String() {
			return true
		}
	}

	return false
}

// RequestTokenHasSomeRole checks whether the request has at least one of a list of roles
func RequestTokenHasSomeRole(c *gin.Context, roles ...Role) bool {

	if len(roles) == 0 {
		return false
	}

	for _, role := range roles {
		if RequestTokenHasRole(c, role) {
			return true
		}
	}

	return false
}

func GetRolesFromRequest(c *gin.Context) (roles []Role) {

	if !RequestTokenIsValid(c) {
		return
	}

	claims := jwt.ExtractClaims(c)
	val, ok := claims["roles"]
	if !ok {
		return
	}

	rolesFromClaim, ok := val.([]interface{})
	if !ok {
		return
	}

	for _, r := range rolesFromClaim {
		if rval, ok := r.(string); ok {
			role := ToRole(rval)
			if role != nil {
				roles = append(roles, *role)
			}
		}
	}

	return
}

func GetPermissionsFromRequest(c *gin.Context) (permissions []Permission) {

	roles := GetRolesFromRequest(c)

	for _, r := range roles {
		permissions = append(permissions, rolesToPermissionMap[r]...)
	}

	return
}

func RequestTokenHasPermission(c *gin.Context, permission Permission) bool {

	permissions := GetPermissionsFromRequest(c)

	for _, p := range permissions {
		if p == permission {
			return true
		}
	}

	return false
}

func GetGroupsFromRequest(c *gin.Context) (groups []string) {

	if !RequestTokenIsValid(c) {
		return
	}

	claims := jwt.ExtractClaims(c)
	val, ok := claims["groups"]
	if !ok {
		return
	}

	groupsFromClaim, ok := val.([]interface{})
	if !ok {
		return
	}

	for _, r := range groupsFromClaim {
		if rval, ok := r.(string); ok {
			groups = append(groups, rval)
		}
	}

	return
}

func GetOrganizationsFromRequest(c *gin.Context) (organizations []string) {

	if !RequestTokenIsValid(c) {
		return
	}

	claims := jwt.ExtractClaims(c)
	val, ok := claims["organizations"]
	if !ok {
		return
	}

	organizationsFromClaim, ok := val.([]interface{})
	if !ok {
		return
	}

	for _, r := range organizationsFromClaim {
		if rval, ok := r.(string); ok {
			organizations = append(organizations, rval)
		}
	}

	return
}

// SetPermissionsFilters adds permission related filters for groups and organizations
func SetPermissionsFilters(c *gin.Context, filters map[helpers.FilterType][]string) map[helpers.FilterType][]string {

	if RequestTokenHasSomeRole(c, RoleOrganizationPipelinesViewer, RoleOrganizationPipelinesOperator) {
		filters[helpers.FilterOrganizations] = GetOrganizationsFromRequest(c)
	} else if RequestTokenHasSomeRole(c, RoleGroupPipelinesViewer, RoleGroupPipelinesOperator) {
		filters[helpers.FilterGroups] = GetGroupsFromRequest(c)
	}

	return filters
}
