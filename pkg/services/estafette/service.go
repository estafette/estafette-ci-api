package estafette

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/bitbucketapi"
	"github.com/estafette/estafette-ci-api/pkg/clients/builderapi"
	"github.com/estafette/estafette-ci-api/pkg/clients/cloudsourceapi"
	"github.com/estafette/estafette-ci-api/pkg/clients/cloudstorage"
	"github.com/estafette/estafette-ci-api/pkg/clients/database"
	"github.com/estafette/estafette-ci-api/pkg/clients/githubapi"
	"github.com/estafette/estafette-ci-api/pkg/clients/prometheus"
	contracts "github.com/estafette/estafette-ci-contracts"
	crypt "github.com/estafette/estafette-ci-crypt"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	yaml "gopkg.in/yaml.v2"
)

const (
	releaseNotAllowed = "Release not allowed on this branch"
)

var (
	ErrNoBuildCreated    = errors.New("No build is created")
	ErrNoReleaseCreated  = errors.New("No release is created")
	ErrNoBotCreated      = errors.New("No bot is created")
	ErrReleaseNotAllowed = &ReleaseError{Message: releaseNotAllowed}
)

type ReleaseError struct {
	Cluster                  string                        `json:"cluster,omitempty"`
	Message                  string                        `json:"message,omitempty"`
	RepositoryReleaseControl *api.RepositoryReleaseControl `json:"repositoryReleaseControl,omitempty"`
}

func (r *ReleaseError) Error() string {
	return r.Message
}

func (r *ReleaseError) Is(target error) bool {
	if target, ok := target.(*ReleaseError); !ok {
		return false
	} else {
		return r.Error() == target.Error()
	}
}

// Service encapsulates build and release creation and re-triggering
//
//go:generate mockgen -package=estafette -destination ./mock.go -source=service.go
type Service interface {
	CreateBuild(ctx context.Context, build contracts.Build) (b *contracts.Build, err error)
	FinishBuild(ctx context.Context, repoSource, repoOwner, repoName string, buildID string, buildStatus contracts.Status) (err error)
	CreateRelease(ctx context.Context, release contracts.Release, mft manifest.EstafetteManifest, repoBranch, repoRevision string) (r *contracts.Release, err error)
	FinishRelease(ctx context.Context, repoSource, repoOwner, repoName string, releaseID string, releaseStatus contracts.Status) (err error)
	CreateBot(ctx context.Context, bot contracts.Bot, mft manifest.EstafetteManifest, repoBranch string) (b *contracts.Bot, err error)
	FinishBot(ctx context.Context, repoSource, repoOwner, repoName string, botID string, botStatus contracts.Status) (err error)
	FireGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (err error)
	FirePipelineTriggers(ctx context.Context, build contracts.Build, event string) (err error)
	FireReleaseTriggers(ctx context.Context, release contracts.Release, event string) (err error)
	FirePubSubTriggers(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) (err error)
	FireCronTriggers(ctx context.Context, cronEvent manifest.EstafetteCronEvent) (err error)
	FireGithubTriggers(ctx context.Context, githubEvent manifest.EstafetteGithubEvent) (err error)
	FireBitbucketTriggers(ctx context.Context, bitbucketEvent manifest.EstafetteBitbucketEvent) (err error)
	Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
	Archive(ctx context.Context, repoSource, repoOwner, repoName string) (err error)
	Unarchive(ctx context.Context, repoSource, repoOwner, repoName string) (err error)
	UpdateBuildStatus(ctx context.Context, event contracts.EstafetteCiBuilderEvent) (err error)
	UpdateJobResources(ctx context.Context, event contracts.EstafetteCiBuilderEvent) (err error)
	GetEventsForJobEnvvars(ctx context.Context, triggers []manifest.EstafetteTrigger, events []manifest.EstafetteEvent) (triggersAsEvents []manifest.EstafetteEvent, err error)
}

// NewService returns a new estafette.Service
func NewService(config *api.APIConfig, databaseClient database.Client, secretHelper crypt.SecretHelper, prometheusClient prometheus.Client, cloudStorageClient cloudstorage.Client, builderapiClient builderapi.Client, githubJobVarsFunc func(context.Context, string, string, string) (string, error), bitbucketJobVarsFunc func(context.Context, string, string, string) (string, error), cloudsourceJobVarsFunc func(context.Context, string, string, string) (string, error)) Service {

	return &service{
		config:                 config,
		databaseClient:         databaseClient,
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
	databaseClient         database.Client
	secretHelper           crypt.SecretHelper
	prometheusClient       prometheus.Client
	cloudStorageClient     cloudstorage.Client
	builderapiClient       builderapi.Client
	githubJobVarsFunc      func(context.Context, string, string, string) (string, error)
	bitbucketJobVarsFunc   func(context.Context, string, string, string) (string, error)
	cloudsourceJobVarsFunc func(context.Context, string, string, string) (string, error)
	triggerConcurrency     int64
}

func (s *service) CreateBuild(ctx context.Context, build contracts.Build) (createdBuild *contracts.Build, err error) {

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
	pipeline, _ := s.databaseClient.GetPipeline(ctx, build.RepoSource, build.RepoOwner, build.RepoName, map[api.FilterType][]string{}, false)

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
		lastBuildsForBranch, _ := s.databaseClient.GetPipelineBuilds(ctx, pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName, 1, 10, map[api.FilterType][]string{api.FilterBranch: []string{pipeline.RepoBranch}}, []api.OrderField{}, false)
		if len(lastBuildsForBranch) == 1 {
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
	environmentVariableWithToken, err := s.getSourceCodeAccessToken(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
	if err != nil {
		return
	}

	jobResources := s.getBuildJobResources(ctx, build)

	// store build in db
	createdBuild, err = s.databaseClient.InsertBuild(ctx, contracts.Build{
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

	triggeredByEvents, err := s.GetEventsForJobEnvvars(ctx, build.Triggers, build.Events)
	if err != nil {
		return
	}

	// define ci builder params
	ciBuilderParams := builderapi.CiBuilderParams{
		BuilderConfig: contracts.BuilderConfig{
			JobType: contracts.JobTypeBuild,
			Git: &contracts.GitConfig{
				RepoSource:   createdBuild.RepoSource,
				RepoOwner:    createdBuild.RepoOwner,
				RepoName:     createdBuild.RepoName,
				RepoBranch:   createdBuild.RepoBranch,
				RepoRevision: createdBuild.RepoRevision,
			},
			Track: &builderTrack,
			Version: &contracts.VersionConfig{
				Version:                 createdBuild.BuildVersion,
				CurrentCounter:          currentCounter,
				AutoIncrement:           &currentCounter,
				MaxCounter:              maxCounter,
				MaxCounterCurrentBranch: maxCounterCurrentBranch,
			},
			Manifest: &mft,
			Build:    createdBuild,
			Events:   triggeredByEvents,
		},
		EnvironmentVariables: environmentVariableWithToken,
		OperatingSystem:      builderOperatingSystem,
		JobResources:         jobResources,
	}

	// create ci builder job
	if hasValidManifest && invalidSecretsErr == nil {
		log.Debug().Msgf("Pipeline %v/%v/%v revision %v has valid manifest, creating build job...", build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision)
		// create ci builder job
		_, err = s.builderapiClient.CreateCiBuilderJob(ctx, ciBuilderParams)
		if err != nil {
			return
		}

		// handle triggers
		go func() {
			// create new context to avoid cancellation impacting execution
			span, _ := opentracing.StartSpanFromContext(ctx, "estafette:AsyncFirePipelineTriggers")
			ctx := opentracing.ContextWithSpan(context.Background(), span)
			defer span.Finish()

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

		insertedBuildLog, err := s.databaseClient.InsertBuildLog(ctx, buildLog)
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

		insertedBuildLog, err := s.databaseClient.InsertBuildLog(ctx, buildLog)
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

func (s *service) FinishBuild(ctx context.Context, repoSource, repoOwner, repoName string, buildID string, buildStatus contracts.Status) error {

	err := s.databaseClient.UpdateBuildStatus(ctx, repoSource, repoOwner, repoName, buildID, buildStatus)
	if err != nil {
		return err
	}

	// handle triggers
	go func() {
		// create new context to avoid cancellation impacting execution
		span, _ := opentracing.StartSpanFromContext(ctx, "estafette:AsyncFirePipelineTriggers")
		ctx := opentracing.ContextWithSpan(context.Background(), span)
		defer span.Finish()

		build, err := s.databaseClient.GetPipelineBuildByID(ctx, repoSource, repoOwner, repoName, buildID, false)
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

func (s *service) CreateRelease(ctx context.Context, release contracts.Release, mft manifest.EstafetteManifest, repoBranch, repoRevision string) (createdRelease *contracts.Release, err error) {
	if blocked, cluster, rc := s.isReleaseBlocked(release, mft, repoBranch); blocked {
		return nil, &ReleaseError{
			Cluster:                  cluster,
			Message:                  releaseNotAllowed,
			RepositoryReleaseControl: rc,
		}
	}
	// create deep copy to ensure no properties are shared through a pointer
	mft = mft.DeepCopy()

	// check if there's secrets restricted to other pipelines
	manifestBytes, err := yaml.Marshal(mft)
	if err != nil {
		return
	}
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
	environmentVariableWithToken, err := s.getSourceCodeAccessToken(ctx, release.RepoSource, release.RepoOwner, release.RepoName)
	if err != nil {
		return
	}

	jobResources := s.getReleaseJobResources(ctx, release)

	// create release in database
	createdRelease, err = s.databaseClient.InsertRelease(ctx, contracts.Release{
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

	maxCounter := currentCounter
	maxCounterCurrentBranch := currentCounter

	triggeredByEvents := release.Events

	// retrieve pipeline if already exists to get counter value
	pipeline, _ := s.databaseClient.GetPipeline(ctx, release.RepoSource, release.RepoOwner, release.RepoName, map[api.FilterType][]string{}, false)
	if pipeline != nil {
		// get max counter for pipeline
		mc := s.getVersionCounter(ctx, pipeline.BuildVersion, mft)
		if mc > maxCounter {
			maxCounter = mc
		}

		// get max counter for same branch
		lastBuildsForBranch, _ := s.databaseClient.GetPipelineBuilds(ctx, pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName, 1, 10, map[api.FilterType][]string{api.FilterBranch: []string{repoBranch}}, []api.OrderField{}, false)
		if len(lastBuildsForBranch) >= 1 {
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
		BuilderConfig: contracts.BuilderConfig{
			JobType: contracts.JobTypeRelease,
			Git: &contracts.GitConfig{
				RepoSource:   createdRelease.RepoSource,
				RepoOwner:    createdRelease.RepoOwner,
				RepoName:     createdRelease.RepoName,
				RepoBranch:   repoBranch,
				RepoRevision: repoRevision,
			},
			Track: &builderTrack,
			Version: &contracts.VersionConfig{
				Version:                 createdRelease.ReleaseVersion,
				CurrentCounter:          currentCounter,
				AutoIncrement:           &currentCounter,
				MaxCounter:              maxCounter,
				MaxCounterCurrentBranch: maxCounterCurrentBranch,
			},
			Manifest: &mft,
			Release:  createdRelease,
			Events:   triggeredByEvents,
		},
		EnvironmentVariables: environmentVariableWithToken,
		OperatingSystem:      builderOperatingSystem,
		JobResources:         jobResources,
	}

	if invalidSecretsErr == nil {
		// create ci release job
		_, err = s.builderapiClient.CreateCiBuilderJob(ctx, ciBuilderParams)
		if err != nil {
			return
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

		insertedReleaseLog, err := s.databaseClient.InsertReleaseLog(ctx, releaseLog)
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
		// create new context to avoid cancellation impacting execution
		span, _ := opentracing.StartSpanFromContext(ctx, "estafette:AsyncFireReleaseTriggers")
		ctx := opentracing.ContextWithSpan(context.Background(), span)
		defer span.Finish()

		err := s.FireReleaseTriggers(ctx, release, "started")
		if err != nil {
			log.Error().Err(err).Msgf("Failed firing release triggers for %v/%v/%v to target %v", release.RepoSource, release.RepoOwner, release.RepoName, release.Name)
		}
	}()

	return
}

func (s *service) FinishRelease(ctx context.Context, repoSource, repoOwner, repoName string, releaseID string, releaseStatus contracts.Status) error {
	err := s.databaseClient.UpdateReleaseStatus(ctx, repoSource, repoOwner, repoName, releaseID, releaseStatus)
	if err != nil {
		return err
	}

	// handle triggers
	go func() {
		// create new context to avoid cancellation impacting execution
		span, _ := opentracing.StartSpanFromContext(ctx, "estafette:AsyncFireReleaseTriggers")
		ctx := opentracing.ContextWithSpan(context.Background(), span)
		defer span.Finish()

		release, err := s.databaseClient.GetPipelineRelease(ctx, repoSource, repoOwner, repoName, releaseID)
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

func (s *service) CreateBot(ctx context.Context, bot contracts.Bot, mft manifest.EstafetteManifest, repoBranch string) (createdBot *contracts.Bot, err error) {

	// create deep copy to ensure no properties are shared through a pointer
	mft = mft.DeepCopy()

	// check if there's secrets restricted to other pipelines
	manifestBytes, err := yaml.Marshal(mft)
	if err != nil {
		return
	}
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
	environmentVariableWithToken, err := s.getSourceCodeAccessToken(ctx, bot.RepoSource, bot.RepoOwner, bot.RepoName)
	if err != nil {
		return
	}

	jobResources := s.getBotJobResources(ctx, bot)

	// create release in database
	createdBot, err = s.databaseClient.InsertBot(ctx, contracts.Bot{
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

	// define ci builder params
	ciBuilderParams := builderapi.CiBuilderParams{
		BuilderConfig: contracts.BuilderConfig{
			JobType: contracts.JobTypeBot,
			Git: &contracts.GitConfig{
				RepoSource: createdBot.RepoSource,
				RepoOwner:  createdBot.RepoOwner,
				RepoName:   createdBot.RepoName,
				RepoBranch: repoBranch,
			},
			Track:    &builderTrack,
			Version:  &contracts.VersionConfig{},
			Manifest: &mft,
			Bot:      createdBot,
			Events:   createdBot.Events,
		},
		EnvironmentVariables: environmentVariableWithToken,
		OperatingSystem:      builderOperatingSystem,
		JobResources:         jobResources,
	}

	if invalidSecretsErr == nil {
		// create ci release job
		_, err = s.builderapiClient.CreateCiBuilderJob(ctx, ciBuilderParams)
		if err != nil {
			return
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

		insertedBotLog, err := s.databaseClient.InsertBotLog(ctx, botLog)
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

func (s *service) FinishBot(ctx context.Context, repoSource, repoOwner, repoName string, botID string, botStatus contracts.Status) error {
	err := s.databaseClient.UpdateBotStatus(ctx, repoSource, repoOwner, repoName, botID, botStatus)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) FireGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) error {

	log.Debug().Msgf("[trigger:git(%v-%v:%v)] Checking if triggers need to be fired...", gitEvent.Repository, gitEvent.Branch, gitEvent.Event)

	// retrieve all pipeline triggers
	pipelines, err := s.databaseClient.GetGitTriggers(ctx, gitEvent)
	if err != nil {
		return err
	}

	e := manifest.EstafetteEvent{
		Fired: true,
		Git:   &gitEvent,
	}

	triggerCount := 0
	firedTriggerCount := 0

	// limit concurrency using a semaphore
	semaphore := semaphore.NewWeighted(s.triggerConcurrency)
	g, ctx := errgroup.WithContext(ctx)

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

				p := p
				t := t
				e := e

				g.Go(func() error {
					err = semaphore.Acquire(ctx, 1)
					if err != nil {
						return err
					}
					defer semaphore.Release(1)

					// create new context to avoid cancellation impacting execution
					span, _ := opentracing.StartSpanFromContext(ctx, "estafette:AsyncFireGitTriggersItem")
					ctx = opentracing.ContextWithSpan(context.Background(), span)
					defer span.Finish()

					// create new build for t.Run
					if t.BuildAction != nil {
						log.Debug().Msgf("[trigger:git(%v-%v:%v)] Firing build action '%v/%v/%v', branch '%v'...", gitEvent.Repository, gitEvent.Branch, gitEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
						err := s.fireBuild(ctx, *p, t, e)
						if err != nil {
							log.Error().Err(err).Msgf("[trigger:git(%v-%v:%v)] Failed starting build action'%v/%v/%v', branch '%v'", gitEvent.Repository, gitEvent.Branch, gitEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
						}
					} else if t.ReleaseAction != nil {
						log.Debug().Msgf("[trigger:git(%v-%v:%v)] Firing release action '%v/%v/%v', target '%v', action '%v'...", gitEvent.Repository, gitEvent.Branch, gitEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
						err := s.fireRelease(ctx, *p, t, e)
						if err != nil {
							log.Error().Err(err).Msgf("[trigger:git(%v-%v:%v)] Failed starting release action '%v/%v/%v', target '%v', action '%v'", gitEvent.Repository, gitEvent.Branch, gitEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
						}
					} else if t.BotAction != nil {
						log.Debug().Msgf("[trigger:git(%v-%v:%v)] Firing bot action '%v/%v/%v', branch '%v'...", gitEvent.Repository, gitEvent.Branch, gitEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName, t.BotAction.Branch)
						err := s.fireBot(ctx, *p, t, e)
						if err != nil {
							log.Error().Err(err).Msgf("[trigger:git(%v-%v:%v)] Failed starting bot action '%v/%v/%v', branch '%v'", gitEvent.Repository, gitEvent.Branch, gitEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName, t.BotAction.Branch)
						}
					}

					return nil
				})
			}
		}
	}

	// wait until all concurrent goroutines are done
	err = g.Wait()
	if err != nil {
		return err
	}

	log.Debug().Msgf("[trigger:git(%v-%v:%v)] Fired %v out of %v triggers for %v pipelines", gitEvent.Repository, gitEvent.Branch, gitEvent.Event, firedTriggerCount, triggerCount, len(pipelines))

	return nil
}

func (s *service) FireGithubTriggers(ctx context.Context, githubEvent manifest.EstafetteGithubEvent) (err error) {

	log.Debug().Msgf("[trigger:github(%v:%v)] Checking if triggers need to be fired...", githubEvent.Repository, githubEvent.Event)

	// retrieve all pipeline triggers
	pipelines, err := s.databaseClient.GetGithubTriggers(ctx, githubEvent)
	if err != nil {
		return err
	}

	log.Debug().Msgf("[trigger:github(%v:%v)] Retrieved %v pipelines...", githubEvent.Repository, githubEvent.Event, len(pipelines))

	e := manifest.EstafetteEvent{
		Fired:  true,
		Github: &githubEvent,
	}

	triggerCount := 0
	firedTriggerCount := 0

	// limit concurrency using a semaphore
	semaphore := semaphore.NewWeighted(s.triggerConcurrency)
	g, ctx := errgroup.WithContext(ctx)

	// check for each trigger whether it should fire
	for _, p := range pipelines {
		for _, t := range p.Triggers {

			log.Debug().Interface("event", githubEvent).Interface("trigger", t).Msgf("[trigger:github(%v:%v)] Checking if pipeline '%v/%v/%v' trigger should fire...", githubEvent.Repository, githubEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName)

			if t.Github == nil {
				continue
			}

			triggerCount++

			if t.Github.Fires(&githubEvent) {

				firedTriggerCount++

				p := p
				t := t
				e := e

				g.Go(func() error {
					err = semaphore.Acquire(ctx, 1)
					if err != nil {
						return err
					}
					defer semaphore.Release(1)

					// create new context to avoid cancellation impacting execution
					span, _ := opentracing.StartSpanFromContext(ctx, "estafette:AsyncFireGithubTriggersItem")
					ctx = opentracing.ContextWithSpan(context.Background(), span)
					defer span.Finish()

					// create new build for t.Run
					if t.BuildAction != nil {
						log.Debug().Msgf("[trigger:github(%v:%v)] Firing build action '%v/%v/%v', branch '%v'...", githubEvent.Repository, githubEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
						err := s.fireBuild(ctx, *p, t, e)
						if err != nil {
							log.Error().Err(err).Msgf("[trigger:github(%v:%v)] Failed starting build action'%v/%v/%v', branch '%v'", githubEvent.Repository, githubEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
						}
					} else if t.ReleaseAction != nil {
						log.Debug().Msgf("[trigger:github(%v:%v)] Firing release action '%v/%v/%v', target '%v', action '%v'...", githubEvent.Repository, githubEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
						err := s.fireRelease(ctx, *p, t, e)
						if err != nil {
							log.Error().Err(err).Msgf("[trigger:github(%v:%v)] Failed starting release action '%v/%v/%v', target '%v', action '%v'", githubEvent.Repository, githubEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
						}
					} else if t.BotAction != nil {
						log.Debug().Msgf("[trigger:github(%v:%v)] Firing bot action '%v/%v/%v', branch '%v'...", githubEvent.Repository, githubEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName, t.BotAction.Branch)
						err := s.fireBot(ctx, *p, t, e)
						if err != nil {
							log.Error().Err(err).Msgf("[trigger:github(%v:%v)] Failed starting bot action '%v/%v/%v', branch '%v'", githubEvent.Repository, githubEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName, t.BotAction.Branch)
						}
					}

					return nil
				})
			}
		}
	}

	// wait until all concurrent goroutines are done
	err = g.Wait()
	if err != nil {
		return err
	}

	log.Debug().Msgf("[trigger:github(%v:%v)] Fired %v out of %v triggers for %v pipelines", githubEvent.Repository, githubEvent.Event, firedTriggerCount, triggerCount, len(pipelines))

	return nil
}

func (s *service) FireBitbucketTriggers(ctx context.Context, bitbucketEvent manifest.EstafetteBitbucketEvent) (err error) {

	log.Debug().Msgf("[trigger:bitbucket(%v:%v)] Checking if triggers need to be fired...", bitbucketEvent.Repository, bitbucketEvent.Event)

	// retrieve all pipeline triggers
	pipelines, err := s.databaseClient.GetBitbucketTriggers(ctx, bitbucketEvent)
	if err != nil {
		return err
	}

	log.Debug().Msgf("[trigger:bitbucket(%v:%v)] Retrieved %v pipelines...", bitbucketEvent.Repository, bitbucketEvent.Event, len(pipelines))

	e := manifest.EstafetteEvent{
		Fired:     true,
		Bitbucket: &bitbucketEvent,
	}

	triggerCount := 0
	firedTriggerCount := 0

	// limit concurrency using a semaphore
	semaphore := semaphore.NewWeighted(s.triggerConcurrency)
	g, ctx := errgroup.WithContext(ctx)

	// check for each trigger whether it should fire
	for _, p := range pipelines {
		for _, t := range p.Triggers {

			log.Debug().Interface("event", bitbucketEvent).Interface("trigger", t).Msgf("[trigger:bitbucket(%v:%v)] Checking if pipeline '%v/%v/%v' trigger should fire...", bitbucketEvent.Repository, bitbucketEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName)

			if t.Bitbucket == nil {
				continue
			}

			triggerCount++

			if t.Bitbucket.Fires(&bitbucketEvent) {

				firedTriggerCount++

				p := p
				t := t
				e := e

				g.Go(func() error {
					err = semaphore.Acquire(ctx, 1)
					if err != nil {
						return err
					}
					defer semaphore.Release(1)

					// create new context to avoid cancellation impacting execution
					span, _ := opentracing.StartSpanFromContext(ctx, "estafette:AsyncFireBitbucketTriggersItem")
					ctx = opentracing.ContextWithSpan(context.Background(), span)
					defer span.Finish()

					// create new build for t.Run
					if t.BuildAction != nil {
						log.Debug().Msgf("[trigger:bitbucket(%v:%v)] Firing build action '%v/%v/%v', branch '%v'...", bitbucketEvent.Repository, bitbucketEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
						err := s.fireBuild(ctx, *p, t, e)
						if err != nil {
							log.Error().Err(err).Msgf("[trigger:bitbucket(%v:%v)] Failed starting build action'%v/%v/%v', branch '%v'", bitbucketEvent.Repository, bitbucketEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
						}
					} else if t.ReleaseAction != nil {
						log.Debug().Msgf("[trigger:bitbucket(%v:%v)] Firing release action '%v/%v/%v', target '%v', action '%v'...", bitbucketEvent.Repository, bitbucketEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
						err := s.fireRelease(ctx, *p, t, e)
						if err != nil {
							log.Error().Err(err).Msgf("[trigger:bitbucket(%v:%v)] Failed starting release action '%v/%v/%v', target '%v', action '%v'", bitbucketEvent.Repository, bitbucketEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
						}
					} else if t.BotAction != nil {
						log.Debug().Msgf("[trigger:bitbucket(%v:%v)] Firing bot action '%v/%v/%v', branch '%v'...", bitbucketEvent.Repository, bitbucketEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName, t.BotAction.Branch)
						err := s.fireBot(ctx, *p, t, e)
						if err != nil {
							log.Error().Err(err).Msgf("[trigger:bitbucket(%v:%v)] Failed starting bot action '%v/%v/%v', branch '%v'", bitbucketEvent.Repository, bitbucketEvent.Event, p.RepoSource, p.RepoOwner, p.RepoName, t.BotAction.Branch)
						}
					}

					return nil
				})
			}
		}
	}

	// wait until all concurrent goroutines are done
	err = g.Wait()
	if err != nil {
		return err
	}

	log.Debug().Msgf("[trigger:bitbucket(%v:%v)] Fired %v out of %v triggers for %v pipelines", bitbucketEvent.Repository, bitbucketEvent.Event, firedTriggerCount, triggerCount, len(pipelines))

	return nil
}

func (s *service) FirePipelineTriggers(ctx context.Context, build contracts.Build, event string) error {

	log.Debug().Msgf("[trigger:pipeline(%v/%v/%v:%v)] Checking if triggers need to be fired...", build.RepoSource, build.RepoOwner, build.RepoName, event)

	// retrieve all pipeline triggers
	pipelines, err := s.databaseClient.GetPipelineTriggers(ctx, build, event)
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

	// limit concurrency using a semaphore
	semaphore := semaphore.NewWeighted(s.triggerConcurrency)
	g, ctx := errgroup.WithContext(ctx)

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

				p := p
				t := t
				e := e

				g.Go(func() error {
					err = semaphore.Acquire(ctx, 1)
					if err != nil {
						return err
					}
					defer semaphore.Release(1)

					// create new context to avoid cancellation impacting execution
					span, _ := opentracing.StartSpanFromContext(ctx, "estafette:AsyncFirePipelineTriggerItem")
					ctx = opentracing.ContextWithSpan(context.Background(), span)
					defer span.Finish()

					// create new build for t.Run
					if t.BuildAction != nil {
						log.Debug().Msgf("[trigger:pipeline(%v/%v/%v:%v)] Firing build action '%v/%v/%v', branch '%v'...", build.RepoSource, build.RepoOwner, build.RepoName, event, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
						err := s.fireBuild(ctx, *p, t, e)
						if err != nil {
							log.Error().Err(err).Msgf("[trigger:pipeline(%v/%v/%v:%v)] Failed starting build action'%v/%v/%v', branch '%v'", build.RepoSource, build.RepoOwner, build.RepoName, event, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
						}
					} else if t.ReleaseAction != nil {
						log.Debug().Msgf("[trigger:pipeline(%v/%v/%v:%v)] Firing release action '%v/%v/%v', target '%v', action '%v'...", build.RepoSource, build.RepoOwner, build.RepoName, event, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
						err := s.fireRelease(ctx, *p, t, e)
						if err != nil {
							log.Error().Err(err).Msgf("[trigger:pipeline(%v/%v/%v:%v)] Failed starting release action '%v/%v/%v', target '%v', action '%v'", build.RepoSource, build.RepoOwner, build.RepoName, event, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
						}
					} else if t.BotAction != nil {
						log.Debug().Msgf("[trigger:pipeline(%v/%v/%v:%v)] Firing bot action '%v/%v/%v', branch '%v'...", build.RepoSource, build.RepoOwner, build.RepoName, event, p.RepoSource, p.RepoOwner, p.RepoName, t.BotAction.Branch)
						err := s.fireBot(ctx, *p, t, e)
						if err != nil {
							log.Error().Err(err).Msgf("[trigger:pipeline(%v/%v/%v:%v)] Failed starting bot action '%v/%v/%v', branch '%v'", build.RepoSource, build.RepoOwner, build.RepoName, event, p.RepoSource, p.RepoOwner, p.RepoName, t.BotAction.Branch)
						}
					}

					return nil
				})
			}
		}
	}

	// wait until all concurrent goroutines are done
	err = g.Wait()
	if err != nil {
		return err
	}

	log.Debug().Msgf("[trigger:pipeline(%v/%v/%v:%v)] Fired %v out of %v triggers for %v pipelines", build.RepoSource, build.RepoOwner, build.RepoName, event, firedTriggerCount, triggerCount, len(pipelines))

	return nil
}

func (s *service) FireReleaseTriggers(ctx context.Context, release contracts.Release, event string) error {

	log.Debug().Msgf("[trigger:release(%v/%v/%v-%v:%v] Checking if triggers need to be fired...", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, event)

	pipelines, err := s.databaseClient.GetReleaseTriggers(ctx, release, event)
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

	// limit concurrency using a semaphore
	semaphore := semaphore.NewWeighted(s.triggerConcurrency)
	g, ctx := errgroup.WithContext(ctx)

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

				p := p
				t := t
				e := e

				g.Go(func() error {
					err = semaphore.Acquire(ctx, 1)
					if err != nil {
						return err
					}
					defer semaphore.Release(1)

					// create new context to avoid cancellation impacting execution
					span, _ := opentracing.StartSpanFromContext(ctx, "estafette:AsyncFireReleaseTriggersItem")
					ctx = opentracing.ContextWithSpan(context.Background(), span)
					defer span.Finish()

					if t.BuildAction != nil {
						log.Debug().Msgf("[trigger:release(%v/%v/%v-%v:%v)] Firing build action '%v/%v/%v', branch '%v'...", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, event, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
						err := s.fireBuild(ctx, *p, t, e)
						if err != nil {
							log.Error().Err(err).Msgf("[trigger:release(%v/%v/%v-%v:%v)] Failed starting build action '%v/%v/%v', branch '%v'", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, event, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
						}
					} else if t.ReleaseAction != nil {
						log.Debug().Msgf("[trigger:release(%v/%v/%v-%v:%v)] Firing release action '%v/%v/%v', target '%v', action '%v'...", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, event, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
						err := s.fireRelease(ctx, *p, t, e)
						if err != nil {
							log.Error().Err(err).Msgf("[trigger:release(%v/%v/%v-%v:%v)] Failed starting release action '%v/%v/%v', target '%v', action '%v'", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, event, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
						}
					} else if t.BotAction != nil {
						log.Debug().Msgf("[trigger:release(%v/%v/%v-%v:%v)] Firing bot action '%v/%v/%v', branch '%v'...", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, event, p.RepoSource, p.RepoOwner, p.RepoName, t.BotAction.Branch)
						err := s.fireBot(ctx, *p, t, e)
						if err != nil {
							log.Error().Err(err).Msgf("[trigger:release(%v/%v/%v-%v:%v)] Failed starting bot action '%v/%v/%v', branch '%v'", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, event, p.RepoSource, p.RepoOwner, p.RepoName, t.BotAction.Branch)
						}
					}

					return nil
				})
			}
		}
	}

	// wait until all concurrent goroutines are done
	err = g.Wait()
	if err != nil {
		return err
	}

	log.Debug().Msgf("[trigger:release(%v/%v/%v-%v:%v] Fired %v out of %v triggers for %v pipelines", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, event, firedTriggerCount, triggerCount, len(pipelines))

	return nil
}

func (s *service) FirePubSubTriggers(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) error {

	log.Debug().Msgf("[trigger:pubsub(projects/%v/topics/%v)] Checking if triggers need to be fired...", pubsubEvent.Project, pubsubEvent.Topic)

	// retrieve all pipeline triggers
	pipelines, err := s.databaseClient.GetPubSubTriggers(ctx)
	if err != nil {
		return err
	}

	e := manifest.EstafetteEvent{
		Fired:  true,
		PubSub: &pubsubEvent,
	}

	triggerCount := 0
	firedTriggerCount := 0

	// limit concurrency using a semaphore
	semaphore := semaphore.NewWeighted(s.triggerConcurrency)
	g, ctx := errgroup.WithContext(ctx)

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

				p := p
				t := t
				e := e

				g.Go(func() error {
					err = semaphore.Acquire(ctx, 1)
					if err != nil {
						return err
					}
					defer semaphore.Release(1)

					// create new context to avoid cancellation impacting execution
					span, _ := opentracing.StartSpanFromContext(ctx, "estafette:AsyncFirePubSubTriggersItem")
					ctx = opentracing.ContextWithSpan(context.Background(), span)
					defer span.Finish()

					// create new build for t.Run
					if t.BuildAction != nil {
						log.Debug().Msgf("[trigger:pubsub(projects/%v/topics/%v)] Firing build action '%v/%v/%v', branch '%v'...", pubsubEvent.Project, pubsubEvent.Topic, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
						err := s.fireBuild(ctx, *p, t, e)
						if err != nil {
							log.Error().Err(err).Msgf("[trigger:pubsub(projects/%v/topics/%v)] Failed starting build action'%v/%v/%v', branch '%v'", pubsubEvent.Project, pubsubEvent.Topic, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
						}
					} else if t.ReleaseAction != nil {
						log.Debug().Msgf("[trigger:pubsub(projects/%v/topics/%v)] Firing release action '%v/%v/%v', target '%v', action '%v'...", pubsubEvent.Project, pubsubEvent.Topic, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
						err := s.fireRelease(ctx, *p, t, e)
						if err != nil {
							log.Error().Err(err).Msgf("[trigger:pubsub(projects/%v/topics/%v)] Failed starting release action '%v/%v/%v', target '%v', action '%v'", pubsubEvent.Project, pubsubEvent.Topic, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
						}
					} else if t.BotAction != nil {
						log.Debug().Msgf("[trigger:pubsub(projects/%v/topics/%v)] Firing bot action '%v/%v/%v', branch '%v'...", pubsubEvent.Project, pubsubEvent.Topic, p.RepoSource, p.RepoOwner, p.RepoName, t.BotAction.Branch)
						err := s.fireBot(ctx, *p, t, e)
						if err != nil {
							log.Error().Err(err).Msgf("[trigger:pubsub(projects/%v/topics/%v)] Failed starting bot action '%v/%v/%v', branch '%v'", pubsubEvent.Project, pubsubEvent.Topic, p.RepoSource, p.RepoOwner, p.RepoName, t.BotAction.Branch)
						}
					}

					return nil
				})
			}
		}
	}

	// wait until all concurrent goroutines are done
	err = g.Wait()
	if err != nil {
		return err
	}

	log.Debug().Msgf("[trigger:pubsub(projects/%v/topics/%v)] Fired %v out of %v triggers for %v pipelines", pubsubEvent.Project, pubsubEvent.Topic, firedTriggerCount, triggerCount, len(pipelines))

	return nil
}

func (s *service) FireCronTriggers(ctx context.Context, cronEvent manifest.EstafetteCronEvent) error {

	e := manifest.EstafetteEvent{
		Fired: true,
		Cron:  &cronEvent,
	}

	log.Debug().Msgf("[trigger:cron(%v)] Checking if triggers need to be fired...", cronEvent.Time)

	pipelines, err := s.databaseClient.GetCronTriggers(ctx)
	if err != nil {
		return err
	}

	triggerCount := 0
	firedTriggerCount := 0

	// limit concurrency using a semaphore
	semaphore := semaphore.NewWeighted(s.triggerConcurrency)
	g, ctx := errgroup.WithContext(ctx)

	// check for each trigger whether it should fire
	for _, p := range pipelines {
		for _, t := range p.Triggers {

			log.Debug().Interface("event", cronEvent).Interface("trigger", t).Msgf("[trigger:cron(%v)] Checking if pipeline '%v/%v/%v' trigger should fire...", cronEvent.Time, p.RepoSource, p.RepoOwner, p.RepoName)

			if t.Cron == nil {
				continue
			}

			triggerCount++

			if t.Cron.Fires(&cronEvent) {

				firedTriggerCount++

				p := p
				t := t
				e := e

				g.Go(func() error {
					err = semaphore.Acquire(ctx, 1)
					if err != nil {
						return err
					}
					defer semaphore.Release(1)

					// create new context to avoid cancellation impacting execution
					span, _ := opentracing.StartSpanFromContext(ctx, "estafette:AsyncFireCronTriggersItem")
					ctx = opentracing.ContextWithSpan(context.Background(), span)
					defer span.Finish()

					// create new build for t.Run
					if t.BuildAction != nil {
						log.Debug().Msgf("[trigger:cron(%v)] Firing build action '%v/%v/%v', branch '%v'...", cronEvent.Time, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
						err := s.fireBuild(ctx, *p, t, e)
						if err != nil {
							log.Error().Err(err).Msgf("[trigger:cron(%v)] Failed starting build action'%v/%v/%v', branch '%v'", cronEvent.Time, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
						}
					} else if t.ReleaseAction != nil {
						log.Debug().Msgf("[trigger:cron(%v)] Firing release action '%v/%v/%v', target '%v', action '%v'...", cronEvent.Time, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
						err := s.fireRelease(ctx, *p, t, e)
						if err != nil {
							log.Error().Err(err).Msgf("[trigger:cron(%v)] Failed starting release action '%v/%v/%v', target '%v', action '%v'", cronEvent.Time, p.RepoSource, p.RepoOwner, p.RepoName, t.ReleaseAction.Target, t.ReleaseAction.Action)
						}
					} else if t.BotAction != nil {
						log.Debug().Msgf("[trigger:cron(%v)] Firing bot action '%v/%v/%v', branch '%v'...", cronEvent.Time, p.RepoSource, p.RepoOwner, p.RepoName, t.BotAction.Branch)
						err := s.fireBot(ctx, *p, t, e)
						if err != nil {
							log.Error().Err(err).Msgf("[trigger:cron(%v)] Failed starting bot action '%v/%v/%v', branch '%v'", cronEvent.Time, p.RepoSource, p.RepoOwner, p.RepoName, t.BotAction.Branch)
						}
					}

					return nil
				})
			}
		}
	}

	// wait until all concurrent goroutines are done
	err = g.Wait()
	if err != nil {
		return err
	}

	log.Debug().Msgf("[trigger:cron(%v)] Fired %v out of %v triggers for %v pipelines", cronEvent.Time, firedTriggerCount, triggerCount, len(pipelines))

	return nil
}

func (s *service) fireBuild(ctx context.Context, p contracts.Pipeline, t manifest.EstafetteTrigger, e manifest.EstafetteEvent) error {
	if t.BuildAction == nil {
		return fmt.Errorf("Trigger to fire does not have a 'builds' property, shouldn't get to here")
	}

	// get last build for branch defined in 'builds' section
	lastBuildForBranch, err := s.databaseClient.GetLastPipelineBuildForBranch(ctx, p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
	if err != nil {
		return err
	}

	if lastBuildForBranch == nil {
		return fmt.Errorf("There's no build for pipeline '%v/%v/%v' branch '%v', cannot trigger one", p.RepoSource, p.RepoOwner, p.RepoName, t.BuildAction.Branch)
	}

	// empty the build version so a new one gets created
	lastBuildForBranch.BuildVersion = ""

	// set event that triggers the build
	lastBuildForBranch.Events = []manifest.EstafetteEvent{e}

	_, err = s.CreateBuild(ctx, *lastBuildForBranch)
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
		succeededBuilds, err := s.databaseClient.GetPipelineBuildsByVersion(ctx, p.RepoSource, p.RepoOwner, p.RepoName, versionToRelease, []contracts.Status{contracts.StatusSucceeded}, 1, false)
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
	}, *mft, repoBranch, repoRevision)
	if err != nil {
		return err
	}
	return nil
}

func (s *service) fireBot(ctx context.Context, p contracts.Pipeline, t manifest.EstafetteTrigger, e manifest.EstafetteEvent) error {
	if t.BotAction == nil {
		return fmt.Errorf("Trigger to fire does not have a 'runs' property, shouldn't get to here")
	}

	_, err := s.CreateBot(ctx, contracts.Bot{
		Name:       t.BotAction.Bot,
		RepoSource: p.RepoSource,
		RepoOwner:  p.RepoOwner,
		RepoName:   p.RepoName,
		Events:     []manifest.EstafetteEvent{e},
	}, *p.ManifestObject, p.RepoBranch)
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

func (s *service) getSourceCodeAccessToken(ctx context.Context, repoSource, repoOwner, repoName string) (environmentVariableWithToken map[string]string, err error) {

	switch {
	case githubapi.IsRepoSourceGithub(repoSource):
		var accessToken string
		accessToken, err = s.githubJobVarsFunc(ctx, repoSource, repoOwner, repoName)
		if err != nil {
			return
		}
		environmentVariableWithToken = map[string]string{"ESTAFETTE_GITHUB_API_TOKEN": accessToken}
		return

	case bitbucketapi.IsRepoSourceBitbucket(repoSource):
		var accessToken string
		accessToken, err = s.bitbucketJobVarsFunc(ctx, repoSource, repoOwner, repoName)
		if err != nil {
			return
		}
		environmentVariableWithToken = map[string]string{"ESTAFETTE_BITBUCKET_API_TOKEN": accessToken}
		return

	case cloudsourceapi.IsRepoSourceCloudSource(repoSource):
		var accessToken string
		accessToken, err = s.cloudsourceJobVarsFunc(ctx, repoSource, repoOwner, repoName)
		if err != nil {
			return
		}
		environmentVariableWithToken = map[string]string{"ESTAFETTE_CLOUDSOURCE_API_TOKEN": accessToken}
		return
	}

	return environmentVariableWithToken, fmt.Errorf("Source %v not supported for generating authenticated repository url", repoSource)
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

		err := s.databaseClient.Rename(ctx, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName)
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
	return s.databaseClient.ArchiveComputedPipeline(ctx, repoSource, repoOwner, repoName)
}

func (s *service) Unarchive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	return s.databaseClient.UnarchiveComputedPipeline(ctx, repoSource, repoOwner, repoName)
}

func (s *service) UpdateBuildStatus(ctx context.Context, ciBuilderEvent contracts.EstafetteCiBuilderEvent) (err error) {

	log.Debug().Msgf("UpdateBuildStatus executing...")

	err = ciBuilderEvent.Validate()
	if err != nil {
		return
	}

	switch ciBuilderEvent.JobType {
	case contracts.JobTypeBuild:

		err = s.FinishBuild(ctx, ciBuilderEvent.Git.RepoSource, ciBuilderEvent.Git.RepoOwner, ciBuilderEvent.Git.RepoName, ciBuilderEvent.Build.ID, ciBuilderEvent.Build.BuildStatus)
		if err != nil {
			return err
		}

		log.Debug().Msgf("Updated build status for job %v to %v", ciBuilderEvent.JobName, ciBuilderEvent.Build.BuildStatus)

		return

	case contracts.JobTypeRelease:

		err = s.FinishRelease(ctx, ciBuilderEvent.Git.RepoSource, ciBuilderEvent.Git.RepoOwner, ciBuilderEvent.Git.RepoName, ciBuilderEvent.Release.ID, ciBuilderEvent.Release.ReleaseStatus)
		if err != nil {
			return err
		}

		log.Debug().Msgf("Updated release status for job %v to %v", ciBuilderEvent.JobName, ciBuilderEvent.Release.ReleaseStatus)
		return

	case contracts.JobTypeBot:

		err = s.FinishBot(ctx, ciBuilderEvent.Git.RepoSource, ciBuilderEvent.Git.RepoOwner, ciBuilderEvent.Git.RepoName, ciBuilderEvent.Bot.ID, ciBuilderEvent.Bot.BotStatus)
		if err != nil {
			return err
		}

		log.Debug().Msgf("Updated bot status for job %v to %v", ciBuilderEvent.JobName, ciBuilderEvent.Bot.BotStatus)
		return

	}

	return fmt.Errorf("CiBuilderEvent has invalid JobType %v, not updating status", ciBuilderEvent.JobType)
}

func (s *service) UpdateJobResources(ctx context.Context, ciBuilderEvent contracts.EstafetteCiBuilderEvent) (err error) {

	log.Debug().Msgf("Updating job resources for pod %v", ciBuilderEvent.PodName)

	err = ciBuilderEvent.Validate()
	if err != nil {
		return
	}

	if ciBuilderEvent.PodName != "" {

		s.prometheusClient.AwaitScrapeInterval(ctx)

		maxCPU, err := s.prometheusClient.GetMaxCPUByPodName(ctx, ciBuilderEvent.PodName)
		if err != nil {
			return err
		}

		log.Debug().Msgf("Max cpu usage for pod %v is %v", ciBuilderEvent.PodName, maxCPU)

		maxMemory, err := s.prometheusClient.GetMaxMemoryByPodName(ctx, ciBuilderEvent.PodName)
		if err != nil {
			return err
		}

		log.Debug().Msgf("Max memory usage for pod %v is %v", ciBuilderEvent.PodName, maxMemory)

		jobResources := database.JobResources{
			CPUMaxUsage:    maxCPU,
			MemoryMaxUsage: maxMemory,
		}

		switch ciBuilderEvent.JobType {
		case contracts.JobTypeBuild:

			err = s.databaseClient.UpdateBuildResourceUtilization(ctx, ciBuilderEvent.Git.RepoSource, ciBuilderEvent.Git.RepoOwner, ciBuilderEvent.Git.RepoName, ciBuilderEvent.Build.ID, jobResources)
			if err != nil {
				return err
			}

		case contracts.JobTypeRelease:

			err = s.databaseClient.UpdateReleaseResourceUtilization(ctx, ciBuilderEvent.Git.RepoSource, ciBuilderEvent.Git.RepoOwner, ciBuilderEvent.Git.RepoName, ciBuilderEvent.Release.ID, jobResources)
			if err != nil {
				return err
			}

		case contracts.JobTypeBot:

			err = s.databaseClient.UpdateBotResourceUtilization(ctx, ciBuilderEvent.Git.RepoSource, ciBuilderEvent.Git.RepoOwner, ciBuilderEvent.Git.RepoName, ciBuilderEvent.Bot.ID, jobResources)
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

func (s *service) getBuildCounter(ctx context.Context, build contracts.Build, shortRepoSource string, hasValidManifest bool, mft manifest.EstafetteManifest, pipeline *contracts.Pipeline) (counter int, updatedBuild contracts.Build, err error) {

	// get or set autoincrement and build version
	if build.BuildVersion == "" {
		// get autoincrementing counter
		counter, err = s.databaseClient.GetAutoIncrement(ctx, shortRepoSource, build.RepoOwner, build.RepoName)
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

func (s *service) getBuildJobResources(ctx context.Context, build contracts.Build) database.JobResources {
	// define resource request and limit values to fit reasonably well inside a n1-standard-8 (8 vCPUs, 30 GB memory) machine
	defaultCPUCores := s.config.Jobs.DefaultCPUCores
	if defaultCPUCores == 0 {
		defaultCPUCores = s.config.Jobs.MaxCPUCores
	}
	defaultMemory := s.config.Jobs.DefaultMemoryBytes
	if defaultMemory == 0 {
		defaultMemory = s.config.Jobs.MaxMemoryBytes
	}

	jobResources := database.JobResources{
		CPURequest:    defaultCPUCores,
		CPULimit:      s.config.Jobs.MaxCPUCores,
		MemoryRequest: defaultMemory,
		MemoryLimit:   s.config.Jobs.MaxMemoryBytes,
	}

	// get max usage from previous builds
	measuredResources, nrRecords, err := s.databaseClient.GetPipelineBuildMaxResourceUtilization(ctx, build.RepoSource, build.RepoOwner, build.RepoName, 25)
	if err != nil {
		log.Warn().Err(err).Msgf("Failed retrieving max resource utilization for recent builds of %v/%v/%v, using defaults...", build.RepoSource, build.RepoOwner, build.RepoName)
	} else if nrRecords < 5 {
		log.Debug().Msgf("Retrieved max resource utilization for recent builds of %v/%v/%v only has %v records, using defaults...", build.RepoSource, build.RepoOwner, build.RepoName, nrRecords)
	} else {
		log.Debug().Msgf("Retrieved max resource utilization for recent builds of %v/%v/%v, checking if they are within lower and upper bound...", build.RepoSource, build.RepoOwner, build.RepoName)

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

func (s *service) getReleaseJobResources(ctx context.Context, release contracts.Release) database.JobResources {

	// define resource request and limit values to fit reasonably well inside a n1-standard-8 (8 vCPUs, 30 GB memory) machine
	jobResources := database.JobResources{
		CPURequest:    s.config.Jobs.MaxCPUCores,
		CPULimit:      s.config.Jobs.MaxCPUCores,
		MemoryRequest: s.config.Jobs.MaxMemoryBytes,
		MemoryLimit:   s.config.Jobs.MaxMemoryBytes,
	}

	// get max usage from previous releases
	measuredResources, nrRecords, err := s.databaseClient.GetPipelineReleaseMaxResourceUtilization(ctx, release.RepoSource, release.RepoOwner, release.RepoName, release.Name, 25)
	if err != nil {
		log.Warn().Err(err).Msgf("Failed retrieving max resource utilization for recent releases of %v/%v/%v target %v, using defaults...", release.RepoSource, release.RepoOwner, release.RepoName, release.Name)
	} else if nrRecords < 5 {
		log.Debug().Msgf("Retrieved max resource utilization for recent releases of %v/%v/%v target %v only has %v records, using defaults...", release.RepoSource, release.RepoOwner, release.RepoName, release.Name, nrRecords)
	} else {
		log.Debug().Msgf("Retrieved max resource utilization for recent releases of %v/%v/%v target %v, checking if they are within lower and upper bound...", release.RepoSource, release.RepoOwner, release.RepoName, release.Name)

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

func (s *service) getBotJobResources(ctx context.Context, bot contracts.Bot) database.JobResources {

	// define resource request and limit values to fit reasonably well inside a n1-standard-8 (8 vCPUs, 30 GB memory) machine
	jobResources := database.JobResources{
		CPURequest:    s.config.Jobs.MaxCPUCores,
		CPULimit:      s.config.Jobs.MaxCPUCores,
		MemoryRequest: s.config.Jobs.MaxMemoryBytes,
		MemoryLimit:   s.config.Jobs.MaxMemoryBytes,
	}

	// get max usage from previous releases
	measuredResources, nrRecords, err := s.databaseClient.GetPipelineReleaseMaxResourceUtilization(ctx, bot.RepoSource, bot.RepoOwner, bot.RepoName, bot.Name, 25)
	if err != nil {
		log.Warn().Err(err).Msgf("Failed retrieving max resource utilization for recent bots of %v/%v/%v target %v, using defaults...", bot.RepoSource, bot.RepoOwner, bot.RepoName, bot.Name)
	} else if nrRecords < 5 {
		log.Debug().Msgf("Retrieved max resource utilization for recent bots of %v/%v/%v target %v only has %v records, using defaults...", bot.RepoSource, bot.RepoOwner, bot.RepoName, bot.Name, nrRecords)
	} else {
		log.Debug().Msgf("Retrieved max resource utilization for recent bots of %v/%v/%v target %v, checking if they are within lower and upper bound...", bot.RepoSource, bot.RepoOwner, bot.RepoName, bot.Name)

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

				lastBuilds, innerErr := s.databaseClient.GetPipelineBuilds(ctx, pipelineNameParts[0], pipelineNameParts[1], pipelineNameParts[2], 1, 1, filters, []api.OrderField{}, false)
				if innerErr != nil {
					return
				}
				if len(lastBuilds) == 0 {
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

				lastReleases, innerErr := s.databaseClient.GetPipelineReleases(ctx, pipelineNameParts[0], pipelineNameParts[1], pipelineNameParts[2], 1, 1, filters, []api.OrderField{})
				if innerErr != nil {
					return
				}
				if len(lastReleases) == 0 {
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

func (s *service) isReleaseBlocked(release contracts.Release, mft manifest.EstafetteManifest, repoBranch string) (bool, string, *api.RepositoryReleaseControl) {
	if s.config.BuildControl == nil || s.config.BuildControl.Release == nil {
		return false, "", nil
	}
	var cluster string
	var checkRequired bool
R:
	for _, r := range mft.Releases {
		if r.Name == release.Name {
			for _, stg := range r.Stages {
				if checkRequired, cluster = s.stageContainsRestrictedCluster(stg); checkRequired {
					break R
				}
			}
		}
	}
	if !checkRequired {
		if s.config.BuildControl.Release.RestrictedClusters.Matches(release.Name) {
			cluster = release.Name
		} else {
			// release is not blocked as cluster is not restricted list
			return false, "", nil
		}
	}
	var releaseControl api.RepositoryReleaseControl
	var ok bool
	if releaseControl, ok = s.config.BuildControl.Release.Repositories[release.RepoName]; !ok {
		if releaseControl, ok = s.config.BuildControl.Release.Repositories[api.AllRepositories]; !ok {
			return false, "", nil
		}
	}
	// blocked branches
	if releaseControl.Blocked.Matches(repoBranch) {
		return true, cluster, &releaseControl
	}
	// allowed branches
	if releaseControl.Allowed.Matches(repoBranch) {
		return false, cluster, &releaseControl
	}
	// block everything else if allowed branches are provided
	return len(releaseControl.Allowed) > 0, cluster, &releaseControl
}

func (s *service) stageContainsRestrictedCluster(stage *manifest.EstafetteStage) (bool, string) {
	if creds, ok := stage.CustomProperties["credentials"]; ok {
		creds1 := strings.TrimPrefix(creds.(string), "gke-")
		if s.config.BuildControl.Release.RestrictedClusters.Matches(creds1) {
			return true, creds1
		}
	}
	for _, stg := range stage.ParallelStages {
		if ok, cluster := s.stageContainsRestrictedCluster(stg); ok {
			return true, cluster
		}
	}
	return false, ""
}
