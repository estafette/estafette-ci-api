package github

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/githubapi"
)

func Test_isBuildBlocked(t *testing.T) {
	genEvent := func(fullName string) githubapi.PushEvent {
		return githubapi.PushEvent{
			Repository: githubapi.Repository{
				FullName: fullName,
				Name:     strings.Split(fullName, "/")[1],
			},
		}
	}
	pushEvent1 := genEvent("estafette/test-repo-1")
	pushEvent2 := genEvent("some-org/test-repo-2")
	pushEvent3 := genEvent("estafette/test-repo-3")
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
	t.Run("AllowedRepo", func(t *testing.T) {
		var s = genService(&api.BuildControl{
			Github: &api.GithubBuildControl{
				Allowed: api.List{
					"test-repo-1",
					"some-org/test-repo-2",
				},
			},
		})
		assertT := assert.New(t)
		assertT.False(s.isBuildBlocked(pushEvent1))
		assertT.False(s.isBuildBlocked(pushEvent2))
		assertT.True(s.isBuildBlocked(pushEvent3))
	})
	t.Run("BlockedRepo", func(t *testing.T) {
		var s = genService(&api.BuildControl{
			Github: &api.GithubBuildControl{
				Blocked: api.List{
					"test-repo-1",
					"estafette/test-repo-3",
				},
			},
		})
		assertT := assert.New(t)
		assertT.True(s.isBuildBlocked(pushEvent1))
		assertT.False(s.isBuildBlocked(pushEvent2))
		assertT.True(s.isBuildBlocked(pushEvent3))
	})
}
