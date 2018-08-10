package estafette

import (
	"fmt"

	manifest "github.com/estafette/estafette-ci-manifest"
)

// InjectSteps injects git-clone and build-status steps if not present in manifest
func InjectSteps(mft manifest.EstafetteManifest, builderTrack, gitSource string) (injectedManifest manifest.EstafetteManifest, err error) {

	injectedManifest = mft

	if !StepExists(injectedManifest.Stages, "git-clone") {
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

	if !StepExists(injectedManifest.Stages, "set-pending-build-status") {
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

	if !StepExists(injectedManifest.Stages, "set-build-status") {
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

// InjectReleaseSteps injects git-clone if release clone is true and stage is not present in manifest
func InjectReleaseSteps(mft manifest.EstafetteManifest, builderTrack, releaseName string) (injectedManifest manifest.EstafetteManifest, err error) {

	injectedManifest = mft

	for _, release := range injectedManifest.Releases {
		if release.Name == releaseName {
			if release.CloneGitRepository {
				if !StepExists(release.Stages, "git-clone") {
					// add git-clone at the start
					gitCloneStep := &manifest.EstafetteStage{
						Name:             "git-clone",
						ContainerImage:   fmt.Sprintf("extensions/git-clone:%v", builderTrack),
						Shell:            "/bin/sh",
						WorkingDirectory: "/estafette-work",
						When:             "status == 'succeeded'",
						AutoInjected:     true,
					}
					release.Stages = append([]*manifest.EstafetteStage{gitCloneStep}, release.Stages...)
				}
			}
			break
		}
	}

	return
}

// StepExists returns true if a step with stepName already exists, false otherwise
func StepExists(stages []*manifest.EstafetteStage, stepName string) bool {
	for _, step := range stages {
		if step.Name == stepName {
			return true
		}
	}
	return false
}
