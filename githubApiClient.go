package main

// GithubApiClient is the object to perform Github api calls with
type GithubApiClient struct {
	baseURL string
}

// New returns an initialized APIClient
func New() *GithubApiClient {

	return &GithubApiClient{
		baseURL: "https://api.github.com/",
	}
}

func (gh *GithubApiClient) requestJWT() (jwt string, err error) {

	// from: https://developer.github.com/apps/building-integrations/setting-up-and-registering-github-apps/about-authentication-options-for-github-apps/

	// require 'openssl'
	// require 'jwt'  # https://rubygems.org/gems/jwt

	// # Private key contents
	// private_pem = File.read(path_to_pem)
	// private_key = OpenSSL::PKey::RSA.new(private_pem)

	// # Generate the JWT
	// payload = {
	// # issued at time
	// iat: Time.now.to_i,
	// # JWT expiration time (10 minute maximum)
	// exp: Time.now.to_i + (10 * 60),
	// # GitHub App's identifier
	// iss: 42 <replace with your id>
	// }

	// jwt = JWT.encode(payload, private_key, "RS256")

	// curl -i -H "Authorization: Bearer $JWT" -H "Accept: application/vnd.github.machine-man-preview+json" https://api.github.com/app

	return
}

func (gh *GithubApiClient) listInstallations() (i []GithubAppInstallation, err error) {

	// curl -i -X POST \
	// -H "Authorization: Bearer $JWT" \
	// -H "Accept: application/vnd.github.machine-man-preview+json" \
	// https://api.github.com/installations/:installation_id/access_tokens

	// response:

	// {
	// "token": "v1.1f699f1069f60xxx",
	// "expires_at": "2016-07-11T22:14:10Z"
	// }

	return
}

func (gh *GithubApiClient) getAuthenticatedRepositoryURL() (url string, err error) {

	// git clone https://x-access-token:<token>@github.com/owner/repo.git

	return
}
