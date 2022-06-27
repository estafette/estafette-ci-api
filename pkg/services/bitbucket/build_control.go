package bitbucket

import (
	"github.com/estafette/estafette-ci-api/pkg/clients/bitbucketapi"
)

// isBuildBlocked check if repository is blocked from builds
func (s *service) isBuildBlocked(pushEvent bitbucketapi.RepositoryPushEvent) bool {
	if s.config.BuildControl == nil || s.config.BuildControl.Bitbucket == nil {
		return false
	}
	if pushEvent.Repository.FullName == "" || pushEvent.Repository.Project == nil {
		return false
	}
	project := pushEvent.Repository.Project
	repoName := pushEvent.GetRepoName()

	if s.config.BuildControl.Bitbucket.Blocked != nil {
		// blocked projects
		blockedProjects := s.config.BuildControl.Bitbucket.Blocked.Projects
		if blockedProjects.Contains(project.Key) ||
			blockedProjects.Contains(project.Name) {
			return true
		}
		// blocked repos
		blockedRepos := s.config.BuildControl.Bitbucket.Blocked.Repos
		if blockedRepos.Contains(repoName) {
			return true
		}
	}

	if s.config.BuildControl.Bitbucket.Allowed != nil {
		// allowed projects
		allowedProjects := s.config.BuildControl.Bitbucket.Allowed.Projects
		if allowedProjects.Contains(project.Key) ||
			allowedProjects.Contains(project.Name) {
			return false
		}
		// allowed repos
		allowedRepos := s.config.BuildControl.Bitbucket.Allowed.Repos
		if allowedRepos.Contains(repoName) {
			return false
		}

		// block everything else if allowed projects/repos are provided
		return len(allowedProjects)+len(allowedRepos) > 0
	}
	return false
}
