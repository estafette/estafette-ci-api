package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	contracts "github.com/estafette/estafette-ci-contracts"
	crypt "github.com/estafette/estafette-ci-crypt"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
	googleoauth2v2 "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
	yaml "gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
)

// APIConfig represent the configuration for the entire api application
type APIConfig struct {
	Integrations        *APIConfigIntegrations                 `yaml:"integrations,omitempty"`
	APIServer           *APIServerConfig                       `yaml:"apiServer,omitempty"`
	Auth                *AuthConfig                            `yaml:"auth,omitempty"`
	Jobs                *JobsConfig                            `yaml:"jobs,omitempty"`
	Database            *DatabaseConfig                        `yaml:"database,omitempty"`
	ManifestPreferences *manifest.EstafetteManifestPreferences `yaml:"manifestPreferences,omitempty"`
	Catalog             *CatalogConfig                         `yaml:"catalog,omitempty"`
	Credentials         []*contracts.CredentialConfig          `yaml:"credentials,omitempty" json:"credentials,omitempty"`
	TrustedImages       []*contracts.TrustedImageConfig        `yaml:"trustedImages,omitempty" json:"trustedImages,omitempty"`
	RegistryMirror      *string                                `yaml:"registryMirror,omitempty" json:"registryMirror,omitempty"`
	DockerDaemonMTU     *string                                `yaml:"dindMtu,omitempty" json:"dindMtu,omitempty"`
	DockerDaemonBIP     *string                                `yaml:"dindBip,omitempty" json:"dindBip,omitempty"`
	DockerNetwork       *contracts.DockerNetworkConfig         `yaml:"dindNetwork,omitempty" json:"dindNetwork,omitempty"`
}

// APIServerConfig represents configuration for the api server
type APIServerConfig struct {
	BaseURL      string              `yaml:"baseURL"`
	ServiceURL   string              `yaml:"serviceURL"`
	LogWriters   []string            `yaml:"logWriters"`
	LogReader    string              `yaml:"logReader"`
	InjectStages *InjectStagesConfig `yaml:"injectStages,omitempty"`
}

type InjectStagesConfig struct {
	Build   *InjectStagesTypeConfig `yaml:"build,omitempty"`
	Release *InjectStagesTypeConfig `yaml:"release,omitempty"`
}

type InjectStagesTypeConfig struct {
	Before []*manifest.EstafetteStage `yaml:"before,omitempty"`
	After  []*manifest.EstafetteStage `yaml:"after,omitempty"`
}

// WriteLogToDatabase indicates if database is in the logWriters config
func (c *APIServerConfig) WriteLogToDatabase() bool {
	return len(c.LogWriters) == 0 || StringArrayContains(c.LogWriters, "database")
}

// WriteLogToCloudStorage indicates if cloudstorage is in the logWriters config
func (c *APIServerConfig) WriteLogToCloudStorage() bool {
	return StringArrayContains(c.LogWriters, "cloudstorage")
}

// ReadLogFromDatabase indicates if logReader config is database
func (c *APIServerConfig) ReadLogFromDatabase() bool {
	return c.LogReader == "" || c.LogReader == "database"
}

// ReadLogFromCloudStorage indicates if logReader config is cloudstorage
func (c *APIServerConfig) ReadLogFromCloudStorage() bool {
	return c.LogReader == "cloudstorage"
}

// AuthConfig determines whether to use IAP for authentication and authorization
type AuthConfig struct {
	JWT            *JWTConfig                `yaml:"jwt"`
	Administrators []string                  `yaml:"administrators"`
	Organizations  []*AuthOrganizationConfig `yaml:"organizations"`
}

// AuthOrganizationConfig configures things relevant to each organization using the system
type AuthOrganizationConfig struct {
	Name           string           `yaml:"name"`
	OAuthProviders []*OAuthProvider `yaml:"oauthProviders"`
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

// JobsConfig configures the lower and upper bounds for automatically setting resources for build/release jobs
type JobsConfig struct {
	Namespace string `yaml:"namespace"`

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
}

type AffinityAndTolerationsConfig struct {
	Affinity    *v1.Affinity    `yaml:"affinity"`
	Tolerations []v1.Toleration `yaml:"tolerations"`
}

// DatabaseConfig contains config for the dabase connection
type DatabaseConfig struct {
	DatabaseName   string `yaml:"databaseName"`
	Host           string `yaml:"host"`
	Insecure       bool   `yaml:"insecure"`
	CertificateDir string `yaml:"certificateDir"`
	Port           int    `yaml:"port"`
	User           string `yaml:"user"`
	Password       string `yaml:"password"`
}

// CatalogConfig configures various aspect of the catalog page
type CatalogConfig struct {
	Filters []string `yaml:"filters,omitempty" json:"filters,omitempty"`
}

// APIConfigIntegrations contains config for 3rd party integrations
type APIConfigIntegrations struct {
	Github       *GithubConfig       `yaml:"github,omitempty"`
	Bitbucket    *BitbucketConfig    `yaml:"bitbucket,omitempty"`
	Slack        *SlackConfig        `yaml:"slack,omitempty"`
	Pubsub       *PubsubConfig       `yaml:"pubsub,omitempty"`
	Prometheus   *PrometheusConfig   `yaml:"prometheus,omitempty"`
	BigQuery     *BigQueryConfig     `yaml:"bigquery,omitempty"`
	CloudStorage *CloudStorageConfig `yaml:"gcs,omitempty"`
	CloudSource  *CloudSourceConfig  `yaml:"cloudsource,omitempty"`
}

// GithubConfig is used to configure github integration
type GithubConfig struct {
	PrivateKeyPath            string                      `yaml:"privateKeyPath"`
	AppID                     string                      `yaml:"appID"`
	ClientID                  string                      `yaml:"clientID"`
	ClientSecret              string                      `yaml:"clientSecret"`
	WebhookSecret             string                      `yaml:"webhookSecret"`
	AllowedInstallations      []int                       `yaml:"allowedInstallations"`
	InstallationOrganizations []InstallationOrganizations `yaml:"installationOrganizations"`
}

// InstallationOrganizations is used to assign organizations to builds triggered through a specific installation
type InstallationOrganizations struct {
	Installation  int                       `yaml:"installation"`
	Organizations []*contracts.Organization `yaml:"organizations"`
}

// BitbucketConfig is used to configure bitbucket integration
type BitbucketConfig struct {
	APIKey             string               `yaml:"apiKey"`
	AppOAuthKey        string               `yaml:"appOAuthKey"`
	AppOAuthSecret     string               `yaml:"appOAuthSecret"`
	AllowedOwners      []string             `yaml:"allowedOwners"`
	OwnerOrganizations []OwnerOrganizations `yaml:"ownerOrganizations"`
}

// OwnerOrganizations is used to assign organizations to builds triggered through a specific owner
type OwnerOrganizations struct {
	Owner         string                    `yaml:"owner"`
	Organizations []*contracts.Organization `yaml:"organizations"`
}

// SlackConfig is used to configure slack integration
type SlackConfig struct {
	ClientID             string `yaml:"clientID"`
	ClientSecret         string `yaml:"clientSecret"`
	AppVerificationToken string `yaml:"appVerificationToken"`
	AppOAuthAccessToken  string `yaml:"appOAuthAccessToken"`
}

// PubsubConfig is used to be able to subscribe to pub/sub topics for triggering pipelines based on pub/sub events
type PubsubConfig struct {
	DefaultProject                 string `yaml:"defaultProject"`
	Endpoint                       string `yaml:"endpoint"`
	Audience                       string `yaml:"audience"`
	ServiceAccountEmail            string `yaml:"serviceAccountEmail"`
	SubscriptionNameSuffix         string `yaml:"subscriptionNameSuffix"`
	SubscriptionIdleExpirationDays int    `yaml:"subscriptionIdleExpirationDays"`
}

// CloudStorageConfig is used to configure a google cloud storage bucket to be used to store logs
type CloudStorageConfig struct {
	ProjectID     string `yaml:"projectID"`
	Bucket        string `yaml:"bucket"`
	LogsDirectory string `yaml:"logsDir"`
}

// PrometheusConfig configures where to find prometheus for retrieving max cpu and memory consumption of build and release jobs
type PrometheusConfig struct {
	ServerURL             string `yaml:"serverURL"`
	ScrapeIntervalSeconds int    `yaml:"scrapeIntervalSeconds"`
}

// BigQueryConfig configures the dataset where to send bigquery events
type BigQueryConfig struct {
	Enable    bool   `yaml:"enable"`
	ProjectID string `yaml:"projectID"`
	Dataset   string `yaml:"dataset"`
}

// CloudSourceConfig is used to configure cloudSource integration
type CloudSourceConfig struct {
	AllowedProjects      []string               `yaml:"allowedProjects"`
	ProjectOrganizations []ProjectOrganizations `yaml:"projectOrganizations"`
}

// ProjectOrganizations is used to assign organizations to builds triggered through a specific project
type ProjectOrganizations struct {
	Project       string                    `yaml:"project"`
	Organizations []*contracts.Organization `yaml:"organizations"`
}

// ConfigReader reads the api config from file
type ConfigReader interface {
	ReadConfigFromFile(string, bool) (*APIConfig, error)
}

type configReaderImpl struct {
	secretHelper crypt.SecretHelper
}

// NewConfigReader returns a new config.ConfigReader
func NewConfigReader(secretHelper crypt.SecretHelper) ConfigReader {
	return &configReaderImpl{
		secretHelper: secretHelper,
	}
}

// ReadConfigFromFile is used to read configuration from a file set from a configmap
func (h *configReaderImpl) ReadConfigFromFile(configPath string, decryptSecrets bool) (config *APIConfig, err error) {

	log.Info().Msgf("Reading %v file...", configPath)

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return config, err
	}

	// decrypt secrets before unmarshalling
	if decryptSecrets {

		decryptedData, err := h.secretHelper.DecryptAllEnvelopes(string(data), "")
		if err != nil {
			log.Fatal().Err(err).Msg("Failed decrypting secrets in config file")
		}

		data = []byte(decryptedData)
	}

	// unmarshal into structs
	if err := yaml.Unmarshal(data, &config); err != nil {
		return config, err
	}

	log.Info().Msgf("Finished reading %v file successfully", configPath)

	return
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
