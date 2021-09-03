package githubapi

import (
	"fmt"
	"strings"

	contracts "github.com/estafette/estafette-ci-contracts"
)

const repoSource = "github.com"

// AnyEvent represents any of the Github webhook events, to check if installation is allowed
type AnyEvent struct {
	Installation GithubInstallation `json:"installation"`
	Repository   *Repository        `json:"repository"`
}

func (ae *AnyEvent) GetRepository() string {
	if ae.Repository == nil {
		return ""
	}

	return fmt.Sprintf("%v/%v", repoSource, ae.Repository.FullName)
}

// PushEvent represents a Github webhook push event
type PushEvent struct {
	After        string             `json:"after"`
	Commits      []Commit           `json:"commits"`
	HeadCommit   Commit             `json:"head_commit"`
	Pusher       Pusher             `json:"pusher"`
	Repository   Repository         `json:"repository"`
	Installation GithubInstallation `json:"installation"`
	Ref          string             `json:"ref"`
}

// Installation represents an installation of a Github app
type GithubInstallation struct {
	ID            int                       `json:"id"`
	AppID         int                       `json:"app_id,omitempty"`
	Account       *Account                  `json:"account,omitempty"`
	Organizations []*contracts.Organization `json:"organizations,omitempty"`
}

type Account struct {
	Login string `json:"login"`
}

// Commit represents a Github commit
type Commit struct {
	Author  Author `json:"author"`
	Message string `json:"message"`
	ID      string `json:"id"`
}

// Author represents a Github author
type Author struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	UserName string `json:"username"`
}

// Pusher represents a Github pusher
type Pusher struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// Repository represents a Github repository
type Repository struct {
	GitURL   string `json:"git_url"`
	HTMLURL  string `json:"html_url"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

// AccessToken represents a Github access token
type AccessToken struct {
	ExpiresAt string `json:"expires_at"`
	Token     string `json:"token"`
}

// RepositoryContent represents a file retrieved via the Github api
type RepositoryContent struct {
	Type     string `json:"type"`
	Encoding string `json:"encoding"`
	Size     int    `json:"size"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Content  string `json:"content"`
	Sha      string `json:"sha"`
}

// GetRepoSource returns the repository source
func (pe *PushEvent) GetRepoSource() string {
	return repoSource
}

// GetRepoOwner returns the repository owner
func (pe *PushEvent) GetRepoOwner() string {
	return strings.Split(pe.Repository.FullName, "/")[0]
}

// GetRepoName returns the repository name
func (pe *PushEvent) GetRepoName() string {
	return pe.Repository.Name
}

// GetRepoFullName returns the repository owner and name
func (pe *PushEvent) GetRepoFullName() string {
	return pe.Repository.FullName
}

// GetRepoBranch returns the branch of the push event
func (pe *PushEvent) GetRepoBranch() string {
	return strings.Replace(pe.Ref, "refs/heads/", "", 1)
}

// GetRepoRevision returns the revision of the push event
func (pe *PushEvent) GetRepoRevision() string {
	return pe.After
}

// GetRepository returns the full path to the repository
func (pe *PushEvent) GetRepository() string {
	return fmt.Sprintf("%v/%v", pe.GetRepoSource(), pe.GetRepoFullName())
}

// RepositoryEvent represents a Github webhook repository event
type RepositoryEvent struct {
	Action       string             `json:"action"`
	Changes      RepositoryChanges  `json:"changes"`
	Repository   Repository         `json:"repository"`
	Installation GithubInstallation `json:"installation"`
}

// RepositoryChanges records changes made to a repository
type RepositoryChanges struct {
	Repository RepositoryChangesRepository `json:"repository"`
}

// RepositoryChangesRepository records changes made to a repository
type RepositoryChangesRepository struct {
	Name RepositoryChangesRepositoryName `json:"name"`
}

// RepositoryChangesRepositoryName records changes made to a repository's name
type RepositoryChangesRepositoryName struct {
	From string `json:"from"`
}

// IsValidRenameEvent returns true if all fields for a repo rename are set
func (pe *RepositoryEvent) IsValidRenameEvent() bool {
	return pe.Action == "renamed" && pe.Changes.Repository.Name.From != "" && pe.Repository.FullName != ""
}

// GetRepoSource returns the repository source
func (pe *RepositoryEvent) GetRepoSource() string {
	return repoSource
}

// GetRepoOwner returns the current repository owner
func (pe *RepositoryEvent) GetRepoOwner() string {
	return strings.Split(pe.Repository.FullName, "/")[0]
}

// GetOldRepoOwner returns the former repository owner
func (pe *RepositoryEvent) GetOldRepoOwner() string {
	return pe.GetRepoOwner()
}

// GetNewRepoOwner returns the new repository owner
func (pe *RepositoryEvent) GetNewRepoOwner() string {
	return pe.GetRepoOwner()
}

// GetRepoName returns the current repository name
func (pe *RepositoryEvent) GetRepoName() string {
	return strings.Split(pe.Repository.FullName, "/")[1]
}

// GetOldRepoName returns the former repository name
func (pe *RepositoryEvent) GetOldRepoName() string {
	return pe.Changes.Repository.Name.From
}

// GetNewRepoName returns the new repository name
func (pe *RepositoryEvent) GetNewRepoName() string {
	return pe.GetRepoName()
}

// IsRepoSourceGithub returns true if the repo source is from github
func IsRepoSourceGithub(repoSourceToCompare string) bool {
	return repoSourceToCompare == repoSource
}

type GithubApp struct {
	ID            int                   `json:"id"`
	Name          string                `json:"name"`
	Slug          string                `json:"slug"`
	PrivateKey    string                `json:"pem"`
	WebhookSecret string                `json:"webhook_secret"`
	ClientID      string                `json:"client_id"`
	ClientSecret  string                `json:"client_secret"`
	Installations []*GithubInstallation `json:"installations"`
}
