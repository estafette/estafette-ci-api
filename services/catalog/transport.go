package catalog

import (
	"fmt"
	"net/http"

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

func (h *Handler) GetCatalogEntityLabels(c *gin.Context) {

	// // ensure the request has the correct permission
	// if !auth.RequestTokenHasPermission(c, auth.PermissionCatalogEntitiesList) {
	// 	c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
	// 	return
	// }

	pageNumber, pageSize, filters, _ := helpers.GetQueryParameters(c)

	ctx := c.Request.Context()

	response, err := helpers.GetPagedListResponse(
		func() ([]interface{}, error) {
			catalogEntityKeys, err := h.cockroachdbClient.GetCatalogEntityLabels(ctx, pageNumber, pageSize, filters)
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
			return h.cockroachdbClient.GetCatalogEntityLabelsCount(ctx, filters)
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
	// if !auth.RequestTokenHasPermission(c, auth.PermissionCatalogEntitiesList) {
	// 	c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
	// 	return
	// }

	pageNumber, pageSize, filters, sortings := helpers.GetQueryParameters(c)

	ctx := c.Request.Context()

	response, err := helpers.GetPagedListResponse(
		func() ([]interface{}, error) {
			catalogEntityKeys, err := h.cockroachdbClient.GetCatalogEntityKeys(ctx, pageNumber, pageSize, filters, sortings)
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
			return h.cockroachdbClient.GetCatalogEntityKeysCount(ctx, filters)
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

func (h *Handler) GetCatalogEntityParentKeys(c *gin.Context) {

	// // ensure the request has the correct permission
	// if !auth.RequestTokenHasPermission(c, auth.PermissionCatalogEntitiesList) {
	// 	c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
	// 	return
	// }

	pageNumber, pageSize, filters, sortings := helpers.GetQueryParameters(c)

	ctx := c.Request.Context()

	response, err := helpers.GetPagedListResponse(
		func() ([]interface{}, error) {
			catalogEntityKeys, err := h.cockroachdbClient.GetCatalogEntityParentKeys(ctx, pageNumber, pageSize, filters, sortings)
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
			return h.cockroachdbClient.GetCatalogEntityParentKeysCount(ctx, filters)
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
func (h *Handler) GetCatalogEntities(c *gin.Context) {

	// // ensure the request has the correct permission
	// if !auth.RequestTokenHasPermission(c, auth.PermissionCatalogEntitiesList) {
	// 	c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
	// 	return
	// }

	pageNumber, pageSize, filters, sortings := helpers.GetQueryParameters(c)

	ctx := c.Request.Context()

	response, err := helpers.GetPagedListResponse(
		func() ([]interface{}, error) {
			catalogEntities, err := h.cockroachdbClient.GetCatalogEntities(ctx, pageNumber, pageSize, filters, sortings)
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
			return h.cockroachdbClient.GetCatalogEntitiesCount(ctx, filters)
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
	if !auth.RequestTokenHasPermission(c, auth.PermissionCatalogEntitiesGet) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	ctx := c.Request.Context()
	id := c.Param("id")

	catalogEntity, err := h.cockroachdbClient.GetCatalogEntityByID(ctx, id)
	if err != nil || catalogEntity == nil {
		log.Error().Err(err).Msgf("Failed retrieving catalogEntity with id %v from db", id)
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound)})
		return
	}

	c.JSON(http.StatusOK, catalogEntity)
}

func (h *Handler) CreateCatalogEntity(c *gin.Context) {

	// ensure the request has the correct permission
	if !auth.RequestTokenHasPermission(c, auth.PermissionCatalogEntitiesCreate) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	var catalogEntity contracts.CatalogEntity
	err := c.BindJSON(&catalogEntity)
	if err != nil {
		errorMessage := fmt.Sprint("Binding CreateCatalogEntity body failed")
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
	if !auth.RequestTokenHasPermission(c, auth.PermissionCatalogEntitiesUpdate) {
		c.JSON(http.StatusForbidden, gin.H{"code": http.StatusText(http.StatusForbidden), "message": "JWT is invalid or request does not have correct permission"})
		return
	}

	var catalogEntity contracts.CatalogEntity
	err := c.BindJSON(&catalogEntity)
	if err != nil {
		errorMessage := fmt.Sprint("Binding UpdateCatalogEntity body failed")
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
	if !auth.RequestTokenHasPermission(c, auth.PermissionCatalogEntitiesDelete) {
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
