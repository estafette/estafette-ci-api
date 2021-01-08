package api

import (
	"fmt"

	manifest "github.com/estafette/estafette-ci-manifest"
)

// InjectStages injects some mandatory and configured stages
func InjectStages(config *APIConfig, mft manifest.EstafetteManifest, builderTrack, gitSource, gitBranch string, supportsBuildStatus bool) (injectedManifest manifest.EstafetteManifest, err error) {

	// get preferences for defaults
	var preferences *manifest.EstafetteManifestPreferences
	if config != nil && config.ManifestPreferences != nil {
		preferences = config.ManifestPreferences
	} else {
		preferences = manifest.GetDefaultManifestPreferences()
	}

	// set preferences DefaultBranch to main if this happens to be used at this moment, so it gets set in the triggers correctly
	// todo figure out the default branch if a non-default branch is built
	if gitBranch == "main" {
		preferences.DefaultBranch = "main"
	}

	operatingSystem := getOperatingSystem(mft, *preferences)

	injectedManifest = mft

	// inject build stages
	injectedManifest.Stages = injectBuildStagesBefore(config, operatingSystem, injectedManifest, builderTrack, gitSource, supportsBuildStatus)
	injectedManifest.Stages = injectBuildStagesAfter(config, operatingSystem, injectedManifest, builderTrack, gitSource, supportsBuildStatus)

	// inject release stages
	for _, r := range injectedManifest.Releases {
		releaseOperatingSystem := operatingSystem
		if r.Builder != nil && r.Builder.OperatingSystem != "" {
			releaseOperatingSystem = r.Builder.OperatingSystem
		}

		r.Stages = injectReleaseStagesBefore(config, releaseOperatingSystem, *r, builderTrack, gitSource, supportsBuildStatus)
		r.Stages = injectReleaseStagesAfter(config, releaseOperatingSystem, *r, builderTrack, gitSource, supportsBuildStatus)
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

func injectBuildStagesBefore(config *APIConfig, operatingSystem string, mft manifest.EstafetteManifest, builderTrack, gitSource string, supportsBuildStatus bool) (stages []*manifest.EstafetteStage) {

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
	if config != nil && config.APIServer != nil && config.APIServer.InjectStagesPerOperatingSystem != nil {
		if injectedStages, found := config.APIServer.InjectStagesPerOperatingSystem[operatingSystem]; found && injectedStages.Build != nil && injectedStages.Build.Before != nil {
			for _, s := range injectedStages.Build.Before {
				injectedStage.ParallelStages = append(injectedStage.ParallelStages, s)
			}
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

func injectBuildStagesAfter(config *APIConfig, operatingSystem string, mft manifest.EstafetteManifest, builderTrack, gitSource string, supportsBuildStatus bool) (stages []*manifest.EstafetteStage) {

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
	if config != nil && config.APIServer != nil && config.APIServer.InjectStagesPerOperatingSystem != nil {
		if injectedStages, found := config.APIServer.InjectStagesPerOperatingSystem[operatingSystem]; found && injectedStages.Build != nil && injectedStages.Build.After != nil {
			for _, s := range injectedStages.Build.After {
				injectedStage.ParallelStages = append(injectedStage.ParallelStages, s)
			}
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

func injectReleaseStagesBefore(config *APIConfig, operatingSystem string, release manifest.EstafetteRelease, builderTrack, gitSource string, supportsBuildStatus bool) (stages []*manifest.EstafetteStage) {

	stages = release.Stages

	injectedStage := &manifest.EstafetteStage{
		Name:           getInjectedStageName("injected-before", stages),
		ParallelStages: []*manifest.EstafetteStage{},
		AutoInjected:   true,
	}

	if release.CloneRepository != nil && *release.CloneRepository {
		injectedStage.ParallelStages = injectIfNotExists(stages, injectedStage.ParallelStages, &manifest.EstafetteStage{
			Name:           "git-clone",
			ContainerImage: fmt.Sprintf("extensions/git-clone:%v", builderTrack),
		})
	}

	// add any configured injected stages
	if config != nil && config.APIServer != nil && config.APIServer.InjectStagesPerOperatingSystem != nil {
		if injectedStages, found := config.APIServer.InjectStagesPerOperatingSystem[operatingSystem]; found && injectedStages.Release != nil && injectedStages.Release.Before != nil {
			for _, s := range injectedStages.Release.Before {
				injectedStage.ParallelStages = append(injectedStage.ParallelStages, s)
			}
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

func injectReleaseStagesAfter(config *APIConfig, operatingSystem string, release manifest.EstafetteRelease, builderTrack, gitSource string, supportsBuildStatus bool) (stages []*manifest.EstafetteStage) {

	stages = release.Stages

	injectedStage := &manifest.EstafetteStage{
		Name:           getInjectedStageName("injected-after", stages),
		ParallelStages: []*manifest.EstafetteStage{},
		AutoInjected:   true,
		When:           "status == 'succeeded' || status == 'failed'",
	}

	// add any configured injected stages
	if config != nil && config.APIServer != nil && config.APIServer.InjectStagesPerOperatingSystem != nil {
		if injectedStages, found := config.APIServer.InjectStagesPerOperatingSystem[operatingSystem]; found && injectedStages.Release != nil && injectedStages.Release.After != nil {
			for _, s := range injectedStages.Release.After {
				injectedStage.ParallelStages = append(injectedStage.ParallelStages, s)
			}
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

func getOperatingSystem(mft manifest.EstafetteManifest, preferences manifest.EstafetteManifestPreferences) string {

	if mft.Builder.OperatingSystem == "" {
		return preferences.BuilderOperatingSystems[0]
	}

	return mft.Builder.OperatingSystem
}
