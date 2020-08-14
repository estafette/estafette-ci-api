package api

import (
	"fmt"

	manifest "github.com/estafette/estafette-ci-manifest"
)

// InjectSteps injects git-clone and build-status steps if not present in manifest
func InjectSteps(config *APIConfig, mft manifest.EstafetteManifest, builderTrack, gitSource string, supportsBuildStatus bool) (injectedManifest manifest.EstafetteManifest, err error) {

	injectedManifest = mft

	injectedManifest.Stages = injectBuildStagesBefore(config, injectedManifest, builderTrack, gitSource, supportsBuildStatus)
	injectedManifest.Stages = injectBuildStagesAfter(config, injectedManifest, builderTrack, gitSource, supportsBuildStatus)

	for _, r := range injectedManifest.Releases {
		r.Stages = injectReleaseStagesBefore(config, *r, builderTrack, gitSource, supportsBuildStatus)
		r.Stages = injectReleaseStagesAfter(config, *r, builderTrack, gitSource, supportsBuildStatus)
	}

	var preferences *manifest.EstafetteManifestPreferences
	if config != nil && config.ManifestPreferences != nil {
		preferences = config.ManifestPreferences
	} else {
		preferences = manifest.GetDefaultManifestPreferences()
	}

	// ensure all injected stages have defaults for shell and working directory matching the target operating system
	injectedManifest.SetDefaults(*preferences)

	return
}

func injectBuildStagesBefore(config *APIConfig, mft manifest.EstafetteManifest, builderTrack, gitSource string, supportsBuildStatus bool) (stages []*manifest.EstafetteStage) {

	stages = mft.Stages

	injectedBeforeStageName := "injected-before"
	if StageExists(stages, injectedBeforeStageName) {
		i := 0
		for StageExists(stages, injectedBeforeStageName) {
			injectedBeforeStageName = fmt.Sprintf("injected-before-%v", i)
			i++
		}
	}

	injectedBeforeStage := &manifest.EstafetteStage{
		Name:           injectedBeforeStageName,
		ParallelStages: []*manifest.EstafetteStage{},
		AutoInjected:   true,
	}

	if !StageExists(stages, "git-clone") {
		injectedBeforeStage.ParallelStages = append(injectedBeforeStage.ParallelStages, &manifest.EstafetteStage{
			Name:           "git-clone",
			ContainerImage: fmt.Sprintf("extensions/git-clone:%v", builderTrack),
			AutoInjected:   true,
		})
	}

	if !StageExists(stages, "set-pending-build-status") && supportsBuildStatus {
		injectedBeforeStage.ParallelStages = append(injectedBeforeStage.ParallelStages, &manifest.EstafetteStage{
			Name:           "set-pending-build-status",
			ContainerImage: fmt.Sprintf("extensions/%v-status:%v", gitSource, builderTrack),
			CustomProperties: map[string]interface{}{
				"status": "pending",
			},
			AutoInjected: true,
		})
	}

	// add any configured injected stages
	if config != nil && config.APIServer != nil && config.APIServer.InjectStages != nil && config.APIServer.InjectStages.Build != nil && config.APIServer.InjectStages.Build.Before != nil {
		for _, s := range config.APIServer.InjectStages.Build.Before {
			s.AutoInjected = true
			injectedBeforeStage.ParallelStages = append(injectedBeforeStage.ParallelStages, s)
		}
	}

	if len(injectedBeforeStage.ParallelStages) > 0 {
		stages = append([]*manifest.EstafetteStage{injectedBeforeStage}, stages...)
	}

	return stages
}

func injectBuildStagesAfter(config *APIConfig, mft manifest.EstafetteManifest, builderTrack, gitSource string, supportsBuildStatus bool) (stages []*manifest.EstafetteStage) {

	stages = mft.Stages

	injectedAfterStageName := "injected-after"
	if StageExists(stages, injectedAfterStageName) {
		i := 0
		for StageExists(stages, injectedAfterStageName) {
			injectedAfterStageName = fmt.Sprintf("inject-after-%v", i)
			i++
		}
	}

	injectedAfterStage := &manifest.EstafetteStage{
		Name:           injectedAfterStageName,
		ParallelStages: []*manifest.EstafetteStage{},
		AutoInjected:   true,
	}

	if !StageExists(stages, "set-build-status") && supportsBuildStatus {
		// add set-build-status at the end if it doesn't exist yet
		injectedAfterStage.ParallelStages = append(injectedAfterStage.ParallelStages, &manifest.EstafetteStage{
			Name:           "set-build-status",
			ContainerImage: fmt.Sprintf("extensions/%v-status:%v", gitSource, builderTrack),
			When:           "status == 'succeeded' || status == 'failed'",
			AutoInjected:   true,
		})
	}

	// add any configured injected stages
	if config != nil && config.APIServer != nil && config.APIServer.InjectStages != nil && config.APIServer.InjectStages.Build != nil && config.APIServer.InjectStages.Build.After != nil {
		for _, s := range config.APIServer.InjectStages.Build.After {
			s.AutoInjected = true
			injectedAfterStage.ParallelStages = append(injectedAfterStage.ParallelStages, s)
		}
	}

	if len(injectedAfterStage.ParallelStages) > 0 {
		stages = append(stages, injectedAfterStage)
	}

	return stages
}

func injectReleaseStagesBefore(config *APIConfig, release manifest.EstafetteRelease, builderTrack, gitSource string, supportsBuildStatus bool) (stages []*manifest.EstafetteStage) {

	stages = release.Stages

	injectedBeforeStageName := "injected-before"
	if StageExists(stages, injectedBeforeStageName) {
		i := 0
		for StageExists(stages, injectedBeforeStageName) {
			injectedBeforeStageName = fmt.Sprintf("injected-before-%v", i)
			i++
		}
	}

	injectedBeforeStage := &manifest.EstafetteStage{
		Name:           injectedBeforeStageName,
		ParallelStages: []*manifest.EstafetteStage{},
		AutoInjected:   true,
	}

	if release.CloneRepository {
		if !StageExists(stages, "git-clone") {
			// add git-clone at the start
			gitCloneStep := &manifest.EstafetteStage{
				Name:           "git-clone",
				ContainerImage: fmt.Sprintf("extensions/git-clone:%v", builderTrack),
				AutoInjected:   true,
			}
			stages = append([]*manifest.EstafetteStage{gitCloneStep}, stages...)
		}
	}

	// add any configured injected stages
	if config != nil && config.APIServer != nil && config.APIServer.InjectStages != nil && config.APIServer.InjectStages.Build != nil && config.APIServer.InjectStages.Build.Before != nil {
		for _, s := range config.APIServer.InjectStages.Build.Before {
			s.AutoInjected = true
			injectedBeforeStage.ParallelStages = append(injectedBeforeStage.ParallelStages, s)
		}
	}

	if len(injectedBeforeStage.ParallelStages) > 0 {
		stages = append([]*manifest.EstafetteStage{injectedBeforeStage}, stages...)
	}

	return stages
}

func injectReleaseStagesAfter(config *APIConfig, release manifest.EstafetteRelease, builderTrack, gitSource string, supportsBuildStatus bool) (stages []*manifest.EstafetteStage) {

	stages = release.Stages

	injectedAfterStageName := "injected-after"
	if StageExists(stages, injectedAfterStageName) {
		i := 0
		for StageExists(stages, injectedAfterStageName) {
			injectedAfterStageName = fmt.Sprintf("inject-after-%v", i)
			i++
		}
	}

	injectedAfterStage := &manifest.EstafetteStage{
		Name:           injectedAfterStageName,
		ParallelStages: []*manifest.EstafetteStage{},
		AutoInjected:   true,
	}

	// add any configured injected stages
	if config != nil && config.APIServer != nil && config.APIServer.InjectStages != nil && config.APIServer.InjectStages.Release != nil && config.APIServer.InjectStages.Release.After != nil {
		for _, s := range config.APIServer.InjectStages.Release.After {
			s.AutoInjected = true
			injectedAfterStage.ParallelStages = append(injectedAfterStage.ParallelStages, s)
		}
	}

	if len(injectedAfterStage.ParallelStages) > 0 {
		stages = append(stages, injectedAfterStage)
	}

	return stages
}

// StageExists returns true if a stage with stageName already exists, false otherwise
func StageExists(stages []*manifest.EstafetteStage, stageName string) bool {
	for _, stage := range stages {
		if stage.Name == stageName {
			return true
		}
	}
	return false
}
