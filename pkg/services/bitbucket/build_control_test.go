package bitbucket

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/bitbucketapi"
)

func Test_isBuildBlocked(t *testing.T) {
	pushEvent1 := bitbucketapi.RepositoryPushEvent{
		Repository: bitbucketapi.Repository{
			FullName: "/test-repo-1",
			Project: &bitbucketapi.Project{
				Name: "project-1",
				Key:  "p1",
			},
		},
	}
	pushEvent2 := bitbucketapi.RepositoryPushEvent{
		Repository: bitbucketapi.Repository{
			FullName: "/test-repo-2",
			Project: &bitbucketapi.Project{
				Name: "project-2",
				Key:  "p2",
			},
		},
	}
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
	})
	t.Run("AllowedProject", func(t *testing.T) {
		var s = genService(&api.BuildControl{
			Bitbucket: api.BitbucketBuildControl{
				Allowed: api.BitbucketProjectsRepos{
					Projects: api.List{
						"p1",
					},
				},
			},
		})
		assertT := assert.New(t)
		assertT.False(s.isBuildBlocked(pushEvent1))
		assertT.True(s.isBuildBlocked(pushEvent2))
	})
	t.Run("BlockedProject", func(t *testing.T) {
		var s = genService(&api.BuildControl{
			Bitbucket: api.BitbucketBuildControl{
				Blocked: api.BitbucketProjectsRepos{
					Projects: api.List{
						"p1",
					},
				},
			},
		})
		assertT := assert.New(t)
		assertT.True(s.isBuildBlocked(pushEvent1))
		assertT.False(s.isBuildBlocked(pushEvent2))
	})
	t.Run("AllowedRepo", func(t *testing.T) {
		var s = genService(&api.BuildControl{
			Bitbucket: api.BitbucketBuildControl{
				Allowed: api.BitbucketProjectsRepos{
					Repos: api.List{
						"test-repo-1",
					},
				},
			},
		})
		assertT := assert.New(t)
		assertT.False(s.isBuildBlocked(pushEvent1))
		assertT.True(s.isBuildBlocked(pushEvent2))
	})
	t.Run("BlockedRepo", func(t *testing.T) {
		var s = genService(&api.BuildControl{
			Bitbucket: api.BitbucketBuildControl{
				Blocked: api.BitbucketProjectsRepos{
					Repos: api.List{
						"test-repo-1",
					},
				},
			},
		})
		assertT := assert.New(t)
		assertT.True(s.isBuildBlocked(pushEvent1))
		assertT.False(s.isBuildBlocked(pushEvent2))
	})
}
