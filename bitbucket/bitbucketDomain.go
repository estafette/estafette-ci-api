package bitbucket

import (
	"regexp"
)

// RepositoryPushEvent represents a Bitbucket push event
type RepositoryPushEvent struct {
	Actor      Owner      `json:"actor"`
	Repository Repository `json:"repository"`
	Push       PushEvent  `json:"push"`
}

// PushEvent represents a Bitbucket push event push info
type PushEvent struct {
	Changes []PushEventChange `json:"changes"`
}

// PushEventChange represents a Bitbucket push change
type PushEventChange struct {
	New       *PushEventChangeObject `json:"new,omitempty"`
	Old       *PushEventChangeObject `json:"old,omitempty"`
	Created   bool                   `json:"created"`
	Closed    bool                   `json:"closed"`
	Forced    bool                   `json:"forced"`
	Commits   []Commit               `json:"commits"`
	Truncated bool                   `json:"truncated"`
}

// PushEventChangeObject represents the state of the reference after a push
type PushEventChangeObject struct {
	Type   string                      `json:"type"`
	Name   string                      `json:"name,omitempty"`
	Target PushEventChangeObjectTarget `json:"target"`
}

// PushEventChangeObjectTarget represents the target of a change
type PushEventChangeObjectTarget struct {
	Hash    string                            `json:"hash"`
	Author  PushEventChangeObjectTargetAuthor `json:"author"`
	Message string                            `json:"message"`
}

// GetCommitMessage extracts the commit message from the Commit Message field
func (t *PushEventChangeObjectTarget) GetCommitMessage() string {

	re := regexp.MustCompile(`^([^\n]+)`)
	match := re.FindStringSubmatch(t.Message)

	if len(match) < 2 {
		return ""
	}

	return match[1]
}

// PushEventChangeObjectTargetAuthor represents the author of a commit
type PushEventChangeObjectTargetAuthor struct {
	Name     string `json:"display_name"`
	Username string `json:"username"`
	Raw      string `json:"raw"`
}

// GetEmailAddress returns the email address extracted from PushEventChangeObjectTargetAuthorUser
func (u *PushEventChangeObjectTargetAuthor) GetEmailAddress() string {

	re := regexp.MustCompile(`[^<]+<([^>]+)>`)
	match := re.FindStringSubmatch(u.Raw)

	if len(match) < 2 {
		return ""
	}

	return match[1]
}

// Owner represents a Bitbucket owner
type Owner struct {
	Type        string `json:"type"`
	UserName    string `json:"username"`
	DisplayName string `json:"display_name"`
}

// Repository represents a Bitbucket repository
type Repository struct {
	Name      string          `json:"name"`
	FullName  string          `json:"full_name"`
	Owner     Owner           `json:"owner"`
	IsPrivate bool            `json:"is_private"`
	Scm       string          `json:"scm"`
	Links     RepositoryLinks `json:"links"`
}

// RepositoryLinks represents a collections of links for a Bitbucket repository
type RepositoryLinks struct {
	HTML Link `json:"html"`
}

// Link represents a single link for Bitbucket
type Link struct {
	Href string `json:"href"`
}

// Commit represents a Bitbucket commit
type Commit struct {
	Author  Author `json:"author"`
	Date    string `json:"date"`
	Hash    string `json:"hash"`
	Message string `json:"message"`
}

// Author represents a Bitbucket author
type Author struct {
	Raw  string `json:"raw"`
	Type string `json:"type"`
	User Owner  `json:"user,omitempty"`
}

// AccessToken represents a token to use for api requests
type AccessToken struct {
	AccessToken  string `json:"access_token"`
	Scopes       string `json:"scopes"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}
