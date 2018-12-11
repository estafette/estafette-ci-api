package estafette

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/estafette/estafette-ci-api/cockroach"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/rs/zerolog/log"
)

// BuildService encapsulates build and release creation and re-triggering
type BuildService interface {
	CreateBuild(build contracts.Build, waitForJobToStart bool) (*contracts.Build, error)
	CreateRelease(release contracts.Release, mft manifest.EstafetteManifest, repoBranch, repoRevision string, waitForJobToStart bool) (*contracts.Release, error)
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
	mft, err := manifest.ReadManifest(build.Manifest)
	if err != nil {
		log.Warn().Err(err).Str("manifest", build.Manifest).Msgf("Deserializing Estafette manifest for pipeline %v/%v/%v and revision %v failed, continuing though so developer gets useful feedback", build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision)
	} else {
		hasValidManifest = true
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
		} else {
			// set build version to autoincrement so there's at least a version in the db and gui
			build.BuildVersion = strconv.Itoa(autoincrement)
		}
	} else {
		// get autoincrement from build version
		autoincrementCandidate := build.BuildVersion
		if hasValidManifest && mft.Version.SemVer != nil {
			re := regexp.MustCompile(`^[0-9]+\.[0-9]+\.([0-9]+)(-[0-9a-zA-Z-/]+)?$`)
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
		}
		build.ReleaseTargets = releaseTargets
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
	})
	if err != nil {
		return
	}

	buildID, err := strconv.Atoi(createdBuild.ID)
	if err != nil {
		return
	}

	// get authenticated url
	authenticatedRepositoryURL, environmentVariableWithToken, err := s.getAuthenticatedRepositoryURL(build.RepoSource, build.RepoOwner, build.RepoName)
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
	}

	return
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

	// get authenticated url
	authenticatedRepositoryURL, environmentVariableWithToken, err := s.getAuthenticatedRepositoryURL(release.RepoSource, release.RepoOwner, release.RepoName)
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

	return
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
	}

	return authenticatedRepositoryURL, environmentVariableWithToken, fmt.Errorf("Source %v not supported for generatnig authenticated repository url", repoSource)
}
