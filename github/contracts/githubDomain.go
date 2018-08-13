package contracts

import "strings"

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
	return "github.com"
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
