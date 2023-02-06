package migration

const (
	BuildLog   LogType = "builds"
	ReleaseLog LogType = "releases"
)

const (
	StatusUnset  Status = -1
	StatusQueued Status = iota
	StatusInProgress
	StatusFailed
	StatusCompleted
	StatusCanceled
)

const (
	StepWaiting Step = iota
	StepReleasesFailed
	StepReleasesDone
	StepReleaseLogsFailed
	StepReleaseLogsDone
	StepReleaseLogObjectsFailed
	StepReleaseLogObjectsDone
	StepBuildsFailed
	StepBuildsDone
	StepBuildLogsFailed
	StepBuildLogsDone
	StepBuildLogObjectsFailed
	StepBuildLogObjectsDone
	StepBuildVersionsFailed
	StepBuildVersionsDone
	StepCallbackFailed
	StepCallbackDone
)

type LogType string
type Step int
type Status int

type Change struct {
	FromID int64
	ToID   int64
}

func StepFrom(str string) Step {
	switch str {
	case "":
		return StepWaiting
	case "releases_failed":
		return StepReleasesFailed
	case "releases_done":
		return StepReleasesDone
	case "release_logs_failed":
		return StepReleaseLogsFailed
	case "release_logs_done":
		return StepReleaseLogsDone
	case "release_log_objects_failed":
		return StepReleaseLogObjectsFailed
	case "release_log_objects_done":
		return StepReleaseLogObjectsDone
	case "builds_failed":
		return StepBuildsFailed
	case "builds_done":
		return StepBuildsDone
	case "build_logs_failed":
		return StepBuildLogsFailed
	case "build_logs_done":
		return StepBuildLogsDone
	case "build_log_objects_failed":
		return StepBuildLogObjectsFailed
	case "build_log_objects_done":
		return StepBuildLogObjectsDone
	case "build_versions_failed":
		return StepBuildVersionsFailed
	case "build_versions_done":
		return StepBuildVersionsDone
	case "callback_failed":
		return StepCallbackFailed
	case "callback_done":
		return StepCallbackDone
	default:
		return -1
	}
}

func (ms Step) String() string {
	switch ms {
	case StepWaiting:
		return "waiting"
	case StepReleasesFailed:
		return "releases_failed"
	case StepReleasesDone:
		return "releases_done"
	case StepReleaseLogsFailed:
		return "release_logs_failed"
	case StepReleaseLogsDone:
		return "release_logs_done"
	case StepReleaseLogObjectsFailed:
		return "release_log_objects_failed"
	case StepReleaseLogObjectsDone:
		return "release_log_objects_done"
	case StepBuildsFailed:
		return "builds_failed"
	case StepBuildsDone:
		return "builds_done"
	case StepBuildLogsFailed:
		return "build_logs_failed"
	case StepBuildLogsDone:
		return "build_logs_done"
	case StepBuildLogObjectsFailed:
		return "build_log_objects_failed"
	case StepBuildLogObjectsDone:
		return "build_log_objects_done"
	case StepBuildVersionsFailed:
		return "build_versions_failed"
	case StepBuildVersionsDone:
		return "build_versions_done"
	case StepCallbackFailed:
		return "callback_failed"
	case StepCallbackDone:
		return "callback_done"
	default:
		return "unknown"
	}
}

func StatusFrom(str string) Status {
	switch str {
	case "queued":
		return StatusQueued
	case "in_progress":
		return StatusInProgress
	case "failed":
		return StatusFailed
	case "completed":
		return StatusCompleted
	case "canceled":
		return StatusCanceled
	case "":
		return StatusUnset
	default:
		return -1
	}
}

func (ms Status) String() string {
	switch ms {
	case StatusQueued:
		return "queued"
	case StatusInProgress:
		return "in_progress"
	case StatusFailed:
		return "failed"
	case StatusCompleted:
		return "completed"
	case StatusCanceled:
		return "canceled"
	case StatusUnset:
		return ""
	default:
		return "unknown"
	}
}
