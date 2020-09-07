package api

import (
	"fmt"

	manifest "github.com/estafette/estafette-ci-manifest"
)

// InjectStages injects some mandatory and configured stages
func InjectStages(config *APIConfig, mft manifest.EstafetteManifest, builderTrack, gitSource string, supportsBuildStatus bool) (injectedManifest manifest.EstafetteManifest, err error) {

	injectedManifest = mft

	// inject build stages
	injectedManifest.Stages = injectBuildStagesBefore(config, injectedManifest, builderTrack, gitSource, supportsBuildStatus)
	injectedManifest.Stages = injectBuildStagesAfter(config, injectedManifest, builderTrack, gitSource, supportsBuildStatus)

	// inject release stages
	for _, r := range injectedManifest.Releases {
		r.Stages = injectReleaseStagesBefore(config, *r, builderTrack, gitSource, supportsBuildStatus)
		r.Stages = injectReleaseStagesAfter(config, *r, builderTrack, gitSource, supportsBuildStatus)
	}

	// get preferences for defaults
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

func getInjectedStageName(stageBaseName string, stages []*manifest.EstafetteStage) string {

	injectedStageName := stageBaseName
	if stageExists(stages, injectedStageName) {
		i := 0
		for stageExists(stages, injectedStageName) {
			injectedStageName = fmt.Sprintf("%v-%v", stageBaseName, i)
			i++
		}
	}

	return injectedStageName
}

func injectIfNotExists(stages, parallelStages []*manifest.EstafetteStage, stageToInject *manifest.EstafetteStage) []*manifest.EstafetteStage {
	if !stageExists(stages, stageToInject.Name) {
		return append(parallelStages, stageToInject)
	}

	return parallelStages
}

func injectBuildStagesBefore(config *APIConfig, mft manifest.EstafetteManifest, builderTrack, gitSource string, supportsBuildStatus bool) (stages []*manifest.EstafetteStage) {

	stages = mft.Stages

	injectedStage := &manifest.EstafetteStage{
		Name:           getInjectedStageName("injected-before", stages),
		ParallelStages: []*manifest.EstafetteStage{},
		AutoInjected:   true,
	}

	injectedStage.ParallelStages = injectIfNotExists(stages, injectedStage.ParallelStages, &manifest.EstafetteStage{
		Name:           "git-clone",
		ContainerImage: fmt.Sprintf("extensions/git-clone:%v", builderTrack),
	})

	if supportsBuildStatus {
		injectedStage.ParallelStages = injectIfNotExists(stages, injectedStage.ParallelStages, &manifest.EstafetteStage{
			Name:           "set-pending-build-status",
			ContainerImage: fmt.Sprintf("extensions/%v-status:%v", gitSource, builderTrack),
			CustomProperties: map[string]interface{}{
				"status": "pending",
			},
		})
	}

	// add any configured injected stages
	if config != nil && config.APIServer != nil && config.APIServer.InjectStages != nil && config.APIServer.InjectStages.Build != nil && config.APIServer.InjectStages.Build.Before != nil {
		for _, s := range config.APIServer.InjectStages.Build.Before {
			injectedStage.ParallelStages = append(injectedStage.ParallelStages, s)
		}
	}

	if len(injectedStage.ParallelStages) > 0 {
		for _, ps := range injectedStage.ParallelStages {
			ps.AutoInjected = true
		}
		stages = append([]*manifest.EstafetteStage{injectedStage}, stages...)
	}

	return stages
}

func injectBuildStagesAfter(config *APIConfig, mft manifest.EstafetteManifest, builderTrack, gitSource string, supportsBuildStatus bool) (stages []*manifest.EstafetteStage) {

	stages = mft.Stages

	injectedStage := &manifest.EstafetteStage{
		Name:           getInjectedStageName("injected-after", stages),
		ParallelStages: []*manifest.EstafetteStage{},
		AutoInjected:   true,
		When:           "status == 'succeeded' || status == 'failed'",
	}

	if supportsBuildStatus {
		injectedStage.ParallelStages = injectIfNotExists(stages, injectedStage.ParallelStages, &manifest.EstafetteStage{
			Name:           "set-build-status",
			ContainerImage: fmt.Sprintf("extensions/%v-status:%v", gitSource, builderTrack),
			When:           "status == 'succeeded' || status == 'failed'",
		})
	}

	// add any configured injected stages
	if config != nil && config.APIServer != nil && config.APIServer.InjectStages != nil && config.APIServer.InjectStages.Build != nil && config.APIServer.InjectStages.Build.After != nil {
		for _, s := range config.APIServer.InjectStages.Build.After {
			injectedStage.ParallelStages = append(injectedStage.ParallelStages, s)
		}
	}

	if len(injectedStage.ParallelStages) > 0 {
		for _, ps := range injectedStage.ParallelStages {
			ps.AutoInjected = true
		}
		stages = append(stages, injectedStage)
	}

	return stages
}

func injectReleaseStagesBefore(config *APIConfig, release manifest.EstafetteRelease, builderTrack, gitSource string, supportsBuildStatus bool) (stages []*manifest.EstafetteStage) {

	stages = release.Stages

	injectedStage := &manifest.EstafetteStage{
		Name:           getInjectedStageName("injected-before", stages),
		ParallelStages: []*manifest.EstafetteStage{},
		AutoInjected:   true,
	}

	if release.CloneRepository {
		injectedStage.ParallelStages = injectIfNotExists(stages, injectedStage.ParallelStages, &manifest.EstafetteStage{
			Name:           "git-clone",
			ContainerImage: fmt.Sprintf("extensions/git-clone:%v", builderTrack),
		})
	}

	// add any configured injected stages
	if config != nil && config.APIServer != nil && config.APIServer.InjectStages != nil && config.APIServer.InjectStages.Build != nil && config.APIServer.InjectStages.Build.Before != nil {
		for _, s := range config.APIServer.InjectStages.Build.Before {
			injectedStage.ParallelStages = append(injectedStage.ParallelStages, s)
		}
	}

	if len(injectedStage.ParallelStages) > 0 {
		for _, ps := range injectedStage.ParallelStages {
			ps.AutoInjected = true
		}
		stages = append([]*manifest.EstafetteStage{injectedStage}, stages...)
	}

	return stages
}

func injectReleaseStagesAfter(config *APIConfig, release manifest.EstafetteRelease, builderTrack, gitSource string, supportsBuildStatus bool) (stages []*manifest.EstafetteStage) {

	stages = release.Stages

	injectedStage := &manifest.EstafetteStage{
		Name:           getInjectedStageName("injected-after", stages),
		ParallelStages: []*manifest.EstafetteStage{},
		AutoInjected:   true,
		When:           "status == 'succeeded' || status == 'failed'",
	}

	// add any configured injected stages
	if config != nil && config.APIServer != nil && config.APIServer.InjectStages != nil && config.APIServer.InjectStages.Release != nil && config.APIServer.InjectStages.Release.After != nil {
		for _, s := range config.APIServer.InjectStages.Release.After {
			injectedStage.ParallelStages = append(injectedStage.ParallelStages, s)
		}
	}

	if len(injectedStage.ParallelStages) > 0 {
		for _, ps := range injectedStage.ParallelStages {
			ps.AutoInjected = true
		}
		stages = append(stages, injectedStage)
	}

	return stages
}

func stageExists(stages []*manifest.EstafetteStage, stageName string) bool {
	for _, stage := range stages {
		if stage.Name == stageName {
			return true
		}
	}
	return false
}
