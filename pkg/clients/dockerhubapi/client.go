package dockerhubapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	"github.com/sethgrid/pester"
)

// Client communicates with docker hub api
//
//go:generate mockgen -package=dockerhubapi -destination ./mock.go -source=client.go
type Client interface {
	GetToken(ctx context.Context, repository string) (token DockerHubToken, err error)
	GetDigest(ctx context.Context, token DockerHubToken, repository string, tag string) (digest DockerImageDigest, err error)
	GetDigestCached(ctx context.Context, repository string, tag string) (digest DockerImageDigest, err error)
}

// NewClient returns a new dockerhubapi.Client
func NewClient() Client {
	return &client{
		tokens:  make(map[string]DockerHubToken),
		digests: make(map[string]DockerImageDigest),
	}
}

type client struct {
	tokens  map[string]DockerHubToken
	digests map[string]DockerImageDigest
}

// GetToken creates an estafette-ci-builder job in Kubernetes to run the estafette build
func (c *client) GetToken(ctx context.Context, repository string) (token DockerHubToken, err error) {

	url := fmt.Sprintf("https://auth.docker.io/token?service=registry.docker.io&scope=repository:%v:pull", repository)

	response, err := pester.Get(url)
	if err != nil {
		return
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return
	}

	// unmarshal json body
	err = json.Unmarshal(body, &token)
	if err != nil {
		return
	}

	return
}

func (c *client) GetDigest(ctx context.Context, token DockerHubToken, repository string, tag string) (digest DockerImageDigest, err error) {

	url := fmt.Sprintf("https://index.docker.io/v2/%v/manifests/%v", repository, tag)

	// create client, in order to add headers
	client := pester.NewExtendedClient(&http.Client{Transport: &nethttp.Transport{}})
	client.MaxRetries = 3
	client.Backoff = pester.ExponentialJitterBackoff
	client.KeepLog = true
	client.Timeout = time.Second * 10
	request, err := http.NewRequest("HEAD", url, nil)
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
	request.Header.Add("Authorization", fmt.Sprintf("%v %v", "Bearer", token.Token))
	request.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	// perform actual request
	response, err := client.Do(request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	if ht != nil {
		ht.Finish()
	}

	digest = DockerImageDigest{
		Digest:    response.Header.Get("Docker-Content-Digest"),
		ExpiresIn: 300,
		FetchedAt: time.Now().UTC(),
	}

	return
}

func (c *client) GetDigestCached(ctx context.Context, repository string, tag string) (digest DockerImageDigest, err error) {

	key := fmt.Sprintf("%v:%v", repository, tag)

	// fetch digest from cache or renew
	if val, ok := c.digests[key]; ok && !val.IsExpired() {
		// digest exists and is still valid
		digest = val
		return
	}

	// fetch token from cache or renew
	var token DockerHubToken
	if val, ok := c.tokens[repository]; !ok || val.IsExpired() {
		// token doesn't exist or is no longer valid, renew
		token, err = c.GetToken(ctx, repository)
		if err != nil {
			return
		}
		c.tokens[repository] = token
	}
	token = c.tokens[repository]

	// digest doesn't exist or is no longer valid, renew
	digest, err = c.GetDigest(ctx, token, repository, tag)
	if err != nil {
		return
	}
	c.digests[key] = digest

	return
}

// DockerHubToken is a bearer token to authenticate requests with
type DockerHubToken struct {
	Token     string    `json:"token"`
	ExpiresIn int       `json:"expires_in"`
	IssuedAt  time.Time `json:"issued_at"`
}

func (t *DockerHubToken) ExpiresAt() time.Time {
	return t.IssuedAt.Add(time.Duration(t.ExpiresIn) * time.Second)
}

func (t *DockerHubToken) IsExpired() bool {
	return time.Now().UTC().After(t.ExpiresAt())
}

type DockerImageDigest struct {
	Digest    string
	ExpiresIn int
	FetchedAt time.Time
}

func (t *DockerImageDigest) ExpiresAt() time.Time {
	return t.FetchedAt.Add(time.Duration(t.ExpiresIn) * time.Second)
}

func (t *DockerImageDigest) IsExpired() bool {
	return time.Now().UTC().After(t.ExpiresAt())
}
