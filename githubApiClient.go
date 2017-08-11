package main

// GithubApiClient is the object to perform Github api calls with
type GithubApiClient struct {
	baseURL string
}

// New returns an initialized APIClient
func New() *GithubApiClient {

	return &GithubApiClient{
		baseURL: "https://api.github.com/",
	}
}

func (gh *GithubApiClient) listInstallations() (i []GithubAppInstallation, err error) {

	return
}
