package config

import (
	"encoding/json"
	"testing"

	crypt "github.com/estafette/estafette-ci-crypt"
	"github.com/stretchr/testify/assert"
)

func TestReadConfigFromFile(t *testing.T) {

	t.Run("ReturnsConfigWithoutErrors", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false))

		// act
		_, err := configReader.ReadConfigFromFile("test-config.yaml", true)

		assert.Nil(t, err)
	})

	t.Run("ReturnsGithubConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false))

		// act
		config, _ := configReader.ReadConfigFromFile("test-config.yaml", true)

		githubConfig := config.Integrations.Github

		assert.Equal(t, "/github-app-key/private-key.pem", githubConfig.PrivateKeyPath)
		assert.Equal(t, "15", githubConfig.AppID)
		assert.Equal(t, "asdas2342", githubConfig.ClientID)
		assert.Equal(t, "this is my secret", githubConfig.ClientSecret)
	})

	t.Run("ReturnsBitbucketConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false))

		// act
		config, _ := configReader.ReadConfigFromFile("test-config.yaml", true)

		bitbucketConfig := config.Integrations.Bitbucket

		assert.Equal(t, "sd9ewiwuejkwejkewk", bitbucketConfig.APIKey)
		assert.Equal(t, "2390w3e90jdsk", bitbucketConfig.AppOAuthKey)
		assert.Equal(t, "this is my secret", bitbucketConfig.AppOAuthSecret)
	})

	t.Run("ReturnsSlackConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false))

		// act
		config, _ := configReader.ReadConfigFromFile("test-config.yaml", true)

		slackConfig := config.Integrations.Slack

		assert.Equal(t, "d9ew90weoijewjke", slackConfig.ClientID)
		assert.Equal(t, "this is my secret", slackConfig.ClientSecret)
		assert.Equal(t, "this is my secret", slackConfig.AppVerificationToken)
		assert.Equal(t, "this is my secret", slackConfig.AppOAuthAccessToken)
	})

	t.Run("ReturnsAPIServerConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false))

		// act
		config, _ := configReader.ReadConfigFromFile("test-config.yaml", true)

		apiServerConfig := config.APIServer

		assert.Equal(t, "https://ci.estafette.io/", apiServerConfig.BaseURL)
		assert.Equal(t, "http://estafette-ci-api.estafette.svc.cluster.local/", apiServerConfig.ServiceURL)
	})

	t.Run("ReturnsAuthConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false))

		// act
		config, _ := configReader.ReadConfigFromFile("test-config.yaml", true)

		authConfig := config.Auth

		assert.True(t, authConfig.IAP.Enable)
		assert.Equal(t, "/projects/***/global/backendServices/***", authConfig.IAP.Audience)
		assert.Equal(t, "this is my secret", authConfig.APIKey)
	})

	t.Run("ReturnsDatabaseConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false))

		// act
		config, _ := configReader.ReadConfigFromFile("test-config.yaml", true)

		databaseConfig := config.Database

		assert.Equal(t, "estafette_ci_api", databaseConfig.DatabaseName)
		assert.Equal(t, "cockroachdb-public.estafette.svc.cluster.local", databaseConfig.Host)
		assert.Equal(t, true, databaseConfig.Insecure)
		assert.Equal(t, "/cockroachdb-certificates/cockroachdb.crt", databaseConfig.CertificateDir)
		assert.Equal(t, 26257, databaseConfig.Port)
		assert.Equal(t, "myuser", databaseConfig.User)
		assert.Equal(t, "this is my secret", databaseConfig.Password)
	})

	t.Run("ReturnsCredentialsConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false))

		// act
		config, _ := configReader.ReadConfigFromFile("test-config.yaml", true)

		credentialsConfig := config.Credentials

		assert.Equal(t, 9, len(credentialsConfig))
		assert.Equal(t, "container-registry-extensions", credentialsConfig[0].Name)
		assert.Equal(t, "container-registry", credentialsConfig[0].Type)
		assert.Equal(t, "extensions", credentialsConfig[0].AdditionalProperties["repository"])
		assert.Equal(t, "slack-webhook-estafette", credentialsConfig[6].Name)
		assert.Equal(t, "slack-webhook", credentialsConfig[6].Type)
		assert.Equal(t, "estafette", credentialsConfig[6].AdditionalProperties["workspace"])
	})

	t.Run("ReturnsTrustedImagesConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false))

		// act
		config, _ := configReader.ReadConfigFromFile("test-config.yaml", true)

		trustedImagesConfig := config.TrustedImages

		assert.Equal(t, 8, len(trustedImagesConfig))
		assert.Equal(t, "extensions/docker", trustedImagesConfig[0].ImagePath)
		assert.True(t, trustedImagesConfig[0].RunDocker)
		assert.Equal(t, 1, len(trustedImagesConfig[0].InjectedCredentialTypes))
		assert.Equal(t, "container-registry", trustedImagesConfig[0].InjectedCredentialTypes[0])

		assert.Equal(t, "multiple-git-sources-test", trustedImagesConfig[7].ImagePath)
		assert.False(t, trustedImagesConfig[7].RunDocker)
		assert.Equal(t, 2, len(trustedImagesConfig[7].InjectedCredentialTypes))
		assert.Equal(t, "bitbucket-api-token", trustedImagesConfig[7].InjectedCredentialTypes[0])
		assert.Equal(t, "github-api-token", trustedImagesConfig[7].InjectedCredentialTypes[1])
	})

	t.Run("ReturnsRegistryMirror", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false))

		// act
		config, _ := configReader.ReadConfigFromFile("test-config.yaml", true)

		registryMirrorConfig := config.RegistryMirror

		assert.NotNil(t, registryMirrorConfig)
		assert.Equal(t, "https://mirror.gcr.io", *registryMirrorConfig)
	})

	t.Run("ReturnsDockerDaemonMTU", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false))

		// act
		config, _ := configReader.ReadConfigFromFile("test-config.yaml", true)

		dockerDaemonMTUConfig := config.DockerDaemonMTU

		assert.NotNil(t, dockerDaemonMTUConfig)
		assert.Equal(t, "1360", *dockerDaemonMTUConfig)
	})

	t.Run("AllowsCredentialConfigWithComplexAdditionalPropertiesToBeJSONMarshalled", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false))

		// act
		config, _ := configReader.ReadConfigFromFile("test-config.yaml", true)

		credentialsConfig := config.Credentials

		bytes, err := json.Marshal(credentialsConfig[2])

		assert.Nil(t, err)
		assert.Equal(t, "{\"name\":\"gke-estafette-production\",\"type\":\"kubernetes-engine\",\"additionalProperties\":{\"cluster\":\"production-europe-west2\",\"defaults\":{\"autoscale\":{\"min\":2},\"container\":{\"repository\":\"estafette\"},\"namespace\":\"estafette\",\"sidecars\":[{\"image\":\"estafette/openresty-sidecar:1.13.6.1-alpine\",\"type\":\"openresty\"}]},\"project\":\"estafette-production\",\"region\":\"europe-west2\",\"serviceAccountKeyfile\":\"{}\"}}", string(bytes))
	})
}
