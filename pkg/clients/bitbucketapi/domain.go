package bitbucketapi

import (
	"fmt"
	"regexp"
	"strings"

	contracts "github.com/estafette/estafette-ci-contracts"
)

const repoSource = "bitbucket.org"

// AccessToken represents a token to use for api requests
type AccessToken struct {
	AccessToken  string `json:"access_token"`
	Scopes       string `json:"scopes"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

// EventCheck helps to check whether the payload is in new or old format
type EventCheck struct {
	Data       *EventCheckData `json:"data"`
	Repository *Repository     `json:"repository"`
}

type EventCheckData struct {
	Repository *Repository `json:"repository"`
}

func (ec *EventCheck) GetRepository() *Repository {
	if ec.Data != nil {
		return ec.Data.Repository
	}

	return ec.Repository
}

func (ec *EventCheck) GetFullRepository() string {
	if ec.Data == nil && ec.Repository == nil {
		return ""
	}

	if ec.Data != nil {
		if ec.Data.Repository == nil {
			return ""
		}
		return fmt.Sprintf("%v/%v", repoSource, ec.Data.Repository.FullName)
	}

	if ec.Repository == nil {
		return ""
	}
	return fmt.Sprintf("%v/%v", repoSource, ec.Repository.FullName)
}

// RepositoryPushEventEnvelope represents a Bitbucket push event in new envelope
type RepositoryPushEventEnvelope struct {
	Event string              `json:"event"`
	Data  RepositoryPushEvent `json:"data"`
}

// RepositoryPushEvent represents a Bitbucket push event payload
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
	Workspace *Workspace      `json:"workspace"`
	Project   *Project        `json:"project"`
}

type Workspace struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	UUID string `json:"uuid"`
}

// RepositoryLinks represents a collections of links for a Bitbucket repository
type RepositoryLinks struct {
	HTML Link `json:"html"`
}

type Project struct {
	Type string `json:"type"`
	Name string `json:"name"`
	UUID string `json:"uuid"`
	Key  string `json:"key"`
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

// GetCommitMessage extracts the commit message from the Commit Message field
func (t *Commit) GetCommitMessage() string {

	re := regexp.MustCompile(`^([^\n]+)`)
	match := re.FindStringSubmatch(t.Message)

	if len(match) < 2 {
		return ""
	}

	return match[1]
}

// Author represents a Bitbucket author
type Author struct {
	Name     string `json:"display_name"`
	Username string `json:"username"`
	Raw      string `json:"raw"`
}

// GetEmailAddress returns the email address extracted from Author
func (u *Author) GetEmailAddress() string {

	re := regexp.MustCompile(`[^<]+<([^>]+)>`)
	match := re.FindStringSubmatch(u.Raw)

	if len(match) < 2 {
		return ""
	}

	return match[1]
}

// GetName returns the name extracted from Author
func (u *Author) GetName() string {

	re := regexp.MustCompile(`([^<]+)<([^>]+)>`)
	match := re.FindStringSubmatch(u.Raw)

	if len(match) < 2 {
		return ""
	}

	return strings.TrimSpace(match[1])
}

// GetRepoSource returns the repository source
func (pe *RepositoryPushEvent) GetRepoSource() string {
	return repoSource
}

// GetRepoOwner returns the repository owner
func (pe *RepositoryPushEvent) GetRepoOwner() string {
	return strings.Split(pe.Repository.FullName, "/")[0]
}

// GetRepoName returns the repository name
func (pe *RepositoryPushEvent) GetRepoName() string {
	return strings.Split(pe.Repository.FullName, "/")[1]
}

// GetRepoFullName returns the repository owner and name
func (pe *RepositoryPushEvent) GetRepoFullName() string {
	return pe.Repository.FullName
}

// GetRepoBranch returns the branch of the push event
func (pe *RepositoryPushEvent) GetRepoBranch() string {
	return pe.Push.Changes[0].New.Name
}

// GetRepoRevision returns the revision of the push event
func (pe *RepositoryPushEvent) GetRepoRevision() string {
	return pe.Push.Changes[0].New.Target.Hash
}

// GetRepository returns the full path to the repository
func (pe *RepositoryPushEvent) GetRepository() string {
	return fmt.Sprintf("%v/%v", pe.GetRepoSource(), pe.GetRepoFullName())
}

// RepoUpdatedEvent represents a Bitbucket repo:updated event
type RepoUpdatedEvent struct {
	Changes RepoUpdatedChanges `json:"changes"`
}

// RepoUpdatedChanges records changes to the repository name
type RepoUpdatedChanges struct {
	FullName RepoUpdatedChangesName `json:"full_name"`
}

// RepoUpdatedChangesName records changes to the repository name
type RepoUpdatedChangesName struct {
	Old string `json:"old"`
	New string `json:"new"`
}

// IsValidRenameEvent returns true if all fields for a repo rename are set
func (pe *RepoUpdatedEvent) IsValidRenameEvent() bool {
	return pe.Changes.FullName.Old != "" && pe.Changes.FullName.New != ""
}

// GetRepoSource returns the repository source
func (pe *RepoUpdatedEvent) GetRepoSource() string {
	return repoSource
}

// GetOldRepoOwner returns the repository owner
func (pe *RepoUpdatedEvent) GetOldRepoOwner() string {
	return strings.Split(pe.Changes.FullName.Old, "/")[0]
}

// GetNewRepoOwner returns the repository owner
func (pe *RepoUpdatedEvent) GetNewRepoOwner() string {
	return strings.Split(pe.Changes.FullName.New, "/")[0]
}

// GetOldRepoName returns the repository name
func (pe *RepoUpdatedEvent) GetOldRepoName() string {
	return strings.Split(pe.Changes.FullName.Old, "/")[1]
}

// GetNewRepoName returns the repository name
func (pe *RepoUpdatedEvent) GetNewRepoName() string {
	return strings.Split(pe.Changes.FullName.New, "/")[1]
}

// IsRepoSourceBitbucket returns true if the repo source is from bitbucket
func IsRepoSourceBitbucket(repoSourceToCompare string) bool {
	return repoSourceToCompare == repoSource
}

// RepoDeletedEvent represents a Bitbucket repo:deleted event
type RepoDeletedEvent struct {
	Actor      Owner      `json:"actor"`
	Repository Repository `json:"repository"`
}

// GetRepoSource returns the repository source
func (pe *RepoDeletedEvent) GetRepoSource() string {
	return repoSource
}

// GetRepoOwner returns the repository owner
func (pe *RepoDeletedEvent) GetRepoOwner() string {
	return strings.Split(pe.Repository.FullName, "/")[0]
}

// GetRepoName returns the repository name
func (pe *RepoDeletedEvent) GetRepoName() string {
	return strings.Split(pe.Repository.FullName, "/")[1]
}

type BitbucketApp struct {
	Key           string                      `json:"key"`
	Installations []*BitbucketAppInstallation `json:"installations"`
}

type BitbucketAppInstallation struct {
	Key           string                    `json:"key"`
	BaseApiURL    string                    `json:"baseApiUrl"`
	ClientKey     string                    `json:"clientKey"`
	SharedSecret  string                    `json:"sharedSecret"`
	Workspace     *Workspace                `json:"workspace"`
	Organizations []*contracts.Organization `json:"organizations,omitempty"`
}

func (ai *BitbucketAppInstallation) GetWorkspaceUUID() string {
	if ai.Workspace != nil {
		return ai.Workspace.UUID
	}

	re := regexp.MustCompile(`ari:cloud:bitbucket::app/({[^}]+})/.+`)
	match := re.FindStringSubmatch(ai.ClientKey)

	if len(match) > 1 {
		return match[1]
	}

	return ""
}
