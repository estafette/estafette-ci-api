package cloudsourceapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPubSubNotification(t *testing.T) {
	t.Run("GetRepoOwnerFromName", func(t *testing.T) {

		u := PubSubNotification{
			Name: "projects/test-project/repos/pubsub-test",
		}

		// act
		repoOwner := u.GetRepoOwner()

		assert.Equal(t, "test-project", repoOwner)
	})

	t.Run("GetRepoNameFromName", func(t *testing.T) {

		u := PubSubNotification{
			Name: "projects/test-project/repos/pubsub-test",
		}

		// act
		repoName := u.GetRepoName()

		assert.Equal(t, "pubsub-test", repoName)
	})

	t.Run("GetRepoBranchFromRefUpdateEvent", func(t *testing.T) {

		u := RefUpdate{
			RefName: "refs/heads/master",
		}

		// act
		repoBranch := u.GetRepoBranch()

		assert.Equal(t, "master", repoBranch)
	})
}
