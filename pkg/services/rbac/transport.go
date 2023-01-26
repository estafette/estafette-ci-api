package rbac

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"net/http"
	"sort"
	"strings"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/bitbucketapi"
	"github.com/estafette/estafette-ci-api/pkg/clients/database"
	"github.com/estafette/estafette-ci-api/pkg/clients/githubapi"
	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/gin-gonic/gin"
	jwtgo "github.com/golang-jwt/jwt/v4"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// NewHandler returns a new rbac.Handler
func NewHandler(config *api.APIConfig, service Service, databaseClient database.Client, bitbucketapiClient bitbucketapi.Client, githubapiClient githubapi.Client) Handler {
	return Handler{
		config:             config,
		service:            service,
		databaseClient:     databaseClient,
		bitbucketapiClient: bitbucketapiClient,
		githubapiClient:    githubapiClient,
	}
}

type Handler struct {
	config             *api.APIConfig
	service            Service
	databaseClient     database.Client
	bitbucketapiClient bitbucketapi.Client
	githubapiClient    githubapi.Client
}

func (h *Handler) GetLoggedInUser(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	id := claims[jwt.IdentityKey].(string)

	ctx := c.Request.Context()

	user, err := h.databaseClient.GetUserByID(ctx, id, map[api.FilterType][]string{})
	if err != nil {
		log.Error().Err(err).Msgf("Retrieving user from db failed with id %v", id)
		c.String(http.StatusInternalServerError, "Retrieving user from db failed")
		return
	}

	c.JSON(http.StatusOK, user)
}
func (h *Handler) GetRoles(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionRolesList) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	ctx := c.Request.Context()
	roles, err := h.service.GetRoles(ctx)

	// sort
	sort.Slice(roles, func(i, j int) bool {
		return roles[i] < roles[j]
	})

	if err != nil {
		log.Error().Err(err).Msg("Retrieving roles failed")
		c.String(http.StatusInternalServerError, "Retrieving roles failed")
		return
	}

	c.JSON(http.StatusOK, roles)
}

func (h *Handler) GetProviders(c *gin.Context) {

	ctx := c.Request.Context()

	providers, err := h.service.GetProviders(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Retrieving oauth providers failed")
		c.String(http.StatusInternalServerError, "Retrieving oauth providers failed")
		return
	}

	responseItems := make([]interface{}, 0)
	for _, provider := range providers {
		responseItems = append(responseItems, map[string]interface{}{
			"id":           provider.Name,
			"organization": provider.Organization,
			"name":         cases.Title(language.Und).String(provider.Name),
			"path":         fmt.Sprintf("/api/auth/login/%v", provider.Name),
		})
	}

	c.JSON(http.StatusOK, responseItems)
}

func (h *Handler) LoginProvider(c *gin.Context) {

	ctx := c.Request.Context()

	name := c.Param("provider")
	organization := c.Query("organization")

	provider, err := h.service.GetProviderByName(ctx, organization, name)
	if err != nil {
		log.Error().Err(err).Msg("Retrieving provider by name failed")
		c.String(http.StatusBadRequest, "Retrieving provider by name failed")
		return
	}

	// add return url as claim if it's set as query param
	optionalClaims := jwtgo.MapClaims{}
	returnURL := c.Query("returnURL")
	if returnURL != "" {
		optionalClaims["returnURL"] = returnURL
	}
	if organization != "" {
		optionalClaims["organization"] = organization
	}

	// generate jwt to use as state
	now := time.Now().UTC()
	expiry := now.Add(time.Duration(10) * time.Minute)
	state, err := api.GenerateJWT(h.config, now, expiry, optionalClaims)
	if err != nil {
		log.Error().Err(err).Msg("Failed generating JWT to use as state")
		c.String(http.StatusInternalServerError, "Failed generating JWT to use as state")
	}

	c.Redirect(http.StatusTemporaryRedirect, provider.AuthCodeURL(h.config.APIServer.BaseURL, state))
}

func (h *Handler) HandleOAuthLoginProviderAuthenticator() func(c *gin.Context) (interface{}, error) {
	return func(c *gin.Context) (interface{}, error) {

		name := c.Param("provider")
		code := c.Query("code")
		state := c.Query("state")

		// validate jwt in state
		claims, err := api.GetClaimsFromJWT(h.config, state)
		if err != nil {
			return nil, err
		}

		// get optional return url from claim and store in gin context to use in LoginResponse handler for jwt middleware
		if returnURL, ok := claims["returnURL"]; ok {
			c.Set("returnURL", returnURL)
		}

		organization := ""
		if organizationClaim, ok := claims["organization"]; ok {
			organization = organizationClaim.(string)
		}

		ctx := c.Request.Context()

		// retrieve configured providers
		provider, err := h.service.GetProviderByName(ctx, organization, name)
		if err != nil {
			return nil, err
		}

		log.Debug().Interface("provider", provider).Msgf("Fetched provider for organization '%v' and name '%v'", organization, name)

		// retrieve oauth config
		cfg := provider.GetConfig(h.config.APIServer.BaseURL)
		token, err := cfg.Exchange(ctx, code)
		if err != nil {
			return nil, err
		}

		// fetch identity from oauth provider api
		identity, err := provider.GetUserIdentity(ctx, cfg, token)
		if err != nil {
			return nil, err
		}

		if identity == nil {
			return nil, fmt.Errorf("Empty identity retrieved from oauth provider api")
		}

		// check if user is allowed for this provider
		isAllowed, err := provider.UserIsAllowed(ctx, identity.Email)
		if err != nil {
			return nil, err
		}
		if !isAllowed {
			return nil, fmt.Errorf("User with email %v is not allowed for provider %v", identity.Email, provider.Name)
		}

		// upsert user
		user, err := h.service.GetUserByIdentity(ctx, *identity)
		if err != nil && errors.Is(err, database.ErrUserNotFound) {
			user, err = h.service.CreateUserFromIdentity(ctx, *identity)
			if err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}

		if user == nil {
			return nil, err
		}

		// update identity
		hasIdentityForProvider := false
		for _, i := range user.Identities {
			if i.Provider == name {
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

		// update user last visit and identity
		lastVisit := time.Now().UTC()
		user.LastVisit = &lastVisit
		user.Active = true
		user.CurrentProvider = name
		user.Name = user.GetName()
		user.Email = user.GetEmail()

		// check if user is part of the organization used to log in, if not add it
		if organization != "" {
			user.CurrentOrganization = organization
			isLinkedToOrganization := false
			for _, o := range user.Organizations {
				if o.Name == organization {
					isLinkedToOrganization = true
				}
			}
			if !isLinkedToOrganization {
				// try and get organization by name
				org, err := h.databaseClient.GetOrganizationByName(ctx, organization)
				if err != nil && errors.Is(err, database.ErrOrganizationNotFound) {
					// organization doesn't exist yet, create it
					org, err = h.service.CreateOrganization(ctx, contracts.Organization{
						Name: organization,
					})
					if err != nil {
						return nil, err
					}
				} else if err != nil {
					return nil, err
				}
				// add the organization to the user
				user.Organizations = append(user.Organizations, &contracts.Organization{
					ID:   org.ID,
					Name: org.Name,
				})
			}
		}

		go func(user contracts.User) {
			// create new context to avoid cancellation impacting execution
			span, _ := opentracing.StartSpanFromContext(c.Request.Context(), "rbac:AsyncUpdateUser")
			ctx := opentracing.ContextWithSpan(context.Background(), span)
			defer span.Finish()

			err = h.service.UpdateUser(ctx, user)
			if err != nil {
				log.Warn().Err(err).Msg("Failed updating user in db")
			}
		}(*user)

		// get all roles the user inherits from groups and organizations
		inheritedRoles, err := h.service.GetInheritedRolesForUser(ctx, *user)
		if err != nil {
			return nil, err
		}
		user.Roles = inheritedRoles

		// get all organizations the user inherits from groups
		inheritedOrganizations, err := h.service.GetInheritedOrganizationsForUser(ctx, *user)
		if err != nil {
			return nil, err
		}
		user.Organizations = inheritedOrganizations

		return user, nil
	}
}

func (h *Handler) HandleClientLoginProviderAuthenticator() func(c *gin.Context) (interface{}, error) {
	return func(c *gin.Context) (interface{}, error) {

		log.Debug().Msg("HandleClientLoginProviderAuthenticator")

		var client contracts.Client
		err := c.BindJSON(&client)
		if err != nil {
			log.Error().Err(err).Msg("Failed binding json for client login")
			return nil, err
		}

		ctx := c.Request.Context()

		// get client from db by clientID
		clientFromDB, err := h.databaseClient.GetClientByClientID(ctx, client.ClientID)
		if err != nil {
			log.Error().Err(err).Msgf("Failed retrieving client by client id %v", client.ClientID)
			return nil, err
		}
		if clientFromDB == nil {
			log.Error().Err(err).Msgf("Client for client id %v is nil", client.ClientID)
			return nil, fmt.Errorf("Client from db is nil")
		}

		// see if secret from binded data and database match
		if client.ClientSecret != clientFromDB.ClientSecret {
			log.Error().Err(err).Msgf("Client secret for client id %v does not match", client.ClientID)
			return nil, fmt.Errorf("Client secret does not match")
		}

		// check if client is active
		if !clientFromDB.Active {
			log.Error().Msgf("Client with id %v is not active", client.ClientID)
			return nil, fmt.Errorf("Client is not active")
		}

		return clientFromDB, nil
	}
}

func (h *Handler) HandleImpersonateAuthenticator() func(c *gin.Context) (interface{}, error) {
	return func(c *gin.Context) (interface{}, error) {

		// ensure the request has the correct permission
		if !api.RequestTokenHasPermission(c, api.PermissionUsersImpersonate) {
			return nil, fmt.Errorf("User is not allowed to impersonate other user")
		}

		ctx := c.Request.Context()
		id := c.Param("id")

		user, err := h.databaseClient.GetUserByID(ctx, id, map[api.FilterType][]string{})
		if err != nil || user == nil {
			return nil, err
		}

		// get all roles the user inherits from groups and organizations
		inheritedRoles, err := h.service.GetInheritedRolesForUser(ctx, *user)
		if err != nil {
			return nil, err
		}
		user.Roles = inheritedRoles

		// get all organizations the user inherits from groups
		inheritedOrganizations, err := h.service.GetInheritedOrganizationsForUser(ctx, *user)
		if err != nil {
			return nil, err
		}
		user.Organizations = inheritedOrganizations

		return user, nil
	}
}

func (h *Handler) GetUsers(c *gin.Context) {

	pageNumber, pageSize, filters, sortings := api.GetQueryParameters(c)

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionUsersList) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	ctx := c.Request.Context()

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			users, err := h.databaseClient.GetUsers(ctx, pageNumber, pageSize, filters, sortings)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(users))
			for i := range users {
				items[i] = users[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetUsersCount(ctx, filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving users from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetUser(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionUsersGet) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	ctx := c.Request.Context()
	id := c.Param("id")

	user, err := h.databaseClient.GetUserByID(ctx, id, map[api.FilterType][]string{})
	if err != nil || user == nil {
		log.Error().Err(err).Msgf("Failed retrieving user with id %v from db", id)
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound)})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *Handler) CreateUser(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionUsersCreate) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	var user contracts.User
	err := c.BindJSON(&user)
	if err != nil {
		errorMessage := "Binding CreateUser body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	ctx := c.Request.Context()

	insertedUser, err := h.service.CreateUser(ctx, user)
	if err != nil {
		log.Error().Err(err).Msg("Failed inserting user")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusCreated, insertedUser)
}

func (h *Handler) UpdateUser(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionUsersUpdate) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	var user contracts.User
	err := c.BindJSON(&user)
	if err != nil {
		errorMessage := "Binding UpdateUser body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	ctx := c.Request.Context()

	id := c.Param("id")
	if user.ID != id {
		log.Error().Err(err).Msg("User id is incorrect")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest)})
		return
	}

	err = h.service.UpdateUser(ctx, user)
	if err != nil {
		log.Error().Err(err).Msg("Failed updating user")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusText(http.StatusOK)})
}

func (h *Handler) DeleteUser(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionUsersDelete) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	ctx := c.Request.Context()
	id := c.Param("id")
	err := h.service.DeleteUser(ctx, id)
	if err != nil {
		log.Error().Err(err).Msg("Failed deleting user")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusText(http.StatusOK)})
}

func (h *Handler) GetGroups(c *gin.Context) {

	pageNumber, pageSize, filters, sortings := api.GetQueryParameters(c)

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionGroupsList) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	ctx := c.Request.Context()

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			groups, err := h.databaseClient.GetGroups(ctx, pageNumber, pageSize, filters, sortings)
			if err != nil {
				return nil, err
			}

			if len(sortings) == 0 {
				sort.Slice(groups, func(i, j int) bool {
					return groups[i].Name < groups[j].Name
				})
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(groups))
			for i := range groups {
				items[i] = groups[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetGroupsCount(ctx, filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving groups from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetGroup(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionGroupsGet) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	ctx := c.Request.Context()
	id := c.Param("id")

	group, err := h.databaseClient.GetGroupByID(ctx, id, map[api.FilterType][]string{})
	if err != nil || group == nil {
		log.Error().Err(err).Msgf("Failed retrieving group with id %v from db", id)
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound)})
		return
	}

	c.JSON(http.StatusOK, group)
}

func (h *Handler) CreateGroup(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionGroupsCreate) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	var group contracts.Group
	err := c.BindJSON(&group)
	if err != nil {
		errorMessage := "Binding CreateGroup body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	ctx := c.Request.Context()

	insertedGroup, err := h.service.CreateGroup(ctx, group)
	if err != nil {
		log.Error().Err(err).Msg("Failed inserting group")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusCreated, insertedGroup)
}

func (h *Handler) UpdateGroup(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionGroupsUpdate) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	var group contracts.Group
	err := c.BindJSON(&group)
	if err != nil {
		errorMessage := "Binding UpdateGroup body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	ctx := c.Request.Context()

	id := c.Param("id")
	if group.ID != id {
		log.Error().Err(err).Msg("Group id is incorrect")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest)})
		return
	}

	err = h.service.UpdateGroup(ctx, group)
	if err != nil {
		log.Error().Err(err).Msg("Failed updating group")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusText(http.StatusOK)})
}

func (h *Handler) DeleteGroup(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionGroupsDelete) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	ctx := c.Request.Context()
	id := c.Param("id")
	err := h.service.DeleteGroup(ctx, id)
	if err != nil {
		log.Error().Err(err).Msg("Failed deleting group")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusText(http.StatusOK)})
}

func (h *Handler) GetOrganizations(c *gin.Context) {

	pageNumber, pageSize, filters, sortings := api.GetQueryParameters(c)

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionOrganizationsList) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	ctx := c.Request.Context()

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			organizations, err := h.databaseClient.GetOrganizations(ctx, pageNumber, pageSize, filters, sortings)
			if err != nil {
				return nil, err
			}

			if len(sortings) == 0 {
				sort.Slice(organizations, func(i, j int) bool {
					return organizations[i].Name < organizations[j].Name
				})
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(organizations))
			for i := range organizations {
				items[i] = organizations[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetOrganizationsCount(ctx, filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving organizations from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetOrganization(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionOrganizationsGet) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	ctx := c.Request.Context()
	id := c.Param("id")

	organization, err := h.databaseClient.GetOrganizationByID(ctx, id)
	if err != nil || organization == nil {
		log.Error().Err(err).Msgf("Failed retrieving organization with id %v from db", id)
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound)})
		return
	}

	c.JSON(http.StatusOK, organization)
}

func (h *Handler) CreateOrganization(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionOrganizationsCreate) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	var organization contracts.Organization
	err := c.BindJSON(&organization)
	if err != nil {
		errorMessage := "Binding CreateOrganization body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	ctx := c.Request.Context()

	insertedOrganization, err := h.service.CreateOrganization(ctx, organization)
	if err != nil {
		log.Error().Err(err).Msg("Failed inserting organization")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusCreated, insertedOrganization)
}

func (h *Handler) UpdateOrganization(c *gin.Context) {

	// ensure the user has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionOrganizationsUpdate) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	var organization contracts.Organization
	err := c.BindJSON(&organization)
	if err != nil {
		errorMessage := "Binding UpdateOrganization body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	id := c.Param("id")
	if organization.ID != id {
		log.Error().Err(err).Msg("Organization id is incorrect")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest)})
		return
	}

	ctx := c.Request.Context()

	err = h.service.UpdateOrganization(ctx, organization)
	if err != nil {
		log.Error().Err(err).Msg("Failed updating organization")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusText(http.StatusOK)})
}

func (h *Handler) DeleteOrganization(c *gin.Context) {

	// ensure the user has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionOrganizationsDelete) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	ctx := c.Request.Context()
	id := c.Param("id")
	err := h.service.DeleteOrganization(ctx, id)
	if err != nil {
		log.Error().Err(err).Msg("Failed deleting organization")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusText(http.StatusOK)})
}

func (h *Handler) GetClients(c *gin.Context) {

	pageNumber, pageSize, filters, sortings := api.GetQueryParameters(c)

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionClientsList) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	ctx := c.Request.Context()

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			clients, err := h.databaseClient.GetClients(ctx, pageNumber, pageSize, filters, sortings)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(clients))
			for i := range clients {
				if !api.RequestTokenHasPermission(c, api.PermissionClientsViewSecret) {
					// obfuscate client secret
					clients[i].ClientSecret = "***"
				}
				items[i] = clients[i]
			}
			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetClientsCount(ctx, filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving clients from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetClient(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionClientsGet) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	ctx := c.Request.Context()
	id := c.Param("id")

	client, err := h.databaseClient.GetClientByID(ctx, id)
	if err != nil || client == nil {
		log.Error().Err(err).Msgf("Failed retrieving client with id %v from db", id)
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound)})
		return
	}

	if !api.RequestTokenHasPermission(c, api.PermissionClientsViewSecret) {
		// obfuscate client secret
		client.ClientSecret = "***"
	}

	c.JSON(http.StatusOK, client)
}

func (h *Handler) CreateClient(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionClientsCreate) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	var client contracts.Client
	err := c.BindJSON(&client)
	if err != nil {
		errorMessage := "Binding CreateClient body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	ctx := c.Request.Context()

	insertedClient, err := h.service.CreateClient(ctx, client)
	if err != nil {
		log.Error().Err(err).Msg("Failed inserting client")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusCreated, insertedClient)
}

func (h *Handler) UpdateClient(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionClientsUpdate) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	var client contracts.Client
	err := c.BindJSON(&client)
	if err != nil {
		errorMessage := "Binding UpdateClient body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	id := c.Param("id")
	if client.ID != id {
		log.Error().Err(err).Msg("Client id is incorrect")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest)})
		return
	}

	ctx := c.Request.Context()

	err = h.service.UpdateClient(ctx, client)
	if err != nil {
		log.Error().Err(err).Msg("Failed updating client")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusText(http.StatusOK)})
}

func (h *Handler) DeleteClient(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionClientsDelete) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	ctx := c.Request.Context()
	id := c.Param("id")
	err := h.service.DeleteClient(ctx, id)
	if err != nil {
		log.Error().Err(err).Msg("Failed deleting client")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusText(http.StatusOK)})
}

func (h *Handler) GetIntegrations(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionIntegrationsGet) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	ctx := c.Request.Context()

	var response struct {
		Github    *githubResponse    `json:"github,omitempty"`
		Bitbucket *bitbucketResponse `json:"bitbucket,omitempty"`
	}

	if h.config != nil && h.config.Integrations != nil && h.config.Integrations.Github != nil && h.config.Integrations.Github.Enable {
		response.Github = &githubResponse{
			Manifest: githubManifest{
				Name:        "estafette-ci",
				Description: "Estafette - The resilient and cloud-native CI/CD platform",
				URL:         h.config.APIServer.BaseURL,
				HookAttributes: &githubHookAttribute{
					URL: fmt.Sprintf("%v/api/integrations/github/events", strings.TrimRight(h.config.APIServer.IntegrationsURL, "/")),
				},
				RedirectURL: fmt.Sprintf("%v/api/integrations/github/redirect", strings.TrimRight(h.config.APIServer.IntegrationsURL, "/")),
				Public:      false,
				DefaultEvents: []string{
					"push",
					"repository",
				},
				DefaultPermissions: map[string]string{
					"contents": "write",
					"issues":   "write",
					"metadata": "read",
					"statuses": "write",
				},
			},
		}

		apps, err := h.githubapiClient.GetApps(ctx)
		if err != nil {
			log.Error().Err(err).Msg("Failed retrieving github apps from configmap")
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
			return
		}

		// clone apps while obfuscating secret properties
		response.Github.Apps = make([]*githubapi.GithubApp, 0)
		for _, app := range apps {
			response.Github.Apps = append(response.Github.Apps, &githubapi.GithubApp{
				ID:            app.ID,
				Name:          app.Name,
				Slug:          app.Slug,
				PrivateKey:    "***",
				WebhookSecret: "***",
				ClientID:      app.ClientID,
				ClientSecret:  "***",
				Installations: app.Installations,
			})
		}
	}

	if h.config != nil && h.config.Integrations != nil && h.config.Integrations.Bitbucket != nil && h.config.Integrations.Bitbucket.Enable {
		response.Bitbucket = &bitbucketResponse{
			RedirectURI: fmt.Sprintf("%v/api/integrations/bitbucket/redirect", strings.TrimRight(h.config.APIServer.IntegrationsURL, "/")),
		}

		apps, err := h.bitbucketapiClient.GetApps(ctx)
		if err != nil {
			log.Error().Err(err).Msg("Failed retrieving bitbucket apps from configmap")
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
			return
		}

		// clone installations while obfuscating secret properties
		response.Bitbucket.Apps = make([]*bitbucketapi.BitbucketApp, 0)
		for _, app := range apps {
			copiedApp := &bitbucketapi.BitbucketApp{
				Key:           app.Key,
				Installations: make([]*bitbucketapi.BitbucketAppInstallation, 0),
			}

			for _, installation := range app.Installations {
				copiedApp.Installations = append(copiedApp.Installations, &bitbucketapi.BitbucketAppInstallation{
					Key:          installation.Key,
					BaseApiURL:   installation.BaseApiURL,
					ClientKey:    installation.ClientKey,
					SharedSecret: "***",
					Workspace: &bitbucketapi.Workspace{
						Slug: installation.Workspace.Slug,
						Name: installation.Workspace.Name,
						UUID: installation.Workspace.UUID,
					},
					Organizations: installation.Organizations,
				})
			}
			response.Bitbucket.Apps = append(response.Bitbucket.Apps, copiedApp)
		}
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) UpdateGithubInstallation(c *gin.Context) {
	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionIntegrationsUpdate) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	ctx := c.Request.Context()

	// bind body
	var installationFromForm githubapi.GithubInstallation
	err := c.BindJSON(&installationFromForm)
	if err != nil {
		errorMessage := "Binding UpdateGithubIntegration body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	// get github installation
	_, installation, err := h.githubapiClient.GetAppAndInstallationByID(ctx, installationFromForm.ID)
	if err != nil {
		log.Error().Err(err).Msgf("Failed retrieving github installation %v from configmap", installationFromForm.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	// update organizations
	installation.Organizations = installationFromForm.Organizations

	err = h.githubapiClient.AddInstallation(ctx, *installation)
	if err != nil {
		log.Error().Err(err).Msgf("Failed updating github installation %v in configmap", installationFromForm.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusText(http.StatusOK)})
}

func (h *Handler) UpdateBitbucketInstallation(c *gin.Context) {
	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionIntegrationsUpdate) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	ctx := c.Request.Context()

	// bind body
	var installationFromForm bitbucketapi.BitbucketAppInstallation
	err := c.BindJSON(&installationFromForm)
	if err != nil {
		errorMessage := "Binding UpdateBitbucketIntegration body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	// get bitbucket installation
	installation, err := h.bitbucketapiClient.GetInstallationByClientKey(ctx, installationFromForm.ClientKey)
	if err != nil {
		log.Error().Err(err).Msgf("Failed retrieving bitbucket installation %v from configmap", installationFromForm.ClientKey)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	// update organizations
	installation.Organizations = installationFromForm.Organizations

	err = h.bitbucketapiClient.AddInstallation(ctx, *installation)
	if err != nil {
		log.Error().Err(err).Msgf("Failed updating bitbucket installation %v in configmap", installationFromForm.ClientKey)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusText(http.StatusOK)})
}

func (h *Handler) GetPipelines(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionPipelinesList) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	pageNumber, pageSize, filters, sortings := api.GetQueryParameters(c)

	ctx := c.Request.Context()

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			pipelines, err := h.databaseClient.GetPipelines(ctx, pageNumber, pageSize, filters, sortings, true)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(pipelines))
			for i := range pipelines {
				items[i] = pipelines[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetPipelinesCount(ctx, filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving pipelines from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetPipeline(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionPipelinesGet) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	ctx := c.Request.Context()
	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	pipeline, err := h.databaseClient.GetPipeline(ctx, source, owner, repo, map[api.FilterType][]string{}, false)
	if err != nil || pipeline == nil {
		log.Error().Err(err).Msgf("Failed retrieving pipeline for %v/%v/%v from db", source, owner, repo)
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound)})
		return
	}

	c.JSON(http.StatusOK, pipeline)
}

func (h *Handler) UpdatePipeline(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionPipelinesUpdate) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	var pipeline contracts.Pipeline
	err := c.BindJSON(&pipeline)
	if err != nil {
		errorMessage := "Binding UpdatePipeline body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	ctx := c.Request.Context()
	source := c.Param("source")
	owner := c.Param("owner")
	repo := c.Param("repo")

	if pipeline.RepoSource != source && pipeline.RepoOwner != owner && pipeline.RepoName != repo {
		log.Error().Err(err).Msg("Pipeline name is incorrect")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest)})
		return
	}

	err = h.service.UpdatePipeline(ctx, pipeline)
	if err != nil {
		log.Error().Err(err).Msg("Failed updating pipeline")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusText(http.StatusOK)})
}

func (h *Handler) BatchUpdateUsers(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionUsersUpdate) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	var body struct {
		Users                 []string                  `json:"users"`
		Role                  *string                   `json:"role"`
		RolesToAdd            []*string                 `json:"rolesToAdd"`
		RolesToRemove         []*string                 `json:"rolesToRemove"`
		Group                 *contracts.Group          `json:"group"`
		GroupsToAdd           []*contracts.Group        `json:"groupsToAdd"`
		GroupsToRemove        []*contracts.Group        `json:"groupsToRemove"`
		Organization          *contracts.Organization   `json:"organization"`
		OrganizationsToAdd    []*contracts.Organization `json:"organizationsToAdd"`
		OrganizationsToRemove []*contracts.Organization `json:"organizationsToRemove"`
	}

	err := c.BindJSON(&body)
	if err != nil {
		errorMessage := "Binding BatchUpdateUsers body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	ctx := c.Request.Context()

	if len(body.Users) == 0 || (body.Role == nil && len(body.RolesToAdd) == 0 && len(body.RolesToRemove) == 0 && body.Group == nil && len(body.GroupsToAdd) == 0 && len(body.GroupsToRemove) == 0 && body.Organization == nil && len(body.OrganizationsToAdd) == 0 && len(body.OrganizationsToRemove) == 0) {
		log.Error().Err(err).Msg("Request body is incorrect")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest)})
		return
	}

	// limit concurrency using a semaphore
	semaphore := semaphore.NewWeighted(10)
	g, ctx := errgroup.WithContext(ctx)

	for _, u := range body.Users {
		u := u

		g.Go(func() error {
			err = semaphore.Acquire(ctx, 1)
			if err != nil {
				return err
			}
			defer semaphore.Release(1)

			user, err := h.databaseClient.GetUserByID(ctx, u, map[api.FilterType][]string{})
			if err != nil {
				return err
			}

			// add role if not present
			if body.Role != nil {
				hasRole := false
				for _, r := range user.Roles {
					if r != nil && *r == *body.Role {
						hasRole = true
					}
				}
				if !hasRole {
					user.Roles = append(user.Roles, body.Role)
				}
			}

			// add role if not present
			for _, roleToAdd := range body.RolesToAdd {
				hasRole := false
				for _, r := range user.Roles {
					if r != nil && *r == *roleToAdd {
						hasRole = true
					}
				}
				if !hasRole {
					user.Roles = append(user.Roles, roleToAdd)
				}
			}

			// remove role if present
			for _, roleToRemove := range body.RolesToRemove {
				newRoles := []*string{}
				for _, r := range user.Roles {
					if r == nil || *r != *roleToRemove {
						newRoles = append(newRoles, r)
					}
				}
				user.Roles = newRoles
			}

			// add group if not present
			if body.Group != nil {
				hasGroup := false
				for _, g := range user.Groups {
					if g != nil && g.Name == body.Group.Name && g.ID == body.Group.ID {
						hasGroup = true
					}
				}
				if !hasGroup {
					user.Groups = append(user.Groups, body.Group)
				}
			}

			// add group if not present
			for _, groupToAdd := range body.GroupsToAdd {
				hasGroup := false
				for _, g := range user.Groups {
					if g != nil && g.Name == groupToAdd.Name && g.ID == groupToAdd.ID {
						hasGroup = true
					}
				}
				if !hasGroup {
					user.Groups = append(user.Groups, groupToAdd)
				}
			}

			// remove group if present
			for _, groupToRemove := range body.GroupsToRemove {
				newGroups := []*contracts.Group{}
				for _, g := range user.Groups {
					if g == nil || g.Name != groupToRemove.Name || g.ID != groupToRemove.ID {
						newGroups = append(newGroups, g)
					}
				}
				user.Groups = newGroups
			}

			// add organization if not present
			if body.Organization != nil {
				hasOrganization := false
				for _, o := range user.Organizations {
					if o != nil && o.Name == body.Organization.Name && o.ID == body.Organization.ID {
						hasOrganization = true
					}
				}
				if !hasOrganization {
					user.Organizations = append(user.Organizations, body.Organization)
				}
			}

			// add organization if not present
			for _, organizationToAdd := range body.OrganizationsToAdd {
				hasOrganization := false
				for _, o := range user.Organizations {
					if o != nil && o.Name == organizationToAdd.Name && o.ID == organizationToAdd.ID {
						hasOrganization = true
					}
				}
				if !hasOrganization {
					user.Organizations = append(user.Organizations, organizationToAdd)
				}
			}

			// remove organization if present
			for _, organizationToRemove := range body.OrganizationsToRemove {
				newOrganizations := []*contracts.Organization{}
				for _, o := range user.Organizations {
					if o == nil || o.Name != organizationToRemove.Name || o.ID != organizationToRemove.ID {
						newOrganizations = append(newOrganizations, o)
					}
				}
				user.Organizations = newOrganizations
			}

			err = h.service.UpdateUser(ctx, *user)
			if err != nil {
				return err
			}

			return nil
		})
	}

	// wait until all concurrent goroutines are done
	err = g.Wait()
	if err != nil {
		log.Error().Err(err).Msg("Failed updating user")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusText(http.StatusOK)})
}

func (h *Handler) BatchUpdateGroups(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionGroupsUpdate) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	var body struct {
		Groups                []string                  `json:"groups"`
		Role                  *string                   `json:"role"`
		RolesToAdd            []*string                 `json:"rolesToAdd"`
		RolesToRemove         []*string                 `json:"rolesToRemove"`
		Organization          *contracts.Organization   `json:"organization"`
		OrganizationsToAdd    []*contracts.Organization `json:"organizationsToAdd"`
		OrganizationsToRemove []*contracts.Organization `json:"organizationsToRemove"`
	}

	err := c.BindJSON(&body)
	if err != nil {
		errorMessage := "Binding BatchUpdateGroups body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	ctx := c.Request.Context()

	if len(body.Groups) == 0 || (body.Role == nil && len(body.RolesToAdd) == 0 && len(body.RolesToRemove) == 0 && body.Organization == nil && len(body.OrganizationsToAdd) == 0 && len(body.OrganizationsToRemove) == 0) {
		log.Error().Err(err).Msg("Request body is incorrect")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest)})
		return
	}

	// limit concurrency using a semaphore
	semaphore := semaphore.NewWeighted(10)
	errgrp, ctx := errgroup.WithContext(ctx)

	for _, g := range body.Groups {
		g := g

		errgrp.Go(func() error {
			err = semaphore.Acquire(ctx, 1)
			if err != nil {
				return err
			}
			defer semaphore.Release(1)

			group, err := h.databaseClient.GetGroupByID(ctx, g, map[api.FilterType][]string{})
			if err != nil {
				return err
			}

			// add role if not present
			if body.Role != nil {
				hasRole := false
				for _, r := range group.Roles {
					if r != nil && *r == *body.Role {
						hasRole = true
					}
				}
				if !hasRole {
					group.Roles = append(group.Roles, body.Role)
				}
			}

			// add role if not present
			for _, roleToAdd := range body.RolesToAdd {
				hasRole := false
				for _, r := range group.Roles {
					if r != nil && *r == *roleToAdd {
						hasRole = true
					}
				}
				if !hasRole {
					group.Roles = append(group.Roles, roleToAdd)
				}
			}

			// remove role if present
			for _, roleToRemove := range body.RolesToRemove {
				newRoles := []*string{}
				for _, r := range group.Roles {
					if r == nil || *r != *roleToRemove {
						newRoles = append(newRoles, r)
					}
				}
				group.Roles = newRoles
			}

			// add organization if not present
			if body.Organization != nil {
				hasOrganization := false
				for _, o := range group.Organizations {
					if o != nil && o.Name == body.Organization.Name && o.ID == body.Organization.ID {
						hasOrganization = true
					}
				}
				if !hasOrganization {
					group.Organizations = append(group.Organizations, body.Organization)
				}
			}

			// add organization if not present
			for _, organizationToAdd := range body.OrganizationsToAdd {
				hasOrganization := false
				for _, o := range group.Organizations {
					if o != nil && o.Name == organizationToAdd.Name && o.ID == organizationToAdd.ID {
						hasOrganization = true
					}
				}
				if !hasOrganization {
					group.Organizations = append(group.Organizations, organizationToAdd)
				}
			}

			// remove organization if present
			for _, organizationToRemove := range body.OrganizationsToRemove {
				newOrganizations := []*contracts.Organization{}
				for _, o := range group.Organizations {
					if o == nil || o.Name != organizationToRemove.Name || o.ID != organizationToRemove.ID {
						newOrganizations = append(newOrganizations, o)
					}
				}
				group.Organizations = newOrganizations
			}

			err = h.service.UpdateGroup(ctx, *group)
			if err != nil {
				return err
			}

			return nil
		})
	}

	// wait until all concurrent goroutines are done
	err = errgrp.Wait()
	if err != nil {
		log.Error().Err(err).Msg("Failed updating group")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusText(http.StatusOK)})
}

func (h *Handler) BatchUpdateOrganizations(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionOrganizationsUpdate) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	var body struct {
		Organizations []string  `json:"organizations"`
		Role          *string   `json:"role"`
		RolesToAdd    []*string `json:"rolesToAdd"`
		RolesToRemove []*string `json:"rolesToRemove"`
	}

	err := c.BindJSON(&body)
	if err != nil {
		errorMessage := "Binding BatchUpdateOrganizations body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	ctx := c.Request.Context()

	if len(body.Organizations) == 0 || (body.Role == nil && len(body.RolesToAdd) == 0 && len(body.RolesToRemove) == 0) {
		log.Error().Err(err).Msg("Request body is incorrect")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest)})
		return
	}

	// limit concurrency using a semaphore
	semaphore := semaphore.NewWeighted(10)
	g, ctx := errgroup.WithContext(ctx)

	for _, o := range body.Organizations {
		o := o

		g.Go(func() error {
			err = semaphore.Acquire(ctx, 1)
			if err != nil {
				return err
			}
			defer semaphore.Release(1)

			organization, err := h.databaseClient.GetOrganizationByID(ctx, o)
			if err != nil {
				return err
			}

			// add role if not present
			if body.Role != nil {
				hasRole := false
				for _, r := range organization.Roles {
					if r != nil && *r == *body.Role {
						hasRole = true
					}
				}
				if !hasRole {
					organization.Roles = append(organization.Roles, body.Role)
				}
			}

			// add role if not present
			for _, roleToAdd := range body.RolesToAdd {
				hasRole := false
				for _, r := range organization.Roles {
					if r != nil && *r == *roleToAdd {
						hasRole = true
					}
				}
				if !hasRole {
					organization.Roles = append(organization.Roles, roleToAdd)
				}
			}

			// remove role if present
			for _, roleToRemove := range body.RolesToRemove {
				newRoles := []*string{}
				for _, r := range organization.Roles {
					if r == nil || *r != *roleToRemove {
						newRoles = append(newRoles, r)
					}
				}
				organization.Roles = newRoles
			}

			err = h.service.UpdateOrganization(ctx, *organization)
			if err != nil {
				return err
			}

			return nil
		})
	}

	// wait until all concurrent goroutines are done
	err = g.Wait()
	if err != nil {
		log.Error().Err(err).Msg("Failed updating organization")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusText(http.StatusOK)})
}

func (h *Handler) BatchUpdateClients(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionOrganizationsUpdate) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	var body struct {
		Clients               []string                  `json:"clients"`
		Role                  *string                   `json:"role"`
		RolesToAdd            []*string                 `json:"rolesToAdd"`
		RolesToRemove         []*string                 `json:"rolesToRemove"`
		Organization          *contracts.Organization   `json:"organization"`
		OrganizationsToAdd    []*contracts.Organization `json:"organizationsToAdd"`
		OrganizationsToRemove []*contracts.Organization `json:"organizationsToRemove"`
	}

	err := c.BindJSON(&body)
	if err != nil {
		errorMessage := "Binding BatchUpdateClients body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	ctx := c.Request.Context()

	if len(body.Clients) == 0 || (body.Role == nil && len(body.RolesToAdd) == 0 && len(body.RolesToRemove) == 0 && body.Organization == nil && len(body.OrganizationsToAdd) == 0 && len(body.OrganizationsToRemove) == 0) {
		log.Error().Err(err).Msg("Request body is incorrect")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest)})
		return
	}

	// limit concurrency using a semaphore
	semaphore := semaphore.NewWeighted(10)
	g, ctx := errgroup.WithContext(ctx)

	for _, c := range body.Clients {
		c := c

		g.Go(func() error {
			err = semaphore.Acquire(ctx, 1)
			if err != nil {
				return err
			}
			defer semaphore.Release(1)

			client, err := h.databaseClient.GetClientByID(ctx, c)
			if err != nil {
				return err
			}

			// add role if not present
			if body.Role != nil {
				hasRole := false
				for _, r := range client.Roles {
					if r != nil && *r == *body.Role {
						hasRole = true
					}
				}
				if !hasRole {
					client.Roles = append(client.Roles, body.Role)
				}
			}

			// add role if not present
			for _, roleToAdd := range body.RolesToAdd {
				hasRole := false
				for _, r := range client.Roles {
					if r != nil && *r == *roleToAdd {
						hasRole = true
					}
				}
				if !hasRole {
					client.Roles = append(client.Roles, roleToAdd)
				}
			}

			// remove role if present
			for _, roleToRemove := range body.RolesToRemove {
				newRoles := []*string{}
				for _, r := range client.Roles {
					if r == nil || *r != *roleToRemove {
						newRoles = append(newRoles, r)
					}
				}
				client.Roles = newRoles
			}

			// add organization if not present
			if body.Organization != nil {
				hasOrganization := false
				for _, o := range client.Organizations {
					if o != nil && o.Name == body.Organization.Name && o.ID == body.Organization.ID {
						hasOrganization = true
					}
				}
				if !hasOrganization {
					client.Organizations = append(client.Organizations, body.Organization)
				}
			}

			// add organization if not present
			for _, organizationToAdd := range body.OrganizationsToAdd {
				hasOrganization := false
				for _, o := range client.Organizations {
					if o != nil && o.Name == organizationToAdd.Name && o.ID == organizationToAdd.ID {
						hasOrganization = true
					}
				}
				if !hasOrganization {
					client.Organizations = append(client.Organizations, organizationToAdd)
				}
			}

			// remove organization if present
			for _, organizationToRemove := range body.OrganizationsToRemove {
				newOrganizations := []*contracts.Organization{}
				for _, o := range client.Organizations {
					if o == nil || o.Name != organizationToRemove.Name || o.ID != organizationToRemove.ID {
						newOrganizations = append(newOrganizations, o)
					}
				}
				client.Organizations = newOrganizations
			}

			err = h.service.UpdateClient(ctx, *client)
			if err != nil {
				return err
			}

			return nil
		})
	}

	// wait until all concurrent goroutines are done
	err = g.Wait()
	if err != nil {
		log.Error().Err(err).Msg("Failed updating client")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusText(http.StatusOK)})
}

func (h *Handler) BatchUpdatePipelines(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionPipelinesUpdate) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	var body struct {
		Pipelines             []string                  `json:"pipelines"`
		Group                 *contracts.Group          `json:"group"`
		GroupsToAdd           []*contracts.Group        `json:"groupsToAdd"`
		GroupsToRemove        []*contracts.Group        `json:"groupsToRemove"`
		Organization          *contracts.Organization   `json:"organization"`
		OrganizationsToAdd    []*contracts.Organization `json:"organizationsToAdd"`
		OrganizationsToRemove []*contracts.Organization `json:"organizationsToRemove"`
	}

	err := c.BindJSON(&body)
	if err != nil {
		errorMessage := "Binding BatchUpdatePipelines body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	ctx := c.Request.Context()

	if len(body.Pipelines) == 0 || (body.Group == nil && len(body.GroupsToAdd) == 0 && len(body.GroupsToRemove) == 0 && body.Organization == nil && len(body.OrganizationsToAdd) == 0 && len(body.OrganizationsToRemove) == 0) {
		log.Error().Err(err).Msg("Request body is incorrect")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest)})
		return
	}

	// limit concurrency using a semaphore
	semaphore := semaphore.NewWeighted(10)
	g, ctx := errgroup.WithContext(ctx)

	for _, p := range body.Pipelines {
		p := p

		g.Go(func() error {
			err = semaphore.Acquire(ctx, 1)
			if err != nil {
				return err
			}
			defer semaphore.Release(1)

			// split pipeline in source, owner and name
			pipelineParts := strings.Split(p, "/")
			if len(pipelineParts) != 3 {
				return fmt.Errorf("Pipeline '%v' has invalid name", p)
			}

			pipeline, err := h.databaseClient.GetPipeline(ctx, pipelineParts[0], pipelineParts[1], pipelineParts[2], map[api.FilterType][]string{}, true)
			if err != nil {
				return err
			}

			// add group if not present
			if body.Group != nil {
				hasGroup := false
				for _, g := range pipeline.Groups {
					if g != nil && g.Name == body.Group.Name && g.ID == body.Group.ID {
						hasGroup = true
					}
				}
				if !hasGroup {
					pipeline.Groups = append(pipeline.Groups, body.Group)
				}
			}

			// add group if not present
			for _, groupToAdd := range body.GroupsToAdd {
				hasGroup := false
				for _, g := range pipeline.Groups {
					if g != nil && g.Name == groupToAdd.Name && g.ID == groupToAdd.ID {
						hasGroup = true
					}
				}
				if !hasGroup {
					pipeline.Groups = append(pipeline.Groups, groupToAdd)
				}
			}

			// remove group if present
			for _, groupToRemove := range body.GroupsToRemove {
				newGroups := []*contracts.Group{}
				for _, g := range pipeline.Groups {
					if g == nil || g.Name != groupToRemove.Name || g.ID != groupToRemove.ID {
						newGroups = append(newGroups, g)
					}
				}
				pipeline.Groups = newGroups
			}

			// add organization if not present
			if body.Organization != nil {
				hasOrganization := false
				for _, o := range pipeline.Organizations {
					if o != nil && o.Name == body.Organization.Name && o.ID == body.Organization.ID {
						hasOrganization = true
					}
				}
				if !hasOrganization {
					pipeline.Organizations = append(pipeline.Organizations, body.Organization)
				}
			}

			// add organization if not present
			for _, organizationToAdd := range body.OrganizationsToAdd {
				hasOrganization := false
				for _, o := range pipeline.Organizations {
					if o != nil && o.Name == organizationToAdd.Name && o.ID == organizationToAdd.ID {
						hasOrganization = true
					}
				}
				if !hasOrganization {
					pipeline.Organizations = append(pipeline.Organizations, organizationToAdd)
				}
			}

			// remove organization if present
			for _, organizationToRemove := range body.OrganizationsToRemove {
				newOrganizations := []*contracts.Organization{}
				for _, o := range pipeline.Organizations {
					if o == nil || o.Name != organizationToRemove.Name || o.ID != organizationToRemove.ID {
						newOrganizations = append(newOrganizations, o)
					}
				}
				pipeline.Organizations = newOrganizations
			}

			err = h.service.UpdatePipeline(ctx, *pipeline)
			if err != nil {
				return err
			}

			return nil
		})
	}

	// wait until all concurrent goroutines are done
	err = g.Wait()
	if err != nil {
		log.Error().Err(err).Msg("Failed updating pipeline")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusText(http.StatusOK)})
}
