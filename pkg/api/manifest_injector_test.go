package api

import (
	"testing"

	"github.com/stretchr/testify/assert"

	manifest "github.com/estafette/estafette-ci-manifest"
)

func TestLabelSelectorMatches(t *testing.T) {
	t.Run("ReturnsTrueIfNoLabelSelectorHasBeenDefined", func(t *testing.T) {
		mft := manifest.EstafetteManifest{
			Labels: map[string]string{
				"language": "golang",
				"team":     "estafette",
			},
		}

		stage := manifest.EstafetteStage{}

		match := labelSelectorMatches(mft, stage)

		assert.True(t, match)
	})

	t.Run("ReturnsTrueIfAllLabelSelectorValuesMatchManifestLabelValues", func(t *testing.T) {
		mft := manifest.EstafetteManifest{
			Labels: map[string]string{
				"language": "golang",
				"team":     "estafette",
			},
		}

		stage := manifest.EstafetteStage{
			CustomProperties: map[string]interface{}{
				"labelSelector": map[string]interface{}{
					"language": "golang",
					"team":     "estafette",
				},
			},
		}

		match := labelSelectorMatches(mft, stage)

		assert.True(t, match)
	})

	t.Run("ReturnsTrueIfAllLabelSelectorValuesRegexesMatchManifestLabelValues", func(t *testing.T) {
		mft := manifest.EstafetteManifest{
			Labels: map[string]string{
				"language": "golang",
				"team":     "estafette",
			},
		}

		stage := manifest.EstafetteStage{
			CustomProperties: map[string]interface{}{
				"labelSelector": map[string]interface{}{
					"language": "golang|node",
					"team":     "estafette",
				},
			},
		}

		match := labelSelectorMatches(mft, stage)

		assert.True(t, match)
	})

	t.Run("ReturnsFalseIfAnyLabelSelectorValueDoesNotMatchManifestLabelValue", func(t *testing.T) {
		mft := manifest.EstafetteManifest{
			Labels: map[string]string{
				"language": "golang",
				"team":     "estafette",
			},
		}

		stage := manifest.EstafetteStage{
			CustomProperties: map[string]interface{}{
				"labelSelector": map[string]interface{}{
					"language": "node",
					"team":     "estafette",
				},
			},
		}

		match := labelSelectorMatches(mft, stage)

		assert.False(t, match)
	})
}

func TestInjectStages(t *testing.T) {

	t.Run("PrependParallelGitCloneStepInInitStage", func(t *testing.T) {

		mft := getManifestWithoutBuildStatusSteps()
		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
		}

		// act
		injectedManifest, err := InjectStages(config, mft, "beta", "github", "main", true)

		assert.Nil(t, err)
		if assert.Equal(t, 3, len(injectedManifest.Stages)) {
			assert.Equal(t, "injected-before-hardcoded", injectedManifest.Stages[0].Name)
			assert.Equal(t, 2, len(injectedManifest.Stages[0].ParallelStages))

			assert.Equal(t, "git-clone", injectedManifest.Stages[0].ParallelStages[0].Name)
			assert.Equal(t, "extensions/git-clone:beta", injectedManifest.Stages[0].ParallelStages[0].ContainerImage)
		}
	})

	t.Run("DoNotPrependParallelGitCloneStepIfGitCloneStepExists", func(t *testing.T) {

		mft := getManifestWithoutBuildStatusStepsAndWithGitClone()
		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
		}

		// act
		injectedManifest, err := InjectStages(config, mft, "dev", "github", "main", true)

		assert.Nil(t, err)
		if assert.Equal(t, 4, len(injectedManifest.Stages)) {
			assert.Equal(t, "injected-before-hardcoded", injectedManifest.Stages[0].Name)
			assert.Equal(t, 1, len(injectedManifest.Stages[0].ParallelStages))
			assert.Equal(t, "set-pending-build-status", injectedManifest.Stages[0].ParallelStages[0].Name)

			assert.Equal(t, "git-clone", injectedManifest.Stages[1].Name)
			assert.Equal(t, "extensions/git-clone:stable", injectedManifest.Stages[1].ContainerImage)
		}
	})

	t.Run("PrependParallelSetPendingBuildStatusStepInInitStage", func(t *testing.T) {

		mft := getManifestWithoutBuildStatusSteps()
		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
		}

		// act
		injectedManifest, err := InjectStages(config, mft, "beta", "github", "main", true)

		assert.Nil(t, err)
		if assert.Equal(t, 3, len(injectedManifest.Stages)) {
			assert.Equal(t, "injected-before-hardcoded", injectedManifest.Stages[0].Name)
			assert.Equal(t, 2, len(injectedManifest.Stages[0].ParallelStages))

			assert.Equal(t, "set-pending-build-status", injectedManifest.Stages[0].ParallelStages[1].Name)
			assert.Equal(t, "extensions/github-status:beta", injectedManifest.Stages[0].ParallelStages[1].ContainerImage)
			assert.Equal(t, "pending", injectedManifest.Stages[0].ParallelStages[1].CustomProperties["status"])
		}
	})

	t.Run("DoNotPrependParallelSetPendingBuildStatusStepIfPendingBuildStatusStepExists", func(t *testing.T) {

		mft := getManifestWithBuildStatusSteps()
		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
		}

		// act
		injectedManifest, err := InjectStages(config, mft, "dev", "github", "main", true)

		assert.Nil(t, err)
		if assert.Equal(t, 4, len(injectedManifest.Stages)) {
			assert.Equal(t, "injected-before-hardcoded", injectedManifest.Stages[0].Name)
			assert.Equal(t, 1, len(injectedManifest.Stages[0].ParallelStages))
			assert.Equal(t, "git-clone", injectedManifest.Stages[0].ParallelStages[0].Name)

			assert.Equal(t, "set-pending-build-status", injectedManifest.Stages[1].Name)
			assert.Equal(t, "extensions/github-status:stable", injectedManifest.Stages[1].ContainerImage)
			assert.Equal(t, "pending", injectedManifest.Stages[1].CustomProperties["status"])
		}
	})

	t.Run("DoNotPrependInitStageIfGitCloneAndSetPendingBuildStatusAndEnvvarsStagesExist", func(t *testing.T) {

		mft := getManifestWithAllSteps()
		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
		}

		// act
		injectedManifest, err := InjectStages(config, mft, "beta", "github", "main", true)

		assert.Nil(t, err)
		if assert.Equal(t, 5, len(injectedManifest.Stages)) {
			assert.Equal(t, "set-pending-build-status", injectedManifest.Stages[0].Name)
			assert.Equal(t, "git-clone", injectedManifest.Stages[1].Name)
			assert.Equal(t, "envvars", injectedManifest.Stages[2].Name)
			assert.Equal(t, "build", injectedManifest.Stages[3].Name)
			assert.Equal(t, "set-build-status", injectedManifest.Stages[4].Name)
		}
	})

	t.Run("DoNotPrependInitStageIfGitCloneAndSetPendingBuildStatusAndEnvvarsStagesExist", func(t *testing.T) {

		mft := getManifestWithAllSteps()
		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
		}

		// act
		injectedManifest, err := InjectStages(config, mft, "beta", "github", "main", true)

		assert.Nil(t, err)
		if assert.Equal(t, 5, len(injectedManifest.Stages)) {
			assert.Equal(t, "set-pending-build-status", injectedManifest.Stages[0].Name)
			assert.Equal(t, "git-clone", injectedManifest.Stages[1].Name)
			assert.Equal(t, "envvars", injectedManifest.Stages[2].Name)
			assert.Equal(t, "build", injectedManifest.Stages[3].Name)
			assert.Equal(t, "set-build-status", injectedManifest.Stages[4].Name)
		}
	})

	t.Run("PrependInitStageWithDifferentNameIfInjectedBeforeStageExist", func(t *testing.T) {

		mft := getManifestWithoutInjectedStepsButWithInjectedBeforeStage()
		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
		}

		// act
		injectedManifest, err := InjectStages(config, mft, "beta", "github", "main", true)

		assert.Nil(t, err)
		if assert.Equal(t, 4, len(injectedManifest.Stages)) {
			assert.Equal(t, "injected-before-hardcoded-0", injectedManifest.Stages[0].Name)
			assert.Equal(t, "injected-before-hardcoded", injectedManifest.Stages[1].Name)
			assert.Equal(t, "build", injectedManifest.Stages[2].Name)
			assert.Equal(t, "injected-after-hardcoded", injectedManifest.Stages[3].Name)
			assert.Equal(t, "set-build-status", injectedManifest.Stages[3].ParallelStages[0].Name)
		}
	})

	t.Run("AppendSetBuildStatusStep", func(t *testing.T) {

		mft := getManifestWithoutBuildStatusSteps()
		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
		}

		// act
		injectedManifest, err := InjectStages(config, mft, "beta", "github", "main", true)

		assert.Nil(t, err)
		if assert.Equal(t, 3, len(injectedManifest.Stages)) {
			assert.Equal(t, "injected-after-hardcoded", injectedManifest.Stages[2].Name)
			assert.Equal(t, "status == 'succeeded' || status == 'failed'", injectedManifest.Stages[2].When)
			assert.Equal(t, "set-build-status", injectedManifest.Stages[2].ParallelStages[0].Name)
			assert.Equal(t, "extensions/github-status:beta", injectedManifest.Stages[2].ParallelStages[0].ContainerImage)
		}
	})

	t.Run("DoNotAppendSetBuildStatusStepIfSetBuildStatusStepExists", func(t *testing.T) {

		mft := getManifestWithBuildStatusSteps()
		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
		}

		// act
		injectedManifest, err := InjectStages(config, mft, "dev", "github", "main", true)

		assert.Nil(t, err)
		if assert.Equal(t, 4, len(injectedManifest.Stages)) {
			assert.Equal(t, "set-build-status", injectedManifest.Stages[3].Name)
			assert.Equal(t, "extensions/github-status:stable", injectedManifest.Stages[3].ContainerImage)
		}
	})

	t.Run("PrependGitCloneStepToReleaseStagesIfCloneRepositoryIsTrue", func(t *testing.T) {

		mft := getManifestWithoutBuildStatusSteps()
		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
		}

		// act
		injectedManifest, err := InjectStages(config, mft, "beta", "github", "main", true)

		assert.Nil(t, err)
		if assert.Equal(t, 2, len(injectedManifest.Releases[1].Stages)) {
			assert.Equal(t, "injected-before-hardcoded", injectedManifest.Releases[1].Stages[0].Name)
			assert.Equal(t, "git-clone", injectedManifest.Releases[1].Stages[0].ParallelStages[0].Name)
			assert.Equal(t, "extensions/git-clone:beta", injectedManifest.Releases[1].Stages[0].ParallelStages[0].ContainerImage)
		}
	})

	t.Run("DoNotPrependGitCloneStepToReleaseStagesIfCloneRepositoryIsFalse", func(t *testing.T) {

		mft := getManifestWithoutBuildStatusSteps()
		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
		}

		// act
		injectedManifest, err := InjectStages(config, mft, "beta", "github", "main", true)

		assert.Nil(t, err)
		if assert.Equal(t, 1, len(injectedManifest.Releases[0].Stages)) {
			assert.Equal(t, "deploy", injectedManifest.Releases[0].Stages[0].Name)
			assert.Equal(t, "extensions/gke", injectedManifest.Releases[0].Stages[0].ContainerImage)
		}
	})

	t.Run("DoNotPrependGitCloneStepToReleaseStagesIfCloneRepositoryIsTrueButStageAlreadyExists", func(t *testing.T) {

		mft := getManifestWithBuildStatusSteps()
		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
		}

		// act
		injectedManifest, err := InjectStages(config, mft, "beta", "github", "main", true)

		assert.Nil(t, err)
		if assert.Equal(t, 2, len(injectedManifest.Releases[0].Stages)) {
			assert.Equal(t, "git-clone", injectedManifest.Releases[0].Stages[0].Name)
			assert.Equal(t, "extensions/git-clone:stable", injectedManifest.Releases[0].Stages[0].ContainerImage)
		}
	})

	t.Run("DoNotPrependParallelSetPendingBuildStatusStepIfSupportsBuildStatusFalse", func(t *testing.T) {

		mft := getManifestWithoutBuildStatusSteps()
		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
		}

		// act
		injectedManifest, err := InjectStages(config, mft, "dev", "source", "main", false)

		assert.Nil(t, err)
		if assert.Equal(t, 2, len(injectedManifest.Stages)) {
			assert.Equal(t, "injected-before-hardcoded", injectedManifest.Stages[0].Name)
			assert.Equal(t, 1, len(injectedManifest.Stages[0].ParallelStages))
			assert.Equal(t, "git-clone", injectedManifest.Stages[0].ParallelStages[0].Name)
		}
	})

	t.Run("InjectsConfiguredStagesIfDoesNotAlreadyExists", func(t *testing.T) {

		mft := manifest.EstafetteManifest{
			Stages: []*manifest.EstafetteStage{
				{
					Name:           "build",
					ContainerImage: "golang:1.10.2-alpine3.7",
				},
			},
			Releases: []*manifest.EstafetteRelease{
				{
					Name: "production",
					Stages: []*manifest.EstafetteStage{
						{
							Name:           "deploy",
							ContainerImage: "extensions/gke",
						},
					},
				},
			},
			Bots: []*manifest.EstafetteBot{
				{
					Name: "bot",
					Stages: []*manifest.EstafetteStage{
						{
							Name:           "bot",
							ContainerImage: "extensions/bot",
						},
					},
				},
			},
		}

		mft.SetDefaults(*manifest.GetDefaultManifestPreferences())

		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
			APIServer: &APIServerConfig{
				InjectStagesPerOperatingSystem: map[manifest.OperatingSystem]InjectStagesConfig{
					manifest.OperatingSystemLinux: {
						Build: &InjectStagesTypeConfig{
							Before: []*manifest.EstafetteStage{
								{
									Name:           "envvar-before",
									ContainerImage: "extensions/envvars:stable",
								},
							},
							After: []*manifest.EstafetteStage{
								{
									Name:           "envvar-after",
									ContainerImage: "extensions/envvars:stable",
								},
							},
						},
						Release: &InjectStagesTypeConfig{
							Before: []*manifest.EstafetteStage{
								{
									Name:           "envvar-before",
									ContainerImage: "extensions/envvars:stable",
								},
							},
							After: []*manifest.EstafetteStage{
								{
									Name:           "envvar-after",
									ContainerImage: "extensions/envvars:stable",
								},
							},
						},
						Bot: &InjectStagesTypeConfig{
							Before: []*manifest.EstafetteStage{
								{
									Name:           "envvar-before",
									ContainerImage: "extensions/envvars:stable",
								},
							},
							After: []*manifest.EstafetteStage{
								{
									Name:           "envvar-after",
									ContainerImage: "extensions/envvars:stable",
								},
							},
						},
					},
				},
			},
		}

		// act
		injectedManifest, err := InjectStages(config, mft, "beta", "github", "main", false)

		assert.Nil(t, err)
		assert.Equal(t, 4, len(injectedManifest.Stages))
		assert.Equal(t, "injected-before-hardcoded", injectedManifest.Stages[0].Name)
		assert.Equal(t, 1, len(injectedManifest.Stages[0].ParallelStages))
		assert.Equal(t, "git-clone", injectedManifest.Stages[0].ParallelStages[0].Name)
		assert.Equal(t, "extensions/git-clone:beta", injectedManifest.Stages[0].ParallelStages[0].ContainerImage)

		assert.Equal(t, "injected-before-configured", injectedManifest.Stages[1].Name)
		assert.Equal(t, 1, len(injectedManifest.Stages[1].ParallelStages))
		assert.Equal(t, "envvar-before", injectedManifest.Stages[1].ParallelStages[0].Name)
		assert.Equal(t, "extensions/envvars:stable", injectedManifest.Stages[1].ParallelStages[0].ContainerImage)

		assert.Equal(t, "injected-after-configured", injectedManifest.Stages[3].Name)
		assert.Equal(t, 1, len(injectedManifest.Stages[3].ParallelStages))
		assert.Equal(t, "envvar-after", injectedManifest.Stages[3].ParallelStages[0].Name)
		assert.Equal(t, "extensions/envvars:stable", injectedManifest.Stages[3].ParallelStages[0].ContainerImage)

		assert.Equal(t, 3, len(injectedManifest.Releases[0].Stages))
		assert.Equal(t, "injected-before-configured", injectedManifest.Releases[0].Stages[0].Name)
		assert.Equal(t, 1, len(injectedManifest.Releases[0].Stages[0].ParallelStages))
		assert.Equal(t, "envvar-before", injectedManifest.Releases[0].Stages[0].ParallelStages[0].Name)
		assert.Equal(t, "extensions/envvars:stable", injectedManifest.Releases[0].Stages[0].ParallelStages[0].ContainerImage)
		assert.Equal(t, "injected-after-configured", injectedManifest.Releases[0].Stages[2].Name)
		assert.Equal(t, 1, len(injectedManifest.Releases[0].Stages[2].ParallelStages))
		assert.Equal(t, "envvar-after", injectedManifest.Releases[0].Stages[2].ParallelStages[0].Name)
		assert.Equal(t, "extensions/envvars:stable", injectedManifest.Releases[0].Stages[2].ParallelStages[0].ContainerImage)

		assert.Equal(t, 3, len(injectedManifest.Bots[0].Stages))
		assert.Equal(t, "injected-before-configured", injectedManifest.Bots[0].Stages[0].Name)
		assert.Equal(t, 1, len(injectedManifest.Bots[0].Stages[0].ParallelStages))
		assert.Equal(t, "envvar-before", injectedManifest.Bots[0].Stages[0].ParallelStages[0].Name)
		assert.Equal(t, "extensions/envvars:stable", injectedManifest.Bots[0].Stages[0].ParallelStages[0].ContainerImage)
		assert.Equal(t, "injected-after-configured", injectedManifest.Bots[0].Stages[2].Name)
		assert.Equal(t, 1, len(injectedManifest.Bots[0].Stages[2].ParallelStages))
		assert.Equal(t, "envvar-after", injectedManifest.Bots[0].Stages[2].ParallelStages[0].Name)
		assert.Equal(t, "extensions/envvars:stable", injectedManifest.Bots[0].Stages[2].ParallelStages[0].ContainerImage)
	})

	t.Run("InjectsConfiguredStagesIfDoesNotAlreadyExistsAndLabelsMatchLabelSelector", func(t *testing.T) {

		mft := manifest.EstafetteManifest{
			Labels: map[string]string{
				"language": "golang",
				"team":     "estafette",
			},
			Stages: []*manifest.EstafetteStage{
				{
					Name:           "build",
					ContainerImage: "golang:1.10.2-alpine3.7",
				},
			},
			Releases: []*manifest.EstafetteRelease{
				{
					Name: "production",
					Stages: []*manifest.EstafetteStage{
						{
							Name:           "deploy",
							ContainerImage: "extensions/gke",
						},
					},
				},
			},
			Bots: []*manifest.EstafetteBot{
				{
					Name: "bot",
					Stages: []*manifest.EstafetteStage{
						{
							Name:           "bot",
							ContainerImage: "extensions/bot",
						},
					},
				},
			},
		}

		mft.SetDefaults(*manifest.GetDefaultManifestPreferences())

		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
			APIServer: &APIServerConfig{
				InjectStagesPerOperatingSystem: map[manifest.OperatingSystem]InjectStagesConfig{
					manifest.OperatingSystemLinux: {
						Build: &InjectStagesTypeConfig{
							Before: []*manifest.EstafetteStage{
								{
									Name:           "envvar-before",
									ContainerImage: "extensions/envvars:stable",
									CustomProperties: map[string]interface{}{
										"labelSelector": map[string]interface{}{
											"language": "golang|node",
											"team":     "estafette",
										},
									},
								},
							},
							After: []*manifest.EstafetteStage{
								{
									Name:           "envvar-after",
									ContainerImage: "extensions/envvars:stable",
									CustomProperties: map[string]interface{}{
										"labelSelector": map[string]interface{}{
											"language": "golang|node",
											"team":     "estafette",
										},
									},
								},
							},
						},
						Release: &InjectStagesTypeConfig{
							Before: []*manifest.EstafetteStage{
								{
									Name:           "envvar-before",
									ContainerImage: "extensions/envvars:stable",
									CustomProperties: map[string]interface{}{
										"labelSelector": map[string]interface{}{
											"language": "golang|node",
											"team":     "estafette",
										},
									},
								},
							},
							After: []*manifest.EstafetteStage{
								{
									Name:           "envvar-after",
									ContainerImage: "extensions/envvars:stable",
									CustomProperties: map[string]interface{}{
										"labelSelector": map[string]interface{}{
											"language": "golang|node",
											"team":     "estafette",
										},
									},
								},
							},
						},
						Bot: &InjectStagesTypeConfig{
							Before: []*manifest.EstafetteStage{
								{
									Name:           "envvar-before",
									ContainerImage: "extensions/envvars:stable",
									CustomProperties: map[string]interface{}{
										"labelSelector": map[string]interface{}{
											"language": "golang|node",
											"team":     "estafette",
										},
									},
								},
							},
							After: []*manifest.EstafetteStage{
								{
									Name:           "envvar-after",
									ContainerImage: "extensions/envvars:stable",
									CustomProperties: map[string]interface{}{
										"labelSelector": map[string]interface{}{
											"language": "golang|node",
											"team":     "estafette",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		// act
		injectedManifest, err := InjectStages(config, mft, "beta", "github", "main", false)

		assert.Nil(t, err)
		assert.Equal(t, 4, len(injectedManifest.Stages))

		assert.Equal(t, "injected-before-hardcoded", injectedManifest.Stages[0].Name)
		assert.Equal(t, 1, len(injectedManifest.Stages[0].ParallelStages))
		assert.Equal(t, "git-clone", injectedManifest.Stages[0].ParallelStages[0].Name)
		assert.Equal(t, "extensions/git-clone:beta", injectedManifest.Stages[0].ParallelStages[0].ContainerImage)

		assert.Equal(t, "injected-before-configured", injectedManifest.Stages[1].Name)
		assert.Equal(t, 1, len(injectedManifest.Stages[1].ParallelStages))
		assert.Equal(t, "envvar-before", injectedManifest.Stages[1].ParallelStages[0].Name)
		assert.Equal(t, "extensions/envvars:stable", injectedManifest.Stages[1].ParallelStages[0].ContainerImage)

		assert.Equal(t, "injected-after-configured", injectedManifest.Stages[3].Name)
		assert.Equal(t, 1, len(injectedManifest.Stages[3].ParallelStages))
		assert.Equal(t, "envvar-after", injectedManifest.Stages[3].ParallelStages[0].Name)
		assert.Equal(t, "extensions/envvars:stable", injectedManifest.Stages[3].ParallelStages[0].ContainerImage)

		assert.Equal(t, 3, len(injectedManifest.Releases[0].Stages))
		assert.Equal(t, "injected-before-configured", injectedManifest.Releases[0].Stages[0].Name)
		assert.Equal(t, 1, len(injectedManifest.Releases[0].Stages[0].ParallelStages))
		assert.Equal(t, "envvar-before", injectedManifest.Releases[0].Stages[0].ParallelStages[0].Name)
		assert.Equal(t, "extensions/envvars:stable", injectedManifest.Releases[0].Stages[0].ParallelStages[0].ContainerImage)
		assert.Equal(t, "injected-after-configured", injectedManifest.Releases[0].Stages[2].Name)
		assert.Equal(t, 1, len(injectedManifest.Releases[0].Stages[2].ParallelStages))
		assert.Equal(t, "envvar-after", injectedManifest.Releases[0].Stages[2].ParallelStages[0].Name)
		assert.Equal(t, "extensions/envvars:stable", injectedManifest.Releases[0].Stages[2].ParallelStages[0].ContainerImage)

		assert.Equal(t, 3, len(injectedManifest.Bots[0].Stages))
		assert.Equal(t, "injected-before-configured", injectedManifest.Bots[0].Stages[0].Name)
		assert.Equal(t, 1, len(injectedManifest.Bots[0].Stages[0].ParallelStages))
		assert.Equal(t, "envvar-before", injectedManifest.Bots[0].Stages[0].ParallelStages[0].Name)
		assert.Equal(t, "extensions/envvars:stable", injectedManifest.Bots[0].Stages[0].ParallelStages[0].ContainerImage)
		assert.Equal(t, "injected-after-configured", injectedManifest.Bots[0].Stages[2].Name)
		assert.Equal(t, 1, len(injectedManifest.Bots[0].Stages[2].ParallelStages))
		assert.Equal(t, "envvar-after", injectedManifest.Bots[0].Stages[2].ParallelStages[0].Name)
		assert.Equal(t, "extensions/envvars:stable", injectedManifest.Bots[0].Stages[2].ParallelStages[0].ContainerImage)
	})

	t.Run("DoesNotInjectConfiguredStagesIfAlreadyExists", func(t *testing.T) {

		mft := manifest.EstafetteManifest{
			Stages: []*manifest.EstafetteStage{
				{
					Name:           "envvar-before",
					ContainerImage: "extensions/envvars:stable",
				},
				{
					Name:           "build",
					ContainerImage: "golang:1.10.2-alpine3.7",
				},
				{
					Name:           "envvar-after",
					ContainerImage: "extensions/envvars:stable",
				},
			},
			Releases: []*manifest.EstafetteRelease{
				{
					Name: "production",
					Stages: []*manifest.EstafetteStage{
						{
							Name:           "envvar-before",
							ContainerImage: "extensions/envvars:stable",
						},
						{
							Name:           "deploy",
							ContainerImage: "extensions/gke",
						},
						{
							Name:           "envvar-after",
							ContainerImage: "extensions/envvars:stable",
						},
					},
				},
			},
			Bots: []*manifest.EstafetteBot{
				{
					Name: "bot",
					Stages: []*manifest.EstafetteStage{
						{
							Name:           "envvar-before",
							ContainerImage: "extensions/envvars:stable",
						},
						{
							Name:           "bot",
							ContainerImage: "extensions/bot",
						},
						{
							Name:           "envvar-after",
							ContainerImage: "extensions/envvars:stable",
						},
					},
				},
			},
		}

		mft.SetDefaults(*manifest.GetDefaultManifestPreferences())

		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
			APIServer: &APIServerConfig{
				InjectStagesPerOperatingSystem: map[manifest.OperatingSystem]InjectStagesConfig{
					manifest.OperatingSystemLinux: {
						Build: &InjectStagesTypeConfig{
							Before: []*manifest.EstafetteStage{
								{
									Name:           "envvar-before",
									ContainerImage: "extensions/envvars:stable",
								},
							},
							After: []*manifest.EstafetteStage{
								{
									Name:           "envvar-after",
									ContainerImage: "extensions/envvars:stable",
								},
							},
						},
						Release: &InjectStagesTypeConfig{
							Before: []*manifest.EstafetteStage{
								{
									Name:           "envvar-before",
									ContainerImage: "extensions/envvars:stable",
								},
							},
							After: []*manifest.EstafetteStage{
								{
									Name:           "envvar-after",
									ContainerImage: "extensions/envvars:stable",
								},
							},
						},
						Bot: &InjectStagesTypeConfig{
							Before: []*manifest.EstafetteStage{
								{
									Name:           "envvar-before",
									ContainerImage: "extensions/envvars:stable",
								},
							},
							After: []*manifest.EstafetteStage{
								{
									Name:           "envvar-after",
									ContainerImage: "extensions/envvars:stable",
								},
							},
						},
					},
				},
			},
		}

		// act
		injectedManifest, err := InjectStages(config, mft, "dev", "source", "main", false)

		assert.Nil(t, err)
		assert.Equal(t, 4, len(injectedManifest.Stages))
		assert.Equal(t, "injected-before-hardcoded", injectedManifest.Stages[0].Name)
		assert.Equal(t, 1, len(injectedManifest.Stages[0].ParallelStages))
		assert.Equal(t, "envvar-before", injectedManifest.Stages[1].Name)
		assert.Equal(t, "extensions/envvars:stable", injectedManifest.Stages[1].ContainerImage)
		assert.Equal(t, "envvar-after", injectedManifest.Stages[3].Name)
		assert.Equal(t, "extensions/envvars:stable", injectedManifest.Stages[3].ContainerImage)

		assert.Equal(t, 3, len(injectedManifest.Releases[0].Stages))
		assert.Equal(t, "envvar-before", injectedManifest.Releases[0].Stages[0].Name)
		assert.Equal(t, "extensions/envvars:stable", injectedManifest.Releases[0].Stages[0].ContainerImage)
		assert.Equal(t, "envvar-after", injectedManifest.Releases[0].Stages[2].Name)
		assert.Equal(t, "extensions/envvars:stable", injectedManifest.Releases[0].Stages[2].ContainerImage)

		assert.Equal(t, 3, len(injectedManifest.Bots[0].Stages))
		assert.Equal(t, "envvar-before", injectedManifest.Bots[0].Stages[0].Name)
		assert.Equal(t, "extensions/envvars:stable", injectedManifest.Bots[0].Stages[0].ContainerImage)
		assert.Equal(t, "envvar-after", injectedManifest.Bots[0].Stages[2].Name)
		assert.Equal(t, "extensions/envvars:stable", injectedManifest.Bots[0].Stages[2].ContainerImage)
	})

	t.Run("DoesNotInjectConfiguredStagesIfLabelSelectorDoesNotMatch", func(t *testing.T) {

		mft := manifest.EstafetteManifest{
			Labels: map[string]string{
				"language": "python",
				"team":     "estafette",
			},
			Stages: []*manifest.EstafetteStage{
				{
					Name:           "build",
					ContainerImage: "golang:1.10.2-alpine3.7",
				},
			},
			Releases: []*manifest.EstafetteRelease{
				{
					Name: "production",
					Stages: []*manifest.EstafetteStage{
						{
							Name:           "deploy",
							ContainerImage: "extensions/gke",
						},
					},
				},
			},
			Bots: []*manifest.EstafetteBot{
				{
					Name: "bot",
					Stages: []*manifest.EstafetteStage{
						{
							Name:           "bot",
							ContainerImage: "extensions/bot",
						},
					},
				},
			},
		}

		mft.SetDefaults(*manifest.GetDefaultManifestPreferences())

		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
			APIServer: &APIServerConfig{
				InjectStagesPerOperatingSystem: map[manifest.OperatingSystem]InjectStagesConfig{
					manifest.OperatingSystemLinux: {
						Build: &InjectStagesTypeConfig{
							Before: []*manifest.EstafetteStage{
								{
									Name:           "envvar-before",
									ContainerImage: "extensions/envvars:stable",
									CustomProperties: map[string]interface{}{
										"labelSelector": map[string]interface{}{
											"language": "golang|node",
											"team":     "estafette",
										},
									},
								},
							},
							After: []*manifest.EstafetteStage{
								{
									Name:           "envvar-after",
									ContainerImage: "extensions/envvars:stable",
									CustomProperties: map[string]interface{}{
										"labelSelector": map[string]interface{}{
											"language": "golang|node",
											"team":     "estafette",
										},
									},
								},
							},
						},
						Release: &InjectStagesTypeConfig{
							Before: []*manifest.EstafetteStage{
								{
									Name:           "envvar-before",
									ContainerImage: "extensions/envvars:stable",
									CustomProperties: map[string]interface{}{
										"labelSelector": map[string]interface{}{
											"language": "golang|node",
											"team":     "estafette",
										},
									},
								},
							},
							After: []*manifest.EstafetteStage{
								{
									Name:           "envvar-after",
									ContainerImage: "extensions/envvars:stable",
									CustomProperties: map[string]interface{}{
										"labelSelector": map[string]interface{}{
											"language": "golang|node",
											"team":     "estafette",
										},
									},
								},
							},
						},
						Bot: &InjectStagesTypeConfig{
							Before: []*manifest.EstafetteStage{
								{
									Name:           "envvar-before",
									ContainerImage: "extensions/envvars:stable",
									CustomProperties: map[string]interface{}{
										"labelSelector": map[string]interface{}{
											"language": "golang|node",
											"team":     "estafette",
										},
									},
								},
							},
							After: []*manifest.EstafetteStage{
								{
									Name:           "envvar-after",
									ContainerImage: "extensions/envvars:stable",
									CustomProperties: map[string]interface{}{
										"labelSelector": map[string]interface{}{
											"language": "golang|node",
											"team":     "estafette",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		// act
		injectedManifest, err := InjectStages(config, mft, "dev", "source", "main", false)

		assert.Nil(t, err)
		assert.Equal(t, 2, len(injectedManifest.Stages))
		assert.Equal(t, 1, len(injectedManifest.Releases[0].Stages))
		assert.Equal(t, 1, len(injectedManifest.Bots[0].Stages))
	})
}

func getManifestWithoutBuildStatusSteps() manifest.EstafetteManifest {

	trueValue := true
	falseValue := false

	return manifest.EstafetteManifest{
		Builder: manifest.EstafetteBuilder{
			Track: "stable",
		},
		Version: manifest.EstafetteVersion{
			SemVer: &manifest.EstafetteSemverVersion{
				Major:         1,
				Minor:         0,
				Patch:         "234",
				LabelTemplate: "{{branch}}",
				ReleaseBranch: manifest.StringOrStringArray{Values: []string{"master"}},
			},
		},
		Labels:        map[string]string{},
		GlobalEnvVars: map[string]string{},
		Stages: []*manifest.EstafetteStage{
			{
				Name:             "build",
				ContainerImage:   "golang:1.10.2-alpine3.7",
				WorkingDirectory: "/go/src/github.com/estafette/${ESTAFETTE_GIT_NAME}",
			},
		},
		Releases: []*manifest.EstafetteRelease{
			{
				Name:            "staging",
				CloneRepository: &falseValue,
				Stages: []*manifest.EstafetteStage{
					{
						Name:           "deploy",
						ContainerImage: "extensions/gke",
					},
				},
			},
			{
				Name:            "production",
				CloneRepository: &trueValue,
				Stages: []*manifest.EstafetteStage{
					{
						Name:           "deploy",
						ContainerImage: "extensions/gke",
						Shell:          "/bin/sh",
						When:           "status == 'succeeded'",
					},
				},
			},
		},
	}
}

func getManifestWithoutBuildStatusStepsAndWithGitClone() manifest.EstafetteManifest {

	trueValue := true
	falseValue := false

	return manifest.EstafetteManifest{
		Builder: manifest.EstafetteBuilder{
			Track: "stable",
		},
		Version: manifest.EstafetteVersion{
			SemVer: &manifest.EstafetteSemverVersion{
				Major:         1,
				Minor:         0,
				Patch:         "234",
				LabelTemplate: "{{branch}}",
				ReleaseBranch: manifest.StringOrStringArray{Values: []string{"master"}},
			},
		},
		Labels:        map[string]string{},
		GlobalEnvVars: map[string]string{},
		Stages: []*manifest.EstafetteStage{
			{
				Name:           "git-clone",
				ContainerImage: "extensions/git-clone:stable",
				Shell:          "/bin/sh",
				When:           "status == 'succeeded'",
			},
			{
				Name:             "build",
				ContainerImage:   "golang:1.10.2-alpine3.7",
				WorkingDirectory: "/go/src/github.com/estafette/${ESTAFETTE_GIT_NAME}",
			},
		},
		Releases: []*manifest.EstafetteRelease{
			{
				Name:            "staging",
				CloneRepository: &falseValue,
				Stages: []*manifest.EstafetteStage{
					{
						Name:           "deploy",
						ContainerImage: "extensions/gke",
					},
				},
			},
			{
				Name:            "production",
				CloneRepository: &trueValue,
				Stages: []*manifest.EstafetteStage{
					{
						Name:           "deploy",
						ContainerImage: "extensions/gke",
						Shell:          "/bin/sh",
						When:           "status == 'succeeded'",
					},
				},
			},
		},
	}
}

func getManifestWithBuildStatusSteps() manifest.EstafetteManifest {

	trueValue := true

	return manifest.EstafetteManifest{
		Builder: manifest.EstafetteBuilder{
			Track: "stable",
		},
		Version: manifest.EstafetteVersion{
			SemVer: &manifest.EstafetteSemverVersion{
				Major:         1,
				Minor:         0,
				Patch:         "234",
				LabelTemplate: "{{branch}}",
				ReleaseBranch: manifest.StringOrStringArray{Values: []string{"master"}},
			},
		},
		Labels:        map[string]string{},
		GlobalEnvVars: map[string]string{},
		Stages: []*manifest.EstafetteStage{
			{
				Name:           "set-pending-build-status",
				ContainerImage: "extensions/github-status:stable",
				CustomProperties: map[string]interface{}{
					"status": "pending",
				},
				Shell:            "/bin/sh",
				WorkingDirectory: "/estafette-work",
				When:             "status == 'succeeded'",
			},
			{
				Name:             "build",
				ContainerImage:   "golang:1.10.2-alpine3.7",
				Shell:            "/bin/sh",
				WorkingDirectory: "/go/src/github.com/estafette/${ESTAFETTE_GIT_NAME}",
				When:             "status == 'succeeded'",
			},
			{
				Name:             "set-build-status",
				ContainerImage:   "extensions/github-status:stable",
				Shell:            "/bin/sh",
				WorkingDirectory: "/estafette-work",
				When:             "status == 'succeeded' || status == 'failed'",
			},
		},
		Releases: []*manifest.EstafetteRelease{
			{
				Name:            "production",
				CloneRepository: &trueValue,
				Stages: []*manifest.EstafetteStage{
					{
						Name:           "git-clone",
						ContainerImage: "extensions/git-clone:stable",
						Shell:          "/bin/sh",
						When:           "status == 'succeeded'",
					},
					{
						Name:           "deploy",
						ContainerImage: "extensions/gke",
						Shell:          "/bin/sh",
						When:           "status == 'succeeded'",
					},
				},
			},
		},
	}
}

func getManifestWithAllSteps() manifest.EstafetteManifest {

	trueValue := true

	return manifest.EstafetteManifest{
		Builder: manifest.EstafetteBuilder{
			Track: "stable",
		},
		Version: manifest.EstafetteVersion{
			SemVer: &manifest.EstafetteSemverVersion{
				Major:         1,
				Minor:         0,
				Patch:         "234",
				LabelTemplate: "{{branch}}",
				ReleaseBranch: manifest.StringOrStringArray{Values: []string{"master"}},
			},
		},
		Labels:        map[string]string{},
		GlobalEnvVars: map[string]string{},
		Stages: []*manifest.EstafetteStage{
			{
				Name:           "set-pending-build-status",
				ContainerImage: "extensions/github-status:stable",
				CustomProperties: map[string]interface{}{
					"status": "pending",
				},
				Shell:            "/bin/sh",
				WorkingDirectory: "/estafette-work",
				When:             "status == 'succeeded'",
			},
			{
				Name:           "git-clone",
				ContainerImage: "extensions/git-clone:stable",
				Shell:          "/bin/sh",
				When:           "status == 'succeeded'",
			},
			{
				Name:           "envvars",
				ContainerImage: "extensions/envvars:stable",
				Shell:          "/bin/sh",
				When:           "status == 'succeeded'",
			},
			{
				Name:             "build",
				ContainerImage:   "golang:1.10.2-alpine3.7",
				Shell:            "/bin/sh",
				WorkingDirectory: "/go/src/github.com/estafette/${ESTAFETTE_GIT_NAME}",
				When:             "status == 'succeeded'",
			},
			{
				Name:             "set-build-status",
				ContainerImage:   "extensions/github-status:stable",
				Shell:            "/bin/sh",
				WorkingDirectory: "/estafette-work",
				When:             "status == 'succeeded' || status == 'failed'",
			},
		},
		Releases: []*manifest.EstafetteRelease{
			{
				Name:            "production",
				CloneRepository: &trueValue,
				Stages: []*manifest.EstafetteStage{
					{
						Name:           "git-clone",
						ContainerImage: "extensions/git-clone:stable",
						Shell:          "/bin/sh",
						When:           "status == 'succeeded'",
					},
					{
						Name:           "deploy",
						ContainerImage: "extensions/gke",
						Shell:          "/bin/sh",
						When:           "status == 'succeeded'",
					},
				},
			},
		},
	}
}

func getManifestWithoutInjectedStepsButWithInjectedBeforeStage() manifest.EstafetteManifest {

	trueValue := true
	falseValue := false

	return manifest.EstafetteManifest{
		Builder: manifest.EstafetteBuilder{
			Track: "stable",
		},
		Version: manifest.EstafetteVersion{
			SemVer: &manifest.EstafetteSemverVersion{
				Major:         1,
				Minor:         0,
				Patch:         "234",
				LabelTemplate: "{{branch}}",
				ReleaseBranch: manifest.StringOrStringArray{Values: []string{"master"}},
			},
		},
		Labels:        map[string]string{},
		GlobalEnvVars: map[string]string{},
		Stages: []*manifest.EstafetteStage{
			{
				Name: "injected-before-hardcoded",
			},
			{
				Name:             "build",
				ContainerImage:   "golang:1.10.2-alpine3.7",
				WorkingDirectory: "/go/src/github.com/estafette/${ESTAFETTE_GIT_NAME}",
			},
		},
		Releases: []*manifest.EstafetteRelease{
			{
				Name:            "staging",
				CloneRepository: &falseValue,
				Stages: []*manifest.EstafetteStage{
					{
						Name:           "deploy",
						ContainerImage: "extensions/gke",
					},
				},
			},
			{
				Name:            "production",
				CloneRepository: &trueValue,
				Stages: []*manifest.EstafetteStage{
					{
						Name:           "deploy",
						ContainerImage: "extensions/gke",
						Shell:          "/bin/sh",
						When:           "status == 'succeeded'",
					},
				},
			},
		},
	}
}

func TestInjectCommands(t *testing.T) {

	t.Run("PrependsBeforeCommandsConfiguredForOperatingSystemAndShell", func(t *testing.T) {

		mft := manifest.EstafetteManifest{
			Builder: manifest.EstafetteBuilder{
				OperatingSystem: manifest.OperatingSystemLinux,
			},
			Stages: []*manifest.EstafetteStage{
				{
					Name:  "build",
					Shell: "/bin/sh",
					Commands: []string{
						"build",
					},
					ParallelStages: []*manifest.EstafetteStage{
						{
							Name:  "parallel",
							Shell: "/bin/sh",
							Commands: []string{
								"parallel",
							},
						},
					},
				},
			},
			Releases: []*manifest.EstafetteRelease{
				{
					Stages: []*manifest.EstafetteStage{
						{
							Name:  "release",
							Shell: "/bin/sh",
							Commands: []string{
								"release",
							},
						},
					},
				},
			},
		}

		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
			APIServer: &APIServerConfig{
				InjectCommandsPerOperatingSystemAndShell: map[manifest.OperatingSystem]map[string]InjectCommandsConfig{
					manifest.OperatingSystemLinux: {
						"/bin/sh": {
							Before: []string{
								"echo first",
							},
						},
					},
				},
			},
		}

		// act
		injectedManifest := InjectCommands(config, mft)

		assert.Equal(t, 2, len(injectedManifest.Stages[0].Commands))
		assert.Equal(t, "echo first", injectedManifest.Stages[0].Commands[0])
		assert.Equal(t, "build", injectedManifest.Stages[0].Commands[1])

		assert.Equal(t, 2, len(injectedManifest.Stages[0].ParallelStages[0].Commands))
		assert.Equal(t, "echo first", injectedManifest.Stages[0].ParallelStages[0].Commands[0])
		assert.Equal(t, "parallel", injectedManifest.Stages[0].ParallelStages[0].Commands[1])

		assert.Equal(t, 2, len(injectedManifest.Releases[0].Stages[0].Commands))
		assert.Equal(t, "echo first", injectedManifest.Releases[0].Stages[0].Commands[0])
		assert.Equal(t, "release", injectedManifest.Releases[0].Stages[0].Commands[1])
	})

	t.Run("DoesNotPrependBeforeCommandsConfiguredForOperatingSystemAndShellIfStageHasNoCommands", func(t *testing.T) {

		mft := manifest.EstafetteManifest{
			Builder: manifest.EstafetteBuilder{
				OperatingSystem: manifest.OperatingSystemLinux,
			},
			Stages: []*manifest.EstafetteStage{
				{
					Name:     "build",
					Shell:    "/bin/sh",
					Commands: []string{},
					ParallelStages: []*manifest.EstafetteStage{
						{
							Name:     "parallel",
							Shell:    "/bin/sh",
							Commands: []string{},
						},
					},
				},
			},
			Releases: []*manifest.EstafetteRelease{
				{
					Stages: []*manifest.EstafetteStage{
						{
							Name:     "release",
							Shell:    "/bin/sh",
							Commands: []string{},
						},
					},
				},
			},
		}

		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
			APIServer: &APIServerConfig{
				InjectCommandsPerOperatingSystemAndShell: map[manifest.OperatingSystem]map[string]InjectCommandsConfig{
					manifest.OperatingSystemLinux: {
						"/bin/sh": {
							Before: []string{
								"echo first",
							},
						},
					},
				},
			},
		}

		// act
		injectedManifest := InjectCommands(config, mft)

		assert.Equal(t, 0, len(injectedManifest.Stages[0].Commands))
		assert.Equal(t, 0, len(injectedManifest.Stages[0].ParallelStages[0].Commands))
		assert.Equal(t, 0, len(injectedManifest.Releases[0].Stages[0].Commands))
	})

	t.Run("DoesNotPrependBeforeCommandsConfiguredForOperatingSystemAndShellIfNoneHaveBeenConfiguredBefore", func(t *testing.T) {

		mft := manifest.EstafetteManifest{
			Builder: manifest.EstafetteBuilder{
				OperatingSystem: manifest.OperatingSystemLinux,
			},
			Stages: []*manifest.EstafetteStage{
				{
					Name:  "build",
					Shell: "/bin/sh",
					Commands: []string{
						"build",
					},
					ParallelStages: []*manifest.EstafetteStage{
						{
							Name:  "parallel",
							Shell: "/bin/sh",
							Commands: []string{
								"parallel",
							},
						},
					},
				},
			},
			Releases: []*manifest.EstafetteRelease{
				{
					Stages: []*manifest.EstafetteStage{
						{
							Name:  "release",
							Shell: "/bin/sh",
							Commands: []string{
								"release",
							},
						},
					},
				},
			},
		}

		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
			APIServer: &APIServerConfig{
				InjectCommandsPerOperatingSystemAndShell: map[manifest.OperatingSystem]map[string]InjectCommandsConfig{
					manifest.OperatingSystemLinux: {
						"/bin/sh": {
							Before: []string{},
						},
					},
				},
			},
		}

		// act
		injectedManifest := InjectCommands(config, mft)

		assert.Equal(t, 1, len(injectedManifest.Stages[0].Commands))
		assert.Equal(t, 1, len(injectedManifest.Stages[0].ParallelStages[0].Commands))
		assert.Equal(t, 1, len(injectedManifest.Releases[0].Stages[0].Commands))
	})

	t.Run("DoesNotAppendAfterCommandsConfiguredForOperatingSystemAndShellIfNoneHaveBeenConfiguredAfter", func(t *testing.T) {

		mft := manifest.EstafetteManifest{
			Builder: manifest.EstafetteBuilder{
				OperatingSystem: manifest.OperatingSystemLinux,
			},
			Stages: []*manifest.EstafetteStage{
				{
					Name:  "build",
					Shell: "/bin/sh",
					Commands: []string{
						"build",
					},
					ParallelStages: []*manifest.EstafetteStage{
						{
							Name:  "parallel",
							Shell: "/bin/sh",
							Commands: []string{
								"parallel",
							},
						},
					},
				},
			},
			Releases: []*manifest.EstafetteRelease{
				{
					Stages: []*manifest.EstafetteStage{
						{
							Name:  "release",
							Shell: "/bin/sh",
							Commands: []string{
								"release",
							},
						},
					},
				},
			},
		}

		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
			APIServer: &APIServerConfig{
				InjectCommandsPerOperatingSystemAndShell: map[manifest.OperatingSystem]map[string]InjectCommandsConfig{
					manifest.OperatingSystemLinux: {
						"/bin/sh": {
							After: []string{},
						},
					},
				},
			},
		}

		// act
		injectedManifest := InjectCommands(config, mft)

		assert.Equal(t, 1, len(injectedManifest.Stages[0].Commands))
		assert.Equal(t, 1, len(injectedManifest.Stages[0].ParallelStages[0].Commands))
		assert.Equal(t, 1, len(injectedManifest.Releases[0].Stages[0].Commands))
	})

	t.Run("DoesNotPrependBeforeCommandsConfiguredForOperatingSystemAndShellIfNoneHaveBeenConfiguredForShell", func(t *testing.T) {

		mft := manifest.EstafetteManifest{
			Builder: manifest.EstafetteBuilder{
				OperatingSystem: manifest.OperatingSystemLinux,
			},
			Stages: []*manifest.EstafetteStage{
				{
					Name:  "build",
					Shell: "/bin/sh",
					Commands: []string{
						"build",
					},
					ParallelStages: []*manifest.EstafetteStage{
						{
							Name:  "parallel",
							Shell: "/bin/sh",
							Commands: []string{
								"parallel",
							},
						},
					},
				},
			},
			Releases: []*manifest.EstafetteRelease{
				{
					Stages: []*manifest.EstafetteStage{
						{
							Name:  "release",
							Shell: "/bin/sh",
							Commands: []string{
								"release",
							},
						},
					},
				},
			},
		}

		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
			APIServer: &APIServerConfig{
				InjectCommandsPerOperatingSystemAndShell: map[manifest.OperatingSystem]map[string]InjectCommandsConfig{
					manifest.OperatingSystemLinux: {},
				},
			},
		}

		// act
		injectedManifest := InjectCommands(config, mft)

		assert.Equal(t, 1, len(injectedManifest.Stages[0].Commands))
		assert.Equal(t, 1, len(injectedManifest.Stages[0].ParallelStages[0].Commands))
		assert.Equal(t, 1, len(injectedManifest.Releases[0].Stages[0].Commands))
	})

	t.Run("DoesNotPrependBeforeCommandsConfiguredForOperatingSystemAndShellIfNoneHaveBeenConfiguredForOperatingSystem", func(t *testing.T) {

		mft := manifest.EstafetteManifest{
			Builder: manifest.EstafetteBuilder{
				OperatingSystem: manifest.OperatingSystemLinux,
			},
			Stages: []*manifest.EstafetteStage{
				{
					Name:  "build",
					Shell: "/bin/sh",
					Commands: []string{
						"build",
					},
					ParallelStages: []*manifest.EstafetteStage{
						{
							Name:  "parallel",
							Shell: "/bin/sh",
							Commands: []string{
								"parallel",
							},
						},
					},
				},
			},
			Releases: []*manifest.EstafetteRelease{
				{
					Stages: []*manifest.EstafetteStage{
						{
							Name:  "release",
							Shell: "/bin/sh",
							Commands: []string{
								"release",
							},
						},
					},
				},
			},
		}

		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
			APIServer: &APIServerConfig{
				InjectCommandsPerOperatingSystemAndShell: map[manifest.OperatingSystem]map[string]InjectCommandsConfig{},
			},
		}

		// act
		injectedManifest := InjectCommands(config, mft)

		assert.Equal(t, 1, len(injectedManifest.Stages[0].Commands))
		assert.Equal(t, 1, len(injectedManifest.Stages[0].ParallelStages[0].Commands))
		assert.Equal(t, 1, len(injectedManifest.Releases[0].Stages[0].Commands))
	})

	t.Run("DoesNotPrependBeforeCommandsConfiguredForOperatingSystemAndShellIfNoneHaveBeenConfiguredAtAll", func(t *testing.T) {

		mft := manifest.EstafetteManifest{
			Builder: manifest.EstafetteBuilder{
				OperatingSystem: manifest.OperatingSystemLinux,
			},
			Stages: []*manifest.EstafetteStage{
				{
					Name:  "build",
					Shell: "/bin/sh",
					Commands: []string{
						"build",
					},
					ParallelStages: []*manifest.EstafetteStage{
						{
							Name:  "parallel",
							Shell: "/bin/sh",
							Commands: []string{
								"parallel",
							},
						},
					},
				},
			},
			Releases: []*manifest.EstafetteRelease{
				{
					Stages: []*manifest.EstafetteStage{
						{
							Name:  "release",
							Shell: "/bin/sh",
							Commands: []string{
								"release",
							},
						},
					},
				},
			},
		}

		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
		}

		// act
		injectedManifest := InjectCommands(config, mft)

		assert.Equal(t, 1, len(injectedManifest.Stages[0].Commands))
		assert.Equal(t, 1, len(injectedManifest.Stages[0].ParallelStages[0].Commands))
		assert.Equal(t, 1, len(injectedManifest.Releases[0].Stages[0].Commands))
	})

	t.Run("AppendsAfterCommandsConfiguredForOperatingSystemAndShell", func(t *testing.T) {

		mft := manifest.EstafetteManifest{
			Builder: manifest.EstafetteBuilder{
				OperatingSystem: manifest.OperatingSystemLinux,
			},
			Stages: []*manifest.EstafetteStage{
				{
					Name:  "build",
					Shell: "/bin/sh",
					Commands: []string{
						"build",
					},
					ParallelStages: []*manifest.EstafetteStage{
						{
							Name:  "parallel",
							Shell: "/bin/sh",
							Commands: []string{
								"parallel",
							},
						},
					},
				},
			},
			Releases: []*manifest.EstafetteRelease{
				{
					Stages: []*manifest.EstafetteStage{
						{
							Name:  "release",
							Shell: "/bin/sh",
							Commands: []string{
								"release",
							},
						},
					},
				},
			},
		}

		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
			APIServer: &APIServerConfig{
				InjectCommandsPerOperatingSystemAndShell: map[manifest.OperatingSystem]map[string]InjectCommandsConfig{
					manifest.OperatingSystemLinux: {
						"/bin/sh": {
							After: []string{
								"echo last",
							},
						},
					},
				},
			},
		}

		// act
		injectedManifest := InjectCommands(config, mft)

		assert.Equal(t, 2, len(injectedManifest.Stages[0].Commands))
		assert.Equal(t, "build", injectedManifest.Stages[0].Commands[0])
		assert.Equal(t, "echo last", injectedManifest.Stages[0].Commands[1])

		assert.Equal(t, 2, len(injectedManifest.Stages[0].ParallelStages[0].Commands))
		assert.Equal(t, "parallel", injectedManifest.Stages[0].ParallelStages[0].Commands[0])
		assert.Equal(t, "echo last", injectedManifest.Stages[0].ParallelStages[0].Commands[1])

		assert.Equal(t, 2, len(injectedManifest.Releases[0].Stages[0].Commands))
		assert.Equal(t, "release", injectedManifest.Releases[0].Stages[0].Commands[0])
		assert.Equal(t, "echo last", injectedManifest.Releases[0].Stages[0].Commands[1])

	})

	t.Run("DoesNotAppendAfterCommandsConfiguredForOperatingSystemAndShellIfStageHasNoCommands", func(t *testing.T) {

		mft := manifest.EstafetteManifest{
			Builder: manifest.EstafetteBuilder{
				OperatingSystem: manifest.OperatingSystemLinux,
			},
			Stages: []*manifest.EstafetteStage{
				{
					Name:     "build",
					Shell:    "/bin/sh",
					Commands: []string{},
					ParallelStages: []*manifest.EstafetteStage{
						{
							Name:     "parallel",
							Shell:    "/bin/sh",
							Commands: []string{},
						},
					},
				},
			},
			Releases: []*manifest.EstafetteRelease{
				{
					Stages: []*manifest.EstafetteStage{
						{
							Name:     "release",
							Shell:    "/bin/sh",
							Commands: []string{},
						},
					},
				},
			},
		}

		config := &APIConfig{
			ManifestPreferences: manifest.GetDefaultManifestPreferences(),
			APIServer: &APIServerConfig{
				InjectCommandsPerOperatingSystemAndShell: map[manifest.OperatingSystem]map[string]InjectCommandsConfig{
					manifest.OperatingSystemLinux: {
						"/bin/sh": {
							After: []string{
								"echo last",
							},
						},
					},
				},
			},
		}

		// act
		injectedManifest := InjectCommands(config, mft)

		assert.Equal(t, 0, len(injectedManifest.Stages[0].Commands))
		assert.Equal(t, 0, len(injectedManifest.Stages[0].ParallelStages[0].Commands))
		assert.Equal(t, 0, len(injectedManifest.Releases[0].Stages[0].Commands))
	})

}
