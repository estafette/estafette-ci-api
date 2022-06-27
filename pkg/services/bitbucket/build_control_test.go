package bitbucket

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/bitbucketapi"
)

func Test_isBuildBlocked(t *testing.T) {
	genEvent := func(fullName, projectName, projectKey string) bitbucketapi.RepositoryPushEvent {
		return bitbucketapi.RepositoryPushEvent{
			Repository: bitbucketapi.Repository{
				FullName: fullName,
				Project: &bitbucketapi.Project{
					Name: projectName,
					Key:  projectKey,
				},
			},
		}
	}
	pushEvent1 := genEvent("/test-repo-1", "project-1", "p1")
	pushEvent2 := genEvent("/test-repo-2", "project-2", "p2")
	pushEvent3 := genEvent("/test-repo-3", "project-3", "p3")
	genService := func(b *api.BuildControl) *service {
		return &service{
			config: &api.APIConfig{
				BuildControl: b,
			},
		}
	}
	t.Run("NoAllowedOrBlocked", func(t *testing.T) {
		var s = genService(&api.BuildControl{})
		assertT := assert.New(t)
		assertT.False(s.isBuildBlocked(pushEvent1))
		assertT.False(s.isBuildBlocked(pushEvent2))
		assertT.False(s.isBuildBlocked(pushEvent3))
	})
	t.Run("AllowedProject", func(t *testing.T) {
		var s = genService(&api.BuildControl{
			Bitbucket: &api.BitbucketBuildControl{
				Allowed: &api.BitbucketProjectsRepos{
					Projects: api.List{
						"p1",
					},
				},
			},
		})
		assertT := assert.New(t)
		assertT.False(s.isBuildBlocked(pushEvent1))
		assertT.True(s.isBuildBlocked(pushEvent2))
		assertT.True(s.isBuildBlocked(pushEvent3))
	})
	t.Run("BlockedProject", func(t *testing.T) {
		var s = genService(&api.BuildControl{
			Bitbucket: &api.BitbucketBuildControl{
				Blocked: &api.BitbucketProjectsRepos{
					Projects: api.List{
						"p1",
					},
				},
			},
		})
		assertT := assert.New(t)
		assertT.True(s.isBuildBlocked(pushEvent1))
		assertT.False(s.isBuildBlocked(pushEvent2))
		assertT.False(s.isBuildBlocked(pushEvent3))
	})
	t.Run("AllowedRepo", func(t *testing.T) {
		var s = genService(&api.BuildControl{
			Bitbucket: &api.BitbucketBuildControl{
				Allowed: &api.BitbucketProjectsRepos{
					Repos: api.List{
						"test-repo-1",
					},
				},
			},
		})
		assertT := assert.New(t)
		assertT.False(s.isBuildBlocked(pushEvent1))
		assertT.True(s.isBuildBlocked(pushEvent2))
		assertT.True(s.isBuildBlocked(pushEvent3))
	})
	t.Run("BlockedRepo", func(t *testing.T) {
		var s = genService(&api.BuildControl{
			Bitbucket: &api.BitbucketBuildControl{
				Blocked: &api.BitbucketProjectsRepos{
					Repos: api.List{
						"test-repo-1",
					},
				},
			},
		})
		assertT := assert.New(t)
		assertT.True(s.isBuildBlocked(pushEvent1))
		assertT.False(s.isBuildBlocked(pushEvent2))
		assertT.False(s.isBuildBlocked(pushEvent3))
	})
	t.Run("BlockedRepoAndAllowedProject", func(t *testing.T) {
		var s = genService(&api.BuildControl{
			Bitbucket: &api.BitbucketBuildControl{
				Allowed: &api.BitbucketProjectsRepos{
					Projects: api.List{
						"pr1",
					},
				},
				Blocked: &api.BitbucketProjectsRepos{
					Repos: api.List{
						"test-repo-1",
					},
				},
			},
		})
		assertT := assert.New(t)
		assertT.True(s.isBuildBlocked(pushEvent1))
		assertT.True(s.isBuildBlocked(pushEvent2))
		assertT.True(s.isBuildBlocked(pushEvent3))
	})
}
