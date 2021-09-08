package builderapi

import (
	"fmt"

	"github.com/estafette/estafette-ci-api/pkg/clients/database"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
)

// CiBuilderParams contains the parameters required to create a ci builder job
type CiBuilderParams struct {
	BuilderConfig        contracts.BuilderConfig
	EnvironmentVariables map[string]string
	OperatingSystem      manifest.OperatingSystem
	JobResources         database.JobResources
}

// GetFullRepoPath returns the full path of the pipeline / build / release repository with source, owner and name
func (cbp *CiBuilderParams) GetFullRepoPath() string {
	if cbp.BuilderConfig.Git == nil {
		return ""
	}
	return fmt.Sprintf("%v/%v/%v", cbp.BuilderConfig.Git.RepoSource, cbp.BuilderConfig.Git.RepoOwner, cbp.BuilderConfig.Git.RepoName)
}

type ZeroLogLine struct {
	TailLogLine *contracts.TailLogLine `json:"tailLogLine"`
}
