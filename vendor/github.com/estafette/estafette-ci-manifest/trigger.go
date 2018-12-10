package manifest

// EstafetteTrigger defines an automated trigger to trigger a build or a release
type EstafetteTrigger struct {
	Event  string                  `yaml:"event,omitempty" json:"event,omitempty"`
	Filter *EstafetteTriggerFilter `yaml:"filter,omitempty" json:"filter,omitempty"`
	Run    *EstafetteTriggerRun    `yaml:"run,omitempty" json:"run,omitempty"`
}

// EstafetteTriggerFilter filters the triggered event
type EstafetteTriggerFilter struct {
	// pipeline related filtering
	Pipeline string `yaml:"pipeline,omitempty" json:"pipeline,omitempty"`
	Target   string `yaml:"target,omitempty" json:"target,omitempty"`
	Action   string `yaml:"action,omitempty" json:"action,omitempty"`
	Status   string `yaml:"status,omitempty" json:"status,omitempty"`
	Branch   string `yaml:"branch,omitempty" json:"branch,omitempty"`

	// cron related filtering
	Cron string `yaml:"cron,omitempty" json:"cron,omitempty"`

	// docker related filtering
	Image string `yaml:"image,omitempty" json:"image,omitempty"`
	Tag   string `yaml:"tag,omitempty" json:"tag,omitempty"`
}

// EstafetteTriggerRun determines what action to take on a trigger
type EstafetteTriggerRun struct {
	Branch string `yaml:"branch,omitempty" json:"branch,omitempty"`
	Action string `yaml:"action,omitempty" json:"action,omitempty"`
}

// UnmarshalYAML customizes unmarshalling an EstafetteTrigger
func (t *EstafetteTrigger) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {

	var aux struct {
		Event  string                  `yaml:"event,omitempty" json:"event,omitempty"`
		Filter *EstafetteTriggerFilter `yaml:"filter,omitempty" json:"filter,omitempty"`
		Run    *EstafetteTriggerRun    `yaml:"run,omitempty" json:"run,omitempty"`
	}

	// unmarshal to auxiliary type
	if err := unmarshal(&aux); err != nil {
		return err
	}

	// map auxiliary properties
	t.Event = aux.Event
	t.Filter = aux.Filter
	t.Run = aux.Run

	t.SetDefaults()

	return nil
}

// SetDefaults sets event-specific defaults
func (t *EstafetteTrigger) SetDefaults() {

	// set filter default
	switch t.Event {
	case "pipeline-build-started",
		"pipeline-build-finished",
		"pipeline-release-started",
		"pipeline-release-finished":
		if t.Filter == nil {
			t.Filter = &EstafetteTriggerFilter{}
		}
	}

	// set filter branch default
	switch t.Event {
	case "pipeline-build-started",
		"pipeline-build-finished":
		if t.Filter.Branch == "" {
			t.Filter.Branch = "master"
		}

	case "pipeline-release-started",
		"pipeline-release-finished":
		if t.Filter.Branch == "" {
			t.Filter.Branch = ".+"
		}
	}

	// set filter pipeline default
	switch t.Event {
	case "pipeline-release-started",
		"pipeline-release-finished":
		if t.Filter.Pipeline == "" {
			t.Filter.Pipeline = "this"
		}
	}

	// set filter pipeline default
	switch t.Event {
	case "pipeline-build-finished",
		"pipeline-release-finished":
		if t.Filter.Status == "" {
			t.Filter.Status = "succeeded"
		}
	}

	// set run default
	switch t.Event {
	case "pipeline-build-started",
		"pipeline-build-finished",
		"pipeline-release-started",
		"pipeline-release-finished":
		if t.Run == nil {
			t.Run = &EstafetteTriggerRun{}
		}
	}

	// set run branch default
	switch t.Event {
	case "pipeline-build-started",
		"pipeline-build-finished":
		if t.Run.Branch == "" {
			t.Run.Branch = "master"
		}

	case "pipeline-release-started",
		"pipeline-release-finished":
		if t.Run.Branch == "" {
			t.Run.Branch = ".+"
		}
	}
}

// Equals returns true if the event, all filters and run options match
func (t *EstafetteTrigger) Equals(trigger *EstafetteTrigger) bool {

	if t == nil && trigger == nil {
		return true
	}

	if t == nil || trigger == nil {
		return false
	}

	if t.Event != trigger.Event {
		return false
	}

	if !t.Filter.Equals(trigger.Filter) {
		return false
	}

	if !t.Run.Equals(trigger.Run) {
		return false
	}

	return true
}

// Equals returns true if all filters options match
func (f *EstafetteTriggerFilter) Equals(filter *EstafetteTriggerFilter) bool {

	if f == nil && filter == nil {
		return true
	}

	if f == nil || filter == nil {
		return false
	}

	if f.Pipeline != filter.Pipeline {
		return false
	}

	if f.Target != filter.Target {
		return false
	}

	if f.Action != filter.Action {
		return false
	}

	if f.Status != filter.Status {
		return false
	}

	if f.Branch != filter.Branch {
		return false
	}

	if f.Cron != filter.Cron {
		return false
	}

	if f.Image != filter.Image {
		return false
	}

	if f.Tag != filter.Tag {
		return false
	}

	return true
}

// Equals returns true if all run options match
func (r *EstafetteTriggerRun) Equals(run *EstafetteTriggerRun) bool {

	if r == nil && run == nil {
		return true
	}

	if r == nil || run == nil {
		return false
	}

	if r.Branch != run.Branch {
		return false
	}

	if r.Action != run.Action {
		return false
	}

	return true
}
