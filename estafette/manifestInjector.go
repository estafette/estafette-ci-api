package estafette

import (
	"fmt"

	manifest "github.com/estafette/estafette-ci-manifest"
)

// InjectSteps injects git-clone and build-status steps if not present in manifest
func InjectSteps(mft manifest.EstafetteManifest, builderTrack, gitSource string) (injectedManifest manifest.EstafetteManifest, err error) {

	injectedManifest = mft

	if !StepExists(injectedManifest, "git-clone") {
		// add git-clone at the start
		gitCloneStep := &manifest.EstafetteStage{
			Name:             "git-clone",
			ContainerImage:   fmt.Sprintf("extensions/git-clone:%v", builderTrack),
			Shell:            "/bin/sh",
			WorkingDirectory: "/estafette-work",
			When:             "status == 'succeeded'",
			AutoInjected:     true,
		}
		injectedManifest.Stages = append([]*manifest.EstafetteStage{gitCloneStep}, injectedManifest.Stages...)
	}

	if !StepExists(injectedManifest, "set-pending-build-status") {
		// add set-pending-build-status at the start if it doesn't exist yet
		setPendingBuildStatusStep := &manifest.EstafetteStage{
			Name:           "set-pending-build-status",
			ContainerImage: fmt.Sprintf("extensions/%v-status:%v", gitSource, builderTrack),
			CustomProperties: map[string]interface{}{
				"status": "pending",
			},
			Shell:            "/bin/sh",
			WorkingDirectory: "/estafette-work",
			When:             "status == 'succeeded'",
			AutoInjected:     true,
		}
		injectedManifest.Stages = append([]*manifest.EstafetteStage{setPendingBuildStatusStep}, injectedManifest.Stages...)
	}

	if !StepExists(injectedManifest, "set-build-status") {
		// add set-build-status at the end if it doesn't exist yet
		setBuildStatusStep := &manifest.EstafetteStage{
			Name:             "set-build-status",
			ContainerImage:   fmt.Sprintf("extensions/%v-status:%v", gitSource, builderTrack),
			Shell:            "/bin/sh",
			WorkingDirectory: "/estafette-work",
			When:             "status == 'succeeded' || status == 'failed'",
			AutoInjected:     true,
		}
		injectedManifest.Stages = append(injectedManifest.Stages, setBuildStatusStep)
	}

	return
}

// StepExists returns true if a step with stepName already exists, false otherwise
func StepExists(mft manifest.EstafetteManifest, stepName string) bool {
	for _, step := range mft.Stages {
		if step.Name == stepName {
			return true
		}
	}
	return false
}
