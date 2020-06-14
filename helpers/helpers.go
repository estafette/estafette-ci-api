package helpers

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/gin-gonic/gin"
)

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
func GetQueryParameters(c *gin.Context) (int, int, map[string][]string, []OrderField) {
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
func GetFilters(c *gin.Context) map[string][]string {
	// get filters (?filter[status]=running,succeeded&filter[since]=1w&filter[labels]=team%3Destafette-team)
	filters := map[string][]string{}
	filters["status"] = GetStatusFilter(c)
	filters["since"] = GetSinceFilter(c)
	filters["labels"] = GetLabelsFilter(c)
	filters["search"] = GetGenericFilter(c, "search")
	filters["recent-committer"] = GetGenericFilter(c, "recent-committer")
	filters["recent-releaser"] = GetGenericFilter(c, "recent-releaser")
	filters["group-id"] = GetGenericFilter(c, "group-id")
	filters["organization-id"] = GetGenericFilter(c, "organization-id")
	return filters
}

// GetStatusFilter extracts a filter on status
func GetStatusFilter(c *gin.Context, defaultValues ...string) []string {
	return GetGenericFilter(c, "status", defaultValues...)
}

// GetLastFilter extracts a filter to select last n items
func GetLastFilter(c *gin.Context, defaultValue int) []string {
	return GetGenericFilter(c, "last", strconv.Itoa(defaultValue))
}

// GetSinceFilter extracts a filter on build/release date
func GetSinceFilter(c *gin.Context) []string {
	return GetGenericFilter(c, "since", "eternity")
}

// GetLabelsFilter extracts a filter to select specific labels
func GetLabelsFilter(c *gin.Context) []string {
	return GetGenericFilter(c, "labels")
}

// GetGenericFilter extracts a filter
func GetGenericFilter(c *gin.Context, filterKey string, defaultValues ...string) []string {

	filterValues, filterExist := c.GetQueryArray(fmt.Sprintf("filter[%v]", filterKey))
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
