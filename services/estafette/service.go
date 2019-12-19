package estafette

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/estafette/estafette-ci-api/clients/builderapi"
	"github.com/estafette/estafette-ci-api/clients/cloudstorage"
	"github.com/estafette/estafette-ci-api/clients/cockroachdb"
	"github.com/estafette/estafette-ci-api/clients/prometheus"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/helpers"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/rs/zerolog/log"
)

var (
	ErrNoBuildCreated   = errors.New("No build is created")
	ErrNoReleaseCreated = errors.New("No release is created")
)

// Service encapsulates build and release creation and re-triggering
type Service interface {
	CreateBuild(ctx context.Context, build contracts.Build, waitForJobToStart bool) (b *contracts.Build, err error)
	FinishBuild(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus string) (err error)
	CreateRelease(ctx context.Context, release contracts.Release, mft manifest.EstafetteManifest, repoBranch, repoRevision string, waitForJobToStart bool) (r *contracts.Release, err error)
	FinishRelease(ctx context.Context, repoSource, repoOwner, repoName string, releaseID int, releaseStatus string) (err error)
	FireGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (err error)
	FirePipelineTriggers(ctx context.Context, build contracts.Build, event string) (err error)
	FireReleaseTriggers(ctx context.Context, release contracts.Release, event string) (err error)
	FirePubSubTriggers(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) (err error)
	FireCronTriggers(ctx context.Context) (err error)
	Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
	UpdateBuildStatus(ctx context.Context, event builderapi.CiBuilderEvent) (err error)
	UpdateJobResources(ctx context.Context, event builderapi.CiBuilderEvent) (err error)
}

// NewService returns a new estafette.Service
func NewService(jobsConfig config.JobsConfig, apiServerConfig config.APIServerConfig, cockroachdbClient cockroachdb.Client, prometheusClient prometheus.Client, cloudStorageClient cloudstorage.Client, builderapiClient builderapi.Client, githubJobVarsFunc func(context.Context, string, string, string) (string, string, error), bitbucketJobVarsFunc func(context.Context, string, string, string) (string, string, error)) Service {

	return &service{
		jobsConfig:           jobsConfig,
		apiServerConfig:      apiServerConfig,
		cockroachdbClient:    cockroachdbClient,
		prometheusClient:     prometheusClient,
		cloudStorageClient:   cloudStorageClient,
		builderapiClient:     builderapiClient,
		githubJobVarsFunc:    githubJobVarsFunc,
		bitbucketJobVarsFunc: bitbucketJobVarsFunc,
	}
}

type service struct {
	jobsConfig           config.JobsConfig
	apiServerConfig      config.APIServerConfig
	cockroachdbClient    cockroachdb.Client
	prometheusClient     prometheus.Client
	cloudStorageClient   cloudstorage.Client
	builderapiClient     builderapi.Client
	githubJobVarsFunc    func(context.Context, string, string, string) (string, string, error)
	bitbucketJobVarsFunc func(context.Context, string, string, string) (string, string, error)
}

func (s *service) CreateBuild(ctx context.Context, build contracts.Build, waitForJobToStart bool) (createdBuild *contracts.Build, err error) {

	// validate manifest
	mft, manifestError := manifest.ReadManifest(build.Manifest)
	hasValidManifest := manifestError == nil

	// if manifest is invalid get the pipeline in order to use same labels, release targets and triggers as before
	var pipeline *contracts.Pipeline
	if !hasValidManifest {
		pipeline, _ = s.cockroachdbClient.GetPipeline(ctx, build.RepoSource, build.RepoOwner, build.RepoName, false)
	}

	// set builder track
	builderTrack := "stable"
	builderOperatingSystem := "linux"
	if hasValidManifest {
		builderTrack = mft.Builder.Track
		builderOperatingSystem = mft.Builder.OperatingSystem
	}

	// get short version of repo source
	shortRepoSource := s.getShortRepoSource(build.RepoSource)

	// set build status
	buildStatus := "failed"
	if hasValidManifest {
		buildStatus = "pending"
	}

	// inject build stages
	if hasValidManifest {
		mft, err = helpers.InjectSteps(mft, builderTrack, shortRepoSource)
		if err != nil {
			log.Error().Err(err).
				Msg("Failed injecting build stages for pipeline %v/%v/%v and revision %v")
			return
		}
	}

	autoincrement, err := s.getBuildAutoIncrement(ctx, build, shortRepoSource, hasValidManifest, mft, pipeline)
	if err != nil {
		return nil, err
	}

	build.Labels = s.getBuildLabels(build, hasValidManifest, mft, pipeline)
	build.ReleaseTargets = s.getBuildReleaseTargets(build, hasValidManifest, mft, pipeline)
	build.Triggers = s.getBuildTriggers(build, hasValidManifest, mft, pipeline)

	// get authenticated url
	authenticatedRepositoryURL, environmentVariableWithToken, err := s.getAuthenticatedRepositoryURL(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
	if err != nil {
		return
	}

	jobResources := s.getBuildJobResources(ctx, build)

	// store build in db
	createdBuild, err = s.cockroachdbClient.InsertBuild(ctx, contracts.Build{
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
		Events:         build.Events,
	}, jobResources)
	if err != nil {
		return
	}
	if createdBuild == nil {
		return nil, ErrNoBuildCreated
	}

	buildID, err := strconv.Atoi(createdBuild.ID)
	if err != nil {
		return
	}

	// define ci builder params
	ciBuilderParams := builderapi.CiBuilderParams{
		JobType:              "build",
		RepoSource:           build.RepoSource,
		RepoOwner:            build.RepoOwner,
		RepoName:             build.RepoName,
		RepoURL:              authenticatedRepositoryURL,
		RepoBranch:           build.RepoBranch,
		RepoRevision:         build.RepoRevision,
		EnvironmentVariables: environmentVariableWithToken,
		Track:                builderTrack,
		OperatingSystem:      builderOperatingSystem,
		AutoIncrement:        autoincrement,
		VersionNumber:        build.BuildVersion,
		Manifest:             mft,
		BuildID:              buildID,
		TriggeredByEvents:    build.Events,
		JobResources:         jobResources,
	}

	// create ci builder job
	if hasValidManifest {
		log.Debug().Msgf("Pipeline %v/%v/%v revision %v has valid manifest, creating build job...", build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision)
		// create ci builder job
		if waitForJobToStart {
			_, err = s.builderapiClient.CreateCiBuilderJob(ctx, ciBuilderParams)
			if err != nil {
				return
			}
		} else {
			go func(ciBuilderParams builderapi.CiBuilderParams) {
				_, err = s.builderapiClient.CreateCiBuilderJob(ctx, ciBuilderParams)
				if err != nil {
					log.Warn().Err(err).Msgf("Failed creating async build job")
				}
			}(ciBuilderParams)
		}

		// handle triggers
		go func() {
			err := s.FirePipelineTriggers(ctx, build, "started")
			if err != nil {
				log.Error().Err(err).Msgf("Failed firing pipeline triggers for build %v/%v/%v revision %v", build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision)
			}
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
			Steps: []*contracts.BuildLogStep{
				&contracts.BuildLogStep{
					Step:         "validate-manifest",
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

		insertedBuildLog, err := s.cockroachdbClient.InsertBuildLog(ctx, buildLog, s.apiServerConfig.WriteLogToDatabase())
		if err != nil {
			log.Warn().Err(err).Msgf("Failed inserting build log for invalid manifest")
		}

		if s.apiServerConfig.WriteLogToCloudStorage() {
			err = s.cloudStorageClient.InsertBuildLog(ctx, insertedBuildLog)
			if err != nil {
				log.Warn().Err(err).Msgf("Failed inserting build log into cloud storage for invalid manifest")
			}
		}
	}

	return
}

func (s *service) FinishBuild(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus string) error {

	err := s.cockroachdbClient.UpdateBuildStatus(ctx, repoSource, repoOwner, repoName, buildID, buildStatus)
	if err != nil {
		return err
	}

	// handle triggers
	go func() {
		build, err := s.cockroachdbClient.GetPipelineBuildByID(ctx, repoSource, repoOwner, repoName, buildID, false)
		if err != nil {
			return
		}
		if build != nil {
			err = s.FirePipelineTriggers(ctx, *build, "finished")
			if err != nil {
				log.Error().Err(err).Msgf("Failed firing pipeline triggers for build %v/%v/%v id %v", repoSource, repoOwner, repoName, buildID)
			}
		}
	}()

	return nil
}

func (s *service) CreateRelease(ctx context.Context, release contracts.Release, mft manifest.EstafetteManifest, repoBranch, repoRevision string, waitForJobToStart bool) (createdRelease *contracts.Release, err error) {

	// set builder track
	builderTrack := mft.Builder.Track
	builderOperatingSystem := mft.Builder.OperatingSystem

	// get builder track override for release if exists
	for _, r := range mft.Releases {
		if r.Name == release.Name {
			if r.Builder != nil {
				builderTrack = r.Builder.Track
				builderOperatingSystem = r.Builder.OperatingSystem
				break
			}
		}
	}

	// get short version of repo source
	shortRepoSource := s.getShortRepoSource(release.RepoSource)

	// set release status
	releaseStatus := "pending"

	// inject build stages
	mft, err = helpers.InjectSteps(mft, builderTrack, shortRepoSource)
	if err != nil {
		log.Error().Err(err).
			Msgf("Failed injecting build stages for release to %v of pipeline %v/%v/%v version %v", release.Name, release.RepoSource, release.RepoOwner, release.RepoName, release.ReleaseVersion)
		return
	}

	// get autoincrement from release version
	autoincrement := s.getReleaseAutoIncrement(ctx, release, mft)

	// get authenticated url
	authenticatedRepositoryURL, environmentVariableWithToken, err := s.getAuthenticatedRepositoryURL(ctx, release.RepoSource, release.RepoOwner, release.RepoName)
	if err != nil {
		return
	}

	jobResources := s.getReleaseJobResources(ctx, release)

	// create release in database
	createdRelease, err = s.cockroachdbClient.InsertRelease(ctx, contracts.Release{
		Name:           release.Name,
		Action:         release.Action,
		RepoSource:     release.RepoSource,
		RepoOwner:      release.RepoOwner,
		RepoName:       release.RepoName,
		ReleaseVersion: release.ReleaseVersion,
		ReleaseStatus:  releaseStatus,
		Events:         release.Events,
	}, jobResources)
	if err != nil {
		return
	}
	if createdRelease == nil {
		return nil, ErrNoReleaseCreated
	}

	insertedReleaseID, err := strconv.Atoi(createdRelease.ID)
	if err != nil {
		return
	}

	// get triggered by from events
	triggeredBy := ""
	if len(release.Events) > 0 {
		for _, e := range release.Events {
			if e.Manual != nil {
				triggeredBy = e.Manual.UserID
			}
		}
	}

	// define ci builder params
	ciBuilderParams := builderapi.CiBuilderParams{
		JobType:              "release",
		RepoSource:           release.RepoSource,
		RepoOwner:            release.RepoOwner,
		RepoName:             release.RepoName,
		RepoURL:              authenticatedRepositoryURL,
		RepoBranch:           repoBranch,
		RepoRevision:         repoRevision,
		EnvironmentVariables: environmentVariableWithToken,
		Track:                builderTrack,
		OperatingSystem:      builderOperatingSystem,
		AutoIncrement:        autoincrement,
		VersionNumber:        release.ReleaseVersion,
		Manifest:             mft,
		ReleaseID:            insertedReleaseID,
		ReleaseName:          release.Name,
		ReleaseAction:        release.Action,
		ReleaseTriggeredBy:   triggeredBy,
		TriggeredByEvents:    release.Events,
		JobResources:         jobResources,
	}

	// create ci release job
	if waitForJobToStart {
		_, err = s.builderapiClient.CreateCiBuilderJob(ctx, ciBuilderParams)
		if err != nil {
			return
		}
	} else {
		go func(ciBuilderParams builderapi.CiBuilderParams) {
			_, err = s.builderapiClient.CreateCiBuilderJob(ctx, ciBuilderParams)
			if err != nil {
				log.Warn().Err(err).Msgf("Failed creating async release job")
			}
		}(ciBuilderParams)
	}

	// handle triggers
	go func() {
		err := s.FireReleaseTriggers(ctx, release, "started")
		if err != nil {
			log.Error().Err(err).Msgf("Failed firing release triggers for %v/%v/%v to target %v", release.RepoSource, release.RepoOwner, release.RepoName, release.Name)
		}
	}()

	return
}

func (s *service) FinishRelease(ctx context.Context, repoSource, repoOwner, repoName string, releaseID int, releaseStatus string) error {
	err := s.cockroachdbClient.UpdateReleaseStatus(ctx, repoSource, repoOwner, repoName, releaseID, releaseStatus)
	if err != nil {
		return err
	}

	// handle triggers
	go func() {
		release, err := s.cockroachdbClient.GetPipelineRelease(ctx, repoSource, repoOwner, repoName, releaseID)
		if err != nil {
			return
		}
		if release != nil {
			err = s.FireReleaseTriggers(ctx, *release, "finished")
			if err != nil {
				log.Error().Err(err).Msgf("Failed firing release triggers for %v/%v/%v id %v", repoSource, repoOwner, repoName, releaseID)
			}
		}
	}()

	return nil
}

func (s *service) FireGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) error {

	log.Info().Msgf("[trigger:git(%v-%v:%v)] Checking if triggers need to be fired...", gitEvent.Repository, gitEvent.Branch, gitEvent.Event)

	// retrieve all pipeline triggers
	pipelines, err := s.cockroachdbClient.GetGitTriggers(ctx, gitEvent)
	if err != nil {
		return err
	}

	e := manifest.EstafetteEvent{
		Git: &gitEvent,
	}

	triggerCount := 0
	firedTriggerCount := 0

	// check for each trigger whether it should fire
	for _, p := range pipelines {
		for _, t := range p.Triggers {

			log.Debug().Interface("event", gitEvent).Interface("trigger", t).Msgf("[trigger:git(%v-%v:%v)] Checking if pipeline '%v/%v/%v' trigger should fire...", gitEvent.Repository, gitEvent.Branch, gitEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName)

			if t.Git == nil {
				continue
			}

			triggerCount++

			if t.Git.Fires(&gitEvent) {

				firedTriggerCount++

				// create new build for t.Run
				if t.BuildAction != nil {
					log.Info().Msgf("[trigger:git(%v-%v:%v)] Firing build action '%v/%v/%v', branch '%v'...", gitEvent.Repository, gitEvent.Branch, gitEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
					err := s.fireBuild(ctx, *p, t, e)
					if err != nil {
						log.Error().Err(err).Msgf("[trigger:git(%v-%v:%v)] Failed starting build action'%v/%v/%v', branch '%v'", gitEvent.Repository, gitEvent.Branch, gitEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
					}
				} else if t.ReleaseAction != nil {
					log.Info().Msgf("[trigger:git(%v-%v:%v)] Firing release action '%v/%v/%v', target '%v', action '%v'...", gitEvent.Repository, gitEvent.Branch, gitEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
					err := s.fireRelease(ctx, *p, t, e)
					if err != nil {
						log.Error().Err(err).Msgf("[trigger:git(%v-%v:%v)] Failed starting release action '%v/%v/%v', target '%v', action '%v'", gitEvent.Repository, gitEvent.Branch, gitEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
					}
				}
			}
		}
	}

	log.Info().Msgf("[trigger:git(%v-%v:%v)] Fired %v out of %v triggers for %v pipelines", gitEvent.Repository, gitEvent.Branch, gitEvent.Event, firedTriggerCount, triggerCount, len(pipelines))

	return nil
}

func (s *service) FirePipelineTriggers(ctx context.Context, build contracts.Build, event string) error {

	log.Info().Msgf("[trigger:pipeline(%v/%v/%v:%v)] Checking if triggers need to be fired...", build.RepoSource, build.RepoOwner, build.RepoName, event)

	// retrieve all pipeline triggers
	pipelines, err := s.cockroachdbClient.GetPipelineTriggers(ctx, build, event)
	if err != nil {
		return err
	}

	// create event object
	pe := manifest.EstafettePipelineEvent{
		BuildVersion: build.BuildVersion,
		RepoSource:   build.RepoSource,
		RepoOwner:    build.RepoOwner,
		RepoName:     build.RepoName,
		Branch:       build.RepoBranch,
		Status:       build.BuildStatus,
		Event:        event,
	}
	e := manifest.EstafetteEvent{
		Pipeline: &pe,
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
					err := s.fireBuild(ctx, *p, t, e)
					if err != nil {
						log.Error().Err(err).Msgf("[trigger:pipeline(%v/%v/%v:%v)] Failed starting build action'%v/%v/%v', branch '%v'", build.RepoSource, build.RepoOwner, build.RepoName, event, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
					}
				} else if t.ReleaseAction != nil {
					log.Info().Msgf("[trigger:pipeline(%v/%v/%v:%v)] Firing release action '%v/%v/%v', target '%v', action '%v'...", build.RepoSource, build.RepoOwner, build.RepoName, event, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
					err := s.fireRelease(ctx, *p, t, e)
					if err != nil {
						log.Error().Err(err).Msgf("[trigger:pipeline(%v/%v/%v:%v)] Failed starting release action '%v/%v/%v', target '%v', action '%v'", build.RepoSource, build.RepoOwner, build.RepoName, event, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
					}
				}
			}
		}
	}

	log.Info().Msgf("[trigger:pipeline(%v/%v/%v:%v)] Fired %v out of %v triggers for %v pipelines", build.RepoSource, build.RepoOwner, build.RepoName, event, firedTriggerCount, triggerCount, len(pipelines))

	return nil
}

func (s *service) FireReleaseTriggers(ctx context.Context, release contracts.Release, event string) error {

	log.Info().Msgf("[trigger:release(%v/%v/%v-%v:%v] Checking if triggers need to be fired...", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, event)

	pipelines, err := s.cockroachdbClient.GetReleaseTriggers(ctx, release, event)
	if err != nil {
		return err
	}

	// create event object
	re := manifest.EstafetteReleaseEvent{
		ReleaseVersion: release.ReleaseVersion,
		RepoSource:     release.RepoSource,
		RepoOwner:      release.RepoOwner,
		RepoName:       release.RepoName,
		Target:         release.Name,
		Status:         release.ReleaseStatus,
		Event:          event,
	}
	e := manifest.EstafetteEvent{
		Release: &re,
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
					err := s.fireBuild(ctx, *p, t, e)
					if err != nil {
						log.Error().Err(err).Msgf("[trigger:release(%v/%v/%v-%v:%v)] Failed starting build action '%v/%v/%v', branch '%v'", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, event, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
					}
				} else if t.ReleaseAction != nil {
					log.Info().Msgf("[trigger:release(%v/%v/%v-%v:%v)] Firing release action '%v/%v/%v', target '%v', action '%v'...", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, event, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
					err := s.fireRelease(ctx, *p, t, e)
					if err != nil {
						log.Error().Err(err).Msgf("[trigger:release(%v/%v/%v-%v:%v)] Failed starting release action '%v/%v/%v', target '%v', action '%v'", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, event, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
					}
				}
			}
		}
	}

	log.Info().Msgf("[trigger:release(%v/%v/%v-%v:%v] Fired %v out of %v triggers for %v pipelines", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, event, firedTriggerCount, triggerCount, len(pipelines))

	return nil
}

func (s *service) FirePubSubTriggers(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) error {

	log.Info().Msgf("[trigger:pubsub(projects/%v/topics/%v)] Checking if triggers need to be fired...", pubsubEvent.Project, pubsubEvent.Topic)

	// retrieve all pipeline triggers
	pipelines, err := s.cockroachdbClient.GetPubSubTriggers(ctx, pubsubEvent)
	if err != nil {
		return err
	}

	e := manifest.EstafetteEvent{
		PubSub: &pubsubEvent,
	}

	triggerCount := 0
	firedTriggerCount := 0

	// check for each trigger whether it should fire
	for _, p := range pipelines {
		for _, t := range p.Triggers {

			log.Debug().Interface("event", pubsubEvent).Interface("trigger", t).Msgf("[trigger:pubsub(projects/%v/topics/%v)] Checking if pipeline '%v/%v/%v' trigger should fire...", pubsubEvent.Project, pubsubEvent.Topic, p.RepoSource, p.RepoOwner, p.RepoName)

			if t.PubSub == nil {
				continue
			}

			triggerCount++

			if t.PubSub.Fires(&pubsubEvent) {

				firedTriggerCount++

				// create new build for t.Run
				if t.BuildAction != nil {
					log.Info().Msgf("[trigger:pubsub(projects/%v/topics/%v)] Firing build action '%v/%v/%v', branch '%v'...", pubsubEvent.Project, pubsubEvent.Topic, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
					err := s.fireBuild(ctx, *p, t, e)
					if err != nil {
						log.Error().Err(err).Msgf("[trigger:pubsub(projects/%v/topics/%v)] Failed starting build action'%v/%v/%v', branch '%v'", pubsubEvent.Project, pubsubEvent.Topic, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
					}
				} else if t.ReleaseAction != nil {
					log.Info().Msgf("[trigger:pubsub(projects/%v/topics/%v)] Firing release action '%v/%v/%v', target '%v', action '%v'...", pubsubEvent.Project, pubsubEvent.Topic, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
					err := s.fireRelease(ctx, *p, t, e)
					if err != nil {
						log.Error().Err(err).Msgf("[trigger:pubsub(projects/%v/topics/%v)] Failed starting release action '%v/%v/%v', target '%v', action '%v'", pubsubEvent.Project, pubsubEvent.Topic, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
					}
				}
			}
		}
	}

	log.Info().Msgf("[trigger:pubsub(projects/%v/topics/%v)] Fired %v out of %v triggers for %v pipelines", pubsubEvent.Project, pubsubEvent.Topic, firedTriggerCount, triggerCount, len(pipelines))

	return nil
}

func (s *service) FireCronTriggers(ctx context.Context) error {

	// create event object
	ce := manifest.EstafetteCronEvent{
		Time: time.Now().UTC(),
	}
	e := manifest.EstafetteEvent{
		Cron: &ce,
	}

	log.Info().Msgf("[trigger:cron(%v)] Checking if triggers need to be fired...", ce.Time)

	pipelines, err := s.cockroachdbClient.GetCronTriggers(ctx)
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
					err := s.fireBuild(ctx, *p, t, e)
					if err != nil {
						log.Error().Err(err).Msgf("[trigger:cron(%v)] Failed starting build action'%v/%v/%v', branch '%v'", ce.Time, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
					}
				} else if t.ReleaseAction != nil {
					log.Info().Msgf("[trigger:cron(%v)] Firing release action '%v/%v/%v', target '%v', action '%v'...", ce.Time, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
					err := s.fireRelease(ctx, *p, t, e)
					if err != nil {
						log.Error().Err(err).Msgf("[trigger:cron(%v)] Failed starting release action '%v/%v/%v', target '%v', action '%v'", ce.Time, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
					}
				}
			}
		}
	}

	log.Info().Msgf("[trigger:cron(%v)] Fired %v out of %v triggers for %v pipelines", ce.Time, firedTriggerCount, triggerCount, len(pipelines))

	return nil
}

func (s *service) fireBuild(ctx context.Context, p contracts.Pipeline, t manifest.EstafetteTrigger, e manifest.EstafetteEvent) error {
	if t.BuildAction == nil {
		return fmt.Errorf("Trigger to fire does not have a 'builds' property, shouldn't get to here")
	}

	// get last build for branch defined in 'builds' section
	lastBuildForBranch, err := s.cockroachdbClient.GetLastPipelineBuildForBranch(ctx, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)

	if lastBuildForBranch == nil {
		return fmt.Errorf("There's no build for pipeline '%v/%v/%v' branch '%v', cannot trigger one", p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
	}

	// empty the build version so a new one gets created
	lastBuildForBranch.BuildVersion = ""

	// set event that triggers the build
	lastBuildForBranch.Events = []manifest.EstafetteEvent{e}

	_, err = s.CreateBuild(ctx, *lastBuildForBranch, true)
	if err != nil {
		return err
	}
	return nil
}

func (s *service) fireRelease(ctx context.Context, p contracts.Pipeline, t manifest.EstafetteTrigger, e manifest.EstafetteEvent) error {
	if t.ReleaseAction == nil {
		return fmt.Errorf("Trigger to fire does not have a 'releases' property, shouldn't get to here")
	}

	// determine version to release
	versionToRelease := p.BuildVersion

	switch t.ReleaseAction.Version {
	case "",
		"latest":
		versionToRelease = p.BuildVersion

	case "same":
		if e.Pipeline != nil {
			versionToRelease = e.Pipeline.BuildVersion
		} else if e.Release != nil {
			versionToRelease = e.Release.ReleaseVersion
		} else {
			log.Warn().Msgf("Can't get build version from event for 'same' release action version for pipeline %v, defaulting to pipeline version", p.GetFullRepoPath())
			versionToRelease = p.BuildVersion
		}

	case "current":
		if t.ReleaseAction.Version == "current" {
			for _, rt := range p.ReleaseTargets {
				if rt.Name == t.ReleaseAction.Target {
					for _, ar := range rt.ActiveReleases {
						if ar.Action == t.ReleaseAction.Action {
							versionToRelease = ar.ReleaseVersion
							break
						}
					}
					break
				}
			}
		}

	default:
		versionToRelease = t.ReleaseAction.Version
	}

	_, err := s.CreateRelease(ctx, contracts.Release{
		Name:           t.ReleaseAction.Target,
		Action:         t.ReleaseAction.Action,
		RepoSource:     p.RepoSource,
		RepoOwner:      p.RepoOwner,
		RepoName:       p.RepoName,
		ReleaseVersion: versionToRelease,
		Events:         []manifest.EstafetteEvent{e},
	}, *p.ManifestObject, p.RepoBranch, p.RepoRevision, true)
	if err != nil {
		return err
	}
	return nil
}

func (s *service) getShortRepoSource(repoSource string) string {

	repoSourceArray := strings.Split(repoSource, ".")

	if len(repoSourceArray) <= 0 {
		return repoSource
	}

	return repoSourceArray[0]
}

func (s *service) getAuthenticatedRepositoryURL(ctx context.Context, repoSource, repoOwner, repoName string) (authenticatedRepositoryURL string, environmentVariableWithToken map[string]string, err error) {

	switch repoSource {
	case "github.com":
		var accessToken string
		accessToken, authenticatedRepositoryURL, err = s.githubJobVarsFunc(ctx, repoSource, repoOwner, repoName)
		if err != nil {
			return
		}
		environmentVariableWithToken = map[string]string{"ESTAFETTE_GITHUB_API_TOKEN": accessToken}
		return

	case "bitbucket.org":
		var accessToken string
		accessToken, authenticatedRepositoryURL, err = s.bitbucketJobVarsFunc(ctx, repoSource, repoOwner, repoName)
		if err != nil {
			return
		}
		environmentVariableWithToken = map[string]string{"ESTAFETTE_BITBUCKET_API_TOKEN": accessToken}
		return
	}

	return authenticatedRepositoryURL, environmentVariableWithToken, fmt.Errorf("Source %v not supported for generating authenticated repository url", repoSource)
}

func (s *service) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {

	nrOfGoroutines := 1
	if s.apiServerConfig.WriteLogToCloudStorage() {
		nrOfGoroutines++
	}
	var wg sync.WaitGroup
	wg.Add(nrOfGoroutines)

	errors := make(chan error, nrOfGoroutines)

	go func(wg *sync.WaitGroup, ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) {
		defer wg.Done()

		shortFromRepoSource := s.getShortRepoSource(fromRepoSource)
		shortToRepoSource := s.getShortRepoSource(toRepoSource)

		err := s.cockroachdbClient.Rename(ctx, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName)
		if err != nil {
			errors <- err
		}
	}(&wg, ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)

	if s.apiServerConfig.WriteLogToCloudStorage() {
		go func(wg *sync.WaitGroup, ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) {
			defer wg.Done()

			err := s.cloudStorageClient.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
			if err != nil {
				errors <- err
			}
		}(&wg, ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
	}

	wg.Wait()

	close(errors)
	for e := range errors {
		return e
	}

	return nil
}

func (s *service) UpdateBuildStatus(ctx context.Context, ciBuilderEvent builderapi.CiBuilderEvent) (err error) {

	log.Debug().Interface("ciBuilderEvent", ciBuilderEvent).Msgf("UpdateBuildStatus executing...")

	if ciBuilderEvent.BuildStatus != "" && ciBuilderEvent.ReleaseID != "" {

		releaseID, err := strconv.Atoi(ciBuilderEvent.ReleaseID)
		if err != nil {
			return err
		}

		log.Debug().Msgf("Converted release id %v", releaseID)

		err = s.FinishRelease(ctx, ciBuilderEvent.RepoSource, ciBuilderEvent.RepoOwner, ciBuilderEvent.RepoName, releaseID, ciBuilderEvent.BuildStatus)
		if err != nil {
			return err
		}

		log.Debug().Msgf("Updated release status for job %v to %v", ciBuilderEvent.JobName, ciBuilderEvent.BuildStatus)

		return err

	} else if ciBuilderEvent.BuildStatus != "" && ciBuilderEvent.BuildID != "" {

		buildID, err := strconv.Atoi(ciBuilderEvent.BuildID)
		if err != nil {
			return err
		}

		log.Debug().Msgf("Converted build id %v", buildID)

		err = s.FinishBuild(ctx, ciBuilderEvent.RepoSource, ciBuilderEvent.RepoOwner, ciBuilderEvent.RepoName, buildID, ciBuilderEvent.BuildStatus)
		if err != nil {
			return err
		}

		log.Debug().Msgf("Updated build status for job %v to %v", ciBuilderEvent.JobName, ciBuilderEvent.BuildStatus)

		return err
	}

	return fmt.Errorf("CiBuilderEvent has invalid state, not updating build status")
}

func (s *service) UpdateJobResources(ctx context.Context, ciBuilderEvent builderapi.CiBuilderEvent) (err error) {

	log.Info().Msgf("Updating job resources for pod %v", ciBuilderEvent.PodName)

	if ciBuilderEvent.PodName != "" {

		s.prometheusClient.AwaitScrapeInterval(ctx)

		maxCPU, err := s.prometheusClient.GetMaxCPUByPodName(ctx, ciBuilderEvent.PodName)
		if err != nil {
			return err
		}

		log.Info().Msgf("Max cpu usage for pod %v is %v", ciBuilderEvent.PodName, maxCPU)

		maxMemory, err := s.prometheusClient.GetMaxMemoryByPodName(ctx, ciBuilderEvent.PodName)
		if err != nil {
			return err
		}

		log.Info().Msgf("Max memory usage for pod %v is %v", ciBuilderEvent.PodName, maxMemory)

		jobResources := cockroachdb.JobResources{
			CPUMaxUsage:    maxCPU,
			MemoryMaxUsage: maxMemory,
		}

		if ciBuilderEvent.BuildStatus != "" && ciBuilderEvent.ReleaseID != "" {

			releaseID, err := strconv.Atoi(ciBuilderEvent.ReleaseID)
			if err != nil {
				return err
			}

			err = s.cockroachdbClient.UpdateReleaseResourceUtilization(ctx, ciBuilderEvent.RepoSource, ciBuilderEvent.RepoOwner, ciBuilderEvent.RepoName, releaseID, jobResources)
			if err != nil {
				return err
			}
		} else if ciBuilderEvent.BuildStatus != "" && ciBuilderEvent.BuildID != "" {

			buildID, err := strconv.Atoi(ciBuilderEvent.BuildID)
			if err != nil {
				return err
			}

			err = s.cockroachdbClient.UpdateBuildResourceUtilization(ctx, ciBuilderEvent.RepoSource, ciBuilderEvent.RepoOwner, ciBuilderEvent.RepoName, buildID, jobResources)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *service) getBuildLabels(build contracts.Build, hasValidManifest bool, mft manifest.EstafetteManifest, pipeline *contracts.Pipeline) []contracts.Label {
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

	return build.Labels
}

func (s *service) getBuildReleaseTargets(build contracts.Build, hasValidManifest bool, mft manifest.EstafetteManifest, pipeline *contracts.Pipeline) []contracts.ReleaseTarget {

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
	return build.ReleaseTargets
}

func (s *service) getBuildTriggers(build contracts.Build, hasValidManifest bool, mft manifest.EstafetteManifest, pipeline *contracts.Pipeline) []manifest.EstafetteTrigger {
	if len(build.Triggers) == 0 {
		if hasValidManifest {
			build.Triggers = mft.GetAllTriggers(build.RepoSource, build.RepoOwner, build.RepoName)
		} else if pipeline != nil {
			log.Debug().Msgf("Copying previous release targets for pipeline %v/%v/%v, because current manifest is invalid...", build.RepoSource, build.RepoOwner, build.RepoName)
			build.Triggers = pipeline.Triggers
		}
	}

	return build.Triggers
}

func (s *service) getBuildAutoIncrement(ctx context.Context, build contracts.Build, shortRepoSource string, hasValidManifest bool, mft manifest.EstafetteManifest, pipeline *contracts.Pipeline) (autoincrement int, err error) {

	// get or set autoincrement and build version
	autoincrement = 0
	if build.BuildVersion == "" {
		// get autoincrement number
		autoincrement, err = s.cockroachdbClient.GetAutoIncrement(ctx, shortRepoSource, build.RepoOwner, build.RepoName)
		if err != nil {
			return autoincrement, err
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

	return autoincrement, nil
}

func (s *service) getReleaseAutoIncrement(ctx context.Context, release contracts.Release, mft manifest.EstafetteManifest) (autoincrement int) {

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

	return autoincrement
}

func (s *service) getBuildJobResources(ctx context.Context, build contracts.Build) cockroachdb.JobResources {
	// define resource request and limit values to fit reasonably well inside a n1-standard-8 (8 vCPUs, 30 GB memory) machine
	jobResources := cockroachdb.JobResources{
		CPURequest:    s.jobsConfig.MaxCPUCores,
		CPULimit:      s.jobsConfig.MaxCPUCores,
		MemoryRequest: s.jobsConfig.MaxMemoryBytes,
		MemoryLimit:   s.jobsConfig.MaxMemoryBytes,
	}

	// get max usage from previous builds
	measuredResources, nrRecords, err := s.cockroachdbClient.GetPipelineBuildMaxResourceUtilization(ctx, build.RepoSource, build.RepoOwner, build.RepoName, 25)
	if err != nil {
		log.Warn().Err(err).Msgf("Failed retrieving max resource utilization for recent builds of %v/%v/%v, using defaults...", build.RepoSource, build.RepoOwner, build.RepoName)
	} else if nrRecords < 5 {
		log.Info().Msgf("Retrieved max resource utilization for recent builds of %v/%v/%v only has %v records, using defaults...", build.RepoSource, build.RepoOwner, build.RepoName, nrRecords)
	} else {
		log.Info().Msgf("Retrieved max resource utilization for recent builds of %v/%v/%v, checking if they are within lower and upper bound...", build.RepoSource, build.RepoOwner, build.RepoName)

		// only override cpu and memory request values if measured values are within min and max
		if measuredResources.CPUMaxUsage > 0 {
			if measuredResources.CPUMaxUsage*s.jobsConfig.CPURequestRatio <= s.jobsConfig.MinCPUCores {
				jobResources.CPURequest = s.jobsConfig.MinCPUCores
			} else if measuredResources.CPUMaxUsage*s.jobsConfig.CPURequestRatio >= s.jobsConfig.MaxCPUCores {
				jobResources.CPURequest = s.jobsConfig.MaxCPUCores
			} else {
				jobResources.CPURequest = measuredResources.CPUMaxUsage * s.jobsConfig.CPURequestRatio
			}
		}

		if measuredResources.MemoryMaxUsage > 0 {
			if measuredResources.MemoryMaxUsage*s.jobsConfig.MemoryRequestRatio <= s.jobsConfig.MinMemoryBytes {
				jobResources.MemoryRequest = s.jobsConfig.MinMemoryBytes
			} else if measuredResources.MemoryMaxUsage*s.jobsConfig.MemoryRequestRatio >= s.jobsConfig.MaxMemoryBytes {
				jobResources.MemoryRequest = s.jobsConfig.MaxMemoryBytes
			} else {
				jobResources.MemoryRequest = measuredResources.MemoryMaxUsage * s.jobsConfig.MemoryRequestRatio
			}
		}
	}

	return jobResources
}

func (s *service) getReleaseJobResources(ctx context.Context, release contracts.Release) cockroachdb.JobResources {

	// define resource request and limit values to fit reasonably well inside a n1-standard-8 (8 vCPUs, 30 GB memory) machine
	jobResources := cockroachdb.JobResources{
		CPURequest:    s.jobsConfig.MaxCPUCores,
		CPULimit:      s.jobsConfig.MaxCPUCores,
		MemoryRequest: s.jobsConfig.MaxMemoryBytes,
		MemoryLimit:   s.jobsConfig.MaxMemoryBytes,
	}

	// get max usage from previous releases
	measuredResources, nrRecords, err := s.cockroachdbClient.GetPipelineReleaseMaxResourceUtilization(ctx, release.RepoSource, release.RepoOwner, release.RepoName, release.Name, 25)
	if err != nil {
		log.Warn().Err(err).Msgf("Failed retrieving max resource utilization for recent releases of %v/%v/%v target %v, using defaults...", release.RepoSource, release.RepoOwner, release.RepoName, release.Name)
	} else if nrRecords < 5 {
		log.Info().Msgf("Retrieved max resource utilization for recent releases of %v/%v/%v target %v only has %v records, using defaults...", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, nrRecords)
	} else {
		log.Info().Msgf("Retrieved max resource utilization for recent releases of %v/%v/%v target %v, checking if they are within lower and upper bound...", release.RepoSource, release.RepoOwner, release.RepoName, release.Name)

		// only override cpu and memory request values if measured values are within min and max
		if measuredResources.CPUMaxUsage > 0 {
			if measuredResources.CPUMaxUsage*s.jobsConfig.CPURequestRatio <= s.jobsConfig.MinCPUCores {
				jobResources.CPURequest = s.jobsConfig.MinCPUCores
			} else if measuredResources.CPUMaxUsage*s.jobsConfig.CPURequestRatio >= s.jobsConfig.MaxCPUCores {
				jobResources.CPURequest = s.jobsConfig.MaxCPUCores
			} else {
				jobResources.CPURequest = measuredResources.CPUMaxUsage * s.jobsConfig.CPURequestRatio
			}
		}

		if measuredResources.MemoryMaxUsage > 0 {
			if measuredResources.MemoryMaxUsage*s.jobsConfig.MemoryRequestRatio <= s.jobsConfig.MinMemoryBytes {
				jobResources.MemoryRequest = s.jobsConfig.MinMemoryBytes
			} else if measuredResources.MemoryMaxUsage*s.jobsConfig.MemoryRequestRatio >= s.jobsConfig.MaxMemoryBytes {
				jobResources.MemoryRequest = s.jobsConfig.MaxMemoryBytes
			} else {
				jobResources.MemoryRequest = measuredResources.MemoryMaxUsage * s.jobsConfig.MemoryRequestRatio
			}
		}
	}

	return jobResources
}
