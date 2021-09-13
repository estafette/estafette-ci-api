package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
	googleoauth2v2 "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
	v1 "k8s.io/api/core/v1"
)

// APIConfig represent the configuration for the entire api application
type APIConfig struct {
	Integrations        *APIConfigIntegrations                 `yaml:"integrations,omitempty" env:",prefix=INTEGRATIONS_"`
	APIServer           *APIServerConfig                       `yaml:"apiServer,omitempty" env:",prefix=APISERVER_"`
	Auth                *AuthConfig                            `yaml:"auth,omitempty" env:",prefix=AUTH_"`
	Jobs                *JobsConfig                            `yaml:"jobs,omitempty" env:",prefix=JOBS_"`
	Database            *DatabaseConfig                        `yaml:"database,omitempty" env:",prefix=DATABASE_"`
	Queue               *QueueConfig                           `yaml:"queue,omitempty" env:",prefix=QUEUE_"`
	ManifestPreferences *manifest.EstafetteManifestPreferences `yaml:"manifestPreferences,omitempty" env:",prefix=MANIFESTPREFERENCES_"`
	Catalog             *CatalogConfig                         `yaml:"catalog,omitempty" env:",prefix=CATALOG_"`
	Credentials         []*contracts.CredentialConfig          `yaml:"credentials,omitempty" json:"credentials,omitempty"`
	TrustedImages       []*contracts.TrustedImageConfig        `yaml:"trustedImages,omitempty" json:"trustedImages,omitempty"`
}

func (c *APIConfig) SetDefaults() {

	if c.Integrations == nil {
		c.Integrations = &APIConfigIntegrations{}
	}
	c.Integrations.SetDefaults()

	if c.APIServer == nil {
		c.APIServer = &APIServerConfig{}
	}
	c.APIServer.SetDefaults()

	if c.Auth == nil {
		c.Auth = &AuthConfig{}
	}
	c.Auth.SetDefaults()

	if c.Jobs == nil {
		c.Jobs = &JobsConfig{}
	}
	c.Jobs.SetDefaults()

	if c.Database == nil {
		c.Database = &DatabaseConfig{}
	}
	c.Database.SetDefaults()

	if c.Queue == nil {
		c.Queue = &QueueConfig{}
	}
	c.Queue.SetDefaults()

	if c.ManifestPreferences == nil {
		c.ManifestPreferences = manifest.GetDefaultManifestPreferences()
	}

	if c.Catalog != nil {
		c.Catalog.SetDefaults()
	}

	if c.Credentials == nil {
		c.Credentials = make([]*contracts.CredentialConfig, 0)
	}
	// for _, credential := range c.Credentials {
	// 	credential.SetDefaults()
	// }

	if c.TrustedImages == nil || len(c.TrustedImages) == 0 {
		c.TrustedImages = make([]*contracts.TrustedImageConfig, 0)

		c.TrustedImages = append(c.TrustedImages, &contracts.TrustedImageConfig{
			ImagePath: "extensions/git-clone",
			InjectedCredentialTypes: []string{
				"bitbucket-api-token",
				"github-api-token",
				"cloudsource-api-token",
			},
		})
		c.TrustedImages = append(c.TrustedImages, &contracts.TrustedImageConfig{
			ImagePath: "extensions/github-status",
			InjectedCredentialTypes: []string{
				"github-api-token",
			},
		})
		c.TrustedImages = append(c.TrustedImages, &contracts.TrustedImageConfig{
			ImagePath: "extensions/github-release",
			InjectedCredentialTypes: []string{
				"github-api-token",
			},
		})
		c.TrustedImages = append(c.TrustedImages, &contracts.TrustedImageConfig{
			ImagePath: "extensions/bitbucket-status",
			InjectedCredentialTypes: []string{
				"bitbucket-api-token",
			},
		})
		c.TrustedImages = append(c.TrustedImages, &contracts.TrustedImageConfig{
			ImagePath:          "extensions/docker",
			RunDocker:          true,
			AllowNotifications: true,
			InjectedCredentialTypes: []string{
				"container-registry",
				"github-api-token",
			},
		})
		c.TrustedImages = append(c.TrustedImages, &contracts.TrustedImageConfig{
			ImagePath: "extensions/gke",
			InjectedCredentialTypes: []string{
				"kubernetes-engine",
			},
		})
		c.TrustedImages = append(c.TrustedImages, &contracts.TrustedImageConfig{
			ImagePath: "extensions/helm",
			InjectedCredentialTypes: []string{
				"kubernetes-engine",
			},
		})
		c.TrustedImages = append(c.TrustedImages, &contracts.TrustedImageConfig{
			ImagePath: "extensions/cloud-function",
			InjectedCredentialTypes: []string{
				"kubernetes-engine",
			},
		})
		c.TrustedImages = append(c.TrustedImages, &contracts.TrustedImageConfig{
			ImagePath:     "bsycorp/kind",
			RunPrivileged: true,
		})
	}
}

func (c *APIConfig) Validate() (err error) {

	err = c.Integrations.Validate()
	if err != nil {
		return
	}

	err = c.APIServer.Validate()
	if err != nil {
		return
	}

	err = c.Auth.Validate()
	if err != nil {
		return
	}

	err = c.Jobs.Validate()
	if err != nil {
		return
	}

	err = c.Database.Validate()
	if err != nil {
		return
	}

	if c.Catalog != nil {
		err = c.Catalog.Validate()
		if err != nil {
			return
		}
	}

	// for _, credential := range c.Credentials {
	// 	err = credential.Validate()
	// 	if err != nil {
	// 		return
	// 	}
	// }

	// for _, trustedImage := range c.TrustedImages {
	// 	err = trustedImage.Validate()
	// 	if err != nil {
	// 		return
	// 	}
	// }

	return nil
}

// APIServerConfig represents configuration for the api server
type APIServerConfig struct {
	BaseURL                                  string                                                       `yaml:"baseURL"`
	IntegrationsURL                          string                                                       `yaml:"integrationsURL"`
	ServiceURL                               string                                                       `yaml:"serviceURL"`
	LogWriters                               []LogTarget                                                  `yaml:"logWriters"`
	LogReader                                LogTarget                                                    `yaml:"logReader"`
	InjectStagesPerOperatingSystem           map[manifest.OperatingSystem]InjectStagesConfig              `yaml:"injectStagesPerOperatingSystem,omitempty"`
	InjectCommandsPerOperatingSystemAndShell map[manifest.OperatingSystem]map[string]InjectCommandsConfig `yaml:"injectCommandsPerOperatingSystemAndShell,omitempty"`
	DockerConfigPerOperatingSystem           map[manifest.OperatingSystem]contracts.DockerConfig          `yaml:"dockerConfigPerOperatingSystem,omitempty" json:"dockerConfigPerOperatingSystem,omitempty"`
}

type LogTarget string

const (
	LogTargetUnknown      LogTarget = ""
	LogTargetDatabase     LogTarget = "database"
	LogTargetCloudStorage LogTarget = "cloudstorage"
)

func (c *APIServerConfig) SetDefaults() {
	if c.ServiceURL == "" {
		c.ServiceURL = "http://estafette-ci-api.estafette-ci.svc.cluster.local"
	}
	if len(c.LogWriters) == 0 {
		c.LogWriters = []LogTarget{
			LogTargetDatabase,
		}
	}
	if c.LogReader == "" {
		c.LogReader = LogTargetDatabase
	}

	if c.DockerConfigPerOperatingSystem == nil || len(c.DockerConfigPerOperatingSystem) == 0 {
		c.DockerConfigPerOperatingSystem = map[manifest.OperatingSystem]contracts.DockerConfig{
			manifest.OperatingSystemLinux: contracts.DockerConfig{
				RunType: contracts.DockerRunTypeDinD,
			},
			manifest.OperatingSystemWindows: contracts.DockerConfig{
				RunType: contracts.DockerRunTypeDoD,
			},
		}
	}
}

func (c *APIServerConfig) Validate() (err error) {
	if c.ServiceURL == "" {
		return errors.New("Configuration item 'apiServer.baseURL' is required; please set it to the full http url for the web ui")
	}
	if c.ServiceURL == "" {
		return errors.New("Configuration item 'apiServer.serviceURL' is required; please set it to a full http url towards the estafette api, for build/release jobs to report status and send logs to")
	}
	if len(c.LogWriters) == 0 {
		return errors.New("At least on value for configuration item 'apiServer.logWriters' is required; please set it to a 'database' or 'cloudstorage' or both")
	}
	if c.LogReader == LogTargetUnknown {
		return errors.New("Configuration item 'apiServer.logReader' is required; please set it to either 'database' or 'cloudstorage'")
	}

	return nil
}

type InjectStagesConfig struct {
	Build   *InjectStagesTypeConfig `yaml:"build,omitempty"`
	Release *InjectStagesTypeConfig `yaml:"release,omitempty"`
	Bot     *InjectStagesTypeConfig `yaml:"bot,omitempty"`
}

type InjectStagesTypeConfig struct {
	Before []*manifest.EstafetteStage `yaml:"before,omitempty"`
	After  []*manifest.EstafetteStage `yaml:"after,omitempty"`
}

type InjectCommandsConfig struct {
	Before []string `yaml:"before,omitempty"`
	After  []string `yaml:"after,omitempty"`
}

// LogTargetArrayContains returns true of a value is present in the array
func LogTargetArrayContains(array []LogTarget, value LogTarget) bool {
	for _, v := range array {
		if v == value {
			return true
		}
	}
	return false
}

// WriteLogToDatabase indicates if database is in the logWriters config
func (c *APIServerConfig) WriteLogToDatabase() bool {
	return len(c.LogWriters) == 0 || LogTargetArrayContains(c.LogWriters, LogTargetDatabase)
}

// WriteLogToCloudStorage indicates if cloudstorage is in the logWriters config
func (c *APIServerConfig) WriteLogToCloudStorage() bool {
	return LogTargetArrayContains(c.LogWriters, LogTargetCloudStorage)
}

// ReadLogFromDatabase indicates if logReader config is database
func (c *APIServerConfig) ReadLogFromDatabase() bool {
	return c.LogReader == LogTargetUnknown || c.LogReader == LogTargetDatabase
}

// ReadLogFromCloudStorage indicates if logReader config is cloudstorage
func (c *APIServerConfig) ReadLogFromCloudStorage() bool {
	return c.LogReader == LogTargetCloudStorage
}

// AuthConfig determines whether to use IAP for authentication and authorization
type AuthConfig struct {
	JWT            *JWTConfig                `yaml:"jwt"`
	Administrators []string                  `yaml:"administrators"`
	Organizations  []*AuthOrganizationConfig `yaml:"organizations"`
}

func (c *AuthConfig) SetDefaults() {

	if c.JWT == nil {
		c.JWT = &JWTConfig{}
	}
	c.JWT.SetDefaults()

}

func (c *AuthConfig) Validate() (err error) {

	err = c.JWT.Validate()
	if err != nil {
		return
	}

	return nil
}

// AuthOrganizationConfig configures things relevant to each organization using the system
type AuthOrganizationConfig struct {
	Name           string           `yaml:"name"`
	OAuthProviders []*OAuthProvider `yaml:"oauthProviders"`
}

func (c *AuthOrganizationConfig) SetDefaults() {
}

func (c *AuthOrganizationConfig) Validate() (err error) {
	return nil
}

// IsConfiguredAsAdministrator returns for a user whether they're configured as administrator
func (config *AuthConfig) IsConfiguredAsAdministrator(email string) bool {
	if email == "" {
		return false
	}

	for _, a := range config.Administrators {
		if email == a {
			return true
		}
	}

	return false
}

// OAuthProvider is used to configure one or more oauth providers like google, github, microsoft
type OAuthProvider struct {
	Name                   string `yaml:"name"`
	ClientID               string `yaml:"clientID"`
	ClientSecret           string `yaml:"clientSecret"`
	AllowedIdentitiesRegex string `yaml:"allowedIdentitiesRegex"`
}

func (c *OAuthProvider) SetDefaults() {
}

func (c *OAuthProvider) Validate() (err error) {
	return nil
}

// OAuthProviderInfo provides non configurable information for oauth providers
type OAuthProviderInfo struct {
	AuthURL  string
	TokenURL string
}

// GetConfig returns the oauth config for the provider
func (p *OAuthProvider) GetConfig(baseURL string) *oauth2.Config {

	redirectPath := "/api/auth/handle/"
	redirectURI := strings.TrimSuffix(baseURL, "/") + redirectPath + p.Name

	oauthConfig := oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		RedirectURL:  redirectURI,
	}

	switch p.Name {
	case "google":
		oauthConfig.Endpoint = endpoints.Google
		oauthConfig.Scopes = []string{
			"https://www.googleapis.com/auth/userinfo.email",
		}
	case "microsoft":
		oauthConfig.Endpoint = endpoints.Microsoft
		oauthConfig.Scopes = []string{
			"user.read+openid+profile+email",
		}
	case "facebook":
		oauthConfig.Endpoint = endpoints.Facebook
	case "github":
		oauthConfig.Endpoint = endpoints.GitHub
	case "bitbucket":
		oauthConfig.Endpoint = endpoints.Bitbucket
	case "gitlab":
		oauthConfig.Endpoint = endpoints.GitLab

	default:
		return nil
	}

	return &oauthConfig
}

// AuthCodeURL returns the url to redirect to for login
func (p *OAuthProvider) AuthCodeURL(baseURL, state string) string {
	return p.GetConfig(baseURL).AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// GetUserIdentity returns the user info after a token has been retrieved
func (p *OAuthProvider) GetUserIdentity(ctx context.Context, config *oauth2.Config, token *oauth2.Token) (identity *contracts.UserIdentity, err error) {
	switch p.Name {
	case "google":
		oauth2Service, err := googleoauth2v2.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx, token)))
		if err != nil {
			return nil, err
		}

		// retrieve userinfo
		userInfo, err := oauth2Service.Userinfo.Get().Do()
		if err != nil {
			return nil, err
		}

		username := userInfo.Name
		if username == "" && (userInfo.GivenName != "" || userInfo.FamilyName != "") {
			username = strings.Trim(fmt.Sprintf("%v %v", userInfo.GivenName, userInfo.FamilyName), " ")
		}
		if username == "" {
			username = userInfo.Email
		}

		// map userinfo to user identity
		identity = &contracts.UserIdentity{
			Provider: p.Name,
			Email:    userInfo.Email,
			Name:     username,
			ID:       userInfo.Id,
			Avatar:   userInfo.Picture,
		}

		return identity, nil

	case "microsoft":

		client := config.Client(ctx, token)

		// retrieve userinfo
		resp, err := client.Get("https://graph.microsoft.com/v1.0/me")
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Calling microsoft graph returned unexpected status code %v", resp.StatusCode)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return identity, err
		}

		var userInfo struct {
			ID      string `json:"id,omitempty"`
			Email   string `json:"mail,omitempty"`
			Name    string `json:"displayName,omitempty"`
			Picture string `json:"picture,omitempty"`
		}

		if err = json.Unmarshal(body, &userInfo); err != nil {
			return nil, fmt.Errorf("Unmarshalling userinfo from microsoft graph body failed: %v", body)
		}

		// map userinfo to user identity
		identity = &contracts.UserIdentity{
			Provider: p.Name,
			Email:    userInfo.Email,
			Name:     userInfo.Name,
			ID:       userInfo.ID,
			Avatar:   userInfo.Picture,
		}

		return identity, nil
	}

	return nil, fmt.Errorf("The GetUser function has not been implemented for provider '%v'", p.Name)
}

// UserIsAllowed checks if user email address matches allowedIdentitiesRegex
func (p *OAuthProvider) UserIsAllowed(ctx context.Context, email string) (isAllowed bool, err error) {

	if p.AllowedIdentitiesRegex == "" {
		return true, nil
	}
	if email == "" {
		return false, fmt.Errorf("Email address is empty, can't check if user is allowed")
	}

	pattern := fmt.Sprintf("^%v$", strings.TrimSpace(p.AllowedIdentitiesRegex))
	match, err := regexp.MatchString(pattern, email)
	if err != nil {
		return false, err
	}

	return match, nil
}

// JWTConfig is used to configure JWT middleware
type JWTConfig struct {
	Domain string `yaml:"domain"`
	// Key to sign JWT; use 256-bit key (or 32 bytes) minimum length
	Key string `yaml:"key"`
}

func (c *JWTConfig) SetDefaults() {
}

func (c *JWTConfig) Validate() (err error) {
	if c.Domain == "" {
		return errors.New("Configuration item 'auth.jwt.domain' is required; please set it to the same host as used in 'apiServer.baseURL'")
	}
	if c.Key == "" {
		return errors.New("Configuration item 'auth.jwt.key' is required; please set it to a 256-bit key (or 32 bytes) at the minimum")
	}

	return nil
}

// JobsConfig configures the lower and upper bounds for automatically setting resources for build/release jobs
type JobsConfig struct {
	Namespace          string `yaml:"namespace"`
	ServiceAccountName string `yaml:"serviceAccount"`

	MinCPUCores     float64 `yaml:"minCPUCores"`
	DefaultCPUCores float64 `yaml:"defaultCPUCores"`
	MaxCPUCores     float64 `yaml:"maxCPUCores"`
	CPURequestRatio float64 `yaml:"cpuRequestRatio"`
	CPULimitRatio   float64 `yaml:"cpuLimitRatio"`

	MinMemoryBytes     float64 `yaml:"minMemoryBytes"`
	DefaultMemoryBytes float64 `yaml:"defaultMemoryBytes"`
	MaxMemoryBytes     float64 `yaml:"maxMemoryBytes"`
	MemoryRequestRatio float64 `yaml:"memoryRequestRatio"`
	MemoryLimitRatio   float64 `yaml:"memoryLimitRatio"`

	BuildAffinityAndTolerations   *AffinityAndTolerationsConfig `yaml:"build"`
	ReleaseAffinityAndTolerations *AffinityAndTolerationsConfig `yaml:"release"`
	BotAffinityAndTolerations     *AffinityAndTolerationsConfig `yaml:"bot"`
}

func (c *JobsConfig) SetDefaults() {

	if c.Namespace == "" {
		// get current namespace
		namespace, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err == nil {
			c.Namespace = fmt.Sprintf("%v-jobs", string(namespace))
		}
	}

	if c.ServiceAccountName == "" {
		c.ServiceAccountName = "estafette-ci-builder"
	}

	if c.MinCPUCores <= 0 {
		// 50m
		c.MinCPUCores = 0.05
	}
	if c.DefaultCPUCores <= 0 {
		// 500m
		c.DefaultCPUCores = 0.5
	}
	if c.MaxCPUCores <= 0 {
		// 1000m
		c.MaxCPUCores = 1.0
	}
	if c.CPURequestRatio <= 0 {
		c.CPURequestRatio = 1.0
	}
	if c.CPULimitRatio <= 0 {
		c.CPULimitRatio = 2.0
	}

	if c.MinMemoryBytes <= 0 {
		// 64Mi
		c.MinMemoryBytes = 67108864
	}
	if c.DefaultMemoryBytes <= 0 {
		// 256Mi
		c.DefaultMemoryBytes = 268435456
	}
	if c.MaxMemoryBytes <= 0 {
		// 8Gi
		c.MaxMemoryBytes = 8589934592
	}
	if c.MemoryRequestRatio <= 0 {
		c.MemoryRequestRatio = 1.25
	}
	if c.MemoryLimitRatio <= 0 {
		c.MemoryLimitRatio = 1.0
	}
}

func (c *JobsConfig) Validate() (err error) {
	if c.Namespace == "" {
		return errors.New("Configuration item 'jobs.namespace' is required; please set it to the namespace where you want your build/release jobs to run")
	}
	if c.ServiceAccountName == "" {
		return errors.New("Configuration item 'jobs.serviceAccount' is required; please set it to the same value as 'serviceAccount.builderServiceAccountName' in the helm chart")
	}

	if c.MinCPUCores <= 0 {
		return errors.New("Configuration item 'jobs.minCPUCores' is required; please set it to (a fraction of) the number of cpu cores you want at the minimum for a build/release job")
	}
	if c.DefaultCPUCores <= 0 {
		return errors.New("Configuration item 'jobs.minCPUCores' is required; please set it to (a fraction of) the number of cpu cores you want a build/release job to use initially; in between 'jobs.minCPUCores' and 'jobs.maxCPUCores'")
	}
	if c.MaxCPUCores <= 0 {
		return errors.New("Configuration item 'jobs.maxCPUCores' is required; please set it to (a fraction of) the number of cpu cores you want at the maximum for a build/release job")
	}
	if c.CPURequestRatio < 1.0 {
		return errors.New("Configuration item 'jobs.cpuRequestRatio' is required; please set it to 1.0 or larger")
	}
	if c.CPULimitRatio < 1.0 {
		return errors.New("Configuration item 'jobs.cpuLimitRatio' is required; please set it to 1.0 or larger")
	}

	if c.MinMemoryBytes <= 0 {
		return errors.New("Configuration item 'jobs.minMemoryBytes' is required; please set it to the number of bytes of memory you want at the minimum for a build/release job")
	}
	if c.DefaultMemoryBytes <= 0 {
		return errors.New("Configuration item 'jobs.defaultMemoryBytes' is required; please set it to the number of bytes of memory you want  a build/release job to use initially; in between 'jobs.minMemoryBytes' and 'jobs.maxMemoryBytes'")
	}
	if c.MaxMemoryBytes <= 0 {
		return errors.New("Configuration item 'jobs.maxMemoryBytes' is required; please set it to the number of bytes of memory you want at the maximum for a build/release job")
	}
	if c.MemoryRequestRatio < 1.0 {
		return errors.New("Configuration item 'jobs.memoryRequestRatio' is required; please set it to 1.0 or larger")
	}
	if c.MemoryLimitRatio < 1.0 {
		return errors.New("Configuration item 'jobs.memoryLimitRatio' is required; please set it to 1.0 or larger")
	}

	return nil
}

type AffinityAndTolerationsConfig struct {
	Affinity    *v1.Affinity    `yaml:"affinity"`
	Tolerations []v1.Toleration `yaml:"tolerations"`
}

// DatabaseConfig contains config for the dabase connection
type DatabaseConfig struct {
	DatabaseName             string `yaml:"databaseName"`
	Host                     string `yaml:"host"`
	Insecure                 bool   `yaml:"insecure"`
	SslMode                  string `yaml:"sslMode"`
	CertificateAuthorityPath string `yaml:"certificateAuthorityPath"`
	CertificatePath          string `yaml:"certificatePath"`
	CertificateKeyPath       string `yaml:"certificateKeyPath"`
	Port                     int    `yaml:"port"`
	User                     string `yaml:"user"`
	Password                 string `yaml:"password"`
	MaxOpenConns             int    `yaml:"maxOpenConnections"`
	MaxIdleConns             int    `yaml:"maxIdleConnections"`
	ConnMaxLifetimeMinutes   int    `yaml:"connectionMaxLifetimeMinutes"`
}

func (c *DatabaseConfig) SetDefaults() {
	if c.DatabaseName == "" {
		c.DatabaseName = "defaultdb"
	}
	if c.Host == "" {
		c.Host = "estafette-ci-db-public"
	}
	if c.SslMode == "" {
		c.SslMode = "verify-full"
	}
	if c.CertificateAuthorityPath == "" {
		c.CertificateAuthorityPath = "/cockroach-certs/ca.crt"
	}
	if c.CertificatePath == "" {
		c.CertificatePath = "/cockroach-certs/tls.crt"
	}
	if c.CertificateKeyPath == "" {
		c.CertificateKeyPath = "/cockroach-certs/tls.key"
	}
	if c.Port <= 0 {
		c.Port = 26257
	}
	if c.User == "" {
		c.User = "root"
	}
	if c.MaxOpenConns < 0 {
		c.MaxOpenConns = 0
	}
	if c.MaxIdleConns <= 0 {
		c.MaxIdleConns = 2
	}
	if c.ConnMaxLifetimeMinutes < 0 {
		c.ConnMaxLifetimeMinutes = 0
	}
}

func (c *DatabaseConfig) Validate() (err error) {
	if c.DatabaseName == "" {
		return errors.New("Configuration item 'database.databaseName' is required; please set it to name of the database used by the api")
	}
	if c.Host == "" {
		return errors.New("Configuration item 'database.host' is required; please set it to hostname of the database server")
	}
	if c.Insecure {
		if c.SslMode == "" {
			return errors.New("Configuration item 'database.sslMode' is required; please set it to 'verify-ca' or 'verify-full'")
		}
		if c.CertificateAuthorityPath == "" {
			return errors.New("Configuration item 'database.certificateAuthorityPath' is required; please set it to '/var/run/secrets/kubernetes.io/serviceaccount/ca.crt' or '/cockroach-certs/ca.crt'")
		}
		if c.CertificatePath == "" {
			return errors.New("Configuration item 'database.certificatePath' is required; please set it to '/cockroach-certs/tls.crt'")
		}
		if c.CertificateKeyPath == "" {
			return errors.New("Configuration item 'database.certificateKeyPath' is required; please set it to '/cockroach-certs/tls.key'")
		}
	}

	if c.Port <= 0 {
		return errors.New("Configuration item 'database.port' is required; please set it to port of the database server")
	}
	if c.User == "" {
		return errors.New("Configuration item 'database.user' is required; please set it to the database user")
	}
	if c.MaxOpenConns > 0 && c.MaxIdleConns > c.MaxOpenConns {
		return errors.New("Configuration item 'database.maxIdleConnections' needs to be less or equal to 'database.maxOpenConnections'; please set it to a valid number")
	}

	return nil
}

// QueueConfig contains config for the dabase connection
type QueueConfig struct {
	Hosts            []string `yaml:"hosts"`
	SubjectCron      string   `yaml:"subjectCron"`
	SubjectGit       string   `yaml:"subjectGit"`
	SubjectGithub    string   `yaml:"subjectGithub"`
	SubjectBitbucket string   `yaml:"subjectBitbucket"`
}

func (c *QueueConfig) SetDefaults() {
	if len(c.Hosts) == 0 {
		c.Hosts = []string{"estafette-ci-queue-0.estafette-ci-queue"}
	}
	if c.SubjectCron == "" {
		c.SubjectCron = "event.cron"
	}
	if c.SubjectGit == "" {
		c.SubjectGit = "event.git"
	}
	if c.SubjectGithub == "" {
		c.SubjectGithub = "event.github"
	}
	if c.SubjectBitbucket == "" {
		c.SubjectBitbucket = "event.bitbucket"
	}
}

func (c *QueueConfig) Validate() (err error) {
	if len(c.Hosts) == 0 {
		return errors.New("Configuration item 'queue.hosts' is required; please set it to name of the queue hosts used by the api")
	}
	if c.SubjectCron == "" {
		return errors.New("Configuration item 'queue.subjectCron' is required; please set it to subject of the queue for cron events")
	}
	if c.SubjectGit == "" {
		return errors.New("Configuration item 'queue.subjectGit' is required; please set it to subject of the queue for git events")
	}
	if c.SubjectGithub == "" {
		return errors.New("Configuration item 'queue.subjectGithub' is required; please set it to subject of the queue for github events")
	}
	if c.SubjectBitbucket == "" {
		return errors.New("Configuration item 'queue.subjectBitbucket' is required; please set it to subject of the queue for bitbucket events")
	}

	return nil
}

// CatalogConfig configures various aspect of the catalog page
type CatalogConfig struct {
	Filters []string `yaml:"filters,omitempty" json:"filters,omitempty"`
}

func (c *CatalogConfig) SetDefaults() {
}

func (c *CatalogConfig) Validate() (err error) {
	return nil
}

// APIConfigIntegrations contains config for 3rd party integrations
type APIConfigIntegrations struct {
	Github       *GithubConfig       `yaml:"github,omitempty" env:",prefix=GITHUB_"`
	Bitbucket    *BitbucketConfig    `yaml:"bitbucket,omitempty" env:",prefix=BITBUCKET_"`
	Slack        *SlackConfig        `yaml:"slack,omitempty" env:",prefix=SLACK_"`
	Pubsub       *PubsubConfig       `yaml:"pubsub,omitempty" env:",prefix=PUBSUB_"`
	Prometheus   *PrometheusConfig   `yaml:"prometheus,omitempty" env:",prefix=PROMETHEUS_"`
	BigQuery     *BigQueryConfig     `yaml:"bigquery,omitempty" env:",prefix=BIGQUERY_"`
	CloudStorage *CloudStorageConfig `yaml:"gcs,omitempty" env:",prefix=CLOUDSTORAGE_"`
	CloudSource  *CloudSourceConfig  `yaml:"cloudsource,omitempty" env:",prefix=CLOUDSOURCE_"`
}

func (c *APIConfigIntegrations) SetDefaults() {

	if c.Github == nil {
		c.Github = &GithubConfig{}
	}
	c.Github.SetDefaults()

	if c.Bitbucket == nil {
		c.Bitbucket = &BitbucketConfig{}
	}
	c.Bitbucket.SetDefaults()

	if c.Slack == nil {
		c.Slack = &SlackConfig{}
	}
	c.Slack.SetDefaults()

	if c.Pubsub == nil {
		c.Pubsub = &PubsubConfig{}
	}
	c.Pubsub.SetDefaults()

	if c.Github == nil {
		c.Github = &GithubConfig{}
	}
	c.Github.SetDefaults()

	if c.Prometheus == nil {
		c.Prometheus = &PrometheusConfig{}
	}
	c.Prometheus.SetDefaults()

	if c.BigQuery == nil {
		c.BigQuery = &BigQueryConfig{}
	}
	c.BigQuery.SetDefaults()

	if c.CloudStorage == nil {
		c.CloudStorage = &CloudStorageConfig{}
	}
	c.CloudStorage.SetDefaults()

	if c.CloudSource == nil {
		c.CloudSource = &CloudSourceConfig{}
	}
	c.CloudSource.SetDefaults()
}

func (c *APIConfigIntegrations) Validate() (err error) {

	err = c.Github.Validate()
	if err != nil {
		return
	}

	err = c.Bitbucket.Validate()
	if err != nil {
		return
	}

	err = c.Slack.Validate()
	if err != nil {
		return
	}

	err = c.Pubsub.Validate()
	if err != nil {
		return
	}

	err = c.Github.Validate()
	if err != nil {
		return
	}

	err = c.Prometheus.Validate()
	if err != nil {
		return
	}

	err = c.BigQuery.Validate()
	if err != nil {
		return
	}

	err = c.CloudStorage.Validate()
	if err != nil {
		return
	}

	err = c.CloudSource.Validate()
	if err != nil {
		return
	}

	return nil
}

// GithubConfig is used to configure github integration
type GithubConfig struct {
	Enable bool `yaml:"enable"`
}

func (c *GithubConfig) SetDefaults() {
	if !c.Enable {
		return
	}
}

func (c *GithubConfig) Validate() (err error) {
	if !c.Enable {
		return nil
	}

	return nil
}

// InstallationOrganizations is used to assign organizations to builds triggered through a specific installation
type InstallationOrganizations struct {
	Installation  int                       `yaml:"installation"`
	Organizations []*contracts.Organization `yaml:"organizations"`
}

// BitbucketConfig is used to configure bitbucket integration
type BitbucketConfig struct {
	Enable bool `yaml:"enable"`
}

func (c *BitbucketConfig) SetDefaults() {

}

func (c *BitbucketConfig) Validate() (err error) {
	if !c.Enable {
		return nil
	}

	return nil
}

// OwnerOrganizations is used to assign organizations to builds triggered through a specific owner
type OwnerOrganizations struct {
	Owner         string                    `yaml:"owner"`
	Organizations []*contracts.Organization `yaml:"organizations"`
}

// SlackConfig is used to configure slack integration
type SlackConfig struct {
	Enable               bool   `yaml:"enable"`
	ClientID             string `yaml:"clientID"`
	ClientSecret         string `yaml:"clientSecret"`
	AppVerificationToken string `yaml:"appVerificationToken"`
	AppOAuthAccessToken  string `yaml:"appOAuthAccessToken"`
}

func (c *SlackConfig) SetDefaults() {
	if !c.Enable {
		return
	}
}

func (c *SlackConfig) Validate() (err error) {
	if !c.Enable {
		return nil
	}

	if c.ClientID == "" {
		return errors.New("Configuration item 'integrations.slack.clientID' is required; please set it to a Slack client id")
	}
	if c.ClientSecret == "" {
		return errors.New("Configuration item 'integrations.slack.clientSecret' is required; please set it to a Slack client secret")
	}
	if c.AppVerificationToken == "" {
		return errors.New("Configuration item 'integrations.slack.appVerificationToken' is required; please set it to a Slack app verification token")
	}
	if c.AppOAuthAccessToken == "" {
		return errors.New("Configuration item 'integrations.slack.appOAuthAccessToken' is required; please set it to a Slack app OAuth access token")
	}

	return nil
}

// PubsubConfig is used to be able to subscribe to pub/sub topics for triggering pipelines based on pub/sub events
type PubsubConfig struct {
	Enable                         bool   `yaml:"enable"`
	DefaultProject                 string `yaml:"defaultProject"`
	Endpoint                       string `yaml:"endpoint"`
	Audience                       string `yaml:"audience"`
	ServiceAccountEmail            string `yaml:"serviceAccountEmail"`
	SubscriptionNameSuffix         string `yaml:"subscriptionNameSuffix"`
	SubscriptionIdleExpirationDays int    `yaml:"subscriptionIdleExpirationDays"`
}

func (c *PubsubConfig) SetDefaults() {
	if !c.Enable {
		return
	}

	if c.Audience == "" {
		c.Audience = "beautiful-butterfly"
	}
	if c.SubscriptionNameSuffix == "" {
		c.SubscriptionNameSuffix = "~estafette-ci-pubsub-trigger"
	}
	if c.SubscriptionIdleExpirationDays <= 0 {
		c.SubscriptionIdleExpirationDays = 365
	}
}

func (c *PubsubConfig) Validate() (err error) {
	if !c.Enable {
		return nil
	}

	if c.DefaultProject == "" {
		return errors.New("Configuration item 'integrations.pubsub.defaultProject' is required; please set it to the Google Cloud project id where this server runs")
	}
	if c.Endpoint == "" {
		return errors.New("Configuration item 'integrations.pubsub.endpoint' is required; please set it to the public http endpoint of Estafette CI suffixed by path /api/integrations/pubsub/events")
	}
	if c.Audience == "" {
		return errors.New("Configuration item 'integrations.pubsub.audience' is required; please set it to an random name for the audience")
	}
	if c.ServiceAccountEmail == "" {
		return errors.New("Configuration item 'integrations.pubsub.serviceAccountEmail' is required; please set it to the service account id used by Estafette CI to create Pub/Sub triggers")
	}
	if c.SubscriptionNameSuffix == "" {
		return errors.New("Configuration item 'integrations.pubsub.subscriptionNameSuffix' is required; please set it to a suffix with characters allowed in a Pub/Sub subscription name")
	}
	if c.SubscriptionIdleExpirationDays <= 0 {
		return errors.New("Configuration item 'integrations.pubsub.subscriptionIdleExpirationDays' is required; please set it to a number of days the subscription will survive without any messages")
	}

	return nil
}

// CloudStorageConfig is used to configure a google cloud storage bucket to be used to store logs
type CloudStorageConfig struct {
	Enable        bool   `yaml:"enable"`
	ProjectID     string `yaml:"projectID"`
	Bucket        string `yaml:"bucket"`
	LogsDirectory string `yaml:"logsDir"`
}

func (c *CloudStorageConfig) SetDefaults() {
	if !c.Enable {
		return
	}

	if c.LogsDirectory == "" {
		c.LogsDirectory = "logs"
	}
}

func (c *CloudStorageConfig) Validate() (err error) {
	if !c.Enable {
		return nil
	}

	if c.ProjectID == "" {
		return errors.New("Configuration item 'integrations.gcs.projectID' is required; please set it to a Google Cloud project id where the cloud storage bucket you want to write to is located")
	}
	if c.Bucket == "" {
		return errors.New("Configuration item 'integrations.gcs.bucket' is required; please set it to a Google Cloud Storage bucket name you want to write logs to")
	}
	if c.LogsDirectory == "" {
		return errors.New("Configuration item 'integrations.gcs.logsDir' is required; please set it to the directory within the Google Cloud Storage bucket you want to write logs to")
	}

	return nil
}

// PrometheusConfig configures where to find prometheus for retrieving max cpu and memory consumption of build and release jobs
type PrometheusConfig struct {
	Enable                *bool  `yaml:"enable"`
	ServerURL             string `yaml:"serverURL"`
	ScrapeIntervalSeconds int    `yaml:"scrapeIntervalSeconds"`
}

func (c *PrometheusConfig) SetDefaults() {
	if c.Enable == nil {
		defaultValue := true
		c.Enable = &defaultValue
	}

	if !*c.Enable {
		return
	}

	if c.ServerURL == "" {
		c.ServerURL = "http://estafette-ci-metrics-server"
	}
	if c.ScrapeIntervalSeconds <= 0 {
		c.ScrapeIntervalSeconds = 5
	}
}

func (c *PrometheusConfig) Validate() (err error) {
	if !*c.Enable {
		return nil
	}

	if c.ServerURL == "" {
		return errors.New("Configuration item 'integrations.prometheus.serverURL' is required; please set it to the http url towards your prometheus server")
	}
	if c.ScrapeIntervalSeconds <= 0 {
		return errors.New("Configuration item 'integrations.prometheus.scrapeIntervalSeconds' is required; please set it to a number of seconds larger than 0")
	}

	return nil
}

// BigQueryConfig configures the dataset where to send bigquery events
type BigQueryConfig struct {
	Enable    bool   `yaml:"enable"`
	ProjectID string `yaml:"projectID"`
	Dataset   string `yaml:"dataset"`
}

func (c *BigQueryConfig) SetDefaults() {
	if !c.Enable {
		return
	}

	if c.Dataset == "" {
		c.Dataset = "estafette_ci"
	}
}

func (c *BigQueryConfig) Validate() (err error) {
	if !c.Enable {
		return nil
	}

	if c.ProjectID == "" {
		return errors.New("Configuration item 'integrations.bigquery.projectID' is required; please set it to a Google Cloud project id where you want the BigQuery data to get written to")
	}
	if c.Dataset == "" {
		return errors.New("Configuration item 'integrations.bigquery.dataset' is required; please set it to a BigQuery dataset name for a BigQuery table to get created in and written to")
	}

	return nil
}

// CloudSourceConfig is used to configure cloudSource integration
type CloudSourceConfig struct {
	Enable               bool                   `yaml:"enable"`
	ProjectOrganizations []ProjectOrganizations `yaml:"projectOrganizations"`
}

func (c *CloudSourceConfig) SetDefaults() {
	if !c.Enable {
		return
	}
}

func (c *CloudSourceConfig) Validate() (err error) {
	if !c.Enable {
		return nil
	}

	return nil
}

// ProjectOrganizations is used to assign organizations to builds triggered through a specific project
type ProjectOrganizations struct {
	Project       string                    `yaml:"project"`
	Organizations []*contracts.Organization `yaml:"organizations"`
}

// UnmarshalYAML customizes unmarshalling an AffinityAndTolerationsConfig
func (c *AffinityAndTolerationsConfig) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {

	var aux struct {
		Affinity    map[string]interface{} `json:"affinity,omitempty" yaml:"affinity"`
		Tolerations []interface{}          `json:"tolerations,omitempty" yaml:"tolerations"`
	}

	// unmarshal to auxiliary type
	if err := unmarshal(&aux); err != nil {
		return err
	}

	// fix for map[interface{}]interface breaking json.marshal - see https://github.com/go-yaml/yaml/issues/139
	aux.Affinity = cleanUpStringMap(aux.Affinity)
	aux.Tolerations = cleanUpInterfaceArray(aux.Tolerations)

	// marshal to json
	jsonBytes, err := json.Marshal(aux)
	if err != nil {
		return err
	}

	var auxJson struct {
		Affinity    *v1.Affinity    `json:"affinity"`
		Tolerations []v1.Toleration `json:"tolerations"`
	}

	err = json.Unmarshal(jsonBytes, &auxJson)
	if err != nil {
		return err
	}

	c.Affinity = auxJson.Affinity
	c.Tolerations = auxJson.Tolerations

	return nil
}

func cleanUpStringMap(in map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range in {
		result[fmt.Sprintf("%v", k)] = cleanUpMapValue(v)
	}
	return result
}

func cleanUpInterfaceMap(in map[interface{}]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range in {
		result[fmt.Sprintf("%v", k)] = cleanUpMapValue(v)
	}
	return result
}

func cleanUpMapValue(v interface{}) interface{} {
	switch v := v.(type) {
	case []interface{}:
		return cleanUpInterfaceArray(v)
	case map[interface{}]interface{}:
		return cleanUpInterfaceMap(v)
	case string:
		return v
	case int:
		return v
	case bool:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func cleanUpInterfaceArray(in []interface{}) []interface{} {
	result := make([]interface{}, len(in))
	for i, v := range in {
		result[i] = cleanUpMapValue(v)
	}
	return result
}
