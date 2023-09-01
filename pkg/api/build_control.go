package api

import (
	"errors"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
)

const (
	AllRepositories = "*"
)

var ErrBlockedRepository = errors.New("repository is blocked from build")

type List []string

type BuildControl struct {
	Bitbucket *BitbucketBuildControl `yaml:"bitbucket,omitempty"`
	Github    *GithubBuildControl    `yaml:"github,omitempty"`
	Release   *ReleaseControl        `yaml:"release,omitempty"`
}

type BitbucketBuildControl struct {
	Allowed *BitbucketProjectsRepos `yaml:"allowed,omitempty"`
	Blocked *BitbucketProjectsRepos `yaml:"blocked,omitempty"`
}

type BitbucketProjectsRepos struct {
	Projects List `yaml:"projects,omitempty"`
	Repos    List `yaml:"repos,omitempty"`
}

type GithubBuildControl struct {
	Allowed List `yaml:"allowed,omitempty"`
	Blocked List `yaml:"blocked,omitempty"`
}

type ReleaseControl struct {
	Repositories       map[string]RepositoryReleaseControl `yaml:"repos" json:"repositories,omitempty"`
	RestrictedClusters List                                `yaml:"restrictedClusters,omitempty" json:"restrictedClusters,omitempty"`
}

type RepositoryReleaseControl struct {
	Allowed List `yaml:"allowed,omitempty" json:"allowed,omitempty"`
	Blocked List `yaml:"blocked,omitempty" json:"blocked,omitempty"`
}

func (l List) Contains(toCheck string) bool {
	for _, existing := range l {
		if existing == toCheck {
			return true
		}
	}
	return false
}

func (l List) Matches(toCheck string) bool {
	if l.Contains(toCheck) {
		return true
	}
	for _, existing := range l {
		if !strings.HasPrefix(existing, "^") {
			existing = "^" + existing
		}
		if !strings.HasSuffix(existing, "$") {
			existing = existing + "$"
		}
		re, err := regexp.Compile(existing)
		if err != nil {
			log.Error().Err(err).Msgf("error compiling regex for '%s'", existing)
			continue
		}
		if re.MatchString(toCheck) {
			return true
		}
	}
	return false
}
