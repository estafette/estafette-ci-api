package rbac

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/estafette/estafette-ci-api/auth"
	"github.com/estafette/estafette-ci-api/clients/cockroachdb"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/helpers"
	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// NewHandler returns a new rbac.Handler
func NewHandler(config *config.APIConfig, service Service, cockroachdbClient cockroachdb.Client) Handler {
	return Handler{
		config:            config,
		service:           service,
		cockroachdbClient: cockroachdbClient,
	}
}

type Handler struct {
	config            *config.APIConfig
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

	// ensure the user has administrator role
	if !auth.RequestTokenHasRole(c, auth.Administrator) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or user does not have administrator role"})
		return
	}

	ctx := c.Request.Context()
	roles, err := h.service.GetRoles(ctx)

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

	ctx := c.Request.Context()

	name := c.Param("provider")

	provider, err := h.service.GetProviderByName(ctx, name)
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

	// generate jwt to use as state
	state, err := auth.GenerateJWT(h.config, time.Duration(10)*time.Minute, optionalClaims)
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
		claims, err := auth.GetClaimsFromJWT(h.config, state)
		if err != nil {
			return nil, err
		}

		// get optional return url from claim and store in gin context to use in LoginResponse handler for jwt middleware
		if returnURL, ok := claims["returnURL"]; ok {
			c.Set("returnURL", returnURL)
		}

		ctx := c.Request.Context()

		// retrieve configured providers
		provider, err := h.service.GetProviderByName(ctx, name)
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

		// upsert user
		user, err := h.cockroachdbClient.GetUserByIdentity(ctx, *identity)
		if err != nil && errors.Is(err, cockroachdb.ErrUserNotFound) {
			user, err = h.service.CreateUser(ctx, *identity)
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

		// check if email matches configured administrators and add/remove administrator role correspondingly
		isAdministrator := false
		for _, a := range h.config.Auth.Administrators {
			if identity.Email == a {
				isAdministrator = true
				break
			}
		}

		if isAdministrator {
			// ensure user has administrator role
			user.AddRole(auth.Administrator.String())
		} else {
			// ensure user does not have administrator role
			user.RemoveRole(auth.Administrator.String())
		}

		go func(user contracts.User) {
			err = h.service.UpdateUser(ctx, user)
			if err != nil {
				log.Warn().Err(err).Msg("Failed updating user in db")
			}
		}(*user)

		return user, nil
	}
}

func (h *Handler) HandleClientLoginProviderAuthenticator() func(c *gin.Context) (interface{}, error) {
	return func(c *gin.Context) (interface{}, error) {

		var client contracts.Client
		err := c.BindJSON(&client)
		if err != nil {
			return nil, err
		}

		ctx := c.Request.Context()

		// get client from db by clientID
		clientFromDB, err := h.cockroachdbClient.GetClientByClientID(ctx, client.ClientID)
		if err != nil {
			return nil, err
		}
		if clientFromDB == nil {
			return nil, fmt.Errorf("Client from db is nil")
		}

		// see if secret from binded data and database match
		if client.ClientSecret != clientFromDB.ClientSecret {
			return nil, fmt.Errorf("Client secret does not match")
		}

		// check if client is active
		if !client.Active {
			return nil, fmt.Errorf("Client is not active")
		}

		return client, nil
	}
}
func (h *Handler) GetUsers(c *gin.Context) {

	pageNumber, pageSize, filters, sortings := helpers.GetQueryParameters(c)

	// ensure the user has administrator role
	if !auth.RequestTokenHasRole(c, auth.Administrator) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or user does not have administrator role"})
		return
	}

	ctx := c.Request.Context()

	response, err := helpers.GetPagedListResponse(
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

func (h *Handler) GetGroups(c *gin.Context) {

	pageNumber, pageSize, filters, sortings := helpers.GetQueryParameters(c)

	// ensure the user has administrator role
	if !auth.RequestTokenHasRole(c, auth.Administrator) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or user does not have administrator role"})
		return
	}

	ctx := c.Request.Context()

	response, err := helpers.GetPagedListResponse(
		func() ([]interface{}, error) {
			groups, err := h.cockroachdbClient.GetGroups(ctx, pageNumber, pageSize, filters, sortings)
			if err != nil {
				return nil, err
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

func (h *Handler) GetOrganizations(c *gin.Context) {

	pageNumber, pageSize, filters, sortings := helpers.GetQueryParameters(c)

	// ensure the user has administrator role
	if !auth.RequestTokenHasRole(c, auth.Administrator) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or user does not have administrator role"})
		return
	}

	ctx := c.Request.Context()

	response, err := helpers.GetPagedListResponse(
		func() ([]interface{}, error) {
			organizations, err := h.cockroachdbClient.GetOrganizations(ctx, pageNumber, pageSize, filters, sortings)
			if err != nil {
				return nil, err
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

func (h *Handler) GetClients(c *gin.Context) {

	pageNumber, pageSize, filters, sortings := helpers.GetQueryParameters(c)

	// ensure the user has administrator role
	if !auth.RequestTokenHasRole(c, auth.Administrator) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or user does not have administrator role"})
		return
	}

	ctx := c.Request.Context()

	response, err := helpers.GetPagedListResponse(
		func() ([]interface{}, error) {
			clients, err := h.cockroachdbClient.GetClients(ctx, pageNumber, pageSize, filters, sortings)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(clients))
			for i := range clients {
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

func (h *Handler) GetUser(c *gin.Context) {

	// ensure the user has administrator role
	if !auth.RequestTokenHasRole(c, auth.Administrator) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or user does not have administrator role"})
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

func (h *Handler) GetGroup(c *gin.Context) {

	// ensure the user has administrator role
	if !auth.RequestTokenHasRole(c, auth.Administrator) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or user does not have administrator role"})
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

func (h *Handler) GetOrganization(c *gin.Context) {

	// ensure the user has administrator role
	if !auth.RequestTokenHasRole(c, auth.Administrator) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or user does not have administrator role"})
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

func (h *Handler) GetClient(c *gin.Context) {

	// ensure the user has administrator role
	if !auth.RequestTokenHasRole(c, auth.Administrator) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or user does not have administrator role"})
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

	c.JSON(http.StatusOK, client)
}

func (h *Handler) CreateGroup(c *gin.Context) {

	// ensure the user has administrator role
	if !auth.RequestTokenHasRole(c, auth.Administrator) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or user does not have administrator role"})
		return
	}

	var group contracts.Group
	err := c.BindJSON(&group)
	if err != nil {
		log.Error().Err(err).Msg("Failed binding group form data")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest)})
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

	// ensure the user has administrator role
	if !auth.RequestTokenHasRole(c, auth.Administrator) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or user does not have administrator role"})
		return
	}

	var group contracts.Group
	err := c.BindJSON(&group)
	if err != nil {
		log.Error().Err(err).Msg("Failed binding group form data")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest)})
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

func (h *Handler) CreateOrganization(c *gin.Context) {

	// ensure the user has administrator role
	if !auth.RequestTokenHasRole(c, auth.Administrator) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or user does not have administrator role"})
		return
	}

	var organization contracts.Organization
	err := c.BindJSON(&organization)
	if err != nil {
		log.Error().Err(err).Msg("Failed binding organization form data")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest)})
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

	// ensure the user has administrator role
	if !auth.RequestTokenHasRole(c, auth.Administrator) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or user does not have administrator role"})
		return
	}

	var organization contracts.Organization
	err := c.BindJSON(&organization)
	if err != nil {
		log.Error().Err(err).Msg("Failed binding organization form data")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest)})
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

func (h *Handler) CreateClient(c *gin.Context) {

	// ensure the user has administrator role
	if !auth.RequestTokenHasRole(c, auth.Administrator) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or user does not have administrator role"})
		return
	}

	var client contracts.Client
	err := c.BindJSON(&client)
	if err != nil {
		log.Error().Err(err).Msg("Failed binding client form data")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest)})
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

	// ensure the user has administrator role
	if !auth.RequestTokenHasRole(c, auth.Administrator) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or user does not have administrator role"})
		return
	}

	var client contracts.Client
	err := c.BindJSON(&client)
	if err != nil {
		log.Error().Err(err).Msg("Failed binding client form data")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest)})
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
