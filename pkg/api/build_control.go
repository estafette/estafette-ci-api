package api

type List []string

type BuildControl struct {
	Bitbucket BitbucketBuildControl `yaml:"bitbucket,omitempty"`
	Github    GithubBuildControl    `yaml:"github,omitempty"`
}

type BitbucketBuildControl struct {
	Allowed BitbucketProjectsRepos `yaml:"allowed,omitempty"`
	Blocked BitbucketProjectsRepos `yaml:"blocked,omitempty"`
}

type BitbucketProjectsRepos struct {
	Projects List `yaml:"projects,omitempty"`
	Repos    List `yaml:"repos,omitempty"`
}

type GithubBuildControl struct {
	Allowed List `yaml:"allowed,omitempty"`
	Blocked List `yaml:"blocked,omitempty"`
}

func (l List) Contains(toCheck string) bool {
	for _, existing := range l {
		if existing == toCheck {
			return true
		}
	}
	return false
}
