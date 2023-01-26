package api

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/gin-gonic/gin"
	jwtgo "github.com/golang-jwt/jwt/v4"
	"github.com/sethgrid/pester"
)

var (
	// ErrInvalidSigningAlgorithm indicates signing algorithm is invalid, needs to be HS256, HS384, HS512, RS256, RS384 or RS512
	ErrInvalidSigningAlgorithm = errors.New("invalid signing algorithm")
)

func GenerateJWT(config *APIConfig, now time.Time, expiry time.Time, optionalClaims jwtgo.MapClaims) (tokenString string, err error) {

	// Create the token
	token := jwtgo.New(jwtgo.SigningMethodHS256)
	claims := token.Claims.(jwtgo.MapClaims)

	// set required claims
	claims["exp"] = expiry.Unix()
	claims["orig_iat"] = now.Unix()

	for key, value := range optionalClaims {
		claims[key] = value
	}

	// sign the token
	return token.SignedString([]byte(config.Auth.JWT.Key))
}

func ValidateJWT(config *APIConfig, tokenString string) (token *jwtgo.Token, err error) {
	return jwtgo.Parse(tokenString, func(t *jwtgo.Token) (interface{}, error) {
		if jwtgo.SigningMethodHS256 != t.Method {
			return nil, ErrInvalidSigningAlgorithm
		}
		return []byte(config.Auth.JWT.Key), nil
	})
}

func GetClaimsFromJWT(config *APIConfig, tokenString string) (claims jwtgo.MapClaims, err error) {
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

	body, err := io.ReadAll(response.Body)
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
func SetPermissionsFilters(c *gin.Context, filters map[FilterType][]string) map[FilterType][]string {

	// filter out archived pipelines
	filters[FilterArchived] = GetGenericFilter(c, FilterArchived, "false")
	filters[FilterOrganizations] = GetGenericFilter(c, FilterOrganizations)
	filters[FilterGroups] = GetGenericFilter(c, FilterGroups)

	if RequestTokenHasRole(c, RoleAdministrator) {
		// admin can see all pipelines for all orgs and groups
		return filters
	}

	// filter pipelines on organizations / groups of request
	requestOrganizations := GetOrganizationsFromRequest(c)
	requestGroups := GetGroupsFromRequest(c)

	if len(requestOrganizations) > 0 {
		filters[FilterOrganizations] = requestOrganizations
	} else if len(requestGroups) > 0 {
		filters[FilterGroups] = requestGroups
	}

	return filters
}

// StringArrayContains returns true of a value is present in the array
func StringArrayContains(array []string, value string) bool {
	for _, v := range array {
		if v == value {
			return true
		}
	}
	return false
}

// GetQueryParameters extracts query parameters specified according to https://jsonapi.org/format/
func GetQueryParameters(c *gin.Context) (int, int, map[FilterType][]string, []OrderField) {
	return GetPageNumber(c), GetPageSize(c), GetFilters(c), GetSorting(c)
}

// GetPageNumber extracts pagination parameters specified according to https://jsonapi.org/format/
func GetPageNumber(c *gin.Context) int {
	// get page number query string value or default to 1
	pageNumberValue := c.DefaultQuery("page[number]", "1")
	pageNumber, err := strconv.Atoi(pageNumberValue)
	if err != nil {
		pageNumber = 1
	}

	return pageNumber
}

// GetPageSize extracts pagination parameters specified according to https://jsonapi.org/format/
func GetPageSize(c *gin.Context) int {
	// get page number query string value or default to 20 (maximize at 100)
	pageSizeValue := c.DefaultQuery("page[size]", "20")
	pageSize, err := strconv.Atoi(pageSizeValue)
	if err != nil {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return pageSize
}

// GetSorting extracts sorting parameters specified according to https://jsonapi.org/format/
func GetSorting(c *gin.Context) (sorting []OrderField) {
	// ?sort=-created,title
	sortValue := c.DefaultQuery("sort", "")
	if sortValue == "" {
		return
	}

	splittedSortValues := strings.Split(sortValue, ",")
	for _, sv := range splittedSortValues {
		direction := "ASC"
		if strings.HasPrefix(sv, "-") {
			direction = "DESC"
		}
		sorting = append(sorting, OrderField{
			FieldName: strings.TrimPrefix(sv, "-"),
			Direction: direction,
		})
	}

	return
}

// GetFilters extracts specific filter parameters specified according to https://jsonapi.org/format/
func GetFilters(c *gin.Context) map[FilterType][]string {
	// get filters (?filter[status]=running,succeeded&filter[since]=1w&filter[labels]=team%3Destafette-team)
	filters := map[FilterType][]string{}
	filters[FilterStatus] = GetStatusFilter(c)
	filters[FilterSince] = GetSinceFilter(c)
	filters[FilterLabels] = GetLabelsFilter(c)
	filters[FilterReleaseTarget] = GetGenericFilter(c, FilterReleaseTarget)
	filters[FilterSearch] = GetGenericFilter(c, FilterSearch)
	filters[FilterRecentCommitter] = GetGenericFilter(c, FilterRecentCommitter)
	filters[FilterRecentReleaser] = GetGenericFilter(c, FilterRecentReleaser)
	filters[FilterGroupID] = GetGenericFilter(c, FilterGroupID)
	filters[FilterOrganizationID] = GetGenericFilter(c, FilterOrganizationID)
	filters[FilterPipeline] = GetGenericFilter(c, FilterPipeline)
	filters[FilterParent] = GetGenericFilter(c, FilterParent)
	filters[FilterEntity] = GetGenericFilter(c, FilterEntity)
	filters[FilterBranch] = GetGenericFilter(c, FilterBranch)
	filters[FilterBotName] = GetGenericFilter(c, FilterBotName)

	return filters
}

func GetPipelineFilters(c *gin.Context) map[FilterType][]string {
	filters := map[FilterType][]string{}

	// filter on organizations / groups
	filters = SetPermissionsFilters(c, filters)

	// filter out archived pipelines
	filters[FilterArchived] = GetGenericFilter(c, FilterArchived, "false")

	return filters
}

// GetStatusFilter extracts a filter on status
func GetStatusFilter(c *gin.Context, defaultValues ...contracts.Status) []string {
	defaultValuesAsStrings := []string{}
	for _, dv := range defaultValues {
		defaultValuesAsStrings = append(defaultValuesAsStrings, string(dv))
	}

	return GetGenericFilter(c, FilterStatus, defaultValuesAsStrings...)
}

// GetLastFilter extracts a filter to select last n items
func GetLastFilter(c *gin.Context, defaultValue int) []string {
	return GetGenericFilter(c, FilterLast, strconv.Itoa(defaultValue))
}

// GetSinceFilter extracts a filter on build/release date
func GetSinceFilter(c *gin.Context) []string {
	return GetGenericFilter(c, FilterSince, "eternity")
}

// GetLabelsFilter extracts a filter to select specific labels
func GetLabelsFilter(c *gin.Context) []string {
	return GetGenericFilter(c, FilterLabels)
}

// GetGenericFilter extracts a filter
func GetGenericFilter(c *gin.Context, filterKey FilterType, defaultValues ...string) []string {

	filterValues, filterExist := c.GetQueryArray(fmt.Sprintf("filter[%v]", filterKey.String()))
	if filterExist && len(filterValues) > 0 && filterValues[0] != "" {
		return filterValues
	}

	return defaultValues
}

// GetPagedListResponse runs a paged item query and a count query in parallel and returns them as a ListResponse
func GetPagedListResponse(itemsFunc func() ([]interface{}, error), countFunc func() (int, error), pageNumber, pageSize int) (contracts.ListResponse, error) {

	type ItemsResult struct {
		items []interface{}
		err   error
	}
	type CountResult struct {
		count int
		err   error
	}

	// run 2 database queries in parallel and return their result via channels
	itemsChannel := make(chan ItemsResult)
	countChannel := make(chan CountResult)

	go func() {
		defer close(itemsChannel)
		items, err := itemsFunc()

		itemsChannel <- ItemsResult{items, err}
	}()

	go func() {
		defer close(countChannel)
		count, err := countFunc()

		countChannel <- CountResult{count, err}
	}()

	itemsResult := <-itemsChannel
	if itemsResult.err != nil {
		return contracts.ListResponse{}, itemsResult.err
	}

	countResult := <-countChannel
	if countResult.err != nil {
		return contracts.ListResponse{}, countResult.err
	}

	response := contracts.ListResponse{
		Items: itemsResult.items,
		Pagination: contracts.Pagination{
			Page:       pageNumber,
			Size:       pageSize,
			TotalItems: countResult.count,
			TotalPages: int(math.Ceil(float64(countResult.count) / float64(pageSize))),
		},
	}

	return response, nil
}
