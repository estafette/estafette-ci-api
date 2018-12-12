package estafette

import (
	"fmt"
	"regexp"

	"github.com/estafette/estafette-ci-manifest"

	"github.com/estafette/estafette-ci-api/cockroach"
	contracts "github.com/estafette/estafette-ci-contracts"
)

// TriggerHandler handles triggers
type TriggerHandler interface {
	FireStartBuildTriggers(build *contracts.Build) error
	FireFinishBuildTriggers(build *contracts.Build) error
	FireStartReleaseTriggers(release *contracts.Release) error
	FireFinishReleaseTriggers(release *contracts.Release) error
}

type triggerHandlerImpl struct {
	cockroachDBClient cockroach.DBClient
	ciBuilderClient   CiBuilderClient
}

// NewTriggerHandler returns a new estafette.TriggerHandler
func NewTriggerHandler(cockroachDBClient cockroach.DBClient, ciBuilderClient CiBuilderClient) (triggerHandler TriggerHandler) {

	triggerHandler = &triggerHandlerImpl{
		cockroachDBClient: cockroachDBClient,
		ciBuilderClient:   ciBuilderClient,
	}

	return
}

func (th *triggerHandlerImpl) FireStartBuildTriggers(build *contracts.Build) (err error) {

	triggers, err := th.cockroachDBClient.GetTriggers(build.RepoSource, build.RepoOwner, build.RepoName, "pipeline-build-started")
	if err != nil {
		return err
	}

	triggersToFire := []*cockroach.EstafetteTriggerDb{}
	for _, t := range triggers {
		branchMatch, err := th.match(t.Trigger.Filter.Branch, build.RepoBranch)
		if err != nil {
			return err
		}
		if !branchMatch {
			continue
		}

		triggersToFire = append(triggersToFire, t)
	}

	return nil
}

func (th *triggerHandlerImpl) FireFinishBuildTriggers(build *contracts.Build) (err error) {

	triggers, err := th.cockroachDBClient.GetTriggers(build.RepoSource, build.RepoOwner, build.RepoName, "pipeline-build-finished")
	if err != nil {
		return err
	}

	triggersToFire := []*cockroach.EstafetteTriggerDb{}
	for _, t := range triggers {
		branchMatch, err := th.match(t.Trigger.Filter.Branch, build.RepoBranch)
		if err != nil {
			return err
		}
		if !branchMatch {
			continue
		}

		statusMatch, err := th.match(t.Trigger.Filter.Status, build.BuildStatus)
		if err != nil {
			return err
		}
		if !statusMatch {
			continue
		}

		triggersToFire = append(triggersToFire, t)
	}

	return nil
}

func (th *triggerHandlerImpl) FireStartReleaseTriggers(release *contracts.Release) (err error) {

	triggers, err := th.cockroachDBClient.GetTriggers(release.RepoSource, release.RepoOwner, release.RepoName, "pipeline-release-started")
	if err != nil {
		return err
	}

	// get build for release
	builds, err := th.cockroachDBClient.GetPipelineBuildsByVersion(release.RepoSource, release.RepoOwner, release.RepoName, release.ReleaseVersion, false)
	if err != nil {
		return err
	}
	if len(builds) == 0 {
		return fmt.Errorf("No builds for release")
	}

	triggersToFire := []*cockroach.EstafetteTriggerDb{}
	for _, t := range triggers {
		branchMatch, err := th.match(t.Trigger.Filter.Branch, builds[0].RepoBranch)
		if err != nil {
			return err
		}
		if !branchMatch {
			continue
		}

		targetMatch, err := th.match(t.Trigger.Filter.Target, release.Name)
		if err != nil {
			return err
		}
		if !targetMatch {
			continue
		}

		actionMatch, err := th.match(t.Trigger.Filter.Action, release.Action)
		if err != nil {
			return err
		}
		if !actionMatch {
			continue
		}

		triggersToFire = append(triggersToFire, t)
	}

	return nil
}

func (th *triggerHandlerImpl) FireFinishReleaseTriggers(release *contracts.Release) (err error) {

	triggers, err := th.cockroachDBClient.GetTriggers(release.RepoSource, release.RepoOwner, release.RepoName, "pipeline-release-finished")
	if err != nil {
		return err
	}

	// get build for release
	builds, err := th.cockroachDBClient.GetPipelineBuildsByVersion(release.RepoSource, release.RepoOwner, release.RepoName, release.ReleaseVersion, false)
	if err != nil {
		return err
	}
	if len(builds) == 0 {
		return fmt.Errorf("No builds for release")
	}

	triggersToFire := []*cockroach.EstafetteTriggerDb{}
	for _, t := range triggers {
		branchMatch, err := th.match(t.Trigger.Filter.Branch, builds[0].RepoBranch)
		if err != nil {
			return err
		}
		if !branchMatch {
			continue
		}

		statusMatch, err := th.match(t.Trigger.Filter.Status, release.ReleaseStatus)
		if err != nil {
			return err
		}
		if !statusMatch {
			continue
		}

		targetMatch, err := th.match(t.Trigger.Filter.Target, release.Name)
		if err != nil {
			return err
		}
		if !targetMatch {
			continue
		}

		actionMatch, err := th.match(t.Trigger.Filter.Action, release.Action)
		if err != nil {
			return err
		}
		if !actionMatch {
			continue
		}

		triggersToFire = append(triggersToFire, t)
	}

	return nil
}

func (th *triggerHandlerImpl) Run(triggers []*cockroach.EstafetteTriggerDb) error {

	for _, t := range triggers {
		build, err := th.cockroachDBClient.GetLastPipelineBuildForBranch(t.RepoSource, t.RepoOwner, t.RepoName, t.Trigger.Run.Branch)
		if err != nil {
			continue
		}

		// check whether the trigger is defined as a build trigger
		for _, bt := range build.ManifestObject.Triggers {
			if bt.Equals(t.Trigger) {
				err = th.RunBuild(t, build)
			}
		}

		// check whether the trigger is defined as a release trigger
		for _, r := range build.ManifestObject.Releases {
			for _, rt := range r.Triggers {
				if rt.Equals(t.Trigger) {
					err = th.RunRelease(t, build, r)
				}
			}
		}
	}

	return nil
}

func (th *triggerHandlerImpl) RunBuild(trigger *cockroach.EstafetteTriggerDb, build *contracts.Build) error {
	// todo insert build and start job

	// switch build.RepoSource {
	// case "github.com":

	// // get autoincrement number
	// autoincrement, err := th.cockroachDBClient.GetAutoIncrement("github", pushEvent.Repository.FullName)
	// if err != nil {
	// 	log.Warn().Err(err).
	// 		Msgf("Failed generating autoincrement for Github repository %v", pushEvent.Repository.FullName)
	// }

	// // set build version number
	// buildVersion := build.ManifestObject.Version.Version(manifest.EstafetteVersionParams{
	// 	AutoIncrement: autoincrement,
	// 	Branch:        pushEvent.GetRepoBranch(),
	// 	Revision:      pushEvent.GetRepoRevision(),
	// })

	// // store build in db
	// insertedBuild, err := th.cockroachDBClient.InsertBuild(contracts.Build{
	// 	RepoSource:     build.RepoSource,
	// 	RepoOwner:      build.RepoOwner,
	// 	RepoName:       build.RepoName,
	// 	RepoBranch:     build.RepoBranch,
	// 	RepoRevision:   build.RepoRevision,
	// 	BuildVersion:   buildVersion,
	// 	BuildStatus:    "running",
	// 	Labels:         failedBuild.Labels,
	// 	ReleaseTargets: failedBuild.ReleaseTargets,
	// 	Manifest:       failedBuild.Manifest,
	// 	Commits:        failedBuild.Commits,
	// })
	// if err != nil {
	// 	errorMessage := fmt.Sprintf("Failed inserting build into db for rebuilding version %v of repository %v/%v/%v for build command issued by %v", buildCommand.BuildVersion, buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, user)
	// 	log.Error().Err(err).Msg(errorMessage)
	// 	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
	// }

	// buildID, err := strconv.Atoi(insertedBuild.ID)
	// if err != nil {
	// 	log.Warn().Err(err).Msgf("Failed to convert build id %v to int for build command issued by %v", insertedBuild.ID, user)
	// }

	// // get authenticated url
	// var authenticatedRepositoryURL string
	// var environmentVariableWithToken map[string]string
	// var gitSource string
	// switch build.RepoSource {
	// case "github.com":
	// 	var accessToken string
	// 	accessToken, authenticatedRepositoryURL, err = h.githubJobVarsFunc(buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName)
	// 	if err != nil {
	// 		errorMessage := fmt.Sprintf("Retrieving access token and authenticated github url for repository %v/%v/%v failed for build command issued by %v", buildCommand.BuildVersion, buildCommand.RepoSource, buildCommand.RepoOwner, user)
	// 		log.Error().Err(err).Msg(errorMessage)
	// 		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
	// 	}
	// 	environmentVariableWithToken = map[string]string{"ESTAFETTE_GITHUB_API_TOKEN": accessToken}
	// 	gitSource = "github"

	// case "bitbucket.org":
	// 	var accessToken string
	// 	accessToken, authenticatedRepositoryURL, err = h.bitbucketJobVarsFunc(buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName)
	// 	if err != nil {
	// 		errorMessage := fmt.Sprintf("Retrieving access token and authenticated bitbucket url for repository %v/%v/%v failed for build command issued by %v", buildCommand.BuildVersion, buildCommand.RepoSource, buildCommand.RepoOwner, user)
	// 		log.Error().Err(err).Msg(errorMessage)
	// 		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
	// 	}
	// 	environmentVariableWithToken = map[string]string{"ESTAFETTE_BITBUCKET_API_TOKEN": accessToken}
	// 	gitSource = "bitbucket"
	// }

	// manifest, err := manifest.ReadManifest(failedBuild.Manifest)
	// if err != nil {
	// 	errorMessage := fmt.Sprintf("Failed reading manifest for rebuilding version %v of repository %v/%v/%v for build command issued by %v", buildCommand.BuildVersion, buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, user)
	// 	log.Error().Err(err).Msg(errorMessage)
	// 	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
	// }

	// // inject steps
	// manifest, err = InjectSteps(manifest, manifest.Builder.Track, gitSource)
	// if err != nil {
	// 	errorMessage := fmt.Sprintf("Failed injecting steps into manifest for rebuilding version %v of repository %v/%v/%v for build command issued by %v", buildCommand.BuildVersion, buildCommand.RepoSource, buildCommand.RepoOwner, buildCommand.RepoName, user)
	// 	log.Error().Err(err).Msg(errorMessage)
	// 	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": http.StatusText(http.StatusInternalServerError), "message": errorMessage})
	// }

	// // get autoincrement from build version
	// autoincrement := 0
	// if manifest.Version.SemVer != nil {

	// 	re := regexp.MustCompile(`^[0-9]+\.[0-9]+\.([0-9]+)(-[0-9a-zA-Z-/]+)?$`)
	// 	match := re.FindStringSubmatch(buildCommand.BuildVersion)

	// 	if len(match) > 1 {
	// 		autoincrement, err = strconv.Atoi(match[1])
	// 	}
	// }

	// // define ci builder params
	// ciBuilderParams := CiBuilderParams{
	// 	JobType:              "build",
	// 	RepoSource:           failedBuild.RepoSource,
	// 	RepoOwner:            failedBuild.RepoOwner,
	// 	RepoName:             failedBuild.RepoName,
	// 	RepoURL:              authenticatedRepositoryURL,
	// 	RepoBranch:           failedBuild.RepoBranch,
	// 	RepoRevision:         failedBuild.RepoRevision,
	// 	EnvironmentVariables: environmentVariableWithToken,
	// 	Track:                manifest.Builder.Track,
	// 	AutoIncrement:        autoincrement,
	// 	VersionNumber:        failedBuild.BuildVersion,
	// 	Manifest:             manifest,
	// 	BuildID:              buildID,
	// }

	// // create ci builder job
	// go func(ciBuilderParams CiBuilderParams) {
	// 	_, err := h.ciBuilderClient.CreateCiBuilderJob(ciBuilderParams)
	// 	if err != nil {
	// 		log.Error().Err(err).Msgf("Failed creating rebuild job for %v/%v/%v/%v/%v version %v", ciBuilderParams.RepoSource, ciBuilderParams.RepoOwner, ciBuilderParams.RepoName, ciBuilderParams.RepoBranch, ciBuilderParams.RepoRevision, ciBuilderParams.VersionNumber)
	// 	}
	// }(ciBuilderParams)

	return nil
}

func (th *triggerHandlerImpl) RunRelease(trigger *cockroach.EstafetteTriggerDb, build *contracts.Build, release *manifest.EstafetteRelease) error {
	// todo insert release and start job
	return nil
}

func (th *triggerHandlerImpl) match(expression string, value string) (bool, error) {
	return regexp.MatchString(fmt.Sprintf("^%v$", expression), value)
}
