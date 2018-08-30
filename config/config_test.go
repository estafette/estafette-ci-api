package config

import (
	"testing"

	crypt "github.com/estafette/estafette-ci-crypt"
	"github.com/stretchr/testify/assert"
)

func TestReadConfigFromFile(t *testing.T) {

	t.Run("ReturnsConfigWithoutErrors", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp"))

		// act
		_, err := configReader.ReadConfigFromFile("test-config.yaml")

		assert.Nil(t, err)
	})

	t.Run("ReturnsGithubConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp"))

		// act
		config, _ := configReader.ReadConfigFromFile("test-config.yaml")

		githubConfig := config.Integrations.Github

		assert.Equal(t, "/github-app-key/private-key.pem", githubConfig.PrivateKeyPath)
		assert.Equal(t, "15", githubConfig.AppID)
		assert.Equal(t, "asdas2342", githubConfig.ClientID)
		assert.Equal(t, "this is my secret", githubConfig.ClientSecret)
		assert.Equal(t, 100, githubConfig.EventChannelBufferSize)
		assert.Equal(t, 5, githubConfig.MaxWorkers)
	})

	t.Run("ReturnsBitbucketConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp"))

		// act
		config, _ := configReader.ReadConfigFromFile("test-config.yaml")

		bitbucketConfig := config.Integrations.Bitbucket

		assert.Equal(t, "sd9ewiwuejkwejkewk", bitbucketConfig.APIKey)
		assert.Equal(t, "2390w3e90jdsk", bitbucketConfig.AppOAuthKey)
		assert.Equal(t, "this is my secret", bitbucketConfig.AppOAuthSecret)
		assert.Equal(t, 100, bitbucketConfig.EventChannelBufferSize)
		assert.Equal(t, 5, bitbucketConfig.MaxWorkers)
	})

	t.Run("ReturnsSlackConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp"))

		// act
		config, _ := configReader.ReadConfigFromFile("test-config.yaml")

		slackConfig := config.Integrations.Slack

		assert.Equal(t, "d9ew90weoijewjke", slackConfig.ClientID)
		assert.Equal(t, "this is my secret", slackConfig.ClientSecret)
		assert.Equal(t, "this is my secret", slackConfig.AppVerificationToken)
		assert.Equal(t, "this is my secret", slackConfig.AppOAuthAccessToken)
		assert.Equal(t, 100, slackConfig.EventChannelBufferSize)
		assert.Equal(t, 5, slackConfig.MaxWorkers)
	})

	t.Run("ReturnsAPIServerConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp"))

		// act
		config, _ := configReader.ReadConfigFromFile("test-config.yaml")

		apiServerConfig := config.APIServer

		assert.Equal(t, "https://ci.estafette.io/", apiServerConfig.BaseURL)
		assert.Equal(t, "http://estafette-ci-api.estafette.svc.cluster.local/", apiServerConfig.ServiceURL)
		assert.Equal(t, "this is my secret", apiServerConfig.APIKey)
		assert.Equal(t, 100, apiServerConfig.EventChannelBufferSize)
		assert.Equal(t, 5, apiServerConfig.MaxWorkers)
	})

	t.Run("ReturnsDatabaseConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp"))

		// act
		config, _ := configReader.ReadConfigFromFile("test-config.yaml")

		databaseConfig := config.Database

		assert.Equal(t, "estafette_ci_api", databaseConfig.DatabaseName)
		assert.Equal(t, "cockroachdb-public.estafette.svc.cluster.local", databaseConfig.Host)
		assert.Equal(t, true, databaseConfig.Insecure)
		assert.Equal(t, "/cockroachdb-certificates/cockroachdb.crt", databaseConfig.CertificateDir)
		assert.Equal(t, 26257, databaseConfig.Port)
		assert.Equal(t, "myuser", databaseConfig.User)
		assert.Equal(t, "this is my secret", databaseConfig.Password)
	})

	t.Run("ReturnsPrivateContainerRegistryConfigs", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp"))

		// act
		config, _ := configReader.ReadConfigFromFile("test-config.yaml")

		assert.Equal(t, "https://gcr.io", config.PrivateContainerRegistries[0].Server)
		assert.Equal(t, "_json_key", config.PrivateContainerRegistries[0].Username)
		assert.Equal(t, "this is my secret", config.PrivateContainerRegistries[0].Password)

		assert.Equal(t, "https://eu.gcr.io", config.PrivateContainerRegistries[1].Server)
		assert.Equal(t, "_json_key", config.PrivateContainerRegistries[1].Username)
		assert.Equal(t, "this is my secret", config.PrivateContainerRegistries[1].Password)
	})
}
