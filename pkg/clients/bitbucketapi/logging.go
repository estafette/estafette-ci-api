package bitbucketapi

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
)

// NewLoggingClient returns a new instance of a logging Client.
func NewLoggingClient(c Client) Client {
	return &loggingClient{c, "bitbucketapi"}
}

type loggingClient struct {
	Client Client
	prefix string
}

func (c *loggingClient) GetAccessTokenByInstallation(ctx context.Context, installation BitbucketAppInstallation) (accesstoken AccessToken, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAccessTokenByInstallation", err) }()

	return c.Client.GetAccessTokenByInstallation(ctx, installation)
}

func (c *loggingClient) GetAccessTokenBySlug(ctx context.Context, workspaceSlug string) (accesstoken AccessToken, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAccessTokenBySlug", err) }()

	return c.Client.GetAccessTokenBySlug(ctx, workspaceSlug)
}

func (c *loggingClient) GetAccessTokenByUUID(ctx context.Context, workspaceUUID string) (accesstoken AccessToken, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAccessTokenByUUID", err) }()

	return c.Client.GetAccessTokenByUUID(ctx, workspaceUUID)
}

func (c *loggingClient) GetAccessTokenByJWTToken(ctx context.Context, jwtToken string) (accesstoken AccessToken, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAccessTokenByJWTToken", err) }()

	return c.Client.GetAccessTokenByJWTToken(ctx, jwtToken)
}

func (c *loggingClient) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event RepositoryPushEvent) (valid bool, manifest string, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetEstafetteManifest", err) }()

	return c.Client.GetEstafetteManifest(ctx, accesstoken, event)
}

func (c *loggingClient) JobVarsFunc(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
	return c.Client.JobVarsFunc(ctx)
}

func (c *loggingClient) ValidateInstallationJWT(ctx context.Context, authorizationHeader string) (installation *BitbucketAppInstallation, err error) {
	return c.Client.ValidateInstallationJWT(ctx, authorizationHeader)
}

func (c *loggingClient) GenerateJWTBySlug(ctx context.Context, workspaceSlug string) (tokenString string, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GenerateJWTBySlug", err) }()

	return c.Client.GenerateJWTBySlug(ctx, workspaceSlug)
}

func (c *loggingClient) GenerateJWTByUUID(ctx context.Context, workspaceUUID string) (tokenString string, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GenerateJWTByUUID", err) }()

	return c.Client.GenerateJWTByUUID(ctx, workspaceUUID)
}

func (c *loggingClient) GenerateJWTByInstallation(ctx context.Context, installation BitbucketAppInstallation) (tokenString string, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GenerateJWTByInstallation", err) }()

	return c.Client.GenerateJWTByInstallation(ctx, installation)
}

func (c *loggingClient) GetInstallationBySlug(ctx context.Context, workspaceSlug string) (installation *BitbucketAppInstallation, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetInstallationBySlug", err) }()

	return c.Client.GetInstallationBySlug(ctx, workspaceSlug)
}

func (c *loggingClient) GetInstallationByUUID(ctx context.Context, workspaceUUID string) (installation *BitbucketAppInstallation, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetInstallationByUUID", err) }()

	return c.Client.GetInstallationByUUID(ctx, workspaceUUID)
}

func (c *loggingClient) GetInstallations(ctx context.Context) (installations []*BitbucketAppInstallation, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetInstallations", err) }()

	return c.Client.GetInstallations(ctx)
}

func (c *loggingClient) AddInstallation(ctx context.Context, installation BitbucketAppInstallation) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "AddInstallation", err) }()

	return c.Client.AddInstallation(ctx, installation)
}

func (c *loggingClient) RemoveInstallation(ctx context.Context, installation BitbucketAppInstallation) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "RemoveInstallation", err) }()

	return c.Client.RemoveInstallation(ctx, installation)
}

func (c *loggingClient) GetWorkspace(ctx context.Context, workspaceUUID string) (workspace *Workspace, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetWorkspace", err) }()

	return c.Client.GetWorkspace(ctx, workspaceUUID)
}
