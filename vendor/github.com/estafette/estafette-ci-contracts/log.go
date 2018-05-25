package contracts

import "time"

// BuildLog represents a build log for a specific revision
type BuildLog struct {
	ID           string    `jsonapi:"primary,build-logs"`
	RepoSource   string    `jsonapi:"attr,repo-source"`
	RepoOwner    string    `jsonapi:"attr,repo-owner"`
	RepoName     string    `jsonapi:"attr,repo-name"`
	RepoBranch   string    `jsonapi:"attr,repo-branch"`
	RepoRevision string    `jsonapi:"attr,repo-revision"`
	LogText      string    `jsonapi:"attr,log-text"`
	InsertedAt   time.Time `jsonapi:"attr,inserted-at"`
	UpdatedAt    time.Time `jsonapi:"attr,updated-at"`
}
