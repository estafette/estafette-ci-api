package bitbucketapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/estafette/estafette-ci-api/pkg/api"
	crypt "github.com/estafette/estafette-ci-crypt"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"github.com/sethgrid/pester"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Client is the interface for communicating with the bitbucket api
//go:generate mockgen -package=bitbucketapi -destination ./mock.go -source=client.go
type Client interface {
	GetAccessToken(ctx context.Context) (accesstoken AccessToken, err error)
	GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event RepositoryPushEvent) (valid bool, manifest string, err error)
	JobVarsFunc(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error)
	GenerateJWT() (tokenString string, err error)
	GetInstallations(ctx context.Context) (installations []*BitbucketAppInstallation, err error)
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

// GetAccessToken returns an access token to access the Bitbucket api
func (c *client) GetAccessToken(ctx context.Context) (accesstoken AccessToken, err error) {

	jtwToken, err := c.GenerateJWT()
	if err != nil {
		return
	}

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
	request.Header.Add("Authorization", fmt.Sprintf("%v %v", "JWT", jtwToken))
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

	body, err := ioutil.ReadAll(response.Body)
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

	body, err := ioutil.ReadAll(response.Body)
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
		accessToken, err := c.GetAccessToken(ctx)
		if err != nil {
			return "", err
		}

		return accessToken.AccessToken, nil
	}
}

func (c *client) GenerateJWT() (tokenString string, err error) {

	// Create the token
	token := jwt.New(jwt.GetSigningMethod("HS256"))
	claims := token.Claims.(jwt.MapClaims)

	now := time.Now().UTC()
	expiry := now.Add(time.Duration(180) * time.Second)

	// set required claims
	claims["iss"] = c.config.Integrations.Bitbucket.Key
	claims["iat"] = now.Unix()
	claims["exp"] = expiry.Unix()
	claims["sub"] = c.config.Integrations.Bitbucket.ClientKey

	// sign the token
	return token.SignedString([]byte(c.config.Integrations.Bitbucket.SharedSecret))
}

var installationsCache []*BitbucketAppInstallation

const bitbucketConfigmapName = "estafette-ci-api.bitbucket"

func (c *client) GetInstallations(ctx context.Context) (installations []*BitbucketAppInstallation, err error) {
	// get from cache
	if installationsCache != nil {
		return installationsCache, nil
	}

	installations = make([]*BitbucketAppInstallation, 0)

	configMap, err := c.kubeClientset.CoreV1().ConfigMaps(c.getCurrentNamespace()).Get(ctx, bitbucketConfigmapName, metav1.GetOptions{})
	if err != nil || configMap == nil {
		return installations, nil
	}

	if data, ok := configMap.Data["installations"]; ok {
		err = json.Unmarshal([]byte(data), &installations)
		if err != nil {
			return
		}

		err = c.decryptSharedSecrets(ctx, installations)
		if err != nil {
			return
		}

		// add to cache
		installationsCache = installations
	}

	return
}

func (c *client) AddInstallation(ctx context.Context, installation BitbucketAppInstallation) (err error) {
	installations, err := c.GetInstallations(ctx)
	if err != nil {
		return
	}

	if installations == nil {
		installations = make([]*BitbucketAppInstallation, 0)
	}

	// check if installation(s) with key and clientKey exists, if not add, otherwise update
	installationExists := false
	for _, inst := range installations {
		if inst.Key == installation.Key && inst.ClientKey == installation.ClientKey {
			installationExists = true

			inst.BaseApiURL = installation.BaseApiURL
			inst.SharedSecret = installation.SharedSecret
		}
	}

	if !installationExists {
		installations = append(installations, &installation)
	}

	err = c.upsertConfigmap(ctx, installations)
	if err != nil {
		return
	}

	return
}

func (c *client) RemoveInstallation(ctx context.Context, installation BitbucketAppInstallation) (err error) {
	installations, err := c.GetInstallations(ctx)
	if err != nil {
		return
	}

	if installations == nil {
		installations = make([]*BitbucketAppInstallation, 0)
	}

	// check if installation(s) with key and clientKey exists, then remove
	for i, inst := range installations {
		if inst.Key == installation.Key && inst.ClientKey == installation.ClientKey {
			installations = append(installations[:i], installations[i+1:]...)
		}
	}

	err = c.upsertConfigmap(ctx, installations)
	if err != nil {
		return
	}

	return
}

func (c *client) upsertConfigmap(ctx context.Context, installations []*BitbucketAppInstallation) (err error) {

	err = c.encryptSharedSecrets(ctx, installations)
	if err != nil {
		return
	}

	// marshal to json
	data, err := json.Marshal(installations)
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
				"installations": string(data),
			},
		}
		_, err = c.kubeClientset.CoreV1().ConfigMaps(c.getCurrentNamespace()).Create(ctx, configMap, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	} else {
		// update configmap
		configMap.Data["installations"] = string(data)
		_, err = c.kubeClientset.CoreV1().ConfigMaps(c.getCurrentNamespace()).Update(ctx, configMap, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	// update cache
	installationsCache = installations

	return
}

func (c *client) getCurrentNamespace() string {
	namespace, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed reading namespace")
	}

	return string(namespace)
}

func (c *client) encryptSharedSecrets(ctx context.Context, installations []*BitbucketAppInstallation) (err error) {
	for _, installation := range installations {
		encryptedSharedSecret, encryptErr := c.secretHelper.EncryptEnvelope(installation.SharedSecret, crypt.DefaultPipelineAllowList)
		if encryptErr != nil {
			return encryptErr
		}
		installation.SharedSecret = encryptedSharedSecret
	}

	return nil
}

func (c *client) decryptSharedSecrets(ctx context.Context, installations []*BitbucketAppInstallation) (err error) {
	for _, installation := range installations {
		decryptedSharedSecret, _, decryptErr := c.secretHelper.DecryptEnvelope(installation.SharedSecret, "")
		if decryptErr != nil {
			return decryptErr
		}
		installation.SharedSecret = decryptedSharedSecret
	}

	return nil
}

func (c *client) GetWorkspace(ctx context.Context, installation BitbucketAppInstallation) (workspace *Workspace, err error) {
	accessToken, err := c.GetAccessToken(ctx)
	if err != nil {
		return
	}

	// create client, in order to add headers
	client := pester.NewExtendedClient(&http.Client{Transport: &nethttp.Transport{}})
	client.MaxRetries = 3
	client.Backoff = pester.ExponentialJitterBackoff
	client.KeepLog = true
	client.Timeout = time.Second * 10

	workspaceAPIUrl := fmt.Sprintf("/2.0/workspaces/%v", installation.GetWorkspaceUUID())

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

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &workspace)
	if err != nil {
		return
	}

	return workspace, nil
}
