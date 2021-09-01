package bitbucketapi

import (
	"fmt"
	"testing"

	"github.com/golang-jwt/jwt/v4"
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

func TestGetWorkspaceUUID(t *testing.T) {
	t.Run("ExtractUUIDFromClientKey", func(t *testing.T) {
		bai := BitbucketAppInstallation{
			ClientKey: "ari:cloud:bitbucket::app/{f786eeda-2b64-4617-8039-a85895b13619}/estafette-ci",
		}

		// act
		uuid := bai.GetWorkspaceUUID()

		assert.Equal(t, "{f786eeda-2b64-4617-8039-a85895b13619}", uuid)
	})
}

func TestValidationErrorIssuedAt(t *testing.T) {
	t.Run("HasNoRemainingErrorsIfOnlyErrorIsValidationErrorIssuedAt", func(t *testing.T) {
		vErr := new(jwt.ValidationError)
		vErr.Inner = fmt.Errorf("Token used before issued")
		vErr.Errors |= jwt.ValidationErrorIssuedAt

		err := error(vErr)

		validationErr, isValidationError := err.(*jwt.ValidationError)
		hasIssuedAtError := isValidationError && validationErr.Errors&jwt.ValidationErrorIssuedAt != 0
		remainingErrors := validationErr.Errors ^ jwt.ValidationErrorIssuedAt

		assert.False(t, vErr.Errors == 0)
		assert.True(t, isValidationError)
		assert.True(t, hasIssuedAtError)
		assert.Equal(t, uint32(0), remainingErrors)
	})

	t.Run("HasRemainingErrorsIfOtherErrorsBesidesValidationErrorIssuedAt", func(t *testing.T) {
		vErr := new(jwt.ValidationError)
		vErr.Inner = fmt.Errorf("Token used before issued")
		vErr.Errors |= jwt.ValidationErrorIssuedAt
		vErr.Errors |= jwt.ValidationErrorExpired

		err := error(vErr)

		validationErr, isValidationError := err.(*jwt.ValidationError)
		hasIssuedAtError := isValidationError && validationErr.Errors&jwt.ValidationErrorIssuedAt != 0
		remainingErrors := validationErr.Errors ^ jwt.ValidationErrorIssuedAt

		assert.False(t, vErr.Errors == 0)
		assert.True(t, isValidationError)
		assert.True(t, hasIssuedAtError)
		assert.NotEqual(t, uint32(0), remainingErrors)
	})
}
