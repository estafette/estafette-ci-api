package estafette

// CiBuilderEvent represents a finished estafette build
type CiBuilderEvent struct {
	JobName string `json:"job_name"`
}
