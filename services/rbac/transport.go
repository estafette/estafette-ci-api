package rbac

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/estafette/estafette-ci-api/api"
	"github.com/estafette/estafette-ci-api/clients/cockroachdb"
	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// NewHandler returns a new rbac.Handler
func NewHandler(config *api.APIConfig, service Service, cockroachdbClient cockroachdb.Client) Handler {
	return Handler{
		config:            config,
		service:           service,
		cockroachdbClient: cockroachdbClient,
	}
}

type Handler struct {
	config            *api.APIConfig
	service           Service
	cockroachdbClient cockroachdb.Client
}

func (h *Handler) GetLoggedInUser(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	id := claims[jwt.IdentityKey].(string)

	ctx := c.Request.Context()

	user, err := h.cockroachdbClient.GetUserByID(ctx, id)
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
	for organization, orgProviders := range providers {
		for _, p := range orgProviders {
			responseItems = append(responseItems, map[string]interface{}{
				"id":           p.Name,
				"organization": organization,
				"name":         strings.Title(p.Name),
				"path":         fmt.Sprintf("/api/auth/login/%v", p.Name),
			})
		}
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
	state, err := api.GenerateJWT(h.config, time.Duration(10)*time.Minute, optionalClaims)
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
		if err != nil && errors.Is(err, cockroachdb.ErrUserNotFound) {
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
				org, err := h.cockroachdbClient.GetOrganizationByName(ctx, organization)
				if err != nil && errors.Is(err, cockroachdb.ErrOrganizationNotFound) {
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

		log.Info().Msg("HandleClientLoginProviderAuthenticator")

		var client contracts.Client
		err := c.BindJSON(&client)
		if err != nil {
			log.Error().Err(err).Msg("Failed binding json for client login")
			return nil, err
		}

		ctx := c.Request.Context()

		// get client from db by clientID
		clientFromDB, err := h.cockroachdbClient.GetClientByClientID(ctx, client.ClientID)
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
			c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
			return nil, fmt.Errorf("User is not allowed to impersonate other user")
		}

		ctx := c.Request.Context()
		id := c.Param("id")

		user, err := h.cockroachdbClient.GetUserByID(ctx, id)
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
			users, err := h.cockroachdbClient.GetUsers(ctx, pageNumber, pageSize, filters, sortings)
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
			return h.cockroachdbClient.GetUsersCount(ctx, filters)
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

	user, err := h.cockroachdbClient.GetUserByID(ctx, id)
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
		errorMessage := fmt.Sprint("Binding CreateUser body failed")
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
		errorMessage := fmt.Sprint("Binding UpdateUser body failed")
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
			groups, err := h.cockroachdbClient.GetGroups(ctx, pageNumber, pageSize, filters, sortings)
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
			return h.cockroachdbClient.GetGroupsCount(ctx, filters)
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

	group, err := h.cockroachdbClient.GetGroupByID(ctx, id)
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
		errorMessage := fmt.Sprint("Binding CreateGroup body failed")
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
		errorMessage := fmt.Sprint("Binding UpdateGroup body failed")
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
			organizations, err := h.cockroachdbClient.GetOrganizations(ctx, pageNumber, pageSize, filters, sortings)
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
			return h.cockroachdbClient.GetOrganizationsCount(ctx, filters)
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

	organization, err := h.cockroachdbClient.GetOrganizationByID(ctx, id)
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
		errorMessage := fmt.Sprint("Binding CreateOrganization body failed")
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
		errorMessage := fmt.Sprint("Binding UpdateOrganization body failed")
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
			clients, err := h.cockroachdbClient.GetClients(ctx, pageNumber, pageSize, filters, sortings)
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
			return h.cockroachdbClient.GetClientsCount(ctx, filters)
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

	client, err := h.cockroachdbClient.GetClientByID(ctx, id)
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
		errorMessage := fmt.Sprint("Binding CreateClient body failed")
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
		errorMessage := fmt.Sprint("Binding UpdateClient body failed")
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
			pipelines, err := h.cockroachdbClient.GetPipelines(ctx, pageNumber, pageSize, filters, sortings, true)
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
			return h.cockroachdbClient.GetPipelinesCount(ctx, filters)
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

	pipeline, err := h.cockroachdbClient.GetPipeline(ctx, source, owner, repo, map[api.FilterType][]string{}, false)
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
		errorMessage := fmt.Sprint("Binding UpdatePipeline body failed")
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
		Users        []string                `json:"users"`
		Role         *string                 `json:"role"`
		Group        *contracts.Group        `json:"group"`
		Organization *contracts.Organization `json:"organization"`
	}

	err := c.BindJSON(&body)
	if err != nil {
		errorMessage := fmt.Sprint("Binding BatchUpdateUsers body failed")
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	ctx := c.Request.Context()

	if len(body.Users) == 0 || (body.Role == nil && body.Group == nil && body.Organization == nil) {
		log.Error().Err(err).Msg("Request body is incorrect")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest)})
		return
	}

	// http://jmoiron.net/blog/limiting-concurrency-in-go/
	concurrency := 10
	semaphore := make(chan bool, concurrency)

	resultChannel := make(chan error, len(body.Users))

	for _, u := range body.Users {
		// try to fill semaphore up to it's full size otherwise wait for a routine to finish
		semaphore <- true

		go func(u string) {
			// lower semaphore once the routine's finished, making room for another one to start
			defer func() { <-semaphore }()

			user, err := h.cockroachdbClient.GetUserByID(ctx, u)
			if err != nil {
				resultChannel <- err
				return
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

			err = h.service.UpdateUser(ctx, *user)
			if err != nil {
				resultChannel <- err
				return
			}

			resultChannel <- nil
		}(u)
	}

	// try to fill semaphore up to it's full size which only succeeds if all routines have finished
	for i := 0; i < cap(semaphore); i++ {
		semaphore <- true
	}

	close(resultChannel)
	for err := range resultChannel {
		if err != nil {
			log.Error().Err(err).Msg("Failed updating user")
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
			return
		}
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
		Groups       []string                `json:"groups"`
		Role         *string                 `json:"role"`
		Organization *contracts.Organization `json:"organization"`
	}

	err := c.BindJSON(&body)
	if err != nil {
		errorMessage := fmt.Sprint("Binding BatchUpdateGroups body failed")
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	ctx := c.Request.Context()

	if len(body.Groups) == 0 || (body.Role == nil && body.Organization == nil) {
		log.Error().Err(err).Msg("Request body is incorrect")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest)})
		return
	}

	// http://jmoiron.net/blog/limiting-concurrency-in-go/
	concurrency := 10
	semaphore := make(chan bool, concurrency)

	resultChannel := make(chan error, len(body.Groups))

	for _, g := range body.Groups {
		// try to fill semaphore up to it's full size otherwise wait for a routine to finish
		semaphore <- true

		go func(g string) {
			// lower semaphore once the routine's finished, making room for another one to start
			defer func() { <-semaphore }()

			group, err := h.cockroachdbClient.GetGroupByID(ctx, g)
			if err != nil {
				resultChannel <- err
				return
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

			err = h.service.UpdateGroup(ctx, *group)
			if err != nil {
				resultChannel <- err
				return
			}

			resultChannel <- nil
		}(g)
	}

	// try to fill semaphore up to it's full size which only succeeds if all routines have finished
	for i := 0; i < cap(semaphore); i++ {
		semaphore <- true
	}

	close(resultChannel)
	for err := range resultChannel {
		if err != nil {
			log.Error().Err(err).Msg("Failed updating group")
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
			return
		}
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
		Organizations []string `json:"organizations"`
		Role          *string  `json:"role"`
	}

	err := c.BindJSON(&body)
	if err != nil {
		errorMessage := fmt.Sprint("Binding BatchUpdateOrganizations body failed")
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	ctx := c.Request.Context()

	if len(body.Organizations) == 0 || body.Role == nil {
		log.Error().Err(err).Msg("Request body is incorrect")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest)})
		return
	}

	// http://jmoiron.net/blog/limiting-concurrency-in-go/
	concurrency := 10
	semaphore := make(chan bool, concurrency)

	resultChannel := make(chan error, len(body.Organizations))

	for _, o := range body.Organizations {
		// try to fill semaphore up to it's full size otherwise wait for a routine to finish
		semaphore <- true

		go func(o string) {
			// lower semaphore once the routine's finished, making room for another one to start
			defer func() { <-semaphore }()

			organization, err := h.cockroachdbClient.GetOrganizationByID(ctx, o)
			if err != nil {
				resultChannel <- err
				return
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

			err = h.service.UpdateOrganization(ctx, *organization)
			if err != nil {
				resultChannel <- err
				return
			}

			resultChannel <- nil
		}(o)
	}

	// try to fill semaphore up to it's full size which only succeeds if all routines have finished
	for i := 0; i < cap(semaphore); i++ {
		semaphore <- true
	}

	close(resultChannel)
	for err := range resultChannel {
		if err != nil {
			log.Error().Err(err).Msg("Failed updating organization")
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
			return
		}
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
		Clients []string `json:"clients"`
		Role    *string  `json:"role"`
	}

	err := c.BindJSON(&body)
	if err != nil {
		errorMessage := fmt.Sprint("Binding BatchUpdateClients body failed")
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	ctx := c.Request.Context()

	if len(body.Clients) == 0 || body.Role == nil {
		log.Error().Err(err).Msg("Request body is incorrect")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest)})
		return
	}

	// http://jmoiron.net/blog/limiting-concurrency-in-go/
	concurrency := 10
	semaphore := make(chan bool, concurrency)

	resultChannel := make(chan error, len(body.Clients))

	for _, c := range body.Clients {
		// try to fill semaphore up to it's full size otherwise wait for a routine to finish
		semaphore <- true

		go func(c string) {
			// lower semaphore once the routine's finished, making room for another one to start
			defer func() { <-semaphore }()

			client, err := h.cockroachdbClient.GetClientByID(ctx, c)
			if err != nil {
				resultChannel <- err
				return
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

			err = h.service.UpdateClient(ctx, *client)
			if err != nil {
				resultChannel <- err
				return
			}

			resultChannel <- nil
		}(c)
	}

	// try to fill semaphore up to it's full size which only succeeds if all routines have finished
	for i := 0; i < cap(semaphore); i++ {
		semaphore <- true
	}

	close(resultChannel)
	for err := range resultChannel {
		if err != nil {
			log.Error().Err(err).Msg("Failed updating client")
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
			return
		}
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
		Pipelines    []string                `json:"pipelines"`
		Group        *contracts.Group        `json:"group"`
		Organization *contracts.Organization `json:"organization"`
	}

	err := c.BindJSON(&body)
	if err != nil {
		errorMessage := fmt.Sprint("Binding BatchUpdatePipelines body failed")
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	ctx := c.Request.Context()

	if len(body.Pipelines) == 0 || (body.Group == nil && body.Organization == nil) {
		log.Error().Err(err).Msg("Request body is incorrect")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest)})
		return
	}

	// http://jmoiron.net/blog/limiting-concurrency-in-go/
	concurrency := 10
	semaphore := make(chan bool, concurrency)

	resultChannel := make(chan error, len(body.Pipelines))

	for _, p := range body.Pipelines {
		// try to fill semaphore up to it's full size otherwise wait for a routine to finish
		semaphore <- true

		go func(p string) {
			// lower semaphore once the routine's finished, making room for another one to start
			defer func() { <-semaphore }()

			// split pipeline in source, owner and name
			pipelineParts := strings.Split(p, "/")
			if len(pipelineParts) != 3 {
				resultChannel <- fmt.Errorf("Pipeline '%v' has invalid name", p)
				return
			}

			pipeline, err := h.cockroachdbClient.GetPipeline(ctx, pipelineParts[0], pipelineParts[1], pipelineParts[2], map[api.FilterType][]string{}, true)
			if err != nil {
				resultChannel <- err
				return
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

			err = h.service.UpdatePipeline(ctx, *pipeline)
			if err != nil {
				resultChannel <- err
				return
			}

			resultChannel <- nil
		}(p)
	}

	// try to fill semaphore up to it's full size which only succeeds if all routines have finished
	for i := 0; i < cap(semaphore); i++ {
		semaphore <- true
	}

	close(resultChannel)
	for err := range resultChannel {
		if err != nil {
			log.Error().Err(err).Msg("Failed updating pipeline")
			c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusText(http.StatusOK)})
}
