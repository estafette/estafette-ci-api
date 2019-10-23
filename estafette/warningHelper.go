package estafette

import (
	"fmt"
	"strings"

	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
)

// WarningHelper checks whether any warnings should be issued
type WarningHelper interface {
	GetManifestWarnings(*manifest.EstafetteManifest, string) ([]contracts.Warning, error)
	GetContainerImageParts(string) (string, string, string)
}

type warningHelperImpl struct {
}

// NewWarningHelper returns a new estafette.WarningHelper
func NewWarningHelper() (warningHelper WarningHelper) {

	warningHelper = &warningHelperImpl{}

	return
}

func (w *warningHelperImpl) GetManifestWarnings(manifest *manifest.EstafetteManifest, gitOwner string) (warnings []contracts.Warning, err error) {
	warnings = []contracts.Warning{}

	// unmarshal then marshal manifest to include defaults
	if manifest != nil {
		// check all build and release stages to have a pinned version
		stagesUsingLatestTag := []string{}
		stagesUsingDevTag := []string{}
		for _, s := range manifest.Stages {
			if len(s.ParallelStages) > 0 {
				for _, ns := range s.ParallelStages {
					containerImageRepo, _, containerImageTag := w.GetContainerImageParts(ns.ContainerImage)
					if containerImageTag == "latest" {
						stagesUsingLatestTag = append(stagesUsingLatestTag, ns.Name)
					}
					if containerImageTag == "dev" && containerImageRepo == "extensions" && gitOwner != "estafette" {
						stagesUsingDevTag = append(stagesUsingDevTag, ns.Name)
					}
				}
			} else {
				containerImageRepo, _, containerImageTag := w.GetContainerImageParts(s.ContainerImage)
				if containerImageTag == "latest" {
					stagesUsingLatestTag = append(stagesUsingLatestTag, s.Name)
				}
				if containerImageTag == "dev" && containerImageRepo == "extensions" && gitOwner != "estafette" {
					stagesUsingDevTag = append(stagesUsingDevTag, s.Name)
				}
			}
		}

		for _, r := range manifest.Releases {
			for _, s := range r.Stages {
				if len(s.ParallelStages) > 0 {
					for _, ns := range s.ParallelStages {
						containerImageRepo, _, containerImageTag := w.GetContainerImageParts(ns.ContainerImage)
						if containerImageTag == "latest" {
							stagesUsingLatestTag = append(stagesUsingLatestTag, ns.Name)
						}
						if containerImageTag == "dev" && containerImageRepo == "extensions" && gitOwner != "estafette" {
							stagesUsingDevTag = append(stagesUsingDevTag, ns.Name)
						}
					}
				} else {
					containerImageRepo, _, containerImageTag := w.GetContainerImageParts(s.ContainerImage)
					if containerImageTag == "latest" {
						stagesUsingLatestTag = append(stagesUsingLatestTag, fmt.Sprintf("%v/%v", r.Name, s.Name))
					}
					if containerImageTag == "dev" && containerImageRepo == "extensions" && gitOwner != "estafette" {
						stagesUsingDevTag = append(stagesUsingDevTag, s.Name)
					}
				}
			}
		}

		if len(stagesUsingLatestTag) > 0 {
			warnings = append(warnings, contracts.Warning{
				Status:  "warning",
				Message: fmt.Sprintf("This pipeline has one or more stages that use **latest** or no tag for its container image: `%v`; it is [best practice](https://estafette.io/usage/best-practices/#pin-image-versions) to pin stage images to specific versions so you don't spend hours tracking down build failures because the used image has changed.", strings.Join(stagesUsingLatestTag, ", ")),
			})
		}
		if len(stagesUsingDevTag) > 0 {
			warnings = append(warnings, contracts.Warning{
				Status:  "warning",
				Message: fmt.Sprintf("This pipeline has one or more stages that use the **dev** tag for its container image: `%v`; it is [best practice](https://estafette.io/usage/best-practices/#avoid-using-estafette-s-dev-or-beta-tags) to avoid the dev tag alltogether, since it can be broken at any time.", strings.Join(stagesUsingDevTag, ", ")),
			})
		}

		if manifest.Builder.Track == "dev" && gitOwner != "estafette" {
			warnings = append(warnings, contracts.Warning{
				Status:  "warning",
				Message: "This pipeline uses the **dev** track for the builder; it is [best practice](https://estafette.io/usage/best-practices/#avoid-using-estafette-s-builder-dev-track) to avoid the dev track, since it can be broken at any time.",
			})
		}
	}

	return
}

func (w *warningHelperImpl) GetContainerImageParts(containerImage string) (repo, name, tag string) {
	containerImageArray := strings.Split(containerImage, ":")
	tag = "latest"
	if len(containerImageArray) > 1 {
		tag = containerImageArray[1]
		containerImageArray = strings.Split(containerImageArray[0], "/")
		if len(containerImageArray) > 0 {
			name = containerImageArray[len(containerImageArray)-1]
			repo = strings.Join(containerImageArray[:len(containerImageArray)-1], "/")
		}
	}

	return
}
