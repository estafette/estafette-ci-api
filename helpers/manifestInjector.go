package helpers

import (
	"fmt"

	manifest "github.com/estafette/estafette-ci-manifest"
)

// InjectSteps injects git-clone and build-status steps if not present in manifest
func InjectSteps(mft manifest.EstafetteManifest, builderTrack, gitSource string) (injectedManifest manifest.EstafetteManifest, err error) {

	injectedManifest = mft

	initStepName := "initialize"
	if StepExists(injectedManifest.Stages, initStepName) {
		i := 0
		for StepExists(injectedManifest.Stages, initStepName) {
			initStepName = fmt.Sprintf("initialize-%v", i)
			i++
		}
	}

	initStep := &manifest.EstafetteStage{
		Name:           initStepName,
		ParallelStages: []*manifest.EstafetteStage{},
		AutoInjected:   true,
	}

	if !StepExists(injectedManifest.Stages, "git-clone") {
		initStep.ParallelStages = append(initStep.ParallelStages, &manifest.EstafetteStage{
			Name:           "git-clone",
			ContainerImage: fmt.Sprintf("extensions/git-clone:%v", builderTrack),
			AutoInjected:   true,
		})
	}

	if !StepExists(injectedManifest.Stages, "set-pending-build-status") {
		initStep.ParallelStages = append(initStep.ParallelStages, &manifest.EstafetteStage{
			Name:           "set-pending-build-status",
			ContainerImage: fmt.Sprintf("extensions/%v-status:%v", gitSource, builderTrack),
			CustomProperties: map[string]interface{}{
				"status": "pending",
			},
			AutoInjected: true,
			Retries:      3,
		})
	}
	if !StepExists(injectedManifest.Stages, "envvars") {
		initStep.ParallelStages = append(initStep.ParallelStages, &manifest.EstafetteStage{
			Name:           "envvars",
			ContainerImage: fmt.Sprintf("extensions/envvars:%v", builderTrack),
			AutoInjected:   true,
		})
	}

	if len(initStep.ParallelStages) > 0 {
		injectedManifest.Stages = append([]*manifest.EstafetteStage{initStep}, injectedManifest.Stages...)
	}

	if !StepExists(injectedManifest.Stages, "set-build-status") {
		// add set-build-status at the end if it doesn't exist yet
		setBuildStatusStep := &manifest.EstafetteStage{
			Name:           "set-build-status",
			ContainerImage: fmt.Sprintf("extensions/%v-status:%v", gitSource, builderTrack),
			When:           "status == 'succeeded' || status == 'failed'",
			AutoInjected:   true,
			Retries:        3,
		}
		injectedManifest.Stages = append(injectedManifest.Stages, setBuildStatusStep)
	}

	// inject git-clone for release stages if CloneRepository is true
	for _, r := range injectedManifest.Releases {
		if r.CloneRepository {
			if !StepExists(r.Stages, "git-clone") {
				// add git-clone at the start
				gitCloneStep := &manifest.EstafetteStage{
					Name:           "git-clone",
					ContainerImage: fmt.Sprintf("extensions/git-clone:%v", builderTrack),
					AutoInjected:   true,
				}
				r.Stages = append([]*manifest.EstafetteStage{gitCloneStep}, r.Stages...)
			}
		}
	}

	// ensure all injected stages have defaults for shell and working directory matching the target operating system
	injectedManifest.SetDefaults()

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
