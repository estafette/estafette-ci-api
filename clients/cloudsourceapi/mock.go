package cloudsourceapi

import "context"

type MockClient struct {
	GetAccessTokenFunc                func(ctx context.Context) (accesstoken AccessToken, err error)
	GetAuthenticatedRepositoryURLFunc func(ctx context.Context, accesstoken AccessToken, htmlURL string) (url string, err error)
	GetEstafetteManifestFunc          func(ctx context.Context, accesstoken AccessToken, notification PubSubNotification, gitClone func(string, string, string) error) (valid bool, manifest string, err error)
	JobVarsFuncFunc                   func(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error)
}

func (c MockClient) GetAccessToken(ctx context.Context) (accesstoken AccessToken, err error) {
	if c.GetAccessTokenFunc == nil {
		return
	}
	return c.GetAccessTokenFunc(ctx)
}

func (c MockClient) GetAuthenticatedRepositoryURL(ctx context.Context, accesstoken AccessToken, htmlURL string) (url string, err error) {
	if c.GetAuthenticatedRepositoryURLFunc == nil {
		return
	}
	return c.GetAuthenticatedRepositoryURLFunc(ctx, accesstoken, htmlURL)
}

func (c MockClient) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, notification PubSubNotification, gitClone func(string, string, string) error) (valid bool, manifest string, err error) {
	if c.GetEstafetteManifestFunc == nil {
		return
	}
	return c.GetEstafetteManifestFunc(ctx, accesstoken, notification, gitClone)
}

func (c MockClient) JobVarsFunc(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
	if c.JobVarsFuncFunc == nil {
		return func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
	}
	return c.JobVarsFuncFunc(ctx)
}
