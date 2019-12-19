package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	manifest "github.com/estafette/estafette-ci-manifest"
)

func TestInjectSteps(t *testing.T) {

	t.Run("PrependParallelGitCloneStepInInitStage", func(t *testing.T) {

		mft := getManifestWithoutBuildStatusSteps()

		// act
		injectedManifest, err := InjectSteps(mft, "beta", "github")

		assert.Nil(t, err)
		if assert.Equal(t, 3, len(injectedManifest.Stages)) {
			assert.Equal(t, "initialize", injectedManifest.Stages[0].Name)
			assert.Equal(t, "git-clone", injectedManifest.Stages[0].ParallelStages[1].Name)
			assert.Equal(t, "extensions/git-clone:beta", injectedManifest.Stages[0].ParallelStages[1].ContainerImage)
		}
	})

	t.Run("PrependGitCloneStepAsNormalStageWhenBuildStatusExists", func(t *testing.T) {

		mft := getManifestWithBuildStatusSteps()

		// act
		injectedManifest, err := InjectSteps(mft, "beta", "github")

		assert.Nil(t, err)
		if assert.Equal(t, 4, len(injectedManifest.Stages)) {
			assert.Equal(t, "git-clone", injectedManifest.Stages[0].Name)
			assert.Equal(t, "extensions/git-clone:beta", injectedManifest.Stages[0].ContainerImage)
		}
	})

	t.Run("PrependGitCloneStepToReleaseStagesIfCloneRepositoryIsTrue", func(t *testing.T) {

		mft := getManifestWithoutBuildStatusSteps()

		// act
		injectedManifest, err := InjectSteps(mft, "beta", "github")

		assert.Nil(t, err)
		if assert.Equal(t, 2, len(injectedManifest.Releases[1].Stages)) {
			assert.Equal(t, "git-clone", injectedManifest.Releases[1].Stages[0].Name)
			assert.Equal(t, "extensions/git-clone:beta", injectedManifest.Releases[1].Stages[0].ContainerImage)
		}
	})

	t.Run("DoNotPrependGitCloneStepToReleaseStagesIfCloneRepositoryIsFalse", func(t *testing.T) {

		mft := getManifestWithoutBuildStatusSteps()

		// act
		injectedManifest, err := InjectSteps(mft, "beta", "github")

		assert.Nil(t, err)
		if assert.Equal(t, 1, len(injectedManifest.Releases[0].Stages)) {
			assert.Equal(t, "deploy", injectedManifest.Releases[0].Stages[0].Name)
			assert.Equal(t, "extensions/gke", injectedManifest.Releases[0].Stages[0].ContainerImage)
		}
	})

	t.Run("DoNotPrependGitCloneStepToReleaseStagesIfCloneRepositoryIsTrueButStageAlreadyExists", func(t *testing.T) {

		mft := getManifestWithBuildStatusSteps()

		// act
		injectedManifest, err := InjectSteps(mft, "beta", "github")

		assert.Nil(t, err)
		if assert.Equal(t, 2, len(injectedManifest.Releases[0].Stages)) {
			assert.Equal(t, "git-clone", injectedManifest.Releases[0].Stages[0].Name)
			assert.Equal(t, "extensions/git-clone:stable", injectedManifest.Releases[0].Stages[0].ContainerImage)
		}
	})

	t.Run("PrependParallelSetPendingBuildStatusStepInInitStage", func(t *testing.T) {

		mft := getManifestWithoutBuildStatusSteps()

		// act
		injectedManifest, err := InjectSteps(mft, "beta", "github")

		assert.Nil(t, err)
		if assert.Equal(t, 3, len(injectedManifest.Stages)) {
			assert.Equal(t, "initialize", injectedManifest.Stages[0].Name)
			assert.Equal(t, "set-pending-build-status", injectedManifest.Stages[0].ParallelStages[0].Name)
			assert.Equal(t, "extensions/github-status:beta", injectedManifest.Stages[0].ParallelStages[0].ContainerImage)
			assert.Equal(t, "pending", injectedManifest.Stages[0].ParallelStages[0].CustomProperties["status"])
		}
	})

	t.Run("PrependSetPendingBuildStatusAsNormalStageWhenGitCloneStageAlreadyExists", func(t *testing.T) {

		mft := getManifestWithoutBuildStatusStepsAndWithGitClone()

		// act
		injectedManifest, err := InjectSteps(mft, "beta", "github")

		assert.Nil(t, err)
		if assert.Equal(t, 4, len(injectedManifest.Stages)) {
			assert.Equal(t, "set-pending-build-status", injectedManifest.Stages[0].Name)
			assert.Equal(t, "extensions/github-status:beta", injectedManifest.Stages[0].ContainerImage)
			assert.Equal(t, "pending", injectedManifest.Stages[0].CustomProperties["status"])
		}
	})

	t.Run("AppendSetBuildStatusStep", func(t *testing.T) {

		mft := getManifestWithoutBuildStatusSteps()

		// act
		injectedManifest, err := InjectSteps(mft, "beta", "github")

		assert.Nil(t, err)
		if assert.Equal(t, 3, len(injectedManifest.Stages)) {
			assert.Equal(t, "set-build-status", injectedManifest.Stages[2].Name)
			assert.Equal(t, "extensions/github-status:beta", injectedManifest.Stages[2].ContainerImage)
		}
	})

	t.Run("PrependGitCloneStepIfBuildStatusStepsExist", func(t *testing.T) {

		mft := getManifestWithBuildStatusSteps()

		// act
		injectedManifest, err := InjectSteps(mft, "dev", "github")

		assert.Nil(t, err)
		if assert.Equal(t, 4, len(injectedManifest.Stages)) {
			assert.Equal(t, "git-clone", injectedManifest.Stages[0].Name)
			assert.Equal(t, "extensions/git-clone:dev", injectedManifest.Stages[0].ContainerImage)
		}
	})

	t.Run("DoNotPrependSetPendingBuildStatusStepIfPendingBuildStatusStepExists", func(t *testing.T) {

		mft := getManifestWithBuildStatusSteps()

		// act
		injectedManifest, err := InjectSteps(mft, "dev", "github")

		assert.Nil(t, err)
		if assert.Equal(t, 4, len(injectedManifest.Stages)) {
			assert.Equal(t, "set-pending-build-status", injectedManifest.Stages[1].Name)
			assert.Equal(t, "extensions/github-status:stable", injectedManifest.Stages[1].ContainerImage)
			assert.Equal(t, "pending", injectedManifest.Stages[1].CustomProperties["status"])
		}
	})

	t.Run("DoNotAppendSetBuildStatusStepIfSetBuildStatusStepExists", func(t *testing.T) {

		mft := getManifestWithBuildStatusSteps()

		// act
		injectedManifest, err := InjectSteps(mft, "dev", "github")

		assert.Nil(t, err)
		if assert.Equal(t, 4, len(injectedManifest.Stages)) {
			assert.Equal(t, "set-build-status", injectedManifest.Stages[3].Name)
			assert.Equal(t, "extensions/github-status:stable", injectedManifest.Stages[3].ContainerImage)
		}
	})
}

func getManifestWithoutBuildStatusSteps() manifest.EstafetteManifest {
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
			&manifest.EstafetteStage{
				Name:             "build",
				ContainerImage:   "golang:1.10.2-alpine3.7",
				WorkingDirectory: "/go/src/github.com/estafette/${ESTAFETTE_GIT_NAME}",
			},
		},
		Releases: []*manifest.EstafetteRelease{
			&manifest.EstafetteRelease{
				Name:            "staging",
				CloneRepository: false,
				Stages: []*manifest.EstafetteStage{
					&manifest.EstafetteStage{
						Name:           "deploy",
						ContainerImage: "extensions/gke",
					},
				},
			},
			&manifest.EstafetteRelease{
				Name:            "production",
				CloneRepository: true,
				Stages: []*manifest.EstafetteStage{
					&manifest.EstafetteStage{
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
			&manifest.EstafetteStage{
				Name:           "git-clone",
				ContainerImage: "extensions/git-clone:stable",
				Shell:          "/bin/sh",
				When:           "status == 'succeeded'",
			},
			&manifest.EstafetteStage{
				Name:             "build",
				ContainerImage:   "golang:1.10.2-alpine3.7",
				WorkingDirectory: "/go/src/github.com/estafette/${ESTAFETTE_GIT_NAME}",
			},
		},
		Releases: []*manifest.EstafetteRelease{
			&manifest.EstafetteRelease{
				Name:            "staging",
				CloneRepository: false,
				Stages: []*manifest.EstafetteStage{
					&manifest.EstafetteStage{
						Name:           "deploy",
						ContainerImage: "extensions/gke",
					},
				},
			},
			&manifest.EstafetteRelease{
				Name:            "production",
				CloneRepository: true,
				Stages: []*manifest.EstafetteStage{
					&manifest.EstafetteStage{
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
			&manifest.EstafetteStage{
				Name:           "set-pending-build-status",
				ContainerImage: "extensions/github-status:stable",
				CustomProperties: map[string]interface{}{
					"status": "pending",
				},
				Shell:            "/bin/sh",
				WorkingDirectory: "/estafette-work",
				When:             "status == 'succeeded'",
			},
			&manifest.EstafetteStage{
				Name:             "build",
				ContainerImage:   "golang:1.10.2-alpine3.7",
				Shell:            "/bin/sh",
				WorkingDirectory: "/go/src/github.com/estafette/${ESTAFETTE_GIT_NAME}",
				When:             "status == 'succeeded'",
			},
			&manifest.EstafetteStage{
				Name:             "set-build-status",
				ContainerImage:   "extensions/github-status:stable",
				Shell:            "/bin/sh",
				WorkingDirectory: "/estafette-work",
				When:             "status == 'succeeded' || status == 'failed'",
			},
		},
		Releases: []*manifest.EstafetteRelease{
			&manifest.EstafetteRelease{
				Name:            "production",
				CloneRepository: true,
				Stages: []*manifest.EstafetteStage{
					&manifest.EstafetteStage{
						Name:           "git-clone",
						ContainerImage: "extensions/git-clone:stable",
						Shell:          "/bin/sh",
						When:           "status == 'succeeded'",
					},
					&manifest.EstafetteStage{
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
