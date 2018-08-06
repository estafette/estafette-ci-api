package contracts

// Release represents a release of a pipeline
type Release struct {
	Name           string `json:"name"`
	ReleaseVersion string `json:"releaseVersion,omitempty"`
	ReleaseStatus  string `json:"releaseStatus,omitempty"`
}
