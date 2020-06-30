package helpers

// OrderField determines sorting direction
type OrderField struct {
	FieldName string
	Direction string
}

type FilterType int

const (
	FilterStatus FilterType = iota
	FilterSince
	FilterLabels
	FilterSearch
	FilterRecentCommitter
	FilterRecentReleaser
	FilterGroupID
	FilterOrganizationID
	FilterPipeline
	FilterParent
	FilterEntity
	FilterGroups
	FilterOrganizations
	FilterLast
)

var filters = []string{
	"status",
	"since",
	"labels",
	"search",
	"recent-committer",
	"recent-releaser",
	"group-id",
	"organization-id",
	"pipeline",
	"parent",
	"entity",
	"groups",
	"organizations",
	"last",
}

func (f FilterType) String() string {
	return filters[f]
}

func ToFilter(f string) *FilterType {
	for i, n := range filters {
		if f == n {
			filter := FilterType(i)
			return &filter
		}
	}

	return nil
}

// Filters returns all allowed values for FilterType
func Filters() []string {
	return filters
}
