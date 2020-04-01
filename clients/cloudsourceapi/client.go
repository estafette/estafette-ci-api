package cloudsourceapi

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/estafette/estafette-ci-api/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	sourcerepo "google.golang.org/api/sourcerepo/v1"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

// Client is the interface for communicating with the Google Cloud Source Repository api
type Client interface {
	GetAccessToken(ctx context.Context) (accesstoken AccessToken, err error)
	GetAuthenticatedRepositoryURL(ctx context.Context, accesstoken AccessToken, htmlURL string) (url string, err error)
	GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, notification PubSubNotification) (valid bool, manifest string, err error)
	JobVarsFunc(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error)
}

// NewClient creates an githubapi.Client to communicate with the Google Cloud Source Repository api
func NewClient(config config.CloudSourceConfig) (Client, error) {

	ctx := context.Background()
	tokenSource, err := google.DefaultTokenSource(ctx, sourcerepo.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	sourcerepoService, err := sourcerepo.New(oauth2.NewClient(ctx, tokenSource))
	if err != nil {
		return nil, err
	}

	return &client{
		service:     sourcerepoService,
		config:      config,
		tokenSource: tokenSource,
	}, nil
}

type client struct {
	service     *sourcerepo.Service
	config      config.CloudSourceConfig
	tokenSource oauth2.TokenSource
}

// GetAccessToken returns an access token to access the Cloud Source api
func (c *client) GetAccessToken(ctx context.Context) (accesstoken AccessToken, err error) {

	token, err := c.tokenSource.Token()
	if err != nil {
		return
	}

	accesstoken = AccessToken{
		AccessToken:  token.AccessToken,
		Expiry:       token.Expiry,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
	}

	return
}

func (c *client) GetAuthenticatedRepositoryURL(ctx context.Context, accesstoken AccessToken, htmlURL string) (url string, err error) {

	url = strings.Replace(htmlURL, "https://source.developers.google.com", fmt.Sprintf("https://estafette:%v@source.developers.google.com", accesstoken.AccessToken), -1)

	return
}

func (c *client) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, notification PubSubNotification) (exists bool, manifest string, err error) {

	repoSource := notification.GetRepoSource()
	repoOwner := notification.GetRepoOwner()
	repoName := notification.GetRepoName()
	repoRefName := ""
	for _, refUpdate := range notification.RefUpdateEvent.RefUpdates {
		repoRefName = refUpdate.RefName
	}

	// Tempdir to clone the repository
	dir, err := ioutil.TempDir("", "sourcerepo-manifest")
	if err != nil {
		return
	}

	defer os.RemoveAll(dir) // clean up

	// Clones the repository into the given dir, just as a normal git clone does
	_, err = git.PlainClone(dir, false, &git.CloneOptions{
		URL:           fmt.Sprintf("https://estafette:%v@%v/p/%v/r/%v", accesstoken.AccessToken, repoSource, repoOwner, repoName),
		ReferenceName: plumbing.ReferenceName(repoRefName),
		Depth:         10,
	})
	if err != nil {
		return
	}

	estafetteFilename := filepath.Join(dir, ".estafette.yaml")
	if _, fileErr := os.Stat(estafetteFilename); os.IsNotExist(fileErr) {
		exists = false
		manifest = ""

		return
	}

	estafetteFile, err := ioutil.ReadFile(estafetteFilename)
	if err != nil {
		return
	}

	exists = true
	manifest = string(estafetteFile)

	return
}

// JobVarsFunc returns a function that can get an access token and authenticated url for a repository
func (c *client) JobVarsFunc(ctx context.Context) func(context.Context, string, string, string) (string, string, error) {
	return func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
		// get access token
		accesstoken, err := c.GetAccessToken(ctx)
		if err != nil {
			return
		}
		token = accesstoken.AccessToken

		// get authenticated url for the repository
		url, err = c.GetAuthenticatedRepositoryURL(ctx, accesstoken, fmt.Sprintf("https://%v/p/%v/r/%v", repoSource, repoOwner, repoName))
		if err != nil {
			return
		}

		return
	}
}
