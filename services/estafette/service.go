package estafette

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/estafette/estafette-ci-api/api"
	"github.com/estafette/estafette-ci-api/clients/bitbucketapi"
	"github.com/estafette/estafette-ci-api/clients/builderapi"
	"github.com/estafette/estafette-ci-api/clients/cloudsourceapi"
	"github.com/estafette/estafette-ci-api/clients/cloudstorage"
	"github.com/estafette/estafette-ci-api/clients/cockroachdb"
	"github.com/estafette/estafette-ci-api/clients/githubapi"
	"github.com/estafette/estafette-ci-api/clients/prometheus"
	contracts "github.com/estafette/estafette-ci-contracts"
	crypt "github.com/estafette/estafette-ci-crypt"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v2"
)

var (
	ErrNoBuildCreated   = errors.New("No build is created")
	ErrNoReleaseCreated = errors.New("No release is created")
	ErrNoBotCreated     = errors.New("No bot is created")
)

// Service encapsulates build and release creation and re-triggering
//go:generate mockgen -package=estafette -destination ./mock.go -source=service.go
type Service interface {
	CreateBuild(ctx context.Context, build contracts.Build, waitForJobToStart bool) (b *contracts.Build, err error)
	FinishBuild(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus contracts.Status) (err error)
	CreateRelease(ctx context.Context, release contracts.Release, mft manifest.EstafetteManifest, repoBranch, repoRevision string, waitForJobToStart bool) (r *contracts.Release, err error)
	FinishRelease(ctx context.Context, repoSource, repoOwner, repoName string, releaseID int, releaseStatus contracts.Status) (err error)
	CreateBot(ctx context.Context, bot contracts.Bot, mft manifest.EstafetteManifest, repoBranch, repoRevision string, waitForJobToStart bool) (b *contracts.Bot, err error)
	FinishBot(ctx context.Context, repoSource, repoOwner, repoName string, botID int, botStatus contracts.Status) (err error)
	FireGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (err error)
	FirePipelineTriggers(ctx context.Context, build contracts.Build, event string) (err error)
	FireReleaseTriggers(ctx context.Context, release contracts.Release, event string) (err error)
	FirePubSubTriggers(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) (err error)
	FireCronTriggers(ctx context.Context) (err error)
	Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
	Archive(ctx context.Context, repoSource, repoOwner, repoName string) (err error)
	Unarchive(ctx context.Context, repoSource, repoOwner, repoName string) (err error)
	UpdateBuildStatus(ctx context.Context, event builderapi.CiBuilderEvent) (err error)
	UpdateJobResources(ctx context.Context, event builderapi.CiBuilderEvent) (err error)
	SubscribeToGitEventsTopic(ctx context.Context, gitEventTopic *api.GitEventTopic)
	GetEventsForJobEnvvars(ctx context.Context, triggers []manifest.EstafetteTrigger, events []manifest.EstafetteEvent) (triggersAsEvents []manifest.EstafetteEvent, err error)
}

// NewService returns a new estafette.Service
func NewService(config *api.APIConfig, cockroachdbClient cockroachdb.Client, secretHelper crypt.SecretHelper, prometheusClient prometheus.Client, cloudStorageClient cloudstorage.Client, builderapiClient builderapi.Client, githubJobVarsFunc func(context.Context, string, string, string) (string, string, error), bitbucketJobVarsFunc func(context.Context, string, string, string) (string, string, error), cloudsourceJobVarsFunc func(context.Context, string, string, string) (string, string, error)) Service {

	return &service{
		config:                 config,
		cockroachdbClient:      cockroachdbClient,
		secretHelper:           secretHelper,
		prometheusClient:       prometheusClient,
		cloudStorageClient:     cloudStorageClient,
		builderapiClient:       builderapiClient,
		githubJobVarsFunc:      githubJobVarsFunc,
		bitbucketJobVarsFunc:   bitbucketJobVarsFunc,
		cloudsourceJobVarsFunc: cloudsourceJobVarsFunc,
		triggerConcurrency:     5,
	}
}

type service struct {
	config                 *api.APIConfig
	cockroachdbClient      cockroachdb.Client
	secretHelper           crypt.SecretHelper
	prometheusClient       prometheus.Client
	cloudStorageClient     cloudstorage.Client
	builderapiClient       builderapi.Client
	githubJobVarsFunc      func(context.Context, string, string, string) (string, string, error)
	bitbucketJobVarsFunc   func(context.Context, string, string, string) (string, string, error)
	cloudsourceJobVarsFunc func(context.Context, string, string, string) (string, string, error)
	triggerConcurrency     int
}

func (s *service) CreateBuild(ctx context.Context, build contracts.Build, waitForJobToStart bool) (createdBuild *contracts.Build, err error) {

	// validate manifest
	mft, manifestError := manifest.ReadManifest(s.config.ManifestPreferences, build.Manifest, true)
	hasValidManifest := manifestError == nil

	// check if there's secrets restricted to other pipelines
	var invalidSecrets []string
	var invalidSecretsErr error
	if hasValidManifest {
		invalidSecrets, invalidSecretsErr = s.secretHelper.GetInvalidRestrictedSecrets(build.Manifest, build.GetFullRepoPath())
	}

	// set builder track
	builderTrack := "stable"
	builderOperatingSystem := manifest.OperatingSystemLinux
	if hasValidManifest {
		builderTrack = mft.Builder.Track
		builderOperatingSystem = mft.Builder.OperatingSystem
	}

	// get short version of repo source
	shortRepoSource := s.getShortRepoSource(build.RepoSource)

	// set build status
	buildStatus := contracts.StatusFailed
	if hasValidManifest && invalidSecretsErr == nil {
		buildStatus = contracts.StatusPending
	}

	// inject build stages
	if hasValidManifest {
		mft, err = api.InjectStages(s.config, mft, builderTrack, shortRepoSource, build.RepoBranch, s.supportsBuildStatus(build.RepoSource))
		if err != nil {
			return nil, errors.Wrapf(err, "Failed injecting build stages for pipeline %v/%v/%v and revision %v", build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision)
		}

		// inject any configured commands
		mft = api.InjectCommands(s.config, mft)
	}

	// retrieve pipeline if already exists to get counter value
	pipeline, _ := s.cockroachdbClient.GetPipeline(ctx, build.RepoSource, build.RepoOwner, build.RepoName, map[api.FilterType][]string{}, false)

	// get counter and update build labels if manifest is invalid
	currentCounter, build, err := s.getBuildCounter(ctx, build, shortRepoSource, hasValidManifest, mft, pipeline)
	if err != nil {
		return nil, err
	}

	maxCounter := currentCounter
	maxCounterCurrentBranch := currentCounter
	if pipeline != nil {
		// get max counter for pipeline
		mc := s.getVersionCounter(ctx, pipeline.BuildVersion, mft)
		if mc > maxCounter {
			maxCounter = mc
		}

		// get max counter for same branch
		lastBuildsForBranch, _ := s.cockroachdbClient.GetPipelineBuilds(ctx, pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName, 1, 10, map[api.FilterType][]string{api.FilterBranch: []string{pipeline.RepoBranch}}, []api.OrderField{}, false)
		if lastBuildsForBranch != nil && len(lastBuildsForBranch) == 1 {
			mc := s.getVersionCounter(ctx, lastBuildsForBranch[0].BuildVersion, mft)
			if mc > maxCounterCurrentBranch {
				maxCounterCurrentBranch = mc
			}
		}
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
		Groups:         build.Groups,
		Organizations:  build.Organizations,
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

	triggeredByEvents, err := s.GetEventsForJobEnvvars(ctx, build.Triggers, build.Events)
	if err != nil {
		return
	}

	// define ci builder params
	ciBuilderParams := builderapi.CiBuilderParams{
		JobType:                 builderapi.JobTypeBuild,
		RepoSource:              build.RepoSource,
		RepoOwner:               build.RepoOwner,
		RepoName:                build.RepoName,
		RepoURL:                 authenticatedRepositoryURL,
		RepoBranch:              build.RepoBranch,
		RepoRevision:            build.RepoRevision,
		EnvironmentVariables:    environmentVariableWithToken,
		Track:                   builderTrack,
		OperatingSystem:         builderOperatingSystem,
		CurrentCounter:          currentCounter,
		MaxCounter:              maxCounter,
		MaxCounterCurrentBranch: maxCounterCurrentBranch,
		VersionNumber:           build.BuildVersion,
		Manifest:                mft,
		BuildID:                 buildID,
		TriggeredByEvents:       triggeredByEvents,
		JobResources:            jobResources,
	}

	// create ci builder job
	if hasValidManifest && invalidSecretsErr == nil {
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
	} else if invalidSecretsErr != nil {
		log.Debug().Interface("invalidSecrets", invalidSecrets).Msgf("Pipeline %v/%v/%v revision %v with build id %v has invalid secrets, storing log...", build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision, build.ID)
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
					Step:         "validate-secrets",
					ExitCode:     1,
					Status:       contracts.LogStatusFailed,
					AutoInjected: true,
					RunIndex:     0,
					LogLines: []contracts.BuildLogLine{
						contracts.BuildLogLine{
							LineNumber: 1,
							Timestamp:  time.Now().UTC(),
							StreamType: "stderr",
							Text:       "The manifest has secrets, restricted to other pipelines; please recreate the following secrets through the pipeline's secrets tab:",
						},
					},
				},
			},
		}

		for i, is := range invalidSecrets {
			buildLog.Steps[0].LogLines = append(buildLog.Steps[0].LogLines, contracts.BuildLogLine{
				LineNumber: i + 1,
				Timestamp:  time.Now().UTC(),
				StreamType: "stdout",
				Text:       is,
			})
		}

		insertedBuildLog, err := s.cockroachdbClient.InsertBuildLog(ctx, buildLog, s.config.APIServer.WriteLogToDatabase())
		if err != nil {
			log.Warn().Err(err).Msgf("Failed inserting build log for invalid manifest")
		}

		if s.config.APIServer.WriteLogToCloudStorage() {
			err = s.cloudStorageClient.InsertBuildLog(ctx, insertedBuildLog)
			if err != nil {
				log.Warn().Err(err).Msgf("Failed inserting build log into cloud storage for invalid manifest")
			}
		}
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
					Status:       contracts.LogStatusFailed,
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

		insertedBuildLog, err := s.cockroachdbClient.InsertBuildLog(ctx, buildLog, s.config.APIServer.WriteLogToDatabase())
		if err != nil {
			log.Warn().Err(err).Msgf("Failed inserting build log for invalid manifest")
		}

		if s.config.APIServer.WriteLogToCloudStorage() {
			err = s.cloudStorageClient.InsertBuildLog(ctx, insertedBuildLog)
			if err != nil {
				log.Warn().Err(err).Msgf("Failed inserting build log into cloud storage for invalid manifest")
			}
		}
	}

	return
}

func (s *service) FinishBuild(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus contracts.Status) error {

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

	// create deep copy to ensure no properties are shared through a pointer
	mft = mft.DeepCopy()

	// check if there's secrets restricted to other pipelines
	manifestBytes, err := yaml.Marshal(mft)
	invalidSecrets, invalidSecretsErr := s.secretHelper.GetInvalidRestrictedSecrets(string(manifestBytes), release.GetFullRepoPath())

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
	releaseStatus := contracts.StatusFailed
	if invalidSecretsErr == nil {
		releaseStatus = contracts.StatusPending
	}

	// inject release stages
	mft, err = api.InjectStages(s.config, mft, builderTrack, shortRepoSource, repoBranch, s.supportsBuildStatus(release.RepoSource))
	if err != nil {
		log.Error().Err(err).Msgf("Failed injecting build stages for release to %v of pipeline %v/%v/%v version %v", release.Name, release.RepoSource, release.RepoOwner, release.RepoName, release.ReleaseVersion)
		return
	}

	// inject any configured commands
	mft = api.InjectCommands(s.config, mft)

	// get autoincrement from release version
	currentCounter := s.getVersionCounter(ctx, release.ReleaseVersion, mft)

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
		Groups:         release.Groups,
		Organizations:  release.Organizations,
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

	maxCounter := currentCounter
	maxCounterCurrentBranch := currentCounter

	triggeredByEvents := release.Events

	// retrieve pipeline if already exists to get counter value
	pipeline, _ := s.cockroachdbClient.GetPipeline(ctx, release.RepoSource, release.RepoOwner, release.RepoName, map[api.FilterType][]string{}, false)
	if pipeline != nil {
		// get max counter for pipeline
		mc := s.getVersionCounter(ctx, pipeline.BuildVersion, mft)
		if mc > maxCounter {
			maxCounter = mc
		}

		// get max counter for same branch
		lastBuildsForBranch, _ := s.cockroachdbClient.GetPipelineBuilds(ctx, pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName, 1, 10, map[api.FilterType][]string{api.FilterBranch: []string{repoBranch}}, []api.OrderField{}, false)
		if lastBuildsForBranch != nil && len(lastBuildsForBranch) == 1 {
			mc := s.getVersionCounter(ctx, lastBuildsForBranch[0].BuildVersion, mft)
			if mc > maxCounterCurrentBranch {
				maxCounterCurrentBranch = mc
			}
		}

		// get events and add non-firing triggers as events
		triggeredByEvents, err = s.GetEventsForJobEnvvars(ctx, pipeline.Triggers, release.Events)
		if err != nil {
			return
		}
	}

	// define ci builder params
	ciBuilderParams := builderapi.CiBuilderParams{
		JobType:                 builderapi.JobTypeRelease,
		RepoSource:              release.RepoSource,
		RepoOwner:               release.RepoOwner,
		RepoName:                release.RepoName,
		RepoURL:                 authenticatedRepositoryURL,
		RepoBranch:              repoBranch,
		RepoRevision:            repoRevision,
		EnvironmentVariables:    environmentVariableWithToken,
		Track:                   builderTrack,
		OperatingSystem:         builderOperatingSystem,
		CurrentCounter:          currentCounter,
		MaxCounter:              maxCounter,
		MaxCounterCurrentBranch: maxCounterCurrentBranch,
		VersionNumber:           release.ReleaseVersion,
		Manifest:                mft,
		ReleaseName:             release.Name,
		ReleaseAction:           release.Action,
		ReleaseID:               insertedReleaseID,
		ReleaseTriggeredBy:      triggeredBy,
		BuildID:                 0,
		TriggeredByEvents:       triggeredByEvents,
		JobResources:            jobResources,
	}

	if invalidSecretsErr == nil {
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
	} else {
		// store log with manifest unmarshalling error
		releaseLog := contracts.ReleaseLog{
			ReleaseID:  createdRelease.ID,
			RepoSource: createdRelease.RepoSource,
			RepoOwner:  createdRelease.RepoOwner,
			RepoName:   createdRelease.RepoName,
			Steps: []*contracts.BuildLogStep{
				&contracts.BuildLogStep{
					Step:         "validate-secrets",
					ExitCode:     1,
					Status:       contracts.LogStatusFailed,
					AutoInjected: true,
					RunIndex:     0,
					LogLines: []contracts.BuildLogLine{
						contracts.BuildLogLine{
							LineNumber: 1,
							Timestamp:  time.Now().UTC(),
							StreamType: "stderr",
							Text:       "The manifest has secrets, restricted to other pipelines; please recreate the following secrets through the pipeline's secrets tab:",
						},
					},
				},
			},
		}

		for i, is := range invalidSecrets {
			releaseLog.Steps[0].LogLines = append(releaseLog.Steps[0].LogLines, contracts.BuildLogLine{
				LineNumber: i + 1,
				Timestamp:  time.Now().UTC(),
				StreamType: "stdout",
				Text:       is,
			})
		}

		insertedReleaseLog, err := s.cockroachdbClient.InsertReleaseLog(ctx, releaseLog, s.config.APIServer.WriteLogToDatabase())
		if err != nil {
			log.Warn().Err(err).Msgf("Failed inserting release log for manifest with restricted secrets")
		}

		if s.config.APIServer.WriteLogToCloudStorage() {
			err = s.cloudStorageClient.InsertReleaseLog(ctx, insertedReleaseLog)
			if err != nil {
				log.Warn().Err(err).Msgf("Failed inserting release log into cloud storage for manifest with restricted secrets")
			}
		}
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

func (s *service) FinishRelease(ctx context.Context, repoSource, repoOwner, repoName string, releaseID int, releaseStatus contracts.Status) error {
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

func (s *service) CreateBot(ctx context.Context, bot contracts.Bot, mft manifest.EstafetteManifest, repoBranch, repoRevision string, waitForJobToStart bool) (createdBot *contracts.Bot, err error) {

	// create deep copy to ensure no properties are shared through a pointer
	mft = mft.DeepCopy()

	// check if there's secrets restricted to other pipelines
	manifestBytes, err := yaml.Marshal(mft)
	invalidSecrets, invalidSecretsErr := s.secretHelper.GetInvalidRestrictedSecrets(string(manifestBytes), bot.GetFullRepoPath())

	// set builder track
	builderTrack := mft.Builder.Track
	builderOperatingSystem := mft.Builder.OperatingSystem

	// get builder track override for release if exists
	for _, r := range mft.Bots {
		if r.Name == bot.Name {
			if r.Builder != nil {
				builderTrack = r.Builder.Track
				builderOperatingSystem = r.Builder.OperatingSystem
				break
			}
		}
	}

	// get short version of repo source
	shortRepoSource := s.getShortRepoSource(bot.RepoSource)

	// set bot status
	botStatus := contracts.StatusFailed
	if invalidSecretsErr == nil {
		botStatus = contracts.StatusPending
	}

	// inject release stages
	mft, err = api.InjectStages(s.config, mft, builderTrack, shortRepoSource, repoBranch, s.supportsBuildStatus(bot.RepoSource))
	if err != nil {
		log.Error().Err(err).Msgf("Failed injecting build stages for bot to %v of pipeline %v/%v/%v", bot.Name, bot.RepoSource, bot.RepoOwner, bot.RepoName)
		return
	}

	// inject any configured commands
	mft = api.InjectCommands(s.config, mft)

	// get authenticated url
	authenticatedRepositoryURL, environmentVariableWithToken, err := s.getAuthenticatedRepositoryURL(ctx, bot.RepoSource, bot.RepoOwner, bot.RepoName)
	if err != nil {
		return
	}

	jobResources := s.getBotJobResources(ctx, bot)

	// create release in database
	createdBot, err = s.cockroachdbClient.InsertBot(ctx, contracts.Bot{
		Name:          bot.Name,
		RepoSource:    bot.RepoSource,
		RepoOwner:     bot.RepoOwner,
		RepoName:      bot.RepoName,
		BotStatus:     botStatus,
		Events:        bot.Events,
		Groups:        bot.Groups,
		Organizations: bot.Organizations,
	}, jobResources)
	if err != nil {
		return
	}
	if createdBot == nil {
		return nil, ErrNoBotCreated
	}

	insertedReleaseID, err := strconv.Atoi(createdBot.ID)
	if err != nil {
		return
	}

	// get triggered by from events
	triggeredBy := ""
	if len(bot.Events) > 0 {
		for _, e := range bot.Events {
			if e.Manual != nil {
				triggeredBy = e.Manual.UserID
			}
		}
	}

	triggeredByEvents := bot.Events

	// define ci builder params
	ciBuilderParams := builderapi.CiBuilderParams{
		JobType:              builderapi.JobTypeBot,
		RepoSource:           bot.RepoSource,
		RepoOwner:            bot.RepoOwner,
		RepoName:             bot.RepoName,
		RepoURL:              authenticatedRepositoryURL,
		RepoBranch:           repoBranch,
		RepoRevision:         repoRevision,
		EnvironmentVariables: environmentVariableWithToken,
		Track:                builderTrack,
		OperatingSystem:      builderOperatingSystem,
		Manifest:             mft,
		BotName:              bot.Name,
		BotID:                insertedReleaseID,
		BotTriggeredBy:       triggeredBy,
		BuildID:              0,
		TriggeredByEvents:    triggeredByEvents,
		JobResources:         jobResources,
	}

	if invalidSecretsErr == nil {
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
	} else {
		// store log with manifest unmarshalling error
		botLog := contracts.BotLog{
			BotID:      createdBot.ID,
			RepoSource: createdBot.RepoSource,
			RepoOwner:  createdBot.RepoOwner,
			RepoName:   createdBot.RepoName,
			Steps: []*contracts.BuildLogStep{
				&contracts.BuildLogStep{
					Step:         "validate-secrets",
					ExitCode:     1,
					Status:       contracts.LogStatusFailed,
					AutoInjected: true,
					RunIndex:     0,
					LogLines: []contracts.BuildLogLine{
						contracts.BuildLogLine{
							LineNumber: 1,
							Timestamp:  time.Now().UTC(),
							StreamType: "stderr",
							Text:       "The manifest has secrets, restricted to other pipelines; please recreate the following secrets through the pipeline's secrets tab:",
						},
					},
				},
			},
		}

		for i, is := range invalidSecrets {
			botLog.Steps[0].LogLines = append(botLog.Steps[0].LogLines, contracts.BuildLogLine{
				LineNumber: i + 1,
				Timestamp:  time.Now().UTC(),
				StreamType: "stdout",
				Text:       is,
			})
		}

		insertedBotLog, err := s.cockroachdbClient.InsertBotLog(ctx, botLog, s.config.APIServer.WriteLogToDatabase())
		if err != nil {
			log.Warn().Err(err).Msgf("Failed inserting bot log for manifest with restricted secrets")
		}

		if s.config.APIServer.WriteLogToCloudStorage() {
			err = s.cloudStorageClient.InsertBotLog(ctx, insertedBotLog)
			if err != nil {
				log.Warn().Err(err).Msgf("Failed inserting bot log into cloud storage for manifest with restricted secrets")
			}
		}
	}

	return
}

func (s *service) FinishBot(ctx context.Context, repoSource, repoOwner, repoName string, botID int, botStatus contracts.Status) error {
	err := s.cockroachdbClient.UpdateBotStatus(ctx, repoSource, repoOwner, repoName, botID, botStatus)
	if err != nil {
		return err
	}

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
		Fired: true,
		Git:   &gitEvent,
	}

	triggerCount := 0
	firedTriggerCount := 0

	// http://jmoiron.net/blog/limiting-concurrency-in-go/
	semaphore := make(chan bool, s.triggerConcurrency)

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

				// try to fill semaphore up to it's full size otherwise wait for a routine to finish
				semaphore <- true

				go func(ctx context.Context, p *contracts.Pipeline, t manifest.EstafetteTrigger, e manifest.EstafetteEvent) {
					// lower semaphore once the routine's finished, making room for another one to start
					defer func() { <-semaphore }()

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
				}(ctx, p, t, e)
			}
		}
	}

	// try to fill semaphore up to it's full size which only succeeds if all routines have finished
	for i := 0; i < cap(semaphore); i++ {
		semaphore <- true
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
		Status:       string(build.BuildStatus),
		Event:        event,
	}
	e := manifest.EstafetteEvent{
		Fired:    true,
		Pipeline: &pe,
	}

	triggerCount := 0
	firedTriggerCount := 0

	// http://jmoiron.net/blog/limiting-concurrency-in-go/
	semaphore := make(chan bool, s.triggerConcurrency)

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

				// try to fill semaphore up to it's full size otherwise wait for a routine to finish
				semaphore <- true

				go func(ctx context.Context, p *contracts.Pipeline, t manifest.EstafetteTrigger, e manifest.EstafetteEvent) {
					// lower semaphore once the routine's finished, making room for another one to start
					defer func() { <-semaphore }()

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
				}(ctx, p, t, e)
			}
		}
	}

	// try to fill semaphore up to it's full size which only succeeds if all routines have finished
	for i := 0; i < cap(semaphore); i++ {
		semaphore <- true
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
		Status:         string(release.ReleaseStatus),
		Event:          event,
	}
	e := manifest.EstafetteEvent{
		Fired:   true,
		Release: &re,
	}

	triggerCount := 0
	firedTriggerCount := 0

	// http://jmoiron.net/blog/limiting-concurrency-in-go/
	semaphore := make(chan bool, s.triggerConcurrency)

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

				// try to fill semaphore up to it's full size otherwise wait for a routine to finish
				semaphore <- true

				go func(ctx context.Context, p *contracts.Pipeline, t manifest.EstafetteTrigger, e manifest.EstafetteEvent) {
					// lower semaphore once the routine's finished, making room for another one to start
					defer func() { <-semaphore }()

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
				}(ctx, p, t, e)
			}
		}
	}

	// try to fill semaphore up to it's full size which only succeeds if all routines have finished
	for i := 0; i < cap(semaphore); i++ {
		semaphore <- true
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
		Fired:  true,
		PubSub: &pubsubEvent,
	}

	triggerCount := 0
	firedTriggerCount := 0

	// http://jmoiron.net/blog/limiting-concurrency-in-go/
	semaphore := make(chan bool, s.triggerConcurrency)

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

				// try to fill semaphore up to it's full size otherwise wait for a routine to finish
				semaphore <- true

				go func(ctx context.Context, p *contracts.Pipeline, t manifest.EstafetteTrigger, e manifest.EstafetteEvent) {
					// lower semaphore once the routine's finished, making room for another one to start
					defer func() { <-semaphore }()

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
				}(ctx, p, t, e)
			}
		}
	}

	// try to fill semaphore up to it's full size which only succeeds if all routines have finished
	for i := 0; i < cap(semaphore); i++ {
		semaphore <- true
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
		Fired: true,
		Cron:  &ce,
	}

	log.Info().Msgf("[trigger:cron(%v)] Checking if triggers need to be fired...", ce.Time)

	pipelines, err := s.cockroachdbClient.GetCronTriggers(ctx)
	if err != nil {
		return err
	}

	triggerCount := 0
	firedTriggerCount := 0

	// http://jmoiron.net/blog/limiting-concurrency-in-go/

	semaphore := make(chan bool, s.triggerConcurrency)

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

				// try to fill semaphore up to it's full size otherwise wait for a routine to finish
				semaphore <- true

				go func(ctx context.Context, p *contracts.Pipeline, t manifest.EstafetteTrigger, e manifest.EstafetteEvent) {
					// lower semaphore once the routine's finished, making room for another one to start
					defer func() { <-semaphore }()

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
				}(ctx, p, t, e)
			}
		}
	}

	// try to fill semaphore up to it's full size which only succeeds if all routines have finished
	for i := 0; i < cap(semaphore); i++ {
		semaphore <- true
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

	// get repobranch and reporevision for actually released build if it's not the most recent build that gets released
	repoBranch := p.RepoBranch
	repoRevision := p.RepoRevision
	mft := p.ManifestObject
	if versionToRelease != p.BuildVersion {
		succeededBuilds, err := s.cockroachdbClient.GetPipelineBuildsByVersion(ctx, p.RepoSource, p.RepoOwner, p.RepoName, versionToRelease, []contracts.Status{contracts.StatusSucceeded}, 1, false)
		if err != nil {
			return err
		}
		if len(succeededBuilds) == 0 {
			return fmt.Errorf("No succeeded builds have been found to fire a release trigger")
		}
		repoBranch = succeededBuilds[0].RepoBranch
		repoRevision = succeededBuilds[0].RepoRevision
		mft = succeededBuilds[0].ManifestObject
	}

	_, err := s.CreateRelease(ctx, contracts.Release{
		Name:           t.ReleaseAction.Target,
		Action:         t.ReleaseAction.Action,
		RepoSource:     p.RepoSource,
		RepoOwner:      p.RepoOwner,
		RepoName:       p.RepoName,
		ReleaseVersion: versionToRelease,
		Events:         []manifest.EstafetteEvent{e},
	}, *mft, repoBranch, repoRevision, true)
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

	switch {
	case githubapi.IsRepoSourceGithub(repoSource):
		var accessToken string
		accessToken, authenticatedRepositoryURL, err = s.githubJobVarsFunc(ctx, repoSource, repoOwner, repoName)
		if err != nil {
			return
		}
		environmentVariableWithToken = map[string]string{"ESTAFETTE_GITHUB_API_TOKEN": accessToken}
		return

	case bitbucketapi.IsRepoSourceBitbucket(repoSource):
		var accessToken string
		accessToken, authenticatedRepositoryURL, err = s.bitbucketJobVarsFunc(ctx, repoSource, repoOwner, repoName)
		if err != nil {
			return
		}
		environmentVariableWithToken = map[string]string{"ESTAFETTE_BITBUCKET_API_TOKEN": accessToken}
		return

	case cloudsourceapi.IsRepoSourceCloudSource(repoSource):
		var accessToken string
		accessToken, authenticatedRepositoryURL, err = s.cloudsourceJobVarsFunc(ctx, repoSource, repoOwner, repoName)
		if err != nil {
			return
		}
		environmentVariableWithToken = map[string]string{"ESTAFETTE_CLOUDSOURCE_API_TOKEN": accessToken}
		return
	}

	return authenticatedRepositoryURL, environmentVariableWithToken, fmt.Errorf("Source %v not supported for generating authenticated repository url", repoSource)
}

func (s *service) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {

	nrOfGoroutines := 1
	if s.config.APIServer.WriteLogToCloudStorage() {
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

	if s.config.APIServer.WriteLogToCloudStorage() {
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
		log.Warn().Err(e).Msgf("Failure to rename pipeline logs from %v/%v/%v to %v/%v/%v", fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
	}
	for e := range errors {
		return e
	}

	return nil
}

func (s *service) Archive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	return s.cockroachdbClient.ArchiveComputedPipeline(ctx, repoSource, repoOwner, repoName)
}

func (s *service) Unarchive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	return s.cockroachdbClient.UnarchiveComputedPipeline(ctx, repoSource, repoOwner, repoName)
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

	} else if ciBuilderEvent.BuildStatus != contracts.StatusUnknown && ciBuilderEvent.BuildID != "" {

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

func (s *service) SubscribeToGitEventsTopic(ctx context.Context, gitEventTopic *api.GitEventTopic) {
	eventChannel := gitEventTopic.Subscribe("estafette.Service")
	for {
		message, ok := <-eventChannel
		if !ok {
			break
		}
		s.FireGitTriggers(message.Ctx, message.Event)
	}
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

func (s *service) getBuildCounter(ctx context.Context, build contracts.Build, shortRepoSource string, hasValidManifest bool, mft manifest.EstafetteManifest, pipeline *contracts.Pipeline) (counter int, updatedBuild contracts.Build, err error) {

	// get or set autoincrement and build version
	counter = 0
	if build.BuildVersion == "" {
		// get autoincrementing counter
		counter, err = s.cockroachdbClient.GetAutoIncrement(ctx, shortRepoSource, build.RepoOwner, build.RepoName)
		if err != nil {
			return counter, build, err
		}

		// set build version number
		if hasValidManifest {
			build.BuildVersion = mft.Version.Version(manifest.EstafetteVersionParams{
				AutoIncrement: counter,
				Branch:        build.RepoBranch,
				Revision:      build.RepoRevision,
			})
		} else if pipeline != nil {
			log.Debug().Msgf("Copying previous versioning for pipeline %v/%v/%v, because current manifest is invalid...", build.RepoSource, build.RepoOwner, build.RepoName)
			previousManifest, err := manifest.ReadManifest(s.config.ManifestPreferences, build.Manifest, false)
			if err != nil {
				build.BuildVersion = previousManifest.Version.Version(manifest.EstafetteVersionParams{
					AutoIncrement: counter,
					Branch:        build.RepoBranch,
					Revision:      build.RepoRevision,
				})
			} else {
				log.Warn().Msgf("Not using previous versioning for pipeline %v/%v/%v, because its manifest is also invalid...", build.RepoSource, build.RepoOwner, build.RepoName)
				build.BuildVersion = strconv.Itoa(counter)
			}
		} else {
			// set build version to autoincrement so there's at least a version in the db and gui
			build.BuildVersion = strconv.Itoa(counter)
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

		counter, err = strconv.Atoi(autoincrementCandidate)
		if err != nil {
			log.Warn().Err(err).Str("buildversion", build.BuildVersion).Msgf("Failed extracting autoincrement from build version %v for pipeline %v/%v/%v revision %v", build.BuildVersion, build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision)
		}
	}

	return counter, build, nil
}

func (s *service) getVersionCounter(ctx context.Context, version string, mft manifest.EstafetteManifest) (counter int) {

	counterCandidate := version
	if mft.Version.SemVer != nil {
		re := regexp.MustCompile(`^[0-9]+\.[0-9]+\.([0-9]+)(-[0-9a-zA-Z-/]+)?$`)
		match := re.FindStringSubmatch(version)

		if len(match) > 1 {
			counterCandidate = match[1]
		}
	}

	counter, err := strconv.Atoi(counterCandidate)
	if err != nil {
		log.Warn().Err(err).Msgf("Failed extracting counter from version %v", version)
	}

	return counter
}

func (s *service) getBuildJobResources(ctx context.Context, build contracts.Build) cockroachdb.JobResources {
	// define resource request and limit values to fit reasonably well inside a n1-standard-8 (8 vCPUs, 30 GB memory) machine
	defaultCPUCores := s.config.Jobs.DefaultCPUCores
	if defaultCPUCores == 0 {
		defaultCPUCores = s.config.Jobs.MaxCPUCores
	}
	defaultMemory := s.config.Jobs.DefaultMemoryBytes
	if defaultMemory == 0 {
		defaultMemory = s.config.Jobs.MaxMemoryBytes
	}

	jobResources := cockroachdb.JobResources{
		CPURequest:    defaultCPUCores,
		CPULimit:      s.config.Jobs.MaxCPUCores,
		MemoryRequest: defaultMemory,
		MemoryLimit:   s.config.Jobs.MaxMemoryBytes,
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
			desiredCPURequest := measuredResources.CPUMaxUsage * s.config.Jobs.CPURequestRatio
			if desiredCPURequest <= s.config.Jobs.MinCPUCores {
				jobResources.CPURequest = s.config.Jobs.MinCPUCores
			} else if desiredCPURequest >= s.config.Jobs.MaxCPUCores {
				jobResources.CPURequest = s.config.Jobs.MaxCPUCores
			} else {
				jobResources.CPURequest = desiredCPURequest
			}

			if s.config.Jobs.CPULimitRatio > 1.0 {
				jobResources.CPULimit = jobResources.CPURequest * s.config.Jobs.CPULimitRatio
			} else if jobResources.CPURequest > jobResources.CPULimit {
				// keep limit at default, unless cpu request is larger
				jobResources.CPULimit = jobResources.CPURequest
			}
		}

		if measuredResources.MemoryMaxUsage > 0 {
			desiredMemoryRequest := measuredResources.MemoryMaxUsage * s.config.Jobs.MemoryRequestRatio
			if desiredMemoryRequest <= s.config.Jobs.MinMemoryBytes {
				jobResources.MemoryRequest = s.config.Jobs.MinMemoryBytes
			} else if desiredMemoryRequest >= s.config.Jobs.MaxMemoryBytes {
				jobResources.MemoryRequest = s.config.Jobs.MaxMemoryBytes
			} else {
				jobResources.MemoryRequest = desiredMemoryRequest
			}

			if s.config.Jobs.MemoryLimitRatio > 1.0 {
				jobResources.MemoryLimit = jobResources.MemoryRequest * s.config.Jobs.MemoryLimitRatio
			} else if jobResources.MemoryRequest > jobResources.MemoryLimit {
				// keep limit at default, unless memory request is larger
				jobResources.MemoryLimit = jobResources.MemoryRequest
			}
		}
	}

	return jobResources
}

func (s *service) getReleaseJobResources(ctx context.Context, release contracts.Release) cockroachdb.JobResources {

	// define resource request and limit values to fit reasonably well inside a n1-standard-8 (8 vCPUs, 30 GB memory) machine
	jobResources := cockroachdb.JobResources{
		CPURequest:    s.config.Jobs.MaxCPUCores,
		CPULimit:      s.config.Jobs.MaxCPUCores,
		MemoryRequest: s.config.Jobs.MaxMemoryBytes,
		MemoryLimit:   s.config.Jobs.MaxMemoryBytes,
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
			if measuredResources.CPUMaxUsage*s.config.Jobs.CPURequestRatio <= s.config.Jobs.MinCPUCores {
				jobResources.CPURequest = s.config.Jobs.MinCPUCores
			} else if measuredResources.CPUMaxUsage*s.config.Jobs.CPURequestRatio >= s.config.Jobs.MaxCPUCores {
				jobResources.CPURequest = s.config.Jobs.MaxCPUCores
			} else {
				jobResources.CPURequest = measuredResources.CPUMaxUsage * s.config.Jobs.CPURequestRatio
			}
		}

		if measuredResources.MemoryMaxUsage > 0 {
			if measuredResources.MemoryMaxUsage*s.config.Jobs.MemoryRequestRatio <= s.config.Jobs.MinMemoryBytes {
				jobResources.MemoryRequest = s.config.Jobs.MinMemoryBytes
			} else if measuredResources.MemoryMaxUsage*s.config.Jobs.MemoryRequestRatio >= s.config.Jobs.MaxMemoryBytes {
				jobResources.MemoryRequest = s.config.Jobs.MaxMemoryBytes
			} else {
				jobResources.MemoryRequest = measuredResources.MemoryMaxUsage * s.config.Jobs.MemoryRequestRatio
			}
		}
	}

	return jobResources
}

func (s *service) getBotJobResources(ctx context.Context, bot contracts.Bot) cockroachdb.JobResources {

	// define resource request and limit values to fit reasonably well inside a n1-standard-8 (8 vCPUs, 30 GB memory) machine
	jobResources := cockroachdb.JobResources{
		CPURequest:    s.config.Jobs.MaxCPUCores,
		CPULimit:      s.config.Jobs.MaxCPUCores,
		MemoryRequest: s.config.Jobs.MaxMemoryBytes,
		MemoryLimit:   s.config.Jobs.MaxMemoryBytes,
	}

	// get max usage from previous releases
	measuredResources, nrRecords, err := s.cockroachdbClient.GetPipelineReleaseMaxResourceUtilization(ctx, bot.RepoSource, bot.RepoOwner, bot.RepoName, bot.Name, 25)
	if err != nil {
		log.Warn().Err(err).Msgf("Failed retrieving max resource utilization for recent bots of %v/%v/%v target %v, using defaults...", bot.RepoSource, bot.RepoOwner, bot.RepoName, bot.Name)
	} else if nrRecords < 5 {
		log.Info().Msgf("Retrieved max resource utilization for recent bots of %v/%v/%v target %v only has %v records, using defaults...", bot.RepoSource, bot.RepoOwner, bot.RepoName, bot.Name, nrRecords)
	} else {
		log.Info().Msgf("Retrieved max resource utilization for recent bots of %v/%v/%v target %v, checking if they are within lower and upper bound...", bot.RepoSource, bot.RepoOwner, bot.RepoName, bot.Name)

		// only override cpu and memory request values if measured values are within min and max
		if measuredResources.CPUMaxUsage > 0 {
			if measuredResources.CPUMaxUsage*s.config.Jobs.CPURequestRatio <= s.config.Jobs.MinCPUCores {
				jobResources.CPURequest = s.config.Jobs.MinCPUCores
			} else if measuredResources.CPUMaxUsage*s.config.Jobs.CPURequestRatio >= s.config.Jobs.MaxCPUCores {
				jobResources.CPURequest = s.config.Jobs.MaxCPUCores
			} else {
				jobResources.CPURequest = measuredResources.CPUMaxUsage * s.config.Jobs.CPURequestRatio
			}
		}

		if measuredResources.MemoryMaxUsage > 0 {
			if measuredResources.MemoryMaxUsage*s.config.Jobs.MemoryRequestRatio <= s.config.Jobs.MinMemoryBytes {
				jobResources.MemoryRequest = s.config.Jobs.MinMemoryBytes
			} else if measuredResources.MemoryMaxUsage*s.config.Jobs.MemoryRequestRatio >= s.config.Jobs.MaxMemoryBytes {
				jobResources.MemoryRequest = s.config.Jobs.MaxMemoryBytes
			} else {
				jobResources.MemoryRequest = measuredResources.MemoryMaxUsage * s.config.Jobs.MemoryRequestRatio
			}
		}
	}

	return jobResources
}

func (s *service) supportsBuildStatus(repoSource string) bool {

	switch {
	case githubapi.IsRepoSourceGithub(repoSource):
		return true

	case bitbucketapi.IsRepoSourceBitbucket(repoSource):
		return true

	case cloudsourceapi.IsRepoSourceCloudSource(repoSource):
		return false
	}

	return false
}

func (s *service) GetEventsForJobEnvvars(ctx context.Context, triggers []manifest.EstafetteTrigger, events []manifest.EstafetteEvent) (triggersAsEvents []manifest.EstafetteEvent, err error) {

	triggersAsEvents = events

	for _, t := range triggers {
		if t.Name != "" && t.Pipeline != nil && t.Pipeline.Event == "finished" {
			// get last green build for pipeline matching the trigger and set as event
			pipelineNameParts := strings.Split(t.Pipeline.Name, "/")
			if len(pipelineNameParts) == 3 {
				filters := map[api.FilterType][]string{
					api.FilterStatus: {
						string(contracts.StatusSucceeded),
					},
				}

				if t.Pipeline.Branch != "" {
					branches := strings.Split(t.Pipeline.Branch, "|")
					if len(branches) > 0 {
						filters[api.FilterBranch] = branches
					}
				}

				lastBuilds, innerErr := s.cockroachdbClient.GetPipelineBuilds(ctx, pipelineNameParts[0], pipelineNameParts[1], pipelineNameParts[2], 1, 1, filters, []api.OrderField{}, false)
				if innerErr != nil {
					return
				}
				if lastBuilds == nil || len(lastBuilds) == 0 {
					return triggersAsEvents, fmt.Errorf("Can't get a successful build for named trigger '%v' linking to pipeline '%v'", t.Name, t.Pipeline.Name)
				}

				lastSuccessfulBuild := lastBuilds[0]

				triggersAsEvents = append(triggersAsEvents, manifest.EstafetteEvent{
					Name:  t.Name,
					Fired: false,
					Pipeline: &manifest.EstafettePipelineEvent{
						BuildVersion: lastSuccessfulBuild.BuildVersion,
						RepoSource:   lastSuccessfulBuild.RepoSource,
						RepoOwner:    lastSuccessfulBuild.RepoOwner,
						RepoName:     lastSuccessfulBuild.RepoName,
						Branch:       lastSuccessfulBuild.RepoBranch,
						Status:       string(lastSuccessfulBuild.BuildStatus),
						Event:        t.Pipeline.Event,
					},
				})
			}
		}

		if t.Name != "" && t.Release != nil && t.Release.Event == "finished" {
			// get last green build for pipeline matching the trigger and set as event
			pipelineNameParts := strings.Split(t.Release.Name, "/")
			if len(pipelineNameParts) == 3 {
				filters := map[api.FilterType][]string{
					api.FilterStatus: {
						string(contracts.StatusSucceeded),
					},
				}

				if t.Release.Target != "" {
					filters[api.FilterReleaseTarget] = []string{t.Release.Target}
				}

				lastReleases, innerErr := s.cockroachdbClient.GetPipelineReleases(ctx, pipelineNameParts[0], pipelineNameParts[1], pipelineNameParts[2], 1, 1, filters, []api.OrderField{})
				if innerErr != nil {
					return
				}
				if lastReleases == nil || len(lastReleases) == 0 {
					return triggersAsEvents, fmt.Errorf("Can't get a successful release for named trigger '%v' linking to pipeline '%v' and target '%v'", t.Name, t.Release.Name, t.Release.Target)
				}

				lastSuccessfulRelease := lastReleases[0]

				triggersAsEvents = append(triggersAsEvents, manifest.EstafetteEvent{
					Name:  t.Name,
					Fired: false,
					Release: &manifest.EstafetteReleaseEvent{
						ReleaseVersion: lastSuccessfulRelease.ReleaseVersion,
						RepoSource:     lastSuccessfulRelease.RepoSource,
						RepoOwner:      lastSuccessfulRelease.RepoOwner,
						RepoName:       lastSuccessfulRelease.RepoName,
						Status:         string(lastSuccessfulRelease.ReleaseStatus),
						Event:          t.Release.Event,
					},
				})
			}
		}
	}

	return triggersAsEvents, nil
}
