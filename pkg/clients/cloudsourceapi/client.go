package cloudsourceapi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"golang.org/x/oauth2"
	"google.golang.org/api/sourcerepo/v1"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

// Client is the interface for communicating with the Google Cloud Source Repository api
//
//go:generate mockgen -package=cloudsourceapi -destination ./mock.go -source=client.go
type Client interface {
	GetAccessToken(ctx context.Context) (accesstoken AccessToken, err error)
	GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, notification PubSubNotification, gitClone func(string, string, string) error) (valid bool, manifest string, err error)
	JobVarsFunc(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error)
}

// NewClient creates an cloudsource.Client to communicate with the Google Cloud Source Repository api
func NewClient(config *api.APIConfig, tokenSource oauth2.TokenSource, sourcerepoService *sourcerepo.Service) Client {
	if config == nil || config.Integrations == nil || config.Integrations.CloudSource == nil || !config.Integrations.CloudSource.Enable {
		return &client{
			enabled: false,
		}
	}

	return &client{
		enabled:     true,
		config:      config,
		service:     sourcerepoService,
		tokenSource: tokenSource,
	}
}

type client struct {
	enabled     bool
	config      *api.APIConfig
	service     *sourcerepo.Service
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

func (c *client) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, notification PubSubNotification, gitClone func(string, string, string) error) (exists bool, manifest string, err error) {

	repoSource := notification.GetRepoSource()
	repoOwner := notification.GetRepoOwner()
	repoName := notification.GetRepoName()
	repoRefName := ""
	for _, refUpdate := range notification.RefUpdateEvent.RefUpdates {
		repoRefName = refUpdate.RefName
	}

	// create /tmp dir if it doesnt exist
	_ = os.Mkdir(os.TempDir(), os.ModeDir)
	// Tempdir to clone the repository
	dir, err := os.MkdirTemp("", "sourcerepo-manifest")
	if err != nil {
		return
	}

	defer os.RemoveAll(dir) // clean up

	gitUrl := fmt.Sprintf("https://estafette:%v@%v/p/%v/r/%v", accesstoken.AccessToken, repoSource, repoOwner, repoName)
	if gitClone == nil {
		err = c.gitClone(dir, gitUrl, repoRefName)
	} else {
		err = gitClone(dir, gitUrl, repoRefName)
	}
	if err != nil {
		return
	}

	estafetteFilename := filepath.Join(dir, ".estafette.yaml")
	if _, fileErr := os.Stat(estafetteFilename); os.IsNotExist(fileErr) {
		exists = false
		manifest = ""

		return
	}

	estafetteFile, err := os.ReadFile(estafetteFilename)
	if err != nil {
		return
	}

	exists = true
	manifest = string(estafetteFile)

	return
}

// JobVarsFunc returns a function that can get an access token and authenticated url for a repository
func (c *client) JobVarsFunc(ctx context.Context) func(context.Context, string, string, string) (string, error) {
	return func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
		// get access token
		accesstoken, err := c.GetAccessToken(ctx)
		if err != nil {
			return
		}
		token = accesstoken.AccessToken

		return
	}
}

func (c *client) gitClone(dir, gitUrl, repoRefName string) error {
	// Clones the repository into the given dir, just as a normal git clone does
	_, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL:           gitUrl,
		ReferenceName: plumbing.ReferenceName(repoRefName),
		Depth:         50,
	})

	return err
}
