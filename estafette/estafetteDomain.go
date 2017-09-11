package estafette

// CiBuilderEvent represents a finished estafette build
type CiBuilderEvent struct {
	JobName string `json:"job_name"`
}

// CiBuilderLogLine represents a line logged by the ci builder
type CiBuilderLogLine struct {
	Time     string `json:"time"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}
