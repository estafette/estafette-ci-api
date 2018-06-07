package estafette

import (
	"testing"

	"github.com/stretchr/testify/assert"

	manifest "github.com/estafette/estafette-ci-manifest"
)

func TestInjectSteps(t *testing.T) {

	t.Run("PrependGitCloneStep", func(t *testing.T) {

		mft := getManifestWithoutBuildStatusSteps()

		// act
		injectedManifest, err := InjectSteps(mft, "beta", "github")

		assert.Nil(t, err)
		assert.Equal(t, 4, len(injectedManifest.Pipelines))
		assert.Equal(t, "git-clone", injectedManifest.Pipelines[1].Name)
		assert.Equal(t, "extensions/git-clone:beta", injectedManifest.Pipelines[1].ContainerImage)
	})

	t.Run("PrependSetPendingBuildStatusStep", func(t *testing.T) {

		mft := getManifestWithoutBuildStatusSteps()

		// act
		injectedManifest, err := InjectSteps(mft, "beta", "github")

		assert.Nil(t, err)
		assert.Equal(t, 4, len(injectedManifest.Pipelines))
		assert.Equal(t, "set-pending-build-status", injectedManifest.Pipelines[0].Name)
		assert.Equal(t, "extensions/github-status:beta", injectedManifest.Pipelines[0].ContainerImage)
		assert.Equal(t, "pending", injectedManifest.Pipelines[0].CustomProperties["status"])
	})

	t.Run("AppendSetBuildStatusStep", func(t *testing.T) {

		mft := getManifestWithoutBuildStatusSteps()

		// act
		injectedManifest, err := InjectSteps(mft, "beta", "github")

		assert.Nil(t, err)
		assert.Equal(t, 4, len(injectedManifest.Pipelines))
		assert.Equal(t, "set-build-status", injectedManifest.Pipelines[3].Name)
		assert.Equal(t, "extensions/github-status:beta", injectedManifest.Pipelines[3].ContainerImage)
	})

	t.Run("PrependGitCloneStepIfBuildStatusStepsExist", func(t *testing.T) {

		mft := getManifestWithBuildStatusSteps()

		// act
		injectedManifest, err := InjectSteps(mft, "dev", "github")

		assert.Nil(t, err)
		assert.Equal(t, 4, len(injectedManifest.Pipelines))
		assert.Equal(t, "git-clone", injectedManifest.Pipelines[0].Name)
		assert.Equal(t, "extensions/git-clone:dev", injectedManifest.Pipelines[0].ContainerImage)
	})

	t.Run("PrependSetPendingBuildStatusStep", func(t *testing.T) {

		mft := getManifestWithBuildStatusSteps()

		// act
		injectedManifest, err := InjectSteps(mft, "dev", "github")

		assert.Nil(t, err)
		assert.Equal(t, 4, len(injectedManifest.Pipelines))
		assert.Equal(t, "set-pending-build-status", injectedManifest.Pipelines[1].Name)
		assert.Equal(t, "extensions/github-status:stable", injectedManifest.Pipelines[1].ContainerImage)
		assert.Equal(t, "pending", injectedManifest.Pipelines[1].CustomProperties["status"])
	})

	t.Run("AppendSetBuildStatusStep", func(t *testing.T) {

		mft := getManifestWithBuildStatusSteps()

		// act
		injectedManifest, err := InjectSteps(mft, "dev", "github")

		assert.Nil(t, err)
		assert.Equal(t, 4, len(injectedManifest.Pipelines))
		assert.Equal(t, "set-build-status", injectedManifest.Pipelines[3].Name)
		assert.Equal(t, "extensions/github-status:stable", injectedManifest.Pipelines[3].ContainerImage)
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
				ReleaseBranch: "master",
			},
		},
		Labels:        map[string]string{},
		GlobalEnvVars: map[string]string{},
		Pipelines: []*manifest.EstafettePipeline{
			&manifest.EstafettePipeline{
				Name:             "build",
				ContainerImage:   "golang:1.10.2-alpine3.7",
				Shell:            "/bin/sh",
				WorkingDirectory: "/go/src/github.com/estafette/${ESTAFETTE_LABEL_APP}",
				When:             "status == 'succeeded'",
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
				ReleaseBranch: "master",
			},
		},
		Labels:        map[string]string{},
		GlobalEnvVars: map[string]string{},
		Pipelines: []*manifest.EstafettePipeline{
			&manifest.EstafettePipeline{
				Name:           "set-pending-build-status",
				ContainerImage: "extensions/github-status:stable",
				CustomProperties: map[string]interface{}{
					"status": "pending",
				},
				Shell:            "/bin/sh",
				WorkingDirectory: "/estafette-work",
				When:             "status == 'succeeded'",
			},
			&manifest.EstafettePipeline{
				Name:             "build",
				ContainerImage:   "golang:1.10.2-alpine3.7",
				Shell:            "/bin/sh",
				WorkingDirectory: "/go/src/github.com/estafette/${ESTAFETTE_LABEL_APP}",
				When:             "status == 'succeeded'",
			},
			&manifest.EstafettePipeline{
				Name:             "set-build-status",
				ContainerImage:   "extensions/github-status:stable",
				Shell:            "/bin/sh",
				WorkingDirectory: "/estafette-work",
				When:             "status == 'succeeded' || status == 'failed'",
			},
		},
	}
}
