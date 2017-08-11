package main

// BitbucketRepositoryPushEvent represents a Bitbucket webhook push event
type BitbucketRepositoryPushEvent struct {
	Actor      BitbucketOwner      `json:"actor"`
	Repository BitbucketRepository `json:"repository"`
	Push       BitbucketPushEvent  `json:"push"`
}

// BitbucketPushEvent represents a Bitbucket push event push info
type BitbucketPushEvent struct {
	Changes []BitbucketPushEventChange `json:"changes"`
}

// BitbucketPushEventChange represents a Bitbucket push change
type BitbucketPushEventChange struct {
	New       BitbucketPushEventChangeObject `json:"new,omitempty"`
	Old       BitbucketPushEventChangeObject `json:"old,omitempty"`
	Created   bool                           `json:"created"`
	Closed    bool                           `json:"closed"`
	Forced    bool                           `json:"forced"`
	Commits   []BitbucketCommit              `json:"commits"`
	Truncated bool                           `json:"truncated"`
}

// BitbucketPushEventChangeObject represents the state of the reference after a push
type BitbucketPushEventChangeObject struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

// BitbucketOwner represents a Bitbucket owern
type BitbucketOwner struct {
	Type        string `json:"type"`
	UserName    string `json:"username"`
	DisplayName string `json:"display_name"`
}

// BitbucketRepository represents a Bitbucket repository
type BitbucketRepository struct {
	Name      string         `json:"name"`
	FullName  string         `json:"full_name"`
	Owner     BitbucketOwner `json:"owner"`
	IsPrivate bool           `json:"is_private"`
	Scm       string         `json:"scm"`
}

// BitbucketCommit represents a Bitbucket commit
type BitbucketCommit struct {
	Author BitbucketAuthor `json:"author"`

	Date    string `json:"date"`
	Hash    string `json:"hash"`
	Message string `json:"message"`
}

// BitbucketAuthor represents a Bitbucket author
type BitbucketAuthor struct {
	Raw  string         `json:"raw"`
	Type string         `json:"type"`
	User BitbucketOwner `json:"user,omitempty"`
}