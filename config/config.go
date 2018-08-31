package config

import (
	"io/ioutil"
	"regexp"

	crypt "github.com/estafette/estafette-ci-crypt"
	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v2"
)

// APIConfig represent the configuration for the entire api application
type APIConfig struct {
	Integrations                   *APIConfigIntegrations                 `yaml:"integrations,omitempty"`
	APIServer                      *APIServerConfig                       `yaml:"apiServer,omitempty"`
	Database                       *DatabaseConfig                        `yaml:"database,omitempty"`
	ContainerRepositoryCredentials []*ContainerRepositoryCredentialConfig `yaml:"containerRepositoryCredentials,omitempty"`
}

// APIServerConfig represents configuration for the api server
type APIServerConfig struct {
	BaseURL                string `yaml:"baseURL"`
	ServiceURL             string `yaml:"serviceURL"`
	APIKey                 string `yaml:"apiKey"`
	EventChannelBufferSize int    `yaml:"eventChannelBufferSize"`
	MaxWorkers             int    `yaml:"maxWorkers"`
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
	Github    *GithubConfig    `yaml:"github,omitempty"`
	Bitbucket *BitbucketConfig `yaml:"bitbucket,omitempty"`
	Slack     *SlackConfig     `yaml:"slack,omitempty"`
}

// GithubConfig is used to configure github integration
type GithubConfig struct {
	PrivateKeyPath         string `yaml:"privateKeyPath"`
	AppID                  string `yaml:"appID"`
	ClientID               string `yaml:"clientID"`
	ClientSecret           string `yaml:"clientSecret"`
	WebhookSecret          string `yaml:"webhookSecret"`
	EventChannelBufferSize int    `yaml:"eventChannelBufferSize"`
	MaxWorkers             int    `yaml:"maxWorkers"`
}

// BitbucketConfig is used to configure bitbucket integration
type BitbucketConfig struct {
	APIKey                 string `yaml:"apiKey"`
	AppOAuthKey            string `yaml:"appOAuthKey"`
	AppOAuthSecret         string `yaml:"appOAuthSecret"`
	EventChannelBufferSize int    `yaml:"eventChannelBufferSize"`
	MaxWorkers             int    `yaml:"maxWorkers"`
}

// SlackConfig is used to configure slack integration
type SlackConfig struct {
	ClientID               string `yaml:"clientID"`
	ClientSecret           string `yaml:"clientSecret"`
	AppVerificationToken   string `yaml:"appVerificationToken"`
	AppOAuthAccessToken    string `yaml:"appOAuthAccessToken"`
	EventChannelBufferSize int    `yaml:"eventChannelBufferSize"`
	MaxWorkers             int    `yaml:"maxWorkers"`
}

// ContainerRepositoryCredentialConfig is used to authenticate for (private) container repositories
type ContainerRepositoryCredentialConfig struct {
	Repository string `yaml:"repository"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
}

// ConfigReader reads the api config from file
type ConfigReader interface {
	ReadConfigFromFile(string) (*APIConfig, error)
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
func (h *configReaderImpl) ReadConfigFromFile(configPath string) (config *APIConfig, err error) {

	log.Info().Msgf("Reading %v file...", configPath)

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return config, err
	}

	// decrypt secrets before unmarshalling
	data = h.DecryptSecrets(data)

	// unmarshal into structs
	if err := yaml.Unmarshal(data, &config); err != nil {
		return config, err
	}

	log.Info().Msgf("Finished reading %v file successfully", configPath)

	return
}

func (h *configReaderImpl) DecryptSecrets(data []byte) (decryptedData []byte) {

	r, err := regexp.Compile(`estafette\.secret\([a-zA-Z0-9.=_-]+\)`)
	if err != nil {
		log.Warn().Err(err).Msg("Failed compiling regexp")
	}

	decryptedData = r.ReplaceAllFunc(data, h.decryptSecret)

	return
}

func (h *configReaderImpl) decryptSecret(encryptedValue []byte) (decryptedValue []byte) {

	r, err := regexp.Compile(`^estafette\.secret\(([a-zA-Z0-9.=_-]+)\)$`)
	if err != nil {
		log.Warn().Err(err).Msg("Failed compiling regexp")
		return encryptedValue
	}

	matches := r.FindStringSubmatch(string(encryptedValue))
	if matches == nil {
		log.Debug().Msgf("No matches, returning %v", string(encryptedValue))
		return encryptedValue
	}

	decryptedValueString, err := h.secretHelper.Decrypt(matches[1])
	if err != nil {
		log.Warn().Err(err).Msg("Failed decrypting secret")
		return encryptedValue
	}

	return []byte(decryptedValueString)
}
