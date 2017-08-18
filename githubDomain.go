package main

// GithubPushEvent represents a Github webhook push event
type GithubPushEvent struct {
	After        string             `json:"after"`
	Commits      []GithubCommit     `json:"commits"`
	HeadCommit   GithubCommit       `json:"head_commit"`
	Pusher       GithubPusher       `json:"pusher"`
	Repository   GithubRepository   `json:"repository"`
	Installation GithubInstallation `json:"installation"`
	Ref          string             `json:"ref"`
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

// GithubAccessToken represents a Github access token
type GithubAccessToken struct {
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
