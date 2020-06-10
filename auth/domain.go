package auth

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
	// Administrator can configure role-based-access-control
	Administrator Role = iota
	// CronTrigger can send a cron event
	CronTrigger
	// LogMigrator is needed to migrate logs from db to cloud storage and vice versa
	LogMigrator
	// RoleViewer allows to view available roles
	RoleViewer
	// UserViewer allows to view users
	UserViewer
	// UserAdmin allows to view, create and update users
	UserAdmin
	// GroupViewer allows to view groups
	GroupViewer
	// GroupAdmin allows to view, create and update groups
	GroupAdmin
	// OrganizationViewer allows to view organizations
	OrganizationViewer
	// OrganizationAdmin allows to view, create and update organizations
	OrganizationAdmin
	// ClientViewer allows to view clients
	ClientViewer
	// ClientAdmin allows to view, create and update clients
	ClientAdmin
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
}

func (r Role) String() string {
	return roles[r]
}

// Roles returns all allowed values for Role
func Roles() []string {
	return roles
}

// Permissions are used to secure endpoints; each role maps to one or more permissions
type Permission int

const (
	RolesList Permission = iota

	UsersList
	UsersGet
	UsersCreate
	UsersUpdate
	UsersDelete

	GroupsList
	GroupsGet
	GroupsCreate
	GroupsUpdate
	GroupsDelete

	OrganizationsList
	OrganizationsGet
	OrganizationsCreate
	OrganizationsUpdate
	OrganizationsDelete

	ClientsList
	ClientsGet
	ClientsViewSecret
	ClientsCreate
	ClientsUpdate
	ClientsDelete

	PipelinesList
	PipelinesGet

	BuildsList
	BuildsGet
	BuildsCancel
	BuildsRebuild

	ReleasesList
	ReleasesGet
	ReleasesCreate
	ReleasesCancel
)

var permissions = []string{
	"rbac.roles.list",

	"rbac.users.list",
	"rbac.users.get",
	"rbac.users.create",
	"rbac.users.update",
	"rbac.users.delete",

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

	"ci.builds.list",
	"ci.builds.get",
	"ci.builds.cancel",
	"ci.builds.rebuild",

	"ci.releases.list",
	"ci.releases.get",
	"ci.releases.create",
	"ci.releases.cancel",
}

func (p Permission) String() string {
	return permissions[p]
}

// Permissions returns all allowed values for Permission
func Permissions() []string {
	return permissions
}
