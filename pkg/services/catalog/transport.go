package catalog

import (
	"net/http"
	"sort"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/database"
	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// NewHandler returns a new rbac.Handler
func NewHandler(config *api.APIConfig, service Service, databaseClient database.Client) Handler {
	return Handler{
		config:         config,
		service:        service,
		databaseClient: databaseClient,
	}
}

type Handler struct {
	config         *api.APIConfig
	service        Service
	databaseClient database.Client
}

func (h *Handler) GetCatalogEntityLabels(c *gin.Context) {

	// // ensure the request has the correct permission
	// if !api.RequestTokenHasPermission(c, api.PermissionCatalogEntitiesList) {
	// 	c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
	// 	return
	// }

	pageNumber, pageSize, filters, _ := api.GetQueryParameters(c)

	ctx := c.Request.Context()

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			catalogEntityKeys, err := h.databaseClient.GetCatalogEntityLabels(ctx, pageNumber, pageSize, filters)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(catalogEntityKeys))
			for i := range catalogEntityKeys {
				items[i] = catalogEntityKeys[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetCatalogEntityLabelsCount(ctx, filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving catalog entity labels from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetCatalogEntityKeys(c *gin.Context) {

	// // ensure the request has the correct permission
	// if !api.RequestTokenHasPermission(c, api.PermissionCatalogEntitiesList) {
	// 	c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
	// 	return
	// }

	pageNumber, pageSize, filters, sortings := api.GetQueryParameters(c)

	ctx := c.Request.Context()

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			catalogEntityKeys, err := h.databaseClient.GetCatalogEntityKeys(ctx, pageNumber, pageSize, filters, sortings)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(catalogEntityKeys))
			for i := range catalogEntityKeys {
				items[i] = catalogEntityKeys[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetCatalogEntityKeysCount(ctx, filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving catalog entity keys from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetCatalogEntityValues(c *gin.Context) {

	// // ensure the request has the correct permission
	// if !api.RequestTokenHasPermission(c, api.PermissionCatalogEntitiesList) {
	// 	c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
	// 	return
	// }

	pageNumber, pageSize, filters, sortings := api.GetQueryParameters(c)

	ctx := c.Request.Context()

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			catalogEntityValues, err := h.databaseClient.GetCatalogEntityValues(ctx, pageNumber, pageSize, filters, sortings)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(catalogEntityValues))
			for i := range catalogEntityValues {
				items[i] = catalogEntityValues[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetCatalogEntityValuesCount(ctx, filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving catalog entity values from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetCatalogEntityParentKeys(c *gin.Context) {

	// // ensure the request has the correct permission
	// if !api.RequestTokenHasPermission(c, api.PermissionCatalogEntitiesList) {
	// 	c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
	// 	return
	// }

	pageNumber, pageSize, filters, sortings := api.GetQueryParameters(c)

	ctx := c.Request.Context()

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			catalogEntityKeys, err := h.databaseClient.GetCatalogEntityParentKeys(ctx, pageNumber, pageSize, filters, sortings)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(catalogEntityKeys))
			for i := range catalogEntityKeys {
				items[i] = catalogEntityKeys[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetCatalogEntityParentKeysCount(ctx, filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving catalog entity parent keys from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetCatalogEntityParentValues(c *gin.Context) {

	// // ensure the request has the correct permission
	// if !api.RequestTokenHasPermission(c, api.PermissionCatalogEntitiesList) {
	// 	c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
	// 	return
	// }

	pageNumber, pageSize, filters, sortings := api.GetQueryParameters(c)

	ctx := c.Request.Context()

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			catalogEntityValues, err := h.databaseClient.GetCatalogEntityParentValues(ctx, pageNumber, pageSize, filters, sortings)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(catalogEntityValues))
			for i := range catalogEntityValues {
				items[i] = catalogEntityValues[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetCatalogEntityParentValuesCount(ctx, filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving catalog entity parent values from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetCatalogEntities(c *gin.Context) {

	// // ensure the request has the correct permission
	// if !api.RequestTokenHasPermission(c, api.PermissionCatalogEntitiesList) {
	// 	c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
	// 	return
	// }

	pageNumber, pageSize, filters, sortings := api.GetQueryParameters(c)

	ctx := c.Request.Context()

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			catalogEntities, err := h.databaseClient.GetCatalogEntities(ctx, pageNumber, pageSize, filters, sortings)
			if err != nil {
				return nil, err
			}

			// convert typed array to interface array O(n)
			items := make([]interface{}, len(catalogEntities))
			for i := range catalogEntities {
				items[i] = catalogEntities[i]
			}

			return items, nil
		},
		func() (int, error) {
			return h.databaseClient.GetCatalogEntitiesCount(ctx, filters)
		},
		pageNumber,
		pageSize)

	if err != nil {
		log.Error().Err(err).Msg("Failed retrieving catalog entities from db")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetCatalogEntity(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionCatalogEntitiesGet) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	ctx := c.Request.Context()
	id := c.Param("id")

	catalogEntity, err := h.databaseClient.GetCatalogEntityByID(ctx, id)
	if err != nil || catalogEntity == nil {
		log.Error().Err(err).Msgf("Failed retrieving catalogEntity with id %v from db", id)
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound)})
		return
	}

	c.JSON(http.StatusOK, catalogEntity)
}

func (h *Handler) CreateCatalogEntity(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionCatalogEntitiesCreate) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	var catalogEntity contracts.CatalogEntity
	err := c.BindJSON(&catalogEntity)
	if err != nil {
		errorMessage := "Binding CreateCatalogEntity body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	ctx := c.Request.Context()

	insertedCatalogEntity, err := h.service.CreateCatalogEntity(ctx, catalogEntity)
	if err != nil {
		log.Error().Err(err).Msg("Failed inserting catalog entity")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusCreated, insertedCatalogEntity)
}

func (h *Handler) UpdateCatalogEntity(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionCatalogEntitiesUpdate) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	var catalogEntity contracts.CatalogEntity
	err := c.BindJSON(&catalogEntity)
	if err != nil {
		errorMessage := "Binding UpdateCatalogEntity body failed"
		log.Error().Err(err).Msg(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest), "message": errorMessage})
		return
	}

	id := c.Param("id")
	if catalogEntity.ID != id {
		log.Error().Err(err).Msg("Catalog entity id is incorrect")
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusText(http.StatusBadRequest)})
		return
	}

	ctx := c.Request.Context()

	err = h.service.UpdateCatalogEntity(ctx, catalogEntity)
	if err != nil {
		log.Error().Err(err).Msg("Failed updating catalog entity")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusText(http.StatusOK)})
}

func (h *Handler) DeleteCatalogEntity(c *gin.Context) {

	// ensure the request has the correct permission
	if !api.RequestTokenHasPermission(c, api.PermissionCatalogEntitiesDelete) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	id := c.Param("id")
	ctx := c.Request.Context()

	err := h.service.DeleteCatalogEntity(ctx, id)
	if err != nil {
		log.Error().Err(err).Msg("Failed deleting catalog entity")
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusText(http.StatusOK)})
}

func (h *Handler) GetCatalogUsers(c *gin.Context) {

	pageNumber, pageSize, filters, sortings := api.GetQueryParameters(c)

	// filter on organizations / groups
	filters = api.SetPermissionsFilters(c, filters)

	ctx := c.Request.Context()

	response, err := api.GetPagedListResponse(
		func() ([]interface{}, error) {
			users, err := h.databaseClient.GetUsers(ctx, pageNumber, pageSize, filters, sortings)
			if err != nil {
				return nil, err
			}

			if len(sortings) == 0 {
				sort.Slice(users, func(i, j int) bool {
					return users[i].Name < users[j].Name
				})
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

func (h *Handler) GetCatalogUser(c *gin.Context) {

	ctx := c.Request.Context()
	id := c.Param("id")

	filters := api.SetPermissionsFilters(c, map[api.FilterType][]string{})

	user, err := h.databaseClient.GetUserByID(ctx, id, filters)
	if err != nil || user == nil {
		log.Error().Err(err).Msgf("Failed retrieving user with id %v from db", id)
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound)})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *Handler) GetCatalogGroups(c *gin.Context) {

	pageNumber, pageSize, filters, sortings := api.GetQueryParameters(c)

	// filter on organizations / groups
	filters = api.SetPermissionsFilters(c, filters)

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

func (h *Handler) GetCatalogGroup(c *gin.Context) {

	ctx := c.Request.Context()
	id := c.Param("id")

	filters := api.SetPermissionsFilters(c, map[api.FilterType][]string{})

	group, err := h.databaseClient.GetGroupByID(ctx, id, filters)
	if err != nil || group == nil {
		log.Error().Err(err).Msgf("Failed retrieving group with id %v from db", id)
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound)})
		return
	}

	c.JSON(http.StatusOK, group)
}
