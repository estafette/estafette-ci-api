package estafette

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/estafette/estafette-ci-api/cockroach"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/rs/zerolog/log"
)

// BuildService encapsulates build and release creation and re-triggering
type BuildService interface {
	CreateBuild(build contracts.Build, waitForJobToStart bool) (*contracts.Build, error)
	FinishBuild(repoSource, repoOwner, repoName string, buildID int, buildStatus string) error
	CreateRelease(release contracts.Release, mft manifest.EstafetteManifest, repoBranch, repoRevision string, waitForJobToStart bool) (*contracts.Release, error)
	FinishRelease(repoSource, repoOwner, repoName string, releaseID int, releaseStatus string) error

	FirePipelineTriggers(build contracts.Build, event string) error
	FireReleaseTriggers(release contracts.Release, event string) error
	FireCronTriggers() error
}

type buildServiceImpl struct {
	cockroachDBClient    cockroach.DBClient
	ciBuilderClient      CiBuilderClient
	githubJobVarsFunc    func(string, string, string) (string, string, error)
	bitbucketJobVarsFunc func(string, string, string) (string, string, error)
}

// NewBuildService returns a new estafette.BuildService
func NewBuildService(cockroachDBClient cockroach.DBClient, ciBuilderClient CiBuilderClient, githubJobVarsFunc func(string, string, string) (string, string, error), bitbucketJobVarsFunc func(string, string, string) (string, string, error)) (buildService BuildService) {

	buildService = &buildServiceImpl{
		cockroachDBClient:    cockroachDBClient,
		ciBuilderClient:      ciBuilderClient,
		githubJobVarsFunc:    githubJobVarsFunc,
		bitbucketJobVarsFunc: bitbucketJobVarsFunc,
	}

	return
}

func (s *buildServiceImpl) CreateBuild(build contracts.Build, waitForJobToStart bool) (createdBuild *contracts.Build, err error) {

	// validate manifest
	hasValidManifest := false
	mft, manifestError := manifest.ReadManifest(build.Manifest)
	if manifestError != nil {
		log.Warn().Err(manifestError).Str("manifest", build.Manifest).Msgf("Deserializing Estafette manifest for pipeline %v/%v/%v and revision %v failed, continuing though so developer gets useful feedback", build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision)
	} else {
		hasValidManifest = true
	}

	// if manifest is invalid get the pipeline in order to use same labels as normally
	var pipeline *contracts.Pipeline
	if !hasValidManifest {
		pipeline, _ = s.cockroachDBClient.GetPipeline(build.RepoSource, build.RepoOwner, build.RepoName, false)
	}

	// set builder track
	builderTrack := "stable"
	if hasValidManifest {
		builderTrack = mft.Builder.Track
	}

	// get short version of repo source
	shortRepoSource := s.getShortRepoSource(build.RepoSource)

	// set build status
	buildStatus := "failed"
	if hasValidManifest {
		buildStatus = "running"
	}

	// inject build stages
	if hasValidManifest {
		mft, err = InjectSteps(mft, builderTrack, shortRepoSource)
		if err != nil {
			log.Error().Err(err).
				Msg("Failed injecting build stages for pipeline %v/%v/%v and revision %v")
			return
		}
	}

	// get or set autoincrement and build version
	autoincrement := 0
	if build.BuildVersion == "" {
		// get autoincrement number
		autoincrement, err = s.cockroachDBClient.GetAutoIncrement(shortRepoSource, build.RepoOwner, build.RepoName)
		if err != nil {
			return
		}

		// set build version number
		if hasValidManifest {
			build.BuildVersion = mft.Version.Version(manifest.EstafetteVersionParams{
				AutoIncrement: autoincrement,
				Branch:        build.RepoBranch,
				Revision:      build.RepoRevision,
			})
		} else if pipeline != nil {
			log.Debug().Msgf("Copying previous versioning for pipeline %v/%v/%v, because current manifest is invalid...", build.RepoSource, build.RepoOwner, build.RepoName)
			previousManifest, err := manifest.ReadManifest(build.Manifest)
			if err != nil {
				build.BuildVersion = previousManifest.Version.Version(manifest.EstafetteVersionParams{
					AutoIncrement: autoincrement,
					Branch:        build.RepoBranch,
					Revision:      build.RepoRevision,
				})
			} else {
				log.Warn().Msgf("Not using previous versioning for pipeline %v/%v/%v, because its manifest is also invalid...", build.RepoSource, build.RepoOwner, build.RepoName)
				build.BuildVersion = strconv.Itoa(autoincrement)
			}
		} else {
			// set build version to autoincrement so there's at least a version in the db and gui
			build.BuildVersion = strconv.Itoa(autoincrement)
		}
	} else {
		// get autoincrement from build version
		autoincrementCandidate := build.BuildVersion
		if hasValidManifest && mft.Version.SemVer != nil {
			re := regexp.MustCompile(`^[0-9]+\.[0-9]+\.([0-9]+)(-[0-9a-z-]+)?$`)
			match := re.FindStringSubmatch(build.BuildVersion)

			if len(match) > 1 {
				autoincrementCandidate = match[1]
			}
		}

		autoincrement, err = strconv.Atoi(autoincrementCandidate)
		if err != nil {
			log.Warn().Err(err).Str("buildversion", build.BuildVersion).Msgf("Failed extracting autoincrement from build version %v for pipeline %v/%v/%v revision %v", build.BuildVersion, build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision)
		}
	}

	if len(build.Labels) == 0 {
		var labels []contracts.Label
		if hasValidManifest {
			for k, v := range mft.Labels {
				labels = append(labels, contracts.Label{
					Key:   k,
					Value: v,
				})
			}
		} else if pipeline != nil {
			log.Debug().Msgf("Copying previous labels for pipeline %v/%v/%v, because current manifest is invalid...", build.RepoSource, build.RepoOwner, build.RepoName)
			labels = pipeline.Labels
		}
		build.Labels = labels
	}

	if len(build.ReleaseTargets) == 0 {
		var releaseTargets []contracts.ReleaseTarget
		if hasValidManifest {
			for _, r := range mft.Releases {
				releaseTarget := contracts.ReleaseTarget{
					Name:    r.Name,
					Actions: make([]manifest.EstafetteReleaseAction, 0),
				}
				if r.Actions != nil && len(r.Actions) > 0 {
					for _, a := range r.Actions {
						releaseTarget.Actions = append(releaseTarget.Actions, *a)
					}
				}
				releaseTargets = append(releaseTargets, releaseTarget)
			}
		} else if pipeline != nil {
			log.Debug().Msgf("Copying previous release targets for pipeline %v/%v/%v, because current manifest is invalid...", build.RepoSource, build.RepoOwner, build.RepoName)
			releaseTargets = pipeline.ReleaseTargets
		}
		build.ReleaseTargets = releaseTargets
	}

	if len(build.Triggers) == 0 {
		if hasValidManifest {
			build.Triggers = mft.GetAllTriggers()
		} else if pipeline != nil {
			log.Debug().Msgf("Copying previous release targets for pipeline %v/%v/%v, because current manifest is invalid...", build.RepoSource, build.RepoOwner, build.RepoName)
			build.Triggers = pipeline.Triggers
		}
	}

	// get authenticated url
	authenticatedRepositoryURL, environmentVariableWithToken, err := s.getAuthenticatedRepositoryURL(build.RepoSource, build.RepoOwner, build.RepoName)
	if err != nil {
		return
	}

	// store build in db
	createdBuild, err = s.cockroachDBClient.InsertBuild(contracts.Build{
		RepoSource:     build.RepoSource,
		RepoOwner:      build.RepoOwner,
		RepoName:       build.RepoName,
		RepoBranch:     build.RepoBranch,
		RepoRevision:   build.RepoRevision,
		BuildVersion:   build.BuildVersion,
		BuildStatus:    buildStatus,
		Labels:         build.Labels,
		ReleaseTargets: build.ReleaseTargets,
		Manifest:       build.Manifest,
		Commits:        build.Commits,
		Triggers:       build.Triggers,
	})
	if err != nil {
		return
	}

	buildID, err := strconv.Atoi(createdBuild.ID)
	if err != nil {
		return
	}

	// define ci builder params
	ciBuilderParams := CiBuilderParams{
		JobType:              "build",
		RepoSource:           build.RepoSource,
		RepoOwner:            build.RepoOwner,
		RepoName:             build.RepoName,
		RepoURL:              authenticatedRepositoryURL,
		RepoBranch:           build.RepoBranch,
		RepoRevision:         build.RepoRevision,
		EnvironmentVariables: environmentVariableWithToken,
		Track:                builderTrack,
		AutoIncrement:        autoincrement,
		VersionNumber:        build.BuildVersion,
		Manifest:             mft,
		BuildID:              buildID,
	}

	// create ci builder job
	if hasValidManifest {
		log.Debug().Msgf("Pipeline %v/%v/%v revision %v has valid manifest, creating build job...", build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision)
		// create ci builder job
		if waitForJobToStart {
			_, err = s.ciBuilderClient.CreateCiBuilderJob(ciBuilderParams)
			if err != nil {
				return
			}
		} else {
			go func(ciBuilderParams CiBuilderParams) {
				_, err = s.ciBuilderClient.CreateCiBuilderJob(ciBuilderParams)
				if err != nil {
					log.Warn().Err(err).Msgf("Failed creating async build job")
				}
			}(ciBuilderParams)
		}

		// handle triggers
		go func() {
			s.FirePipelineTriggers(build, "started")
		}()
	} else if manifestError != nil {
		log.Debug().Msgf("Pipeline %v/%v/%v revision %v with build id %v has invalid manifest, storing log...", build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision, build.ID)
		// store log with manifest unmarshalling error
		buildLog := contracts.BuildLog{
			BuildID:      createdBuild.ID,
			RepoSource:   createdBuild.RepoSource,
			RepoOwner:    createdBuild.RepoOwner,
			RepoName:     createdBuild.RepoName,
			RepoBranch:   createdBuild.RepoBranch,
			RepoRevision: createdBuild.RepoRevision,
			Steps: []contracts.BuildLogStep{
				contracts.BuildLogStep{
					Step: "validate-manifest",
					Image: &contracts.BuildLogStepDockerImage{
						Name:         "estafette/estafette-ci-builder",
						Tag:          "stable",
						IsPulled:     true,
						ImageSize:    0,
						PullDuration: time.Duration(0),
						Error:        "",
						IsTrusted:    true,
					},
					ExitCode:     1,
					Status:       "FAILED",
					AutoInjected: true,
					RunIndex:     0,
					LogLines: []contracts.BuildLogLine{
						contracts.BuildLogLine{
							LineNumber: 1,
							Timestamp:  time.Now().UTC(),
							StreamType: "stderr",
							Text:       manifestError.Error(),
						},
					},
				},
			},
		}

		err = s.cockroachDBClient.InsertBuildLog(buildLog)
		if err != nil {
			log.Warn().Err(err).Msgf("Failed inserting build log for invalid manifest")
		}
	}

	return
}

func (s *buildServiceImpl) FinishBuild(repoSource, repoOwner, repoName string, buildID int, buildStatus string) error {

	err := s.cockroachDBClient.UpdateBuildStatus(repoSource, repoOwner, repoName, buildID, buildStatus)
	if err != nil {
		return err
	}

	// handle triggers
	go func() {
		build, err := s.cockroachDBClient.GetPipelineBuildByID(repoSource, repoOwner, repoName, buildID, false)
		if err != nil {
			return
		}
		if build != nil {
			s.FirePipelineTriggers(*build, "finished")
		}
	}()

	return nil
}

func (s *buildServiceImpl) CreateRelease(release contracts.Release, mft manifest.EstafetteManifest, repoBranch, repoRevision string, waitForJobToStart bool) (createdRelease *contracts.Release, err error) {

	// set builder track
	builderTrack := mft.Builder.Track

	// get short version of repo source
	shortRepoSource := s.getShortRepoSource(release.RepoSource)

	// set release status
	releaseStatus := "running"

	// inject build stages
	mft, err = InjectSteps(mft, builderTrack, shortRepoSource)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed injecting build stages for release to %v of pipeline %v/%v/%v version %v", release.Name, release.RepoSource, release.RepoOwner, release.RepoName, release.ReleaseVersion)
		return
	}

	// get autoincrement from release version
	autoincrementCandidate := release.ReleaseVersion
	if mft.Version.SemVer != nil {
		re := regexp.MustCompile(`^[0-9]+\.[0-9]+\.([0-9]+)(-[0-9a-zA-Z-/]+)?$`)
		match := re.FindStringSubmatch(release.ReleaseVersion)

		if len(match) > 1 {
			autoincrementCandidate = match[1]
		}
	}

	autoincrement, err := strconv.Atoi(autoincrementCandidate)
	if err != nil {
		log.Warn().Err(err).Str("releaseversion", release.ReleaseVersion).Msgf("Failed extracting autoincrement from build version %v for pipeline %v/%v/%v", release.ReleaseVersion, release.RepoSource, release.RepoOwner, release.RepoName)
	}

	// get authenticated url
	authenticatedRepositoryURL, environmentVariableWithToken, err := s.getAuthenticatedRepositoryURL(release.RepoSource, release.RepoOwner, release.RepoName)
	if err != nil {
		return
	}

	// create release in database
	createdRelease, err = s.cockroachDBClient.InsertRelease(contracts.Release{
		Name:           release.Name,
		Action:         release.Action,
		RepoSource:     release.RepoSource,
		RepoOwner:      release.RepoOwner,
		RepoName:       release.RepoName,
		ReleaseVersion: release.ReleaseVersion,
		ReleaseStatus:  releaseStatus,
		TriggeredBy:    release.TriggeredBy,
	})
	if err != nil {
		return
	}

	insertedReleaseID, err := strconv.Atoi(createdRelease.ID)
	if err != nil {
		return
	}

	// define ci builder params
	ciBuilderParams := CiBuilderParams{
		JobType:              "release",
		RepoSource:           release.RepoSource,
		RepoOwner:            release.RepoOwner,
		RepoName:             release.RepoName,
		RepoURL:              authenticatedRepositoryURL,
		RepoBranch:           repoBranch,
		RepoRevision:         repoRevision,
		EnvironmentVariables: environmentVariableWithToken,
		Track:                builderTrack,
		AutoIncrement:        autoincrement,
		VersionNumber:        release.ReleaseVersion,
		Manifest:             mft,
		ReleaseID:            insertedReleaseID,
		ReleaseName:          release.Name,
		ReleaseAction:        release.Action,
		ReleaseTriggeredBy:   release.TriggeredBy,
	}

	// create ci release job
	if waitForJobToStart {
		_, err = s.ciBuilderClient.CreateCiBuilderJob(ciBuilderParams)
		if err != nil {
			return
		}
	} else {
		go func(ciBuilderParams CiBuilderParams) {
			_, err = s.ciBuilderClient.CreateCiBuilderJob(ciBuilderParams)
			if err != nil {
				log.Warn().Err(err).Msgf("Failed creating async release job")
			}
		}(ciBuilderParams)
	}

	// handle triggers
	go func() {
		s.FireReleaseTriggers(release, "started")
	}()

	return
}

func (s *buildServiceImpl) FinishRelease(repoSource, repoOwner, repoName string, releaseID int, releaseStatus string) error {
	err := s.cockroachDBClient.UpdateReleaseStatus(repoSource, repoOwner, repoName, releaseID, releaseStatus)
	if err != nil {
		return err
	}

	// handle triggers
	go func() {
		release, err := s.cockroachDBClient.GetPipelineRelease(repoSource, repoOwner, repoName, releaseID)
		if err != nil {
			return
		}
		if release != nil {
			s.FireReleaseTriggers(*release, "finished")
		}
	}()

	return nil
}

func (s *buildServiceImpl) FirePipelineTriggers(build contracts.Build, event string) error {

	log.Info().Msgf("[trigger:pipeline(%v/%v/%v:%v)] Checking if triggers need to be fired...", build.RepoSource, build.RepoOwner, build.RepoName, event)

	// retrieve all pipeline triggers
	pipelines, err := s.cockroachDBClient.GetPipelineTriggers(build, event)
	if err != nil {
		return err
	}

	// create event object
	pe := manifest.EstafettePipelineEvent{
		RepoSource: build.RepoSource,
		RepoOwner:  build.RepoOwner,
		RepoName:   build.RepoName,
		Branch:     build.RepoBranch,
		Status:     build.BuildStatus,
		Event:      event,
	}

	triggerCount := 0
	firedTriggerCount := 0

	// check for each trigger whether it should fire
	for _, p := range pipelines {
		for _, t := range p.Triggers {

			log.Debug().Interface("event", pe).Interface("trigger", t).Msgf("[trigger:pipeline(%v/%v/%v:%v)] Checking if pipeline '%v/%v/%v' trigger should fire...", build.RepoSource, build.RepoOwner, build.RepoName, event, p.RepoSource, p.RepoOwner, p.RepoName)

			if t.Pipeline == nil {
				continue
			}

			triggerCount++

			if t.Pipeline.Fires(&pe) {

				firedTriggerCount++

				// create new build for t.Run
				if t.BuildAction != nil {
					log.Info().Msgf("[trigger:pipeline(%v/%v/%v:%v)] Firing build action '%v/%v/%v', branch '%v'...", build.RepoSource, build.RepoOwner, build.RepoName, event, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
					err := s.fireBuild(*p, t)
					if err != nil {
						log.Info().Msgf("[trigger:pipeline(%v/%v/%v:%v)] Failed starting build action'%v/%v/%v', branch '%v'", build.RepoSource, build.RepoOwner, build.RepoName, event, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
					}
				} else if t.ReleaseAction != nil {
					log.Info().Msgf("[trigger:pipeline(%v/%v/%v:%v)] Firing release action '%v/%v/%v', target '%v', action '%v'...", build.RepoSource, build.RepoOwner, build.RepoName, event, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
					err := s.fireRelease(*p, t, fmt.Sprintf("trigger.pipeline { name: %v/%v/%v, event: %v }", build.RepoSource, build.RepoOwner, build.RepoName, event))
					if err != nil {
						log.Info().Msgf("[trigger:pipeline(%v/%v/%v:%v)] Failed starting release action '%v/%v/%v', target '%v', action '%v'", build.RepoSource, build.RepoOwner, build.RepoName, event, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
					}
				}
			}
		}
	}

	log.Info().Msgf("[trigger:pipeline(%v/%v/%v:%v)] Fired %v out of %v triggers for %v pipelines", build.RepoSource, build.RepoOwner, build.RepoName, event, firedTriggerCount, triggerCount, len(pipelines))

	return nil
}

func (s *buildServiceImpl) FireReleaseTriggers(release contracts.Release, event string) error {

	log.Info().Msgf("[trigger:release(%v/%v/%v-%v:%v] Checking if triggers need to be fired...", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, event)

	pipelines, err := s.cockroachDBClient.GetReleaseTriggers(release, event)
	if err != nil {
		return err
	}

	// create event object
	re := manifest.EstafetteReleaseEvent{
		RepoSource: release.RepoSource,
		RepoOwner:  release.RepoOwner,
		RepoName:   release.RepoName,
		Target:     release.Name,
		Status:     release.ReleaseStatus,
		Event:      event,
	}

	triggerCount := 0
	firedTriggerCount := 0

	// check for each trigger whether it should fire
	for _, p := range pipelines {
		for _, t := range p.Triggers {

			log.Debug().Interface("event", re).Interface("trigger", t).Msgf("[trigger:release(%v/%v/%v-%v:%v)] Checking if pipeline '%v/%v/%v' trigger should fire...", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, event, p.RepoSource, p.RepoOwner, p.RepoName)

			if t.Release == nil {
				continue
			}

			triggerCount++

			if t.Release.Fires(&re) {

				firedTriggerCount++

				if t.BuildAction != nil {
					log.Info().Msgf("[trigger:release(%v/%v/%v-%v:%v)] Firing build action '%v/%v/%v', branch '%v'...", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, event, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
					err := s.fireBuild(*p, t)
					if err != nil {
						log.Info().Msgf("[trigger:release(%v/%v/%v-%v:%v)] Failed starting build action '%v/%v/%v', branch '%v'", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, event, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
					}
				} else if t.ReleaseAction != nil {
					log.Info().Msgf("[trigger:release(%v/%v/%v-%v:%v)] Firing release action '%v/%v/%v', target '%v', action '%v'...", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, event, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
					err := s.fireRelease(*p, t, fmt.Sprintf("trigger.release { name: %v/%v/%v, target: %v, event: %v }", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, event))
					if err != nil {
						log.Info().Msgf("[trigger:release(%v/%v/%v-%v:%v)] Failed starting release action '%v/%v/%v', target '%v', action '%v'", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, event, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
					}
				}
			}
		}
	}

	log.Info().Msgf("[trigger:release(%v/%v/%v-%v:%v] Fired %v out of %v triggers for %v pipelines", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, event, firedTriggerCount, triggerCount, len(pipelines))

	return nil
}

func (s *buildServiceImpl) FireCronTriggers() error {

	// create event object
	ce := manifest.EstafetteCronEvent{
		Time: time.Now().UTC(),
	}

	log.Info().Msgf("[trigger:cron(%v)] Checking if triggers need to be fired...", ce.Time)

	pipelines, err := s.cockroachDBClient.GetCronTriggers()
	if err != nil {
		return err
	}

	triggerCount := 0
	firedTriggerCount := 0

	// check for each trigger whether it should fire
	for _, p := range pipelines {
		for _, t := range p.Triggers {

			log.Debug().Interface("event", ce).Interface("trigger", t).Msgf("[trigger:cron(%v)] Checking if pipeline '%v/%v/%v' trigger should fire...", ce.Time, p.RepoSource, p.RepoOwner, p.RepoName)

			if t.Cron == nil {
				continue
			}

			triggerCount++

			if t.Cron.Fires(&ce) {

				firedTriggerCount++

				// create new build for t.Run
				if t.BuildAction != nil {
					log.Info().Msgf("[trigger:cron(%v)] Firing build action '%v/%v/%v', branch '%v'...", ce.Time, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
					err := s.fireBuild(*p, t)
					if err != nil {
						log.Info().Msgf("[trigger:cron(%v)] Failed starting build action'%v/%v/%v', branch '%v'", ce.Time, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
					}
				} else if t.ReleaseAction != nil {
					log.Info().Msgf("[trigger:cron(%v)] Firing release action '%v/%v/%v', target '%v', action '%v'...", ce.Time, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
					err := s.fireRelease(*p, t, fmt.Sprintf("trigger.cron { time: %v }", ce.Time))
					if err != nil {
						log.Info().Msgf("[trigger:cron(%v)] Failed starting release action '%v/%v/%v', target '%v', action '%v'", ce.Time, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
					}
				}
			}
		}
	}

	log.Info().Msgf("[trigger:cron(%v)] Fired %v out of %v triggers for %v pipelines", ce.Time, firedTriggerCount, triggerCount, len(pipelines))

	return nil
}

func (s *buildServiceImpl) fireBuild(p contracts.Pipeline, t manifest.EstafetteTrigger) error {
	if t.BuildAction == nil {
		return fmt.Errorf("Trigger to fire does not have a 'builds' property, shouldn't get to here")
	}

	// get last build for branch defined in 'builds' section
	lastBuildForBranch, err := s.cockroachDBClient.GetLastPipelineBuildForBranch(p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)

	if lastBuildForBranch == nil {
		return fmt.Errorf("There's no build for pipeline '%v/%v/%v' branch '%v', cannot trigger one", p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
	}

	// empty the build version so a new one gets created
	lastBuildForBranch.BuildVersion = ""

	_, err = s.CreateBuild(*lastBuildForBranch, true)
	if err != nil {
		return err
	}
	return nil
}

func (s *buildServiceImpl) fireRelease(p contracts.Pipeline, t manifest.EstafetteTrigger, triggeredBy string) error {
	if t.ReleaseAction == nil {
		return fmt.Errorf("Trigger to fire does not have a 'releases' property, shouldn't get to here")
	}

	_, err := s.CreateRelease(contracts.Release{
		Name:           t.ReleaseAction.Target,
		Action:         t.ReleaseAction.Action,
		RepoSource:     p.RepoSource,
		RepoOwner:      p.RepoOwner,
		RepoName:       p.RepoName,
		ReleaseVersion: p.BuildVersion,
		TriggeredBy:    triggeredBy,
	}, *p.ManifestObject, p.RepoBranch, p.RepoRevision, true)
	if err != nil {
		return err
	}
	return nil
}

func (s *buildServiceImpl) getShortRepoSource(repoSource string) string {

	repoSourceArray := strings.Split(repoSource, ".")

	if len(repoSourceArray) <= 0 {
		return repoSource
	}

	return repoSourceArray[0]
}

func (s *buildServiceImpl) getAuthenticatedRepositoryURL(repoSource, repoOwner, repoName string) (authenticatedRepositoryURL string, environmentVariableWithToken map[string]string, err error) {

	switch repoSource {
	case "github.com":
		var accessToken string
		accessToken, authenticatedRepositoryURL, err = s.githubJobVarsFunc(repoSource, repoOwner, repoName)
		if err != nil {
			return
		}
		environmentVariableWithToken = map[string]string{"ESTAFETTE_GITHUB_API_TOKEN": accessToken}
		return

	case "bitbucket.org":
		var accessToken string
		accessToken, authenticatedRepositoryURL, err = s.bitbucketJobVarsFunc(repoSource, repoOwner, repoName)
		if err != nil {
			return
		}
		environmentVariableWithToken = map[string]string{"ESTAFETTE_BITBUCKET_API_TOKEN": accessToken}
		return
	}

	return authenticatedRepositoryURL, environmentVariableWithToken, fmt.Errorf("Source %v not supported for generating authenticated repository url", repoSource)
}
