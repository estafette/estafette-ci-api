package bitbucket

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPushEventChangeObjectTargetAuthorUser(t *testing.T) {
	t.Run("GetEmailAddressExtractsEmailAddressFromRawField", func(t *testing.T) {

		u := PushEventChangeObjectTargetAuthorUser{
			Raw: "Someones Name \u003csomeone@server.com\u003e",
		}

		// act
		emailAddress := u.GetEmailAddress()

		assert.Equal(t, "someone@server.com", emailAddress)
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
