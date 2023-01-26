package githubapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/estafette/estafette-ci-api/pkg/api"
	crypt "github.com/estafette/estafette-ci-crypt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"github.com/sethgrid/pester"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	ErrMissingInstallation = errors.New("installation is missing")
)

// Client is the interface for communicating with the github api
//
//go:generate mockgen -package=githubapi -destination ./mock.go -source=client.go
type Client interface {
	GetGithubAppToken(ctx context.Context, app GithubApp) (token string, err error)
	GetAppAndInstallationByOwner(ctx context.Context, repoOwner string) (app *GithubApp, installation *GithubInstallation, err error)
	GetAppAndInstallationByID(ctx context.Context, installationID int) (app *GithubApp, installation *GithubInstallation, err error)
	GetInstallationToken(ctx context.Context, app GithubApp, installation GithubInstallation) (accessToken AccessToken, err error)
	GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event PushEvent) (valid bool, manifest string, err error)
	JobVarsFunc(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error)
	ConvertAppManifestCode(ctx context.Context, code string) (err error)
	GetApps(ctx context.Context) (apps []*GithubApp, err error)
	GetAppByID(ctx context.Context, id int) (app *GithubApp, err error)
	AddApp(ctx context.Context, app GithubApp) (err error)
	RemoveApp(ctx context.Context, app GithubApp) (err error)
	AddInstallation(ctx context.Context, installation GithubInstallation) (err error)
	RemoveInstallation(ctx context.Context, installation GithubInstallation) (err error)
}

// NewClient creates an githubapi.Client to communicate with the Github api
func NewClient(config *api.APIConfig, kubeClientset *kubernetes.Clientset, secretHelper crypt.SecretHelper) Client {
	return &client{
		enabled:       config != nil && config.Integrations != nil && config.Integrations.Github != nil && config.Integrations.Github.Enable,
		config:        config,
		kubeClientset: kubeClientset,
		secretHelper:  secretHelper,
	}
}

type client struct {
	enabled       bool
	config        *api.APIConfig
	kubeClientset *kubernetes.Clientset
	secretHelper  crypt.SecretHelper
}

// GetGithubAppToken returns a Github app token with which to retrieve an installation token
func (c *client) GetGithubAppToken(ctx context.Context, app GithubApp) (githubAppToken string, err error) {

	// https://developer.github.com/apps/building-integrations/setting-up-and-registering-github-apps/about-authentication-options-for-github-apps/

	// load private key from pem file
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(app.PrivateKey))
	if err != nil {
		return
	}

	// create a new token object, specifying signing method and the claims you would like it to contain.
	epoch := time.Now().Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		// issued at time
		"iat": epoch,
		// JWT expiration time (10 minute maximum)
		"exp": epoch + 500,
		// GitHub App's identifier
		"iss": app.ID,
	})

	// sign and get the complete encoded token as a string using the private key
	githubAppToken, err = token.SignedString(privateKey)
	if err != nil {
		return
	}

	return
}

// GetInstallationID returns the id for an installation of a Github app
func (c *client) GetAppAndInstallationByOwner(ctx context.Context, repoOwner string) (app *GithubApp, installation *GithubInstallation, err error) {

	// get installation and app by repoOwner
	apps, err := c.GetApps(ctx)
	if err != nil {
		return
	}

	for _, a := range apps {
		for _, i := range a.Installations {
			if i.Account != nil && i.Account.Login == repoOwner {
				return a, i, nil
			}
		}
	}

	return nil, nil, ErrMissingInstallation
}

func (c *client) GetAppAndInstallationByID(ctx context.Context, installationID int) (app *GithubApp, installation *GithubInstallation, err error) {

	// get installation and app by repoOwner
	apps, err := c.GetApps(ctx)
	if err != nil {
		return
	}

	for _, a := range apps {
		for _, i := range a.Installations {
			if i.ID == installationID {
				return a, i, nil
			}
		}
	}

	return nil, nil, ErrMissingInstallation
}

// GetInstallationToken returns an access token for an installation of a Github app
func (c *client) GetInstallationToken(ctx context.Context, app GithubApp, installation GithubInstallation) (accessToken AccessToken, err error) {

	githubAppToken, err := c.GetGithubAppToken(ctx, app)
	if err != nil {
		return
	}

	_, body, err := c.callGithubAPI(ctx, "POST", fmt.Sprintf("https://api.github.com/app/installations/%v/access_tokens", installation.ID), nil, "Bearer", githubAppToken)
	if err != nil {
		return
	}

	// unmarshal json body
	err = json.Unmarshal(body, &accessToken)
	if err != nil {
		return
	}

	return
}

func (c *client) GetEstafetteManifest(ctx context.Context, accessToken AccessToken, pushEvent PushEvent) (exists bool, manifest string, err error) {

	// https://developer.github.com/v3/repos/contents/

	manifestSourceAPIUrl := fmt.Sprintf("https://api.github.com/repos/%v/contents/.estafette.yaml?ref=%v", pushEvent.Repository.FullName, pushEvent.After)
	statusCode, body, err := c.callGithubAPI(ctx, "GET", manifestSourceAPIUrl, nil, "token", accessToken.Token)
	if err != nil {
		return
	}

	if statusCode == http.StatusNotFound {
		return
	}

	if statusCode != http.StatusOK {
		err = fmt.Errorf("Retrieving estafette manifest from %v failed with status code %v", manifestSourceAPIUrl, statusCode)
		return
	}

	var content RepositoryContent

	// unmarshal json body
	err = json.Unmarshal(body, &content)
	if err != nil {
		return
	}

	if content.Type == "file" && content.Encoding == "base64" {
		data, err := base64.StdEncoding.DecodeString(content.Content)
		if err != nil {
			return false, "", err
		}
		exists = true
		manifest = string(data)
	}

	return
}

// JobVarsFunc returns a function that can get an access token and authenticated url for a repository
func (c *client) JobVarsFunc(ctx context.Context) func(context.Context, string, string, string) (string, error) {
	return func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
		// get installation id with just the repo owner
		app, installation, err := c.GetAppAndInstallationByOwner(ctx, repoOwner)
		if err != nil {
			return "", err
		}
		if app == nil {
			return "", fmt.Errorf("App for repoOwner %v is nil", repoOwner)
		}
		if installation == nil {
			return "", fmt.Errorf("Installation for repoOwner %v is nil", repoOwner)
		}

		// get access token
		accessToken, err := c.GetInstallationToken(ctx, *app, *installation)
		if err != nil {
			return "", err
		}

		return accessToken.Token, nil
	}
}

func (c *client) callGithubAPI(ctx context.Context, method, url string, params interface{}, authorizationType, token string) (statusCode int, body []byte, err error) {

	// convert params to json if they're present
	var requestBody io.Reader
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return 0, body, err
		}
		requestBody = bytes.NewReader(data)
	}

	// create client, in order to add headers
	client := pester.NewExtendedClient(&http.Client{Transport: &nethttp.Transport{}})
	client.MaxRetries = 3
	client.Backoff = pester.ExponentialJitterBackoff
	client.KeepLog = true
	client.Timeout = time.Second * 10
	request, err := http.NewRequest(method, url, requestBody)
	if err != nil {
		return
	}

	span := opentracing.SpanFromContext(ctx)
	var ht *nethttp.Tracer
	if span != nil {
		// add tracing context
		request = request.WithContext(opentracing.ContextWithSpan(request.Context(), span))

		// collect additional information on setting up connections
		request, ht = nethttp.TraceRequest(span.Tracer(), request)
	}

	// add headers
	request.Header.Add("Authorization", fmt.Sprintf("%v %v", authorizationType, token))
	request.Header.Add("Accept", "application/vnd.github.machine-man-preview+json")

	// perform actual request
	response, err := client.Do(request)
	if err != nil {
		return
	}

	defer response.Body.Close()
	if ht != nil {
		ht.Finish()
	}

	statusCode = response.StatusCode

	body, err = io.ReadAll(response.Body)
	if err != nil {
		return
	}

	// unmarshal json body
	var b interface{}
	err = json.Unmarshal(body, &b)
	if err != nil {
		log.Error().Err(err).
			Str("url", url).
			Str("requestMethod", method).
			Interface("requestBody", params).
			Interface("requestHeaders", request.Header).
			Interface("responseHeaders", response.Header).
			Str("responseBody", string(body)).
			Msg("Deserializing response for '%v' Github api call failed")

		return
	}

	return
}

func (c *client) ConvertAppManifestCode(ctx context.Context, code string) (err error) {

	// https://docs.github.com/en/developers/apps/building-github-apps/creating-a-github-app-from-a-manifest#3-you-exchange-the-temporary-code-to-retrieve-the-app-configuration
	// swap code for id (GitHub App ID), pem (private key), and webhook_secret at POST /app-manifests/{code}/conversions

	url := fmt.Sprintf("https://api.github.com/app-manifests/%v/conversions", code)

	// create client, in order to add headers
	client := pester.NewExtendedClient(&http.Client{Transport: &nethttp.Transport{}})
	client.MaxRetries = 3
	client.Backoff = pester.ExponentialJitterBackoff
	client.KeepLog = true
	client.Timeout = time.Second * 10
	request, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return
	}

	span := opentracing.SpanFromContext(ctx)
	var ht *nethttp.Tracer
	if span != nil {
		// add tracing context
		request = request.WithContext(opentracing.ContextWithSpan(request.Context(), span))

		// collect additional information on setting up connections
		request, ht = nethttp.TraceRequest(span.Tracer(), request)
	}

	// perform actual request
	response, err := client.Do(request)
	if err != nil {
		return
	}
	if response == nil {
		return fmt.Errorf("Response for request %v is nil", url)
	}

	defer response.Body.Close()
	if ht != nil {
		ht.Finish()
	}

	if response.StatusCode != http.StatusCreated {
		return fmt.Errorf("Failed requesting %v with status code %v", url, response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return
	}

	log.Debug().Str("body", string(body)).Msgf("Received response from %v", url)

	// unmarshal json body
	var app GithubApp
	err = json.Unmarshal(body, &app)
	if err != nil {
		log.Warn().Err(err).Str("body", string(body)).Msgf("Failed unmarshalling response from %v", url)
		return
	}

	err = c.AddApp(ctx, app)
	if err != nil {
		return
	}

	return
}

type appsCacheItem struct {
	Apps      []*GithubApp
	ExpiresIn int
	FetchedAt time.Time
}

func (c *appsCacheItem) ExpiresAt() time.Time {
	return c.FetchedAt.Add(time.Duration(c.ExpiresIn) * time.Second)
}

func (c *appsCacheItem) IsExpired() bool {
	return time.Now().UTC().After(c.ExpiresAt())
}

var appsCache appsCacheItem
var appsCacheMutex = sync.RWMutex{}

const githubConfigmapName = "estafette-ci-api.github"

func (c *client) GetApps(ctx context.Context) (apps []*GithubApp, err error) {

	// get from cache
	appsCacheMutex.RLock()
	if appsCache.Apps != nil && !appsCache.IsExpired() {
		appsCacheMutex.RUnlock()
		return appsCache.Apps, nil
	}
	appsCacheMutex.RUnlock()

	apps = make([]*GithubApp, 0)

	configMap, err := c.kubeClientset.CoreV1().ConfigMaps(c.getCurrentNamespace()).Get(ctx, githubConfigmapName, metav1.GetOptions{})
	if err != nil || configMap == nil {
		return apps, nil
	}

	if data, ok := configMap.Data["apps"]; ok {
		err = json.Unmarshal([]byte(data), &apps)
		if err != nil {
			return
		}

		err = c.decryptAppSecrets(ctx, apps)
		if err != nil {
			return
		}

		// add to cache
		appsCacheMutex.Lock()
		appsCache = appsCacheItem{
			Apps:      apps,
			ExpiresIn: 30,
			FetchedAt: time.Now().UTC(),
		}
		appsCacheMutex.Unlock()
	}

	return
}

func (c *client) GetAppByID(ctx context.Context, id int) (app *GithubApp, err error) {
	apps, err := c.GetApps(ctx)
	if err != nil {
		return
	}

	if apps == nil {
		return nil, fmt.Errorf("Apps for id %v are nil", id)
	}

	for _, a := range apps {
		if a.ID == id {
			return a, nil
		}
	}

	return nil, fmt.Errorf("App for id %v is unknown", id)
}

func (c *client) AddApp(ctx context.Context, app GithubApp) (err error) {

	apps, err := c.GetApps(ctx)
	if err != nil {
		return
	}

	if apps == nil {
		apps = make([]*GithubApp, 0)
	}

	// check if app(s) with id exists, if not add, otherwise update
	appExists := false
	for _, ap := range apps {
		if ap.ID == app.ID {
			appExists = true

			ap.PrivateKey = app.PrivateKey
			ap.WebhookSecret = app.WebhookSecret
		}
	}

	if !appExists {
		apps = append(apps, &app)
	}

	err = c.upsertConfigmap(ctx, apps)
	if err != nil {
		return
	}

	return
}

func (c *client) RemoveApp(ctx context.Context, app GithubApp) (err error) {
	apps, err := c.GetApps(ctx)
	if err != nil {
		return
	}

	if apps == nil {
		apps = make([]*GithubApp, 0)
	}

	// check if app(s) with id exists, then remove
	for i, ap := range apps {
		if ap.ID == app.ID {
			apps = append(apps[:i], apps[i+1:]...)
		}
	}

	err = c.upsertConfigmap(ctx, apps)
	if err != nil {
		return
	}

	return
}

func (c *client) AddInstallation(ctx context.Context, installation GithubInstallation) (err error) {

	apps, err := c.GetApps(ctx)
	if err != nil {
		return
	}

	if apps == nil {
		apps = make([]*GithubApp, 0)
	}

	// check if installation(s) with id exists, if not add, otherwise update
	for _, app := range apps {
		if app.ID == installation.AppID {
			if app.Installations == nil {
				app.Installations = make([]*GithubInstallation, 0)
			}
			installationExists := false
			for _, inst := range app.Installations {
				if inst.ID == installation.ID {
					installationExists = true

					// update fields editable from gui
					inst.Organizations = installation.Organizations
				}
			}
			if !installationExists {
				app.Installations = append(app.Installations, &installation)
			}
		}
	}

	err = c.upsertConfigmap(ctx, apps)
	if err != nil {
		return
	}

	return
}

func (c *client) RemoveInstallation(ctx context.Context, installation GithubInstallation) (err error) {

	apps, err := c.GetApps(ctx)
	if err != nil {
		return
	}

	if apps == nil {
		apps = make([]*GithubApp, 0)
	}

	// check if installation(s) with id exists, then remove
	for _, app := range apps {
		if app.ID == installation.AppID {
			if app.Installations == nil {
				app.Installations = make([]*GithubInstallation, 0)
			}
			for i, inst := range app.Installations {
				if inst.ID == installation.ID {
					app.Installations = append(app.Installations[:i], app.Installations[i+1:]...)
				}
			}
		}
	}

	err = c.upsertConfigmap(ctx, apps)
	if err != nil {
		return
	}

	return

}

func (c *client) upsertConfigmap(ctx context.Context, apps []*GithubApp) (err error) {

	err = c.encryptAppSecrets(ctx, apps)
	if err != nil {
		return
	}

	// marshal to json
	data, err := json.Marshal(apps)
	if err != nil {
		return err
	}

	// store in configmap
	configMap, err := c.kubeClientset.CoreV1().ConfigMaps(c.getCurrentNamespace()).Get(ctx, githubConfigmapName, metav1.GetOptions{})
	if err != nil || configMap == nil {
		// create configmap
		configMap = &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      githubConfigmapName,
				Namespace: c.getCurrentNamespace(),
			},
			Data: map[string]string{
				"apps": string(data),
			},
		}
		_, err = c.kubeClientset.CoreV1().ConfigMaps(c.getCurrentNamespace()).Create(ctx, configMap, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	} else {
		// update configmap
		configMap.Data["apps"] = string(data)
		_, err = c.kubeClientset.CoreV1().ConfigMaps(c.getCurrentNamespace()).Update(ctx, configMap, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	// add to cache
	appsCacheMutex.Lock()
	appsCache = appsCacheItem{
		Apps:      apps,
		ExpiresIn: 30,
		FetchedAt: time.Now().UTC(),
	}
	appsCacheMutex.Unlock()

	return
}

func (c *client) getCurrentNamespace() string {
	namespace, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed reading namespace")
	}

	return string(namespace)
}

func (c *client) encryptAppSecrets(ctx context.Context, apps []*GithubApp) (err error) {
	for _, app := range apps {
		encryptedPrivateKey, encryptErr := c.secretHelper.EncryptEnvelope(app.PrivateKey, crypt.DefaultPipelineAllowList)
		if encryptErr != nil {
			return encryptErr
		}
		app.PrivateKey = encryptedPrivateKey

		encryptedWebhookSecret, encryptErr := c.secretHelper.EncryptEnvelope(app.WebhookSecret, crypt.DefaultPipelineAllowList)
		if encryptErr != nil {
			return encryptErr
		}
		app.WebhookSecret = encryptedWebhookSecret

		encryptedClientSecret, encryptErr := c.secretHelper.EncryptEnvelope(app.ClientSecret, crypt.DefaultPipelineAllowList)
		if encryptErr != nil {
			return encryptErr
		}
		app.ClientSecret = encryptedClientSecret
	}

	return nil
}

func (c *client) decryptAppSecrets(ctx context.Context, apps []*GithubApp) (err error) {
	for _, app := range apps {
		decryptedPrivateKey, _, decryptErr := c.secretHelper.DecryptEnvelope(app.PrivateKey, "")
		if decryptErr != nil {
			return decryptErr
		}
		app.PrivateKey = decryptedPrivateKey

		decryptedWebhookSecret, _, decryptErr := c.secretHelper.DecryptEnvelope(app.WebhookSecret, "")
		if decryptErr != nil {
			return decryptErr
		}
		app.WebhookSecret = decryptedWebhookSecret

		decryptedClientSecret, _, decryptErr := c.secretHelper.DecryptEnvelope(app.ClientSecret, "")
		if decryptErr != nil {
			return decryptErr
		}
		app.ClientSecret = decryptedClientSecret
	}

	return nil
}
