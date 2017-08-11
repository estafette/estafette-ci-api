package main

// GithubPushEvent represents a Github webhook push event
type GithubPushEvent struct {
	After        string             `json:"after"`
	Commits      []GithubCommit     `json:"commits"`
	HeadCommit   GithubCommit       `json:"head_commit"`
	Pusher       GithubPusher       `json:"pusher"`
	Repository   GithubRepository   `json:"repository"`
	Installation GithubInstallation `json:"installation"`
}

// GithubInstallation represents an installation of a Github app
type GithubInstallation struct {
	ID int `json:"id"`
}

// GithubCommit represents a Github commit
type GithubCommit struct {
	Author  GithubAuthor `json:"author"`
	Message string       `json:"message"`
	ID      string       `json:"id"`
}

// GithubAuthor represents a Github author
type GithubAuthor struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	UserName string `json:"username"`
}

// GithubPusher represents a Github pusher
type GithubPusher struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// GithubRepository represents a Github repository
type GithubRepository struct {
	GitURL   string `json:"git_url"`
	HTMLURL  string `json:"html_url"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

// GithubAccessTokenResponse represents the response returned when requesting a Github access token
type GithubAccessTokenResponse struct {
}
