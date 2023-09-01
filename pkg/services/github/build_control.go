package github

import (
	"github.com/estafette/estafette-ci-api/pkg/clients/githubapi"
)

// isBuildBlocked check if repository is blocked from builds
func (s *service) isBuildBlocked(pushEvent githubapi.PushEvent) bool {
	if s.config.BuildControl == nil || s.config.BuildControl.Github == nil {
		return false
	}
	if pushEvent.Repository.FullName == "" {
		return false
	}
	repoName := pushEvent.GetRepoName()
	repoFullName := pushEvent.GetRepoFullName()

	if len(s.config.BuildControl.Github.Blocked) > 0 {
		// blocked repos
		blockedRepos := s.config.BuildControl.Github.Blocked
		if blockedRepos.Contains(repoFullName) || blockedRepos.Contains(repoName) {
			return true
		}
	}
	if s.config.BuildControl.Github.Allowed != nil {
		// allowed projects
		allowedProjects := s.config.BuildControl.Github.Allowed
		allowedRepos := s.config.BuildControl.Github.Allowed
		if allowedRepos.Contains(repoFullName) || allowedRepos.Contains(repoName) {
			return false
		}
		// block everything else if allowed projects/repos are provided
		return len(allowedProjects)+len(allowedRepos) > 0
	}
	return false
}
