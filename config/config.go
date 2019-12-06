package config

import (
	"io/ioutil"

	"github.com/estafette/estafette-ci-api/helpers"
	contracts "github.com/estafette/estafette-ci-contracts"
	crypt "github.com/estafette/estafette-ci-crypt"
	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v2"
)

// APIConfig represent the configuration for the entire api application
type APIConfig struct {
	Integrations    *APIConfigIntegrations          `yaml:"integrations,omitempty"`
	APIServer       *APIServerConfig                `yaml:"apiServer,omitempty"`
	Auth            *AuthConfig                     `yaml:"auth,omitempty"`
	Jobs            *JobsConfig                     `yaml:"jobs,omitempty"`
	Database        *DatabaseConfig                 `yaml:"database,omitempty"`
	Credentials     []*contracts.CredentialConfig   `yaml:"credentials,omitempty" json:"credentials,omitempty"`
	TrustedImages   []*contracts.TrustedImageConfig `yaml:"trustedImages,omitempty" json:"trustedImages,omitempty"`
	RegistryMirror  *string                         `yaml:"registryMirror,omitempty" json:"registryMirror,omitempty"`
	DockerDaemonMTU *string                         `yaml:"dindMtu,omitempty" json:"dindMtu,omitempty"`
	DockerDaemonBIP *string                         `yaml:"dindBip,omitempty" json:"dindBip,omitempty"`
	DockerNetwork   *contracts.DockerNetworkConfig  `yaml:"dindNetwork,omitempty" json:"dindNetwork,omitempty"`
}

// APIServerConfig represents configuration for the api server
type APIServerConfig struct {
	BaseURL    string   `yaml:"baseURL"`
	ServiceURL string   `yaml:"serviceURL"`
	LogWriters []string `yaml:"logWriters"`
	LogReader  string   `yaml:"logReader"`
}

// WriteLogToDatabase indicates if database is in the logWriters config
func (c *APIServerConfig) WriteLogToDatabase() bool {
	return len(c.LogWriters) == 0 || helpers.StringArrayContains(c.LogWriters, "database")
}

// WriteLogToCloudStorage indicates if cloudstorage is in the logWriters config
func (c *APIServerConfig) WriteLogToCloudStorage() bool {
	return helpers.StringArrayContains(c.LogWriters, "cloudstorage")
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
	IAP    *IAPAuthConfig `yaml:"iap"`
	APIKey string         `yaml:"apiKey"`
}

// JobsConfig configures the lower and upper bounds for automatically setting resources for build/release jobs
type JobsConfig struct {
	Namespace          string  `yaml:"namespace"`
	MinCPUCores        float64 `yaml:"minCPUCores"`
	MaxCPUCores        float64 `yaml:"maxCPUCores"`
	CPURequestRatio    float64 `yaml:"cpuRequestRatio"`
	MinMemoryBytes     float64 `yaml:"minMemoryBytes"`
	MaxMemoryBytes     float64 `yaml:"maxMemoryBytes"`
	MemoryRequestRatio float64 `yaml:"memoryRequestRatio"`
}

// IAPAuthConfig sets iap config in case it's used for authentication and authorization
type IAPAuthConfig struct {
	Enable   bool   `yaml:"enable"`
	Audience string `yaml:"audience"`
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

// APIConfigIntegrations contains config for 3rd party integrations
type APIConfigIntegrations struct {
	Github       *GithubConfig       `yaml:"github,omitempty"`
	Bitbucket    *BitbucketConfig    `yaml:"bitbucket,omitempty"`
	Slack        *SlackConfig        `yaml:"slack,omitempty"`
	Pubsub       *PubsubConfig       `yaml:"pubsub,omitempty"`
	Prometheus   *PrometheusConfig   `yaml:"prometheus,omitempty"`
	BigQuery     *BigQueryConfig     `yaml:"bigquery,omitempty"`
	CloudStorage *CloudStorageConfig `yaml:"gcs,omitempty"`
}

// GithubConfig is used to configure github integration
type GithubConfig struct {
	PrivateKeyPath           string `yaml:"privateKeyPath"`
	AppID                    string `yaml:"appID"`
	ClientID                 string `yaml:"clientID"`
	ClientSecret             string `yaml:"clientSecret"`
	WebhookSecret            string `yaml:"webhookSecret"`
	WhitelistedInstallations []int  `yaml:"whitelistedInstallations"`
}

// BitbucketConfig is used to configure bitbucket integration
type BitbucketConfig struct {
	APIKey            string   `yaml:"apiKey"`
	AppOAuthKey       string   `yaml:"appOAuthKey"`
	AppOAuthSecret    string   `yaml:"appOAuthSecret"`
	WhitelistedOwners []string `yaml:"whitelistedOwners"`
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

		decryptedData, err := h.secretHelper.DecryptAllEnvelopes(string(data))
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
