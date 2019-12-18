package estafette

import (
	"context"

	"github.com/estafette/estafette-ci-api/clients/builderapi"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
)

type MockService struct {
	CreateBuildFunc          func(ctx context.Context, build contracts.Build, waitForJobToStart bool) (b *contracts.Build, err error)
	FinishBuildFunc          func(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus string) (err error)
	CreateReleaseFunc        func(ctx context.Context, release contracts.Release, mft manifest.EstafetteManifest, repoBranch, repoRevision string, waitForJobToStart bool) (r *contracts.Release, err error)
	FinishReleaseFunc        func(ctx context.Context, repoSource, repoOwner, repoName string, releaseID int, releaseStatus string) (err error)
	FireGitTriggersFunc      func(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (err error)
	FirePipelineTriggersFunc func(ctx context.Context, build contracts.Build, event string) (err error)
	FireReleaseTriggersFunc  func(ctx context.Context, release contracts.Release, event string) (err error)
	FirePubSubTriggersFunc   func(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) (err error)
	FireCronTriggersFunc     func(ctx context.Context) (err error)
	RenameFunc               func(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
	UpdateBuildStatusFunc    func(ctx context.Context, event builderapi.CiBuilderEvent) (err error)
	UpdateJobResourcesFunc   func(ctx context.Context, event builderapi.CiBuilderEvent) (err error)
}

func (s *MockService) CreateBuild(ctx context.Context, build contracts.Build, waitForJobToStart bool) (b *contracts.Build, err error) {
	if s.CreateBuildFunc == nil {
		return
	}
	return s.CreateBuildFunc(ctx, build, waitForJobToStart)
}

func (s *MockService) FinishBuild(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus string) (err error) {
	if s.FinishBuildFunc == nil {
		return
	}
	return s.FinishBuildFunc(ctx, repoSource, repoOwner, repoName, buildID, buildStatus)
}

func (s *MockService) CreateRelease(ctx context.Context, release contracts.Release, mft manifest.EstafetteManifest, repoBranch, repoRevision string, waitForJobToStart bool) (r *contracts.Release, err error) {
	if s.CreateReleaseFunc == nil {
		return
	}
	return s.CreateReleaseFunc(ctx, release, mft, repoBranch, repoRevision, waitForJobToStart)
}

func (s *MockService) FinishRelease(ctx context.Context, repoSource, repoOwner, repoName string, releaseID int, releaseStatus string) (err error) {
	if s.FinishReleaseFunc == nil {
		return
	}
	return s.FinishReleaseFunc(ctx, repoSource, repoOwner, repoName, releaseID, releaseStatus)
}

func (s *MockService) FireGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (err error) {
	if s.FireGitTriggersFunc == nil {
		return
	}
	return s.FireGitTriggersFunc(ctx, gitEvent)
}

func (s *MockService) FirePipelineTriggers(ctx context.Context, build contracts.Build, event string) (err error) {
	if s.FirePipelineTriggersFunc == nil {
		return
	}
	return s.FirePipelineTriggersFunc(ctx, build, event)
}

func (s *MockService) FireReleaseTriggers(ctx context.Context, release contracts.Release, event string) (err error) {
	if s.FireReleaseTriggersFunc == nil {
		return
	}
	return s.FireReleaseTriggersFunc(ctx, release, event)
}

func (s *MockService) FirePubSubTriggers(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) (err error) {
	if s.FirePubSubTriggersFunc == nil {
		return
	}
	return s.FirePubSubTriggersFunc(ctx, pubsubEvent)
}

func (s *MockService) FireCronTriggers(ctx context.Context) (err error) {
	if s.FireCronTriggersFunc == nil {
		return
	}
	return s.FireCronTriggersFunc(ctx)
}

func (s *MockService) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	if s.RenameFunc == nil {
		return
	}
	return s.RenameFunc(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *MockService) UpdateBuildStatus(ctx context.Context, event builderapi.CiBuilderEvent) (err error) {
	if s.UpdateBuildStatusFunc == nil {
		return
	}
	return s.UpdateBuildStatusFunc(ctx, event)
}

func (s *MockService) UpdateJobResources(ctx context.Context, event builderapi.CiBuilderEvent) (err error) {
	if s.UpdateJobResourcesFunc == nil {
		return
	}
	return s.UpdateJobResourcesFunc(ctx, event)
}
