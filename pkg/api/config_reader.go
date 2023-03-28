package api

import (
	"os"
	"path/filepath"
	"strings"

	crypt "github.com/estafette/estafette-ci-crypt"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

// ConfigReader reads the api config from file
type ConfigReader interface {
	GetConfigFilePaths(configPath string) (configFilePaths []string, err error)
	ReadConfigFromFiles(configPath string, decryptSecrets bool) (config *APIConfig, err error)
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

// GetConfigFilePaths returns all matching yaml config file paths inside configPath directory
func (h *configReaderImpl) GetConfigFilePaths(configPath string) (configFilePaths []string, err error) {

	resolvedConfigPath, _ := filepath.EvalSymlinks(configPath)

	// get paths to all yaml files
	err = filepath.WalkDir(resolvedConfigPath, func(path string, entry os.DirEntry, err error) error {
		// add all yaml files
		if err == nil && !entry.IsDir() && filepath.Ext(path) == ".yaml" {
			configFilePaths = append(configFilePaths, filepath.Join(configPath, filepath.Base(path)))
		}

		return nil
	})
	if err != nil {
		return
	}

	return
}

// ReadConfigFromFiles is used to read configuration from multiple files set from a configmap
func (h *configReaderImpl) ReadConfigFromFiles(configPath string, decryptSecrets bool) (config *APIConfig, err error) {

	log.Info().Msgf("Reading configs from directory %v...", configPath)

	configFilePaths, err := h.GetConfigFilePaths(configPath)
	if err != nil {
		return
	}

	log.Debug().Msgf("Found %v config files: %v", len(configFilePaths), strings.Join(configFilePaths, ","))

	combinedData := []byte{}

	for _, configFilePath := range configFilePaths {
		data, err := os.ReadFile(configFilePath)
		if err != nil {
			return config, err
		}

		// decrypt secrets before unmarshalling
		if decryptSecrets {
			decryptedData, err := h.secretHelper.DecryptAllEnvelopes(string(data), "")
			if err != nil {
				log.Fatal().Err(err).Msgf("Failed decrypting secrets in config file %v", configFilePath)
			}

			data = []byte(decryptedData)
		}

		combinedData = append(combinedData, data...)
		combinedData = append(combinedData, []byte("\n")...)
	}

	// unmarshal into structs
	if err := yaml.Unmarshal(combinedData, &config); err != nil {
		return config, err
	}

	if config == nil {
		config = &APIConfig{}
	}

	// override values from envvars
	err = OverrideFromEnv(config, "ESCI", os.Environ())
	if err != nil {
		return config, err
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

	log.Debug().Msgf("Finished reading configs from directory %v successfully", configPath)

	return
}
