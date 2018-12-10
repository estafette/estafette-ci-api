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
	return nil
}

func (th *triggerHandlerImpl) RunRelease(trigger *cockroach.EstafetteTriggerDb, build *contracts.Build, release *manifest.EstafetteRelease) error {
	// todo insert release and start job
	return nil
}

func (th *triggerHandlerImpl) match(expression string, value string) (bool, error) {
	return regexp.MatchString(fmt.Sprintf("^%v$", expression), value)
}
