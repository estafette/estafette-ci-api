package githubapi

import (
	"context"
)

type MockClient struct {
	GetGithubAppTokenFunc             func(ctx context.Context) (token string, err error)
	GetInstallationIDFunc             func(ctx context.Context, repoOwner string) (installationID int, err error)
	GetInstallationTokenFunc          func(ctx context.Context, installationID int) (token AccessToken, err error)
	GetAuthenticatedRepositoryURLFunc func(ctx context.Context, accesstoken AccessToken, htmlURL string) (url string, err error)
	GetEstafetteManifestFunc          func(ctx context.Context, accesstoken AccessToken, event PushEvent) (valid bool, manifest string, err error)
	JobVarsFuncFunc                   func(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error)
}

func (c MockClient) GetGithubAppToken(ctx context.Context) (token string, err error) {
	if c.GetGithubAppTokenFunc == nil {
		return
	}
	return c.GetGithubAppTokenFunc(ctx)
}

func (c MockClient) GetInstallationID(ctx context.Context, repoOwner string) (installationID int, err error) {
	if c.GetInstallationIDFunc == nil {
		return
	}
	return c.GetInstallationIDFunc(ctx, repoOwner)
}

func (c MockClient) GetInstallationToken(ctx context.Context, installationID int) (token AccessToken, err error) {
	if c.GetInstallationTokenFunc == nil {
		return
	}
	return c.GetInstallationTokenFunc(ctx, installationID)
}

func (c MockClient) GetAuthenticatedRepositoryURL(ctx context.Context, accesstoken AccessToken, htmlURL string) (url string, err error) {
	if c.GetAuthenticatedRepositoryURLFunc == nil {
		return
	}
	return c.GetAuthenticatedRepositoryURLFunc(ctx, accesstoken, htmlURL)
}

func (c MockClient) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event PushEvent) (valid bool, manifest string, err error) {
	if c.GetEstafetteManifestFunc == nil {
		return
	}
	return c.GetEstafetteManifestFunc(ctx, accesstoken, event)
}

func (c MockClient) JobVarsFunc(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
	if c.JobVarsFuncFunc == nil {
		return nil
	}
	return c.JobVarsFuncFunc(ctx)
}
