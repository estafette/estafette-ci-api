package rbac

import (
	"github.com/estafette/estafette-ci-api/pkg/clients/bitbucketapi"
	"github.com/estafette/estafette-ci-api/pkg/clients/githubapi"
)

type githubResponse struct {
	Manifest githubManifest         `json:"manifest,omitempty"`
	Apps     []*githubapi.GithubApp `json:"apps,omitempty"`
}

type githubManifest struct {
	Name               string               `json:"name,omitempty"`
	Description        string               `json:"description,omitempty"`
	URL                string               `json:"url,omitempty"`
	HookAttributes     *githubHookAttribute `json:"hook_attributes,omitempty"`
	RedirectURL        string               `json:"redirect_url,omitempty"`
	SetupURL           string               `json:"setup_url,omitempty"`
	CallbackURLs       []string             `json:"callback_urls,omitempty"`
	Public             bool                 `json:"public,omitempty"`
	DefaultPermissions map[string]string    `json:"default_permissions,omitempty"`
	DefaultEvents      []string             `json:"default_events,omitempty"`
}

type githubHookAttribute struct {
	URL string `json:"url,omitempty"`
}

type bitbucketResponse struct {
	RedirectURI string                       `json:"redirectURI,omitempty"`
	Apps        []*bitbucketapi.BitbucketApp `json:"apps,omitempty"`
}
