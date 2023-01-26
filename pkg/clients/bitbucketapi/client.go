package bitbucketapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
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
	ErrInvalidAuthorizationHeader = errors.New("invalid authorization header")
	ErrInvalidSigningAlgorithm    = errors.New("invalid signing algorithm")
	ErrInvalidToken               = errors.New("invalid token")
	ErrNoInstallations            = errors.New("no installations")
	ErrMissingApp                 = errors.New("app for key is missing")
	ErrMissingInstallation        = errors.New("installation for clientKey is missing")
	ErrMissingClaims              = errors.New("token has no claims")
)

// Client is the interface for communicating with the bitbucket api
//
//go:generate mockgen -package=bitbucketapi -destination ./mock.go -source=client.go
type Client interface {
	GetAccessTokenByInstallation(ctx context.Context, installation BitbucketAppInstallation) (accesstoken AccessToken, err error)
	GetAccessTokenBySlug(ctx context.Context, workspaceSlug string) (accesstoken AccessToken, err error)
	GetAccessTokenByUUID(ctx context.Context, workspaceUUID string) (accesstoken AccessToken, err error)
	GetAccessTokenByJWTToken(ctx context.Context, jwtToken string) (accesstoken AccessToken, err error)
	GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event RepositoryPushEvent) (valid bool, manifest string, err error)
	JobVarsFunc(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error)
	ValidateInstallationJWT(ctx context.Context, authorizationHeader string) (installation *BitbucketAppInstallation, err error)
	GenerateJWTBySlug(ctx context.Context, workspaceSlug string) (tokenString string, err error)
	GenerateJWTByUUID(ctx context.Context, workspaceUUID string) (tokenString string, err error)
	GenerateJWTByInstallation(ctx context.Context, installation BitbucketAppInstallation) (tokenString string, err error)
	GetApps(ctx context.Context) (apps []*BitbucketApp, err error)
	GetAppByKey(ctx context.Context, key string) (app *BitbucketApp, err error)
	GetInstallationBySlug(ctx context.Context, workspaceSlug string) (installation *BitbucketAppInstallation, err error)
	GetInstallationByUUID(ctx context.Context, workspaceUUID string) (installation *BitbucketAppInstallation, err error)
	GetInstallationByClientKey(ctx context.Context, clientKey string) (installation *BitbucketAppInstallation, err error)
	AddApp(ctx context.Context, app BitbucketApp) (err error)
	AddInstallation(ctx context.Context, installation BitbucketAppInstallation) (err error)
	RemoveInstallation(ctx context.Context, installation BitbucketAppInstallation) (err error)
	GetWorkspace(ctx context.Context, installation BitbucketAppInstallation) (workspace *Workspace, err error)
}

// NewClient returns a new bitbucket.Client
func NewClient(config *api.APIConfig, kubeClientset *kubernetes.Clientset, secretHelper crypt.SecretHelper) Client {
	return &client{
		enabled:       config != nil && config.Integrations != nil && config.Integrations.Bitbucket != nil && config.Integrations.Bitbucket.Enable,
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

func (c *client) GetAccessTokenByInstallation(ctx context.Context, installation BitbucketAppInstallation) (accesstoken AccessToken, err error) {
	jtwToken, err := c.GenerateJWTByInstallation(ctx, installation)
	if err != nil {
		return
	}

	return c.GetAccessTokenByJWTToken(ctx, jtwToken)
}

func (c *client) GetAccessTokenBySlug(ctx context.Context, workspaceSlug string) (accesstoken AccessToken, err error) {
	jtwToken, err := c.GenerateJWTBySlug(ctx, workspaceSlug)
	if err != nil {
		return
	}

	return c.GetAccessTokenByJWTToken(ctx, jtwToken)
}

func (c *client) GetAccessTokenByUUID(ctx context.Context, workspaceUUID string) (accesstoken AccessToken, err error) {
	jtwToken, err := c.GenerateJWTByUUID(ctx, workspaceUUID)
	if err != nil {
		return
	}

	return c.GetAccessTokenByJWTToken(ctx, jtwToken)
}

func (c *client) GetAccessTokenByJWTToken(ctx context.Context, jwtToken string) (accesstoken AccessToken, err error) {

	// form values
	data := url.Values{}
	data.Set("grant_type", "urn:bitbucket:oauth2:jwt")

	// create client, in order to add headers
	client := pester.NewExtendedClient(&http.Client{Transport: &nethttp.Transport{}})
	client.MaxRetries = 3
	client.Backoff = pester.ExponentialJitterBackoff
	client.KeepLog = true
	client.Timeout = time.Second * 10
	request, err := http.NewRequest("POST", "https://bitbucket.org/site/oauth2/access_token", bytes.NewBufferString(data.Encode()))
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
	request.Header.Add("Authorization", fmt.Sprintf("%v %v", "JWT", jwtToken))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// perform actual request
	response, err := client.Do(request)
	if err != nil {
		return
	}

	defer response.Body.Close()
	if ht != nil {
		ht.Finish()
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return
	}

	// unmarshal json body
	err = json.Unmarshal(body, &accesstoken)
	if err != nil {
		log.Warn().Str("body", string(body)).Msg("Failed unmarshalling access token")
		return
	}

	return
}

func (c *client) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, pushEvent RepositoryPushEvent) (exists bool, manifest string, err error) {

	// create client, in order to add headers
	client := pester.NewExtendedClient(&http.Client{Transport: &nethttp.Transport{}})
	client.MaxRetries = 3
	client.Backoff = pester.ExponentialJitterBackoff
	client.KeepLog = true
	client.Timeout = time.Second * 10

	manifestSourceAPIUrl := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%v/src/%v/.estafette.yaml", pushEvent.Repository.FullName, pushEvent.Push.Changes[0].New.Target.Hash)

	request, err := http.NewRequest("GET", manifestSourceAPIUrl, nil)

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
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %v", accesstoken.AccessToken))

	// perform actual request
	response, err := client.Do(request)
	if err != nil {
		return
	}

	defer response.Body.Close()
	if ht != nil {
		ht.Finish()
	}

	if response.StatusCode == http.StatusNotFound {
		return
	}

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("Retrieving estafette manifest from %v failed with status code %v", manifestSourceAPIUrl, response.StatusCode)
		return
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return
	}

	exists = true
	manifest = string(body)

	return
}

// JobVarsFunc returns a function that can get an access token and authenticated url for a repository
func (c *client) JobVarsFunc(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
	return func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
		// get access token
		accessToken, err := c.GetAccessTokenBySlug(ctx, repoOwner)
		if err != nil {
			return "", err
		}

		return accessToken.AccessToken, nil
	}
}

func (c *client) ValidateInstallationJWT(ctx context.Context, authorizationHeader string) (installation *BitbucketAppInstallation, err error) {
	if !strings.HasPrefix(authorizationHeader, "JWT ") {
		return nil, ErrInvalidAuthorizationHeader
	}
	jwtTokenString := strings.TrimPrefix(authorizationHeader, "JWT ")

	apps, err := c.GetApps(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Bitbucket JWT validation failed on retrieving apps")
		return nil, err
	}

	if apps == nil && len(apps) == 0 {
		log.Warn().Msg("Bitbucket JWT validation retrieved 0 apps")
		return nil, ErrNoInstallations
	}

	token, err := jwt.Parse(jwtTokenString, func(token *jwt.Token) (interface{}, error) {
		// check algorithm is correct
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidSigningAlgorithm
		}

		// get shared secret for client key (iss claim)
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			clientKey := claims["iss"].(string)

			for _, app := range apps {
				for _, inst := range app.Installations {
					if inst.ClientKey == clientKey {
						installation = inst

						return []byte(inst.SharedSecret), nil
					}
				}
			}

			return nil, ErrMissingInstallation
		}

		return nil, ErrMissingClaims
	})

	if err != nil {
		// https://github.com/golang-jwt/jwt/issues/98

		// ignore error if only error is ValidationErrorIssuedAt
		validationErr, isValidationError := err.(*jwt.ValidationError)
		if !isValidationError {
			log.Warn().Err(err).Msg("Bitbucket JWT parsing failed, but is not a ValidationError")
			return nil, err
		}

		hasIssuedAtValidationError := validationErr.Errors&jwt.ValidationErrorIssuedAt != 0
		if !hasIssuedAtValidationError {
			log.Warn().Err(err).Msg("Bitbucket JWT parsing failed, but is not a ValidationErrorIssuedAt")
			return nil, err
		}

		// toggle ValidationErrorIssuedAt and check if it was the only validation error
		remainingErrors := validationErr.Errors ^ jwt.ValidationErrorIssuedAt
		if remainingErrors > 0 {
			log.Warn().Err(err).Msg("Bitbucket JWT parsing failed, but has other errors besides ValidationErrorIssuedAt")
			return nil, err
		}

		iat := c.getIat(token)
		if iat > 0 {
			log.Info().Err(err).Str("now", time.Now().UTC().Format(time.RFC3339)).Str("iat", time.Unix(iat, 0).Format(time.RFC3339)).Msg("Bitbucket JWT has 'issued at' validation error, ignoring it")
		} else {
			log.Info().Err(err).Msg("Bitbucket JWT has 'issued at' validation error, ignoring it")
		}

		// token is valid except for ValidationErrorIssuedAt, set to true
		token.Valid = true
	}

	if !token.Valid {
		log.Warn().Err(err).Msg("Bitbucket JWT validation has invalid token")
		return nil, ErrInvalidToken
	}

	return installation, nil
}

func (c *client) GenerateJWTBySlug(ctx context.Context, workspaceSlug string) (tokenString string, err error) {
	installation, err := c.GetInstallationBySlug(ctx, workspaceSlug)
	if err != nil {
		return
	}

	return c.GenerateJWTByInstallation(ctx, *installation)
}

func (c *client) GenerateJWTByUUID(ctx context.Context, workspaceUUID string) (tokenString string, err error) {
	installation, err := c.GetInstallationByUUID(ctx, workspaceUUID)
	if err != nil {
		return
	}

	return c.GenerateJWTByInstallation(ctx, *installation)
}

func (c *client) GenerateJWTByInstallation(ctx context.Context, installation BitbucketAppInstallation) (tokenString string, err error) {
	// Create the token
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)

	now := time.Now().UTC()
	expiry := now.Add(time.Duration(180) * time.Second)

	// set required claims
	claims["iss"] = installation.Key
	claims["iat"] = now.Unix()
	claims["exp"] = expiry.Unix()
	claims["sub"] = installation.ClientKey

	// sign the token
	return token.SignedString([]byte(installation.SharedSecret))

}

func (c *client) GetInstallationBySlug(ctx context.Context, workspaceSlug string) (installation *BitbucketAppInstallation, err error) {
	apps, err := c.GetApps(ctx)
	if err != nil {
		return
	}

	for _, app := range apps {
		for _, inst := range app.Installations {
			if inst != nil && inst.Workspace != nil && inst.Workspace.Slug == workspaceSlug {
				return inst, nil
			}
		}
	}

	return nil, ErrMissingInstallation
}

func (c *client) GetInstallationByUUID(ctx context.Context, workspaceUUID string) (installation *BitbucketAppInstallation, err error) {
	apps, err := c.GetApps(ctx)
	if err != nil {
		return
	}

	for _, app := range apps {
		for _, inst := range app.Installations {
			if inst != nil && inst.GetWorkspaceUUID() == workspaceUUID {
				return inst, nil
			}
		}
	}

	return nil, ErrMissingInstallation
}

func (c *client) GetInstallationByClientKey(ctx context.Context, clientKey string) (installation *BitbucketAppInstallation, err error) {
	apps, err := c.GetApps(ctx)
	if err != nil {
		return
	}

	for _, app := range apps {
		for _, inst := range app.Installations {
			if inst != nil && inst.ClientKey == clientKey {
				return inst, nil
			}
		}
	}

	return nil, ErrMissingInstallation
}

type appsCacheItem struct {
	Apps      []*BitbucketApp
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

const bitbucketConfigmapName = "estafette-ci-api.bitbucket"

func (c *client) GetApps(ctx context.Context) (apps []*BitbucketApp, err error) {
	// get from cache
	appsCacheMutex.RLock()
	if appsCache.Apps != nil && !appsCache.IsExpired() {
		appsCacheMutex.RUnlock()
		return appsCache.Apps, nil
	}
	appsCacheMutex.RUnlock()

	apps = make([]*BitbucketApp, 0)

	configMap, err := c.kubeClientset.CoreV1().ConfigMaps(c.getCurrentNamespace()).Get(ctx, bitbucketConfigmapName, metav1.GetOptions{})
	if err != nil || configMap == nil {
		return apps, nil
	}

	if data, ok := configMap.Data["apps"]; ok {
		err = json.Unmarshal([]byte(data), &apps)
		if err != nil {
			return
		}

		err = c.decryptSharedSecrets(ctx, apps)
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

func (c *client) GetAppByKey(ctx context.Context, key string) (app *BitbucketApp, err error) {
	apps, err := c.GetApps(ctx)
	if err != nil {
		return
	}

	for _, a := range apps {
		if a != nil && a.Key == key {
			return a, nil
		}
	}

	return nil, ErrMissingApp
}

func (c *client) AddApp(ctx context.Context, app BitbucketApp) (err error) {
	apps, err := c.GetApps(ctx)
	if err != nil {
		return
	}

	if apps == nil {
		apps = make([]*BitbucketApp, 0)
	}

	// check if app with key, if not add
	appExists := false
	for _, a := range apps {
		if a != nil && a.Key == app.Key {
			appExists = true
			break
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

func (c *client) AddInstallation(ctx context.Context, installation BitbucketAppInstallation) (err error) {
	apps, err := c.GetApps(ctx)
	if err != nil {
		return
	}

	if apps == nil {
		apps = make([]*BitbucketApp, 0)
	}

	// check if installation(s) with key and clientKey exists, if not add, otherwise update
	appExists := false
	for _, app := range apps {
		if app.Key == installation.Key {
			appExists = true

			installationExists := false
			for _, inst := range app.Installations {
				if inst.ClientKey == installation.ClientKey {
					installationExists = true

					// update fields editable from gui
					inst.Organizations = installation.Organizations

					break
				}
			}

			if !installationExists {
				app.Installations = append(app.Installations, &installation)
			}
		}
	}

	if !appExists {
		apps = append(apps, &BitbucketApp{
			Key:           installation.Key,
			Installations: []*BitbucketAppInstallation{&installation},
		})
	}

	err = c.upsertConfigmap(ctx, apps)
	if err != nil {
		return
	}

	return
}

func (c *client) RemoveInstallation(ctx context.Context, installation BitbucketAppInstallation) (err error) {
	apps, err := c.GetApps(ctx)
	if err != nil {
		return
	}

	if apps == nil {
		apps = make([]*BitbucketApp, 0)
	}

	// check if installation(s) with key and clientKey exists, then remove
	for _, app := range apps {
		for i, inst := range app.Installations {
			if inst.Key == installation.Key && inst.ClientKey == installation.ClientKey {
				app.Installations = append(app.Installations[:i], app.Installations[i+1:]...)
			}
		}
	}

	err = c.upsertConfigmap(ctx, apps)
	if err != nil {
		return
	}

	return
}

func (c *client) upsertConfigmap(ctx context.Context, apps []*BitbucketApp) (err error) {

	err = c.encryptSharedSecrets(ctx, apps)
	if err != nil {
		return
	}

	// marshal to json
	data, err := json.Marshal(apps)
	if err != nil {
		return err
	}

	// store in configmap
	configMap, err := c.kubeClientset.CoreV1().ConfigMaps(c.getCurrentNamespace()).Get(ctx, bitbucketConfigmapName, metav1.GetOptions{})
	if err != nil || configMap == nil {
		// create configmap
		configMap = &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      bitbucketConfigmapName,
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

func (c *client) encryptSharedSecrets(ctx context.Context, apps []*BitbucketApp) (err error) {
	for _, app := range apps {
		for _, installation := range app.Installations {
			encryptedSharedSecret, encryptErr := c.secretHelper.EncryptEnvelope(installation.SharedSecret, crypt.DefaultPipelineAllowList)
			if encryptErr != nil {
				return encryptErr
			}
			installation.SharedSecret = encryptedSharedSecret
		}
	}

	return nil
}

func (c *client) decryptSharedSecrets(ctx context.Context, apps []*BitbucketApp) (err error) {
	for _, app := range apps {
		for _, installation := range app.Installations {
			decryptedSharedSecret, _, decryptErr := c.secretHelper.DecryptEnvelope(installation.SharedSecret, "")
			if decryptErr != nil {
				return decryptErr
			}
			installation.SharedSecret = decryptedSharedSecret
		}
	}

	return nil
}

func (c *client) GetWorkspace(ctx context.Context, installation BitbucketAppInstallation) (workspace *Workspace, err error) {
	accessToken, err := c.GetAccessTokenByInstallation(ctx, installation)
	if err != nil {
		return
	}

	// create client, in order to add headers
	client := pester.NewExtendedClient(&http.Client{Transport: &nethttp.Transport{}})
	client.MaxRetries = 3
	client.Backoff = pester.ExponentialJitterBackoff
	client.KeepLog = true
	client.Timeout = time.Second * 10

	workspaceAPIUrl := fmt.Sprintf("https://api.bitbucket.org/2.0/workspaces/%v", installation.GetWorkspaceUUID())

	request, err := http.NewRequest("GET", workspaceAPIUrl, nil)
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
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %v", accessToken.AccessToken))

	// perform actual request
	response, err := client.Do(request)
	if err != nil {
		return
	}

	defer response.Body.Close()
	if ht != nil {
		ht.Finish()
	}

	if response.StatusCode == http.StatusNotFound {
		return
	}

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("Retrieving workspace from %v failed with status code %v", workspaceAPIUrl, response.StatusCode)
		return
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &workspace)
	if err != nil {
		return
	}

	return workspace, nil
}

func (c *client) getIat(token *jwt.Token) (iat int64) {
	if token == nil || token.Claims == nil {
		return -1
	}

	switch claimType := token.Claims.(type) {
	case jwt.RegisteredClaims:
		log.Debug().Msgf("Token has claim type %T", claimType)
		return claimType.IssuedAt.Unix()
	case jwt.MapClaims:
		log.Debug().Msgf("Token has claim type %T", claimType)
		iat, hasIat := claimType["iat"]
		if hasIat {
			switch iatType := iat.(type) {
			case float64:
				return int64(iatType)
			case json.Number:
				v, _ := iatType.Int64()
				return v
			default:
				log.Warn().Msgf("IAT type %T for token is unknown", iatType)
			}
		}
	default:
		log.Warn().Msgf("Claim type %T for token is unknown", claimType)
	}

	return -1
}
