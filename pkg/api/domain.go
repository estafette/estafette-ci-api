package api

// {
// 	"keys" : [
// 	   {
// 		  "alg" : "ES256",
// 		  "crv" : "P-256",
// 		  "kid" : "f9R3yg",
// 		  "kty" : "EC",
// 		  "use" : "sig",
// 		  "x" : "SqCmEwytkqG6tL6a2GTQGmSNI4jHYo5MeDUs7DpETVg",
// 		  "y" : "Ql1yyBay4NrGbzasPEhp56Jy6HqoDkqkXYyQRreCOo0"
// 	   },

// IAPJSONWebKey is the IAP JWT json web key to represent the public key of the key used to encrypt the IAP JWT
type IAPJSONWebKey struct {
	Algorithm    string `json:"alg"`
	Curve        string `json:"crv"`
	KeyID        string `json:"kid"`
	KeyType      string `json:"kty"`
	PublicKeyUse string `json:"use"`
	X            string `json:"x"`
	Y            string `json:"y"`
}

// IAPJWKResponse as returned by https://www.gstatic.com/iap/verify/public_key-jwk
type IAPJWKResponse struct {
	Keys []IAPJSONWebKey `json:"keys"`
}

// {
// 	"keys": [
// 	  {
// 		"e": "AQAB",
// 		"kty": "RSA",
// 		"alg": "RS256",
// 		"n": "0QW_fsq8WFtNPeOp8cJO1zoToB_E2HBs1Y4ceJB_3qgJmATBCffGwTm7waYEgIlQbJ7fqP1ttgdab-5yQTDGrE51_KS1_3jlB_EDYZPciT3uzHo69BE0v4h9A29fG2MTR1iwkjqDuWE-JN1TNQUeYZ554WYktX1d0qnaiOhM8jNLcuU948LW9d-9xwd7NwnKD_PakCOWRUqXZVYnS7EsTMG4aZpZk0ZB-695tsH-NmwqISPfXI7sEjINRd2PdD9mvs2xAfp-T7eaCV-C3fTfoHDGB3Vwkfn1rG2p-hFB57vzUYB8vEdRgR8ehhEgWndLU6fovvVToWFnPcvkm-ZFxw",
// 		"use": "sig",
// 		"kid": "05a02649a5b45c90fdfe4da1ebefa9c079ab593e"
// 	  },

// GoogleJSONWebKey is the Google JWT json web key to represent the public key of the key used to encrypt the Google JWT
type GoogleJSONWebKey struct {
	E            string `json:"e"`
	KeyType      string `json:"kty"`
	Algorithm    string `json:"alg"`
	N            string `json:"n"`
	PublicKeyUse string `json:"use"`
	KeyID        string `json:"kid"`
}

// GoogleJWKResponse as returned by https://www.googleapis.com/oauth2/v3/certs
type GoogleJWKResponse struct {
	Keys []GoogleJSONWebKey `json:"keys"`
}

// Role is used to hand out permissions to users and clients
type Role int

const (
	// RoleAdministrator can configure role-based-access-control
	RoleAdministrator Role = iota
	// RoleCronTrigger can send a cron event
	RoleCronTrigger
	// RoleLogMigrator is needed to migrate logs from db to cloud storage and vice versa
	RoleLogMigrator
	// RoleRoleViewer allows to view available roles
	RoleRoleViewer
	// RoleUserViewer allows to view users
	RoleUserViewer
	// RoleUserAdmin allows to view, create and update users
	RoleUserAdmin
	// RoleGroupViewer allows to view groups
	RoleGroupViewer
	// RoleGroupAdmin allows to view, create and update groups
	RoleGroupAdmin
	// RoleOrganizationViewer allows to view organizations
	RoleOrganizationViewer
	// RoleOrganizationAdmin allows to view, create and update organizations
	RoleOrganizationAdmin
	// RoleClientViewer allows to view clients
	RoleClientViewer
	// RoleClientAdmin allows to view, create and update clients
	RoleClientAdmin
	// RoleOrganizationPipelinesViewer allows to view all pipelines linked to an organization
	RoleOrganizationPipelinesViewer
	// RoleOrganizationPipelinesOperator allows to operate all pipelines linked to an organization
	RoleOrganizationPipelinesOperator
	// RoleGroupPipelinesViewer allows to view all pipelines linked to a group
	RoleGroupPipelinesViewer
	// RoleGroupPipelinesOperator allows to operate all pipelines linked to a group
	RoleGroupPipelinesOperator
	// RoleCatalogEntitiesViewer allows to view all catalog entities
	RoleCatalogEntitiesViewer
	// RoleCatalogEntitiesAdmin allows to view, create, update and delete catalog entities
	RoleCatalogEntitiesAdmin
)

var roles = []string{
	"administrator",
	"cron.trigger",
	"log.migrator",
	"role.viewer",
	"user.viewer",
	"user.admin",
	"group.viewer",
	"group.admin",
	"organization.viewer",
	"organization.admin",
	"client.viewer",
	"client.admin",
	"organization.pipelines.viewer",
	"organization.pipelines.operator",
	"group.pipelines.viewer",
	"group.pipelines.operator",
	"catalog.entities.viewer",
	"catalog.entities.admin",
}

func (r Role) String() string {
	return roles[r]
}

func ToRole(r string) *Role {
	for i, n := range roles {
		if r == n {
			role := Role(i)
			return &role
		}
	}

	return nil
}

// Roles returns all allowed values for Role
func Roles() []string {
	return roles
}

// Permissions are used to secure endpoints; each role maps to one or more permissions
type Permission int

const (
	PermissionRolesList Permission = iota

	PermissionUsersList
	PermissionUsersGet
	PermissionUsersCreate
	PermissionUsersUpdate
	PermissionUsersDelete
	PermissionUsersImpersonate

	PermissionGroupsList
	PermissionGroupsGet
	PermissionGroupsCreate
	PermissionGroupsUpdate
	PermissionGroupsDelete

	PermissionOrganizationsList
	PermissionOrganizationsGet
	PermissionOrganizationsCreate
	PermissionOrganizationsUpdate
	PermissionOrganizationsDelete

	PermissionClientsList
	PermissionClientsGet
	PermissionClientsViewSecret
	PermissionClientsCreate
	PermissionClientsUpdate
	PermissionClientsDelete

	PermissionIntegrationsGet
	PermissionIntegrationsUpdate

	PermissionPipelinesList
	PermissionPipelinesGet
	PermissionPipelinesUpdate
	PermissionPipelinesArchive

	PermissionBuildsList
	PermissionBuildsGet
	PermissionBuildsCancel
	PermissionBuildsRebuild

	PermissionReleasesList
	PermissionReleasesGet
	PermissionReleasesCreate
	PermissionReleasesCancel

	PermissionCatalogEntitiesList
	PermissionCatalogEntitiesGet
	PermissionCatalogEntitiesCreate
	PermissionCatalogEntitiesUpdate
	PermissionCatalogEntitiesDelete
)

var permissions = []string{
	"rbac.roles.list",

	"rbac.users.list",
	"rbac.users.get",
	"rbac.users.create",
	"rbac.users.update",
	"rbac.users.delete",
	"rbac.users.impersonate",

	"rbac.groups.list",
	"rbac.groups.get",
	"rbac.groups.create",
	"rbac.groups.update",
	"rbac.groups.delete",

	"rbac.organizations.list",
	"rbac.organizations.get",
	"rbac.organizations.create",
	"rbac.organizations.update",
	"rbac.organizations.delete",

	"rbac.clients.list",
	"rbac.clients.get",
	"rbac.clients.create",
	"rbac.clients.update",
	"rbac.clients.delete",

	"ci.pipelines.list",
	"ci.pipelines.get",
	"ci.pipelines.update",
	"ci.pipelines.archive",

	"ci.builds.list",
	"ci.builds.get",
	"ci.builds.cancel",
	"ci.builds.rebuild",

	"ci.releases.list",
	"ci.releases.get",
	"ci.releases.create",
	"ci.releases.cancel",

	"catalog.entities.list",
	"catalog.entities.get",
	"catalog.entities.create",
	"catalog.entities.update",
	"catalog.entities.delete",
}

func (p Permission) String() string {
	return permissions[p]
}

func ToPermission(p string) *Permission {
	for i, n := range permissions {
		if p == n {
			permission := Permission(i)
			return &permission
		}
	}

	return nil
}

// Permissions returns all allowed values for Permission
func Permissions() []string {
	return permissions
}

var rolesToPermissionMap = map[Role][]Permission{
	RoleAdministrator: {
		PermissionRolesList,
		PermissionUsersList,
		PermissionUsersGet,
		PermissionUsersCreate,
		PermissionUsersUpdate,
		PermissionUsersDelete,
		// PermissionUsersImpersonate,
		PermissionGroupsList,
		PermissionGroupsGet,
		PermissionGroupsCreate,
		PermissionGroupsUpdate,
		PermissionGroupsDelete,
		PermissionOrganizationsList,
		PermissionOrganizationsGet,
		PermissionOrganizationsCreate,
		PermissionOrganizationsUpdate,
		PermissionOrganizationsDelete,
		PermissionClientsList,
		PermissionClientsGet,
		PermissionClientsViewSecret,
		PermissionClientsCreate,
		PermissionClientsUpdate,
		PermissionClientsDelete,
		PermissionIntegrationsGet,
		PermissionIntegrationsUpdate,
		PermissionPipelinesList,
		PermissionPipelinesGet,
		PermissionPipelinesUpdate,
		PermissionPipelinesArchive,
		PermissionBuildsList,
		PermissionBuildsGet,
		PermissionBuildsCancel,
		PermissionBuildsRebuild,
		PermissionReleasesList,
		PermissionReleasesGet,
		PermissionReleasesCreate,
		PermissionReleasesCancel,
		PermissionCatalogEntitiesList,
		PermissionCatalogEntitiesGet,
		PermissionCatalogEntitiesCreate,
		PermissionCatalogEntitiesUpdate,
		PermissionCatalogEntitiesDelete,
	},
	RoleRoleViewer: {
		PermissionRolesList,
	},
	RoleUserViewer: {
		PermissionUsersList,
		PermissionUsersGet,
	},
	RoleUserAdmin: {
		PermissionUsersList,
		PermissionUsersGet,
		PermissionUsersCreate,
		PermissionUsersUpdate,
		PermissionUsersDelete,
	},
	RoleGroupViewer: {
		PermissionGroupsList,
		PermissionGroupsGet,
	},
	RoleGroupAdmin: {
		PermissionGroupsList,
		PermissionGroupsGet,
		PermissionGroupsCreate,
		PermissionGroupsUpdate,
		PermissionGroupsDelete,
	},
	RoleOrganizationViewer: {
		PermissionOrganizationsList,
		PermissionOrganizationsGet,
	},
	RoleOrganizationAdmin: {
		PermissionOrganizationsList,
		PermissionOrganizationsGet,
		PermissionOrganizationsCreate,
		PermissionOrganizationsUpdate,
		PermissionOrganizationsDelete,
	},
	RoleClientViewer: {
		PermissionClientsList,
		PermissionClientsGet,
	},
	RoleClientAdmin: {
		PermissionClientsList,
		PermissionClientsGet,
		PermissionClientsViewSecret,
		PermissionClientsCreate,
		PermissionClientsUpdate,
		PermissionClientsDelete,
	},
	RoleOrganizationPipelinesViewer: {
		PermissionPipelinesList,
		PermissionPipelinesGet,
		PermissionBuildsList,
		PermissionBuildsGet,
		PermissionReleasesList,
		PermissionReleasesGet,
	},
	RoleOrganizationPipelinesOperator: {
		PermissionPipelinesList,
		PermissionPipelinesGet,
		PermissionBuildsList,
		PermissionBuildsGet,
		PermissionBuildsCancel,
		PermissionBuildsRebuild,
		PermissionReleasesList,
		PermissionReleasesGet,
		PermissionReleasesCreate,
		PermissionReleasesCancel,
	},
	RoleGroupPipelinesViewer: {
		PermissionPipelinesList,
		PermissionPipelinesGet,
		PermissionBuildsList,
		PermissionBuildsGet,
		PermissionReleasesList,
		PermissionReleasesGet,
	},
	RoleGroupPipelinesOperator: {
		PermissionPipelinesList,
		PermissionPipelinesGet,
		PermissionBuildsList,
		PermissionBuildsGet,
		PermissionBuildsCancel,
		PermissionBuildsRebuild,
		PermissionReleasesList,
		PermissionReleasesGet,
		PermissionReleasesCreate,
		PermissionReleasesCancel,
	},
	RoleCatalogEntitiesViewer: {
		PermissionCatalogEntitiesList,
		PermissionCatalogEntitiesGet,
	},
	RoleCatalogEntitiesAdmin: {
		PermissionCatalogEntitiesList,
		PermissionCatalogEntitiesGet,
		PermissionCatalogEntitiesCreate,
		PermissionCatalogEntitiesUpdate,
		PermissionCatalogEntitiesDelete,
	},
}

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
	FilterReleaseTarget
	FilterSearch
	FilterBranch
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
	FilterArchived
	FilterBotName
)

var filters = []string{
	"status",
	"since",
	"labels",
	"target",
	"search",
	"branch",
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
	"archived",
	"bot",
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
