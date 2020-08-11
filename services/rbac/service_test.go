package rbac

import (
	"testing"

	contracts "github.com/estafette/estafette-ci-contracts"
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

func TestDedupeOrganizations(t *testing.T) {

	t.Run("DedupesDoubleOrganizationsOnID", func(t *testing.T) {

		org1 := contracts.Organization{
			ID:   "123323",
			Name: "Org A",
		}
		org2 := contracts.Organization{
			ID:   "349430943",
			Name: "Org B",
		}
		org3 := contracts.Organization{
			ID:   "123323",
			Name: "org a",
		}

		retrievedOrganizations := []*contracts.Organization{
			&org1,
			&org2,
			&org3,
		}

		service := &service{}

		// act
		organizations := service.dedupeOrganizations(retrievedOrganizations)

		assert.Equal(t, 2, len(organizations))
	})
}
