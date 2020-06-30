package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFiltersToString(t *testing.T) {
	t.Run("AllFiltersCanBeConvertedToString", func(t *testing.T) {

		filters := Filters()

		for i := 0; i < len(filters); i++ {
			r := FilterType(i)
			roleAsString := r.String()

			assert.Equal(t, filters[i], roleAsString)
		}
	})

	t.Run("AllFiltersCanBeConvertedToFilterType", func(t *testing.T) {

		filters := Filters()

		for _, f := range filters {
			filter := ToFilter(f)
			assert.NotNil(t, filter)
			assert.Equal(t, f, filter.String())
		}
	})
}
