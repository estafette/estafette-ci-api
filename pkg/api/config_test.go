package api

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	contracts "github.com/estafette/estafette-ci-contracts"
	crypt "github.com/estafette/estafette-ci-crypt"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestReadConfigFromFiles(t *testing.T) {

	t.Run("MergesMultipleYamlConfigFiles", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)

		assert.Nil(t, err)
		assert.Equal(t, "https://ci.estafette.io/", config.APIServer.BaseURL)
		assert.Equal(t, 9, len(config.Credentials))
		assert.Equal(t, 8, len(config.TrustedImages))
	})

	t.Run("ReturnsConfigWithoutErrors", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		_, err := configReader.ReadConfigFromFiles("configs", true)

		assert.Nil(t, err)
	})

	t.Run("ReturnsGithubConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)

		githubConfig := config.Integrations.Github

		assert.Nil(t, err)
		assert.True(t, githubConfig.Enable)
	})

	t.Run("ReturnsBitbucketConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)

		bitbucketConfig := config.Integrations.Bitbucket

		assert.Nil(t, err)
		assert.True(t, bitbucketConfig.Enable)
	})

	t.Run("ReturnsCloudsourceConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)

		cloudsourceConfig := config.Integrations.CloudSource

		assert.Nil(t, err)
		assert.True(t, cloudsourceConfig.Enable)
		assert.Equal(t, 1, len(cloudsourceConfig.ProjectOrganizations))
		assert.Equal(t, "estafette", cloudsourceConfig.ProjectOrganizations[0].Project)
		assert.Equal(t, 1, len(cloudsourceConfig.ProjectOrganizations[0].Organizations))
		assert.Equal(t, "Estafette", cloudsourceConfig.ProjectOrganizations[0].Organizations[0].Name)
	})

	t.Run("ReturnsPubsubConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)

		pubsubConfig := config.Integrations.Pubsub

		assert.Nil(t, err)
		assert.True(t, pubsubConfig.Enable)
		assert.Equal(t, "estafette", pubsubConfig.DefaultProject)
		assert.Equal(t, "https://ci-integrations.estafette.io/api/integrations/pubsub/events", pubsubConfig.Endpoint)
		assert.Equal(t, "estafette-audience", pubsubConfig.Audience)
		assert.Equal(t, "estafette@estafette.iam.gserviceaccount.com", pubsubConfig.ServiceAccountEmail)
		assert.Equal(t, "~estafette-ci-pubsub-trigger", pubsubConfig.SubscriptionNameSuffix)
		assert.Equal(t, 365, pubsubConfig.SubscriptionIdleExpirationDays)
	})

	t.Run("ReturnsSlackConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)

		slackConfig := config.Integrations.Slack

		assert.Nil(t, err)
		assert.True(t, slackConfig.Enable)
		assert.Equal(t, "d9ew90weoijewjke", slackConfig.ClientID)
		assert.Equal(t, "this is my secret", slackConfig.ClientSecret)
		assert.Equal(t, "this is my secret", slackConfig.AppVerificationToken)
		assert.Equal(t, "this is my secret", slackConfig.AppOAuthAccessToken)
	})

	t.Run("ReturnsPrometheusConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)

		prometheusConfig := config.Integrations.Prometheus

		assert.Nil(t, err)
		assert.True(t, *prometheusConfig.Enable)
		assert.Equal(t, "http://prometheus-server.monitoring.svc.cluster.local", prometheusConfig.ServerURL)
		assert.Equal(t, 10, prometheusConfig.ScrapeIntervalSeconds)
	})

	t.Run("ReturnsBigQueryConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)

		bigqueryConfig := config.Integrations.BigQuery

		assert.Nil(t, err)
		assert.True(t, bigqueryConfig.Enable)
		assert.Equal(t, "my-gcp-project", bigqueryConfig.ProjectID)
		assert.Equal(t, "my-dataset", bigqueryConfig.Dataset)
	})

	t.Run("ReturnsCloudStorageConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)

		cloudStorageConfig := config.Integrations.CloudStorage

		assert.Nil(t, err)
		assert.True(t, cloudStorageConfig.Enable)
		assert.Equal(t, "my-gcp-project", cloudStorageConfig.ProjectID)
		assert.Equal(t, "my-bucket", cloudStorageConfig.Bucket)
		assert.Equal(t, "logs", cloudStorageConfig.LogsDirectory)
	})

	t.Run("ReturnsAPIServerConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)

		apiServerConfig := config.APIServer

		assert.Nil(t, err)
		assert.Equal(t, "https://ci.estafette.io/", apiServerConfig.BaseURL)
		assert.Equal(t, "https://ci-integrations.estafette.io/", apiServerConfig.IntegrationsURL)
		assert.Equal(t, "http://estafette-ci-api.estafette.svc.cluster.local/", apiServerConfig.ServiceURL)
		assert.Equal(t, 2, len(apiServerConfig.LogWriters))
		assert.Equal(t, LogTargetDatabase, apiServerConfig.LogWriters[0])
		assert.Equal(t, LogTargetCloudStorage, apiServerConfig.LogWriters[1])
		assert.Equal(t, LogTargetDatabase, apiServerConfig.LogReader)

		assert.Equal(t, "envvars", apiServerConfig.InjectStagesPerOperatingSystem[manifest.OperatingSystemLinux].Build.Before[0].Name)
		assert.Equal(t, "extensions/envvars:stable", apiServerConfig.InjectStagesPerOperatingSystem[manifest.OperatingSystemLinux].Build.Before[0].ContainerImage)

		assert.Equal(t, "snyk", apiServerConfig.InjectStagesPerOperatingSystem[manifest.OperatingSystemLinux].Build.After[0].Name)
		assert.Equal(t, "extensions/snyk:stable-golang", apiServerConfig.InjectStagesPerOperatingSystem[manifest.OperatingSystemLinux].Build.After[0].ContainerImage)
		assert.Equal(t, map[string]interface{}{
			"language": "golang",
		}, apiServerConfig.InjectStagesPerOperatingSystem[manifest.OperatingSystemLinux].Build.After[0].CustomProperties["labelSelector"])

		assert.Equal(t, "envvars", apiServerConfig.InjectStagesPerOperatingSystem[manifest.OperatingSystemLinux].Release.After[0].Name)
		assert.Equal(t, "extensions/envvars:dev", apiServerConfig.InjectStagesPerOperatingSystem[manifest.OperatingSystemLinux].Release.After[0].ContainerImage)
		assert.Equal(t, "envvars", apiServerConfig.InjectStagesPerOperatingSystem[manifest.OperatingSystemLinux].Bot.After[0].Name)
		assert.Equal(t, "extensions/envvars:dev", apiServerConfig.InjectStagesPerOperatingSystem[manifest.OperatingSystemLinux].Bot.After[0].ContainerImage)

		assert.Equal(t, "envvars", apiServerConfig.InjectStagesPerOperatingSystem[manifest.OperatingSystemWindows].Build.Before[0].Name)
		assert.Equal(t, "extensions/envvars:windowsservercore-ltsc2019", apiServerConfig.InjectStagesPerOperatingSystem[manifest.OperatingSystemWindows].Build.Before[0].ContainerImage)
		assert.Equal(t, "envvars", apiServerConfig.InjectStagesPerOperatingSystem[manifest.OperatingSystemWindows].Release.After[0].Name)
		assert.Equal(t, "extensions/envvars:windowsservercore-ltsc2019", apiServerConfig.InjectStagesPerOperatingSystem[manifest.OperatingSystemWindows].Release.After[0].ContainerImage)
		assert.Equal(t, "envvars", apiServerConfig.InjectStagesPerOperatingSystem[manifest.OperatingSystemWindows].Bot.After[0].Name)
		assert.Equal(t, "extensions/envvars:windowsservercore-ltsc2019", apiServerConfig.InjectStagesPerOperatingSystem[manifest.OperatingSystemWindows].Bot.After[0].ContainerImage)

		assert.Equal(t, 0, len(apiServerConfig.InjectCommandsPerOperatingSystemAndShell[manifest.OperatingSystemLinux]["/bin/sh"].Before))
		assert.Equal(t, 0, len(apiServerConfig.InjectCommandsPerOperatingSystemAndShell[manifest.OperatingSystemLinux]["/bin/sh"].After))

		assert.Equal(t, 0, len(apiServerConfig.InjectCommandsPerOperatingSystemAndShell[manifest.OperatingSystemLinux]["/bin/bash"].Before))
		assert.Equal(t, 0, len(apiServerConfig.InjectCommandsPerOperatingSystemAndShell[manifest.OperatingSystemLinux]["/bin/bash"].After))

		assert.Equal(t, 1, len(apiServerConfig.InjectCommandsPerOperatingSystemAndShell[manifest.OperatingSystemWindows]["cmd"].Before))
		assert.Equal(t, "netsh interface ipv4 set subinterface 31 mtu=1410", apiServerConfig.InjectCommandsPerOperatingSystemAndShell[manifest.OperatingSystemWindows]["cmd"].Before[0])
		assert.Equal(t, 0, len(apiServerConfig.InjectCommandsPerOperatingSystemAndShell[manifest.OperatingSystemWindows]["cmd"].After))

		assert.Equal(t, 1, len(apiServerConfig.InjectCommandsPerOperatingSystemAndShell[manifest.OperatingSystemWindows]["powershell"].Before))
		assert.Equal(t, "Get-NetAdapter | Where-Object Name -like \"*Ethernet*\" | ForEach-Object { & netsh interface ipv4 set subinterface $_.InterfaceIndex mtu=1410 store=persistent }", apiServerConfig.InjectCommandsPerOperatingSystemAndShell[manifest.OperatingSystemWindows]["powershell"].Before[0])
		assert.Equal(t, 0, len(apiServerConfig.InjectCommandsPerOperatingSystemAndShell[manifest.OperatingSystemWindows]["powershell"].After))

		assert.Equal(t, contracts.DockerRunTypeDinD, apiServerConfig.DockerConfigPerOperatingSystem[manifest.OperatingSystemLinux].RunType)
		assert.Equal(t, 1460, apiServerConfig.DockerConfigPerOperatingSystem[manifest.OperatingSystemLinux].MTU)
		assert.Equal(t, "192.168.1.1/24", apiServerConfig.DockerConfigPerOperatingSystem[manifest.OperatingSystemLinux].BIP)
		assert.Equal(t, "estafette", apiServerConfig.DockerConfigPerOperatingSystem[manifest.OperatingSystemLinux].Networks[0].Name)
		assert.Equal(t, "192.168.2.1/24", apiServerConfig.DockerConfigPerOperatingSystem[manifest.OperatingSystemLinux].Networks[0].Subnet)
		assert.Equal(t, "192.168.2.1", apiServerConfig.DockerConfigPerOperatingSystem[manifest.OperatingSystemLinux].Networks[0].Gateway)
		assert.Equal(t, "https://mirror.gcr.io", apiServerConfig.DockerConfigPerOperatingSystem[manifest.OperatingSystemLinux].RegistryMirror)

		assert.Equal(t, contracts.DockerRunTypeDoD, apiServerConfig.DockerConfigPerOperatingSystem[manifest.OperatingSystemWindows].RunType)
		assert.Equal(t, 1410, apiServerConfig.DockerConfigPerOperatingSystem[manifest.OperatingSystemWindows].MTU)
	})

	t.Run("ReturnsAuthConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)

		authConfig := config.Auth

		assert.Nil(t, err)
		assert.Equal(t, "ci.estafette.io", authConfig.JWT.Domain)
		assert.Equal(t, "this is my secret", authConfig.JWT.Key)

		assert.Equal(t, "google", authConfig.Google.Name)
		assert.Equal(t, "abcdasa", authConfig.Google.ClientID)
		assert.Equal(t, "asdsddsfdfs", authConfig.Google.ClientSecret)
		assert.Equal(t, ".+@estafette\\.io", authConfig.Google.AllowedIdentitiesRegex)

		assert.Equal(t, "github", authConfig.Github.Name)
		assert.Equal(t, "abcdasa", authConfig.Github.ClientID)
		assert.Equal(t, "asdsddsfdfs", authConfig.Github.ClientSecret)
		assert.Equal(t, ".+@estafette\\.io", authConfig.Github.AllowedIdentitiesRegex)

		assert.Equal(t, 3, len(authConfig.Organizations))
		assert.Equal(t, "Org A", authConfig.Organizations[0].Name)
		assert.Equal(t, 1, len(authConfig.Organizations[0].OAuthProviders))
		assert.Equal(t, "google", authConfig.Organizations[0].OAuthProviders[0].Name)
		assert.Equal(t, "abcdasa", authConfig.Organizations[0].OAuthProviders[0].ClientID)
		assert.Equal(t, "asdsddsfdfs", authConfig.Organizations[0].OAuthProviders[0].ClientSecret)
		assert.Equal(t, ".+@estafette\\.io", authConfig.Organizations[0].OAuthProviders[0].AllowedIdentitiesRegex)

		assert.Equal(t, "Org B", authConfig.Organizations[1].Name)
		assert.Equal(t, 1, len(authConfig.Organizations[1].OAuthProviders))

		assert.Equal(t, "Org C", authConfig.Organizations[2].Name)
		assert.Equal(t, 1, len(authConfig.Organizations[2].OAuthProviders))

		assert.Equal(t, 2, len(authConfig.Administrators))
		assert.Equal(t, "admin1@server.com", authConfig.Administrators[0])
		assert.Equal(t, "admin2@server.com", authConfig.Administrators[1])
	})

	t.Run("ReturnsJobsConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)

		jobsConfig := config.Jobs

		assert.Nil(t, err)
		assert.Equal(t, "estafette-ci-jobs", jobsConfig.Namespace)
		assert.Equal(t, "estafette-ci-builder", jobsConfig.ServiceAccountName)
		assert.Equal(t, 2.0, jobsConfig.DefaultCPUCores)
		assert.Equal(t, 0.2, jobsConfig.MinCPUCores)
		assert.Equal(t, 60.0, jobsConfig.MaxCPUCores)
		assert.Equal(t, 1.0, jobsConfig.CPURequestRatio)
		assert.Equal(t, 2.0, jobsConfig.CPULimitRatio)

		assert.Equal(t, 8*math.Pow(2, 10)*math.Pow(2, 10)*math.Pow(2, 10), jobsConfig.DefaultMemoryBytes) // 8Gi
		assert.Equal(t, 128*math.Pow(2, 10)*math.Pow(2, 10), jobsConfig.MinMemoryBytes)                   // 128Mi
		assert.Equal(t, 200*math.Pow(2, 10)*math.Pow(2, 10)*math.Pow(2, 10), jobsConfig.MaxMemoryBytes)   // 200Gi
		assert.Equal(t, 1.25, jobsConfig.MemoryRequestRatio)
		assert.Equal(t, 1.0, jobsConfig.MemoryLimitRatio)
	})

	t.Run("ReturnsJobsConfigAffinityAndTolerations", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)

		jobsConfig := config.Jobs

		assert.Nil(t, err)

		// build affinity
		assert.Equal(t, "role", jobsConfig.BuildAffinityAndTolerations.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Key)
		assert.Equal(t, v1.NodeSelectorOpIn, jobsConfig.BuildAffinityAndTolerations.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Operator)
		assert.Equal(t, 1, len(jobsConfig.BuildAffinityAndTolerations.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values))
		assert.Equal(t, "privileged", jobsConfig.BuildAffinityAndTolerations.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0])

		assert.Equal(t, int32(10), jobsConfig.BuildAffinityAndTolerations.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].Weight)
		assert.Equal(t, "cloud.google.com/gke-preemptible", jobsConfig.BuildAffinityAndTolerations.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].Preference.MatchExpressions[0].Key)
		assert.Equal(t, v1.NodeSelectorOpIn, jobsConfig.BuildAffinityAndTolerations.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].Preference.MatchExpressions[0].Operator)
		assert.Equal(t, "true", jobsConfig.BuildAffinityAndTolerations.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].Preference.MatchExpressions[0].Values[0])

		// build tolerations
		assert.Equal(t, []v1.Toleration{
			{
				Key:      "role",
				Operator: v1.TolerationOpEqual,
				Value:    "privileged",
				Effect:   v1.TaintEffectNoSchedule,
			},
			{
				Key:      "cloud.google.com/gke-preemptible",
				Operator: v1.TolerationOpEqual,
				Value:    "true",
				Effect:   v1.TaintEffectNoSchedule,
			},
		}, jobsConfig.BuildAffinityAndTolerations.Tolerations)

		// release affinity
		assert.Equal(t, "role", jobsConfig.ReleaseAffinityAndTolerations.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Key)
		assert.Equal(t, v1.NodeSelectorOpIn, jobsConfig.ReleaseAffinityAndTolerations.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Operator)
		assert.Equal(t, "privileged", jobsConfig.ReleaseAffinityAndTolerations.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0])
		assert.Equal(t, "cloud.google.com/gke-preemptible", jobsConfig.ReleaseAffinityAndTolerations.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[1].Key)
		assert.Equal(t, v1.NodeSelectorOpDoesNotExist, jobsConfig.ReleaseAffinityAndTolerations.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[1].Operator)

		// release tolerations
		assert.Equal(t, []v1.Toleration{
			{
				Key:      "role",
				Operator: v1.TolerationOpEqual,
				Value:    "privileged",
				Effect:   v1.TaintEffectNoSchedule,
			},
		}, jobsConfig.ReleaseAffinityAndTolerations.Tolerations)

		// bot affinity
		assert.Equal(t, "role", jobsConfig.BotAffinityAndTolerations.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Key)
		assert.Equal(t, v1.NodeSelectorOpIn, jobsConfig.BotAffinityAndTolerations.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Operator)
		assert.Equal(t, "privileged", jobsConfig.BotAffinityAndTolerations.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0])
		assert.Equal(t, "cloud.google.com/gke-preemptible", jobsConfig.BotAffinityAndTolerations.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[1].Key)
		assert.Equal(t, v1.NodeSelectorOpDoesNotExist, jobsConfig.BotAffinityAndTolerations.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[1].Operator)

		// bot tolerations
		assert.Equal(t, []v1.Toleration{
			{
				Key:      "role",
				Operator: v1.TolerationOpEqual,
				Value:    "privileged",
				Effect:   v1.TaintEffectNoSchedule,
			},
		}, jobsConfig.BotAffinityAndTolerations.Tolerations)
	})

	t.Run("ReturnsDatabaseConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)

		databaseConfig := config.Database

		assert.Nil(t, err)
		assert.Equal(t, "estafette_ci_api", databaseConfig.DatabaseName)
		assert.Equal(t, "cockroachdb-public.estafette.svc.cluster.local", databaseConfig.Host)
		assert.Equal(t, true, databaseConfig.Insecure)
		assert.Equal(t, "verify-full", databaseConfig.SslMode)
		assert.Equal(t, "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt", databaseConfig.CertificateAuthorityPath)
		assert.Equal(t, "/cockroach-certs/cert", databaseConfig.CertificatePath)
		assert.Equal(t, "/cockroach-certs/key", databaseConfig.CertificateKeyPath)
		assert.Equal(t, 26257, databaseConfig.Port)
		assert.Equal(t, "myuser", databaseConfig.User)
		assert.Equal(t, "this is my secret", databaseConfig.Password)
	})

	t.Run("ReturnsQueueConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)

		queueConfig := config.Queue

		assert.Nil(t, err)
		assert.NotNil(t, queueConfig)
		assert.Equal(t, 1, len(queueConfig.Hosts))
		assert.Equal(t, "estafette-ci-queue-0.estafette-ci-queue", queueConfig.Hosts[0])
		assert.Equal(t, "event.cron", queueConfig.SubjectCron)
		assert.Equal(t, "event.git", queueConfig.SubjectGit)
		assert.Equal(t, "event.github", queueConfig.SubjectGithub)
		assert.Equal(t, "event.bitbucket", queueConfig.SubjectBitbucket)
	})

	t.Run("ReturnsManifestPreferences", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)

		if !assert.Nil(t, err) {
			return
		}

		if !assert.NotNil(t, config.ManifestPreferences) {
			return
		}

		assert.Equal(t, 1, len(config.ManifestPreferences.LabelRegexes))
		assert.Equal(t, "api|web|library|container", config.ManifestPreferences.LabelRegexes["type"])
		assert.Equal(t, 2, len(config.ManifestPreferences.BuilderOperatingSystems))
		assert.Equal(t, manifest.OperatingSystemLinux, config.ManifestPreferences.BuilderOperatingSystems[0])
		assert.Equal(t, manifest.OperatingSystemWindows, config.ManifestPreferences.BuilderOperatingSystems[1])
		assert.Equal(t, 2, len(config.ManifestPreferences.BuilderTracksPerOperatingSystem))
		assert.Equal(t, 3, len(config.ManifestPreferences.BuilderTracksPerOperatingSystem[manifest.OperatingSystemLinux]))
		assert.Equal(t, 3, len(config.ManifestPreferences.BuilderTracksPerOperatingSystem[manifest.OperatingSystemWindows]))
	})

	t.Run("ReturnsCatalogConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)

		catalogConfig := config.Catalog

		assert.Nil(t, err)
		assert.Equal(t, 2, len(catalogConfig.Filters))
		assert.Equal(t, "type", catalogConfig.Filters[0])
		assert.Equal(t, "team", catalogConfig.Filters[1])
	})

	t.Run("ReturnsCredentialsConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)

		credentialsConfig := config.Credentials

		assert.Nil(t, err)
		assert.Equal(t, 9, len(credentialsConfig))
		assert.Equal(t, "container-registry-extensions", credentialsConfig[0].Name)
		assert.Equal(t, "container-registry", credentialsConfig[0].Type)
		assert.Equal(t, "extensions", credentialsConfig[0].AdditionalProperties["repository"])
		assert.Equal(t, "slack-webhook-estafette", credentialsConfig[6].Name)
		assert.Equal(t, "slack-webhook", credentialsConfig[6].Type)
		assert.Equal(t, "estafette", credentialsConfig[6].AdditionalProperties["workspace"])
	})

	t.Run("ReturnsBuildControlConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)

		assertT := assert.New(t)
		assertT.Nil(err)

		buildControlConfig := config.BuildControl
		assertT.Equal(List{"project1", "project2"}, buildControlConfig.Bitbucket.Allowed.Projects)
		assertT.Equal(List{"repo1"}, buildControlConfig.Bitbucket.Allowed.Repos)
		assertT.Equal(List{"project3", "project4"}, buildControlConfig.Bitbucket.Blocked.Projects)
		assertT.Equal(List{"repo2"}, buildControlConfig.Bitbucket.Blocked.Repos)
		assertT.Equal(List{"repo3"}, buildControlConfig.Github.Allowed)
		assertT.Equal(List{"repo4"}, buildControlConfig.Github.Blocked)
		assertT.Equal(List{"main", "master"}, buildControlConfig.Release.Repositories[AllRepositories].Allowed)
		assertT.Equal(List{"bugfix"}, buildControlConfig.Release.Repositories[AllRepositories].Blocked)
		assertT.Equal(List{"trvx-prd"}, buildControlConfig.Release.RestrictedClusters)
	})

	t.Run("ReturnsTrustedImagesConfig", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)

		trustedImagesConfig := config.TrustedImages

		assert.Nil(t, err)
		assert.Equal(t, 8, len(trustedImagesConfig))
		assert.Equal(t, "extensions/docker", trustedImagesConfig[0].ImagePath)
		assert.True(t, trustedImagesConfig[0].RunDocker)
		assert.Equal(t, 1, len(trustedImagesConfig[0].InjectedCredentialTypes))
		assert.Equal(t, "container-registry", trustedImagesConfig[0].InjectedCredentialTypes[0])

		assert.Equal(t, "multiple-git-sources-test", trustedImagesConfig[7].ImagePath)
		assert.False(t, trustedImagesConfig[7].RunDocker)
		assert.Equal(t, 3, len(trustedImagesConfig[7].InjectedCredentialTypes))
		assert.Equal(t, "bitbucket-api-token", trustedImagesConfig[7].InjectedCredentialTypes[0])
		assert.Equal(t, "github-api-token", trustedImagesConfig[7].InjectedCredentialTypes[1])
		assert.Equal(t, "cloudsource-api-token", trustedImagesConfig[7].InjectedCredentialTypes[2])
	})

	t.Run("AllowsCredentialConfigWithComplexAdditionalPropertiesToBeJSONMarshalled", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)
		assert.Nil(t, err)

		credentialsConfig := config.Credentials

		bytes, err := json.Marshal(credentialsConfig[2])

		assert.Nil(t, err)
		assert.Equal(t, "{\"name\":\"gke-estafette-production\",\"type\":\"kubernetes-engine\",\"additionalProperties\":{\"cluster\":\"production-europe-west2\",\"defaults\":{\"autoscale\":{\"min\":2},\"container\":{\"repository\":\"estafette\"},\"namespace\":\"estafette\",\"sidecars\":[{\"image\":\"estafette/openresty-sidecar:1.13.6.1-alpine\",\"type\":\"openresty\"}]},\"project\":\"estafette-production\",\"region\":\"europe-west2\",\"serviceAccountKeyfile\":\"{}\"}}", string(bytes))
	})

	t.Run("Overrides_Integrations_Github_Enabled_From_Envvar", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")
		_ = os.Setenv("ESCI_INTEGRATIONS_GITHUB_ENABLE", "false")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)

		assert.Nil(t, err)
		assert.Equal(t, false, config.Integrations.Github.Enable)
		_ = os.Setenv("ESCI_INTEGRATIONS_GITHUB_ENABLE", "")
	})

	t.Run("Overrides_BuildControl_From_Envvar", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")
		_ = os.Setenv("ESCI_BUILDCONTROL_BITBUCKET_ALLOWED_PROJECTS", "project1,project2")
		_ = os.Setenv("ESCI_BUILDCONTROL_BITBUCKET_ALLOWED_REPOS", "repo1,repo2")

		// act
		config, err := configReader.ReadConfigFromFiles("configs", true)

		assert.Nil(t, err)
		assert.Equal(t, List{"project1", "project2"}, config.BuildControl.Bitbucket.Allowed.Projects)
		assert.Equal(t, List{"repo1", "repo2"}, config.BuildControl.Bitbucket.Allowed.Repos)
		_ = os.Unsetenv("ESCI_BUILDCONTROL_BITBUCKET_ALLOWED_PROJECTS")
		_ = os.Unsetenv("ESCI_BUILDCONTROL_BITBUCKET_ALLOWED_REPOS")
	})

	t.Run("Keeps_Job_AffinitiesAndTolerations_Nil_For_Minimal_Config", func(t *testing.T) {

		configReader := NewConfigReader(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false), "za4BeKbXyMJVsX6gLU2AF352DEu9J5qE")

		// act
		config, err := configReader.ReadConfigFromFiles("minimal-configs", true)

		assert.Nil(t, err)
		assert.Nil(t, config.Jobs.BuildAffinityAndTolerations)
		assert.Nil(t, config.Jobs.ReleaseAffinityAndTolerations)
		assert.Nil(t, config.Jobs.BotAffinityAndTolerations)
	})
}

func TestWriteLogToDatabase(t *testing.T) {

	t.Run("ReturnsTrueIfLogWritersIsEmpty", func(t *testing.T) {

		config := APIServerConfig{
			LogWriters: []LogTarget{},
		}

		// act
		result := config.WriteLogToDatabase()

		assert.True(t, result)
	})

	t.Run("ReturnsTrueIfLogWritersContainsDatabase", func(t *testing.T) {

		config := APIServerConfig{
			LogWriters: []LogTarget{
				LogTargetCloudStorage,
				LogTargetDatabase,
			},
		}

		// act
		result := config.WriteLogToDatabase()

		assert.True(t, result)
	})

	t.Run("ReturnsFalseIfLogWritersDoesNotContainDatabase", func(t *testing.T) {

		config := APIServerConfig{
			LogWriters: []LogTarget{
				LogTargetCloudStorage,
			},
		}

		// act
		result := config.WriteLogToDatabase()

		assert.False(t, result)
	})
}

func TestWriteLogToCloudStorage(t *testing.T) {

	t.Run("ReturnsFalseIfLogWritersIsEmpty", func(t *testing.T) {

		config := APIServerConfig{
			LogWriters: []LogTarget{},
		}

		// act
		result := config.WriteLogToCloudStorage()

		assert.False(t, result)
	})

	t.Run("ReturnsTrueIfLogWritersContainsCloudStorage", func(t *testing.T) {

		config := APIServerConfig{
			LogWriters: []LogTarget{
				LogTargetCloudStorage,
				LogTargetDatabase,
			},
		}

		// act
		result := config.WriteLogToCloudStorage()

		assert.True(t, result)
	})

	t.Run("ReturnsFalseIfLogWritersDoesNotContainCloudStorage", func(t *testing.T) {

		config := APIServerConfig{
			LogWriters: []LogTarget{
				LogTargetDatabase,
			},
		}

		// act
		result := config.WriteLogToCloudStorage()

		assert.False(t, result)
	})
}

func TestReadLogFromDatabase(t *testing.T) {

	t.Run("ReturnsTrueIfLogReaderIsEmpty", func(t *testing.T) {

		config := APIServerConfig{
			LogReader: LogTargetUnknown,
		}

		// act
		result := config.ReadLogFromDatabase()

		assert.True(t, result)
	})

	t.Run("ReturnsTrueIfLogReaderEqualsDatabase", func(t *testing.T) {

		config := APIServerConfig{
			LogReader: LogTargetDatabase,
		}

		// act
		result := config.ReadLogFromDatabase()

		assert.True(t, result)
	})

	t.Run("ReturnsFalseIfLogReaderDoesNotEqualDatabase", func(t *testing.T) {

		config := APIServerConfig{
			LogReader: LogTargetCloudStorage,
		}

		// act
		result := config.ReadLogFromDatabase()

		assert.False(t, result)
	})
}

func TestReadLogFromCloudStorage(t *testing.T) {

	t.Run("ReturnsFalseIfLogReaderIsEmpty", func(t *testing.T) {

		config := APIServerConfig{
			LogReader: LogTargetUnknown,
		}

		// act
		result := config.ReadLogFromCloudStorage()

		assert.False(t, result)
	})

	t.Run("ReturnsTrueIfLogReaderEqualsCloudStorage", func(t *testing.T) {

		config := APIServerConfig{
			LogReader: LogTargetCloudStorage,
		}

		// act
		result := config.ReadLogFromCloudStorage()

		assert.True(t, result)
	})

	t.Run("ReturnsFalseIfLogReaderDoesNotCloudStorage", func(t *testing.T) {

		config := APIServerConfig{
			LogReader: LogTargetDatabase,
		}

		// act
		result := config.ReadLogFromCloudStorage()

		assert.False(t, result)
	})
}
