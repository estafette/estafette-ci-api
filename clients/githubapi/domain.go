package githubapi

import (
	"fmt"
	"strings"
)

const repoSource = "github.com"

// AnyEvent represents any of the Github webhook events, to check if installation is whitelisted
type AnyEvent struct {
	Installation Installation `json:"installation"`
}

// PushEvent represents a Github webhook push event
type PushEvent struct {
	After        string       `json:"after"`
	Commits      []Commit     `json:"commits"`
	HeadCommit   Commit       `json:"head_commit"`
	Pusher       Pusher       `json:"pusher"`
	Repository   Repository   `json:"repository"`
	Installation Installation `json:"installation"`
	Ref          string       `json:"ref"`
}

// Installation represents an installation of a Github app
type Installation struct {
	ID int `json:"id"`
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
	Action       string            `json:"action"`
	Changes      RepositoryChanges `json:"changes"`
	Repository   Repository        `json:"repository"`
	Installation Installation      `json:"installation"`
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

// GetOldRepoOwner returns the repository owner
func (pe *RepositoryEvent) GetOldRepoOwner() string {
	return strings.Split(pe.Repository.FullName, "/")[0]
}

// GetNewRepoOwner returns the repository owner
func (pe *RepositoryEvent) GetNewRepoOwner() string {
	return strings.Split(pe.Repository.FullName, "/")[0]
}

// GetOldRepoName returns the repository name
func (pe *RepositoryEvent) GetOldRepoName() string {
	return pe.Changes.Repository.Name.From
}

// GetNewRepoName returns the repository name
func (pe *RepositoryEvent) GetNewRepoName() string {
	return strings.Split(pe.Repository.FullName, "/")[1]
}

// IsRepoSourceGithub returns true if the repo source is from github
func IsRepoSourceGithub(repoSourceToCompare string) bool {
	return repoSourceToCompare == repoSource
}
