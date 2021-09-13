package api

import (
	"context"
	"io/ioutil"

	crypt "github.com/estafette/estafette-ci-crypt"
	"github.com/rs/zerolog/log"
	"github.com/sethvargo/go-envconfig"
	yaml "gopkg.in/yaml.v2"
)

// ConfigReader reads the api config from file
type ConfigReader interface {
	ReadConfigFromFile(string, bool) (*APIConfig, error)
}

type configReaderImpl struct {
	secretHelper crypt.SecretHelper
	jwtKey       string
}

// NewConfigReader returns a new config.ConfigReader
func NewConfigReader(secretHelper crypt.SecretHelper, jwtKey string) ConfigReader {
	return &configReaderImpl{
		secretHelper: secretHelper,
		jwtKey:       jwtKey,
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

	// override values from envvars
	lookuper := envconfig.PrefixLookuper("ESCI_", envconfig.OsLookuper())
	if err = envconfig.ProcessWith(context.Background(), config, lookuper); err != nil {
		return
	}

	// fill in all the defaults for empty values
	config.SetDefaults()

	// set jwt key from secret
	if config.Auth.JWT.Key == "" {
		config.Auth.JWT.Key = h.jwtKey
	}

	// validate the config
	err = config.Validate()
	if err != nil {
		return
	}

	log.Info().Msgf("Finished reading %v file successfully", configPath)

	return
}
