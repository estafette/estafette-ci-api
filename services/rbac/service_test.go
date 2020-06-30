package rbac

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDedupeRoles(t *testing.T) {

	t.Run("DedupesDoubleRoles", func(t *testing.T) {

		role1 := "administrator"
		role2 := "group.pipelines.operator"
		role3 := "organization.pipelines.operator"
		role4 := "organization.pipelines.viewer"
		role5 := "group.pipelines.operator"
		role6 := "organization.pipelines.viewer"
		role7 := "organization.pipelines.viewer"

		retrievedRoles := []*string{
			&role1,
			&role2,
			&role3,
			&role4,
			&role5,
			&role6,
			&role7,
		}

		service := &service{}

		// act
		roles := service.dedupeRoles(retrievedRoles)

		assert.Equal(t, 4, len(roles))
	})
}
