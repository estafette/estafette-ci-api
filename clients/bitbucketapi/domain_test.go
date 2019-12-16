package bitbucketapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPushEventChangeObjectTargetAuthorUser(t *testing.T) {
	t.Run("GetEmailAddressExtractsEmailAddressFromRawField", func(t *testing.T) {

		u := PushEventChangeObjectTargetAuthor{
			Raw: "Someones Name \u003csomeone@server.com\u003e",
		}

		// act
		emailAddress := u.GetEmailAddress()

		assert.Equal(t, "someone@server.com", emailAddress)
	})
}

func TestAuthor(t *testing.T) {
	t.Run("GetEmailAddressExtractsEmailAddressFromRawField", func(t *testing.T) {

		u := Author{
			Raw: "Someones Name \u003csomeone@server.com\u003e",
		}

		// act
		emailAddress := u.GetEmailAddress()

		assert.Equal(t, "someone@server.com", emailAddress)
	})

	t.Run("GetNameFromRawField", func(t *testing.T) {

		u := Author{
			Raw: "Someones Name \u003csomeone@server.com\u003e",
		}

		// act
		emailAddress := u.GetName()

		assert.Equal(t, "Someones Name", emailAddress)
	})
}

func TestPushEventChangeObjectTarget(t *testing.T) {
	t.Run("GetCommitMessageExtractsTitleFromMessageField", func(t *testing.T) {

		c := PushEventChangeObjectTarget{
			Message: "log api call response body on error only\n\nmore detail",
		}

		// act
		message := c.GetCommitMessage()

		assert.Equal(t, "log api call response body on error only", message)
	})
}

func TestCommit(t *testing.T) {
	t.Run("GetCommitMessageExtractsTitleFromMessageField", func(t *testing.T) {

		c := Commit{
			Message: "log api call response body on error only\n\nmore detail",
		}

		// act
		message := c.GetCommitMessage()

		assert.Equal(t, "log api call response body on error only", message)
	})
}
