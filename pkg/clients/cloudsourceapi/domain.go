package cloudsourceapi

import (
	"fmt"
	"strings"
	"time"
)

const repoSource = "source.developers.google.com"

// AccessToken represents a token to use for api requests
type AccessToken struct {
	AccessToken  string    `json:"access_token"`
	Expiry       time.Time `json:"expiry"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
}

// PubSubNotification represents a PubSub message coming from Cloud Source Repo
type PubSubNotification struct {
	Name           string          `json:"name"`
	Url            string          `json:"url"`
	EventTime      time.Time       `json:"eventTime"`
	RefUpdateEvent *RefUpdateEvent `json:"refUpdateEvent,omitempty"`
}

// RefUpdateEvent represents the Update event from Cloud Source Repo
type RefUpdateEvent struct {
	Email      string               `json:"email"`
	RefUpdates map[string]RefUpdate `json:"refUpdates"`
}

// RefUpdate represents the Update for each reference from a Cloud Source Repo
type RefUpdate struct {
	RefName    string `json:"refName"`
	UpdateType string `json:"updateType"`
	OldId      string `json:"oldId"`
	NewId      string `json:"newId"`
}

// GetRepoSource returns the repository source
func (psn *PubSubNotification) GetRepoSource() string {
	return repoSource
}

// GetRepoOwner returns the repository owner
func (psn *PubSubNotification) GetRepoOwner() string {
	return strings.Split(psn.Name, "/")[1]
}

// GetRepoName returns the repository name
func (psn *PubSubNotification) GetRepoName() string {
	return strings.Split(psn.Name, "/")[3]
}

// GetAuthorName returns the revision of the push event
func (rue *RefUpdateEvent) GetAuthorName() string {
	return strings.Split(rue.Email, "@")[0]
}

// GetRepoBranch returns the branch of the push event
func (ru *RefUpdate) GetRepoBranch() string {
	return strings.Replace(ru.RefName, "refs/heads/", "", 1)
}

// GetRepository returns the full path to the repository
func (psn *PubSubNotification) GetRepository() string {
	return fmt.Sprintf("%v/%v", psn.GetRepoSource(), psn.Name)
}

// IsRepoSourceCloudSource returns true if the repo source is from cloud source
func IsRepoSourceCloudSource(repoSourceToCompare string) bool {
	return repoSourceToCompare == repoSource
}
