package estafette

import (
	"fmt"
	"strings"

	"github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
)

// WarningHelper checks whether any warnings should be issued
type WarningHelper interface {
	GetManifestWarnings(*manifest.EstafetteManifest) ([]contracts.Warning, error)
}

type warningHelperImpl struct {
}

// NewWarningHelper returns a new estafette.WarningHelper
func NewWarningHelper() (warningHelper WarningHelper) {

	warningHelper = &warningHelperImpl{}

	return
}

func (w *warningHelperImpl) GetManifestWarnings(manifest *manifest.EstafetteManifest) (warnings []contracts.Warning, err error) {
	warnings = []contracts.Warning{}

	// unmarshal then marshal manifest to include defaults
	if manifest != nil {
		// check all build and release stages to have a pinned version
		stagesUsingLatestTag := []string{}
		stagesUsingDevTag := []string{}
		for _, s := range manifest.Stages {
			containerImageTag := getContainerImageTag(s.ContainerImage)
			if containerImageTag == "latest" {
				stagesUsingLatestTag = append(stagesUsingLatestTag, s.Name)
			}
			if containerImageTag == "dev" {
				stagesUsingDevTag = append(stagesUsingDevTag, s.Name)
			}
		}

		for _, r := range manifest.Releases {
			for _, s := range r.Stages {
				containerImageTag := getContainerImageTag(s.ContainerImage)
				if containerImageTag == "latest" {
					stagesUsingLatestTag = append(stagesUsingLatestTag, fmt.Sprintf("%v/%v", r.Name, s.Name))
				}
				if containerImageTag == "dev" {
					stagesUsingDevTag = append(stagesUsingDevTag, s.Name)
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
	}

	return
}
