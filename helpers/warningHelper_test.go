package helpers

import (
	"testing"

	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/stretchr/testify/assert"
)

var (
	helper = NewWarningHelper()
)

func TestGetManifestWarnings(t *testing.T) {

	t.Run("ReturnsNoWarningsIfStageContainerImagesUsePinnedVersions", func(t *testing.T) {

		mft := &manifest.EstafetteManifest{
			Stages: []*manifest.EstafetteStage{
				&manifest.EstafetteStage{
					Name:           "build",
					ContainerImage: "golang:1.12.4-alpine3.9",
				},
			},
		}

		// act
		warnings, err := helper.GetManifestWarnings(mft, "extensions")

		assert.Nil(t, err)
		assert.Equal(t, 0, len(warnings))
	})

	t.Run("ReturnsNoWarningsIfStageContainerImageIsEmptyDueToNestedStages", func(t *testing.T) {

		mft := &manifest.EstafetteManifest{
			Stages: []*manifest.EstafetteStage{
				&manifest.EstafetteStage{
					Name:           "build",
					ContainerImage: "",
					ParallelStages: []*manifest.EstafetteStage{
						&manifest.EstafetteStage{
							Name:           "build",
							ContainerImage: "golang:1.12.4-alpine3.9",
						},
					},
				},
			},
		}

		// act
		warnings, err := helper.GetManifestWarnings(mft, "extensions")

		assert.Nil(t, err)
		assert.Equal(t, 0, len(warnings))
	})

	t.Run("ReturnsWarningIfStageContainerImageUsesLatestTag", func(t *testing.T) {

		mft := &manifest.EstafetteManifest{
			Stages: []*manifest.EstafetteStage{
				&manifest.EstafetteStage{
					Name:           "build",
					ContainerImage: "golang:latest",
				},
			},
		}

		// act
		warnings, err := helper.GetManifestWarnings(mft, "extensions")

		assert.Nil(t, err)
		assert.Equal(t, 1, len(warnings))
		assert.Equal(t, "warning", warnings[0].Status)
		assert.Equal(t, "This pipeline has one or more stages that use **latest** or no tag for its container image: `build`; it is [best practice](https://estafette.io/usage/best-practices/#pin-image-versions) to pin stage images to specific versions so you don't spend hours tracking down build failures because the used image has changed.", warnings[0].Message)
	})

	t.Run("ReturnsWarningIfStageContainerImageUsesNoTag", func(t *testing.T) {

		mft := &manifest.EstafetteManifest{
			Stages: []*manifest.EstafetteStage{
				&manifest.EstafetteStage{
					Name:           "build",
					ContainerImage: "golang",
				},
			},
		}

		// act
		warnings, err := helper.GetManifestWarnings(mft, "extensions")

		assert.Nil(t, err)
		assert.Equal(t, 1, len(warnings))
		assert.Equal(t, "warning", warnings[0].Status)
		assert.Equal(t, "This pipeline has one or more stages that use **latest** or no tag for its container image: `build`; it is [best practice](https://estafette.io/usage/best-practices/#pin-image-versions) to pin stage images to specific versions so you don't spend hours tracking down build failures because the used image has changed.", warnings[0].Message)
	})

	t.Run("ReturnsWarningIfStageContainerImageUsesExtensionWithDevTag", func(t *testing.T) {

		mft := &manifest.EstafetteManifest{
			Stages: []*manifest.EstafetteStage{
				&manifest.EstafetteStage{
					Name:           "build",
					ContainerImage: "extensions/docker:dev",
				},
			},
		}

		// act
		warnings, err := helper.GetManifestWarnings(mft, "extensions")

		assert.Nil(t, err)
		assert.Equal(t, 1, len(warnings))
		assert.Equal(t, "warning", warnings[0].Status)
		assert.Equal(t, "This pipeline has one or more stages that use the **dev** tag for its container image: `build`; it is [best practice](https://estafette.io/usage/best-practices/#avoid-using-estafette-s-dev-or-beta-tags) to avoid the dev tag alltogether, since it can be broken at any time.", warnings[0].Message)
	})

	t.Run("ReturnsNoWarningIfStageContainerImageUsesDevTagForNonExtensionImage", func(t *testing.T) {

		mft := &manifest.EstafetteManifest{
			Stages: []*manifest.EstafetteStage{
				&manifest.EstafetteStage{
					Name:           "build",
					ContainerImage: "golang:dev",
				},
			},
		}

		// act
		warnings, err := helper.GetManifestWarnings(mft, "extensions")

		assert.Nil(t, err)
		assert.Equal(t, 0, len(warnings))
	})

	t.Run("ReturnsNoWarningIfStageContainerImageUsesExtensionWithDevTagForAnEstafetteBuild", func(t *testing.T) {

		mft := &manifest.EstafetteManifest{
			Stages: []*manifest.EstafetteStage{
				&manifest.EstafetteStage{
					Name:           "build",
					ContainerImage: "extensions/docker:dev",
				},
			},
		}

		// act
		warnings, err := helper.GetManifestWarnings(mft, "estafette")

		assert.Nil(t, err)
		assert.Equal(t, 0, len(warnings))
	})

	t.Run("ReturnsWarningIfBuilderTrackIsSetToDev", func(t *testing.T) {

		mft := &manifest.EstafetteManifest{
			Builder: manifest.EstafetteBuilder{
				Track: "dev",
			},
		}

		// act
		warnings, err := helper.GetManifestWarnings(mft, "extensions")

		assert.Nil(t, err)
		assert.Equal(t, 1, len(warnings))
		assert.Equal(t, "warning", warnings[0].Status)
		assert.Equal(t, "This pipeline uses the **dev** track for the builder; it is [best practice](https://estafette.io/usage/best-practices/#avoid-using-estafette-s-builder-dev-track) to avoid the dev track, since it can be broken at any time.", warnings[0].Message)
	})
}

func TestGetContainerImageParts(t *testing.T) {

	t.Run("ReturnsEmptyRepoIfOfficialDockerHubImage", func(t *testing.T) {

		containerImage := "golang:1.12.4-alpine3.9"

		// act
		repo, _, _ := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "", repo)
	})

	t.Run("ReturnsPathAsNameIfOfficialDockerHubImage", func(t *testing.T) {

		containerImage := "golang:1.12.4-alpine3.9"

		// act
		_, name, _ := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "golang", name)
	})

	t.Run("ReturnsTagIfProvidedForOfficialDockerHubImage", func(t *testing.T) {

		containerImage := "golang:1.12.4-alpine3.9"

		// act
		_, _, tag := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "1.12.4-alpine3.9", tag)
	})

	t.Run("ReturnsLatestTagIfNoTagProvidedForOfficialDockerHubImage", func(t *testing.T) {

		containerImage := "golang"

		// act
		_, _, tag := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "latest", tag)
	})

	t.Run("ReturnsPartUpToNameIfRepositoryDockerHubImage", func(t *testing.T) {

		containerImage := "estafette/ci-builder:dev"

		// act
		repo, _, _ := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "estafette", repo)
	})

	t.Run("ReturnsPathAsNameIfRepositoryDockerHubImage", func(t *testing.T) {

		containerImage := "estafette/ci-builder:dev"

		// act
		_, name, _ := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "ci-builder", name)
	})

	t.Run("ReturnsTagIfProvidedForRepositoryDockerHubImage", func(t *testing.T) {

		containerImage := "estafette/ci-builder:dev"

		// act
		_, _, tag := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "dev", tag)
	})

	t.Run("ReturnsLatestTagIfNoTagProvidedForRepositoryDockerHubImage", func(t *testing.T) {

		containerImage := "estafette/ci-builder"

		// act
		_, _, tag := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "latest", tag)
	})

	t.Run("ReturnsPartUpToNameIfGoogleContainerRegistryImage", func(t *testing.T) {

		containerImage := "gcr.io/estafette/ci-builder:dev"

		// act
		repo, _, _ := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "gcr.io/estafette", repo)
	})

	t.Run("ReturnsPathAsNameIfGoogleContainerRegistryImage", func(t *testing.T) {

		containerImage := "gcr.io/estafette/ci-builder:dev"

		// act
		_, name, _ := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "ci-builder", name)
	})

	t.Run("ReturnsTagIfProvidedForGoogleContainerRegistryImage", func(t *testing.T) {

		containerImage := "gcr.io/estafette/ci-builder:dev"

		// act
		_, _, tag := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "dev", tag)
	})

	t.Run("ReturnsLatestTagIfNoTagProvidedForGoogleContainerRegistryImage", func(t *testing.T) {

		containerImage := "gcr.io/estafette/ci-builder"

		// act
		_, _, tag := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "latest", tag)
	})
}
