package migration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	ReleasesStage          StageName = "releases"
	ReleaseLogsStage       StageName = "release_logs"
	ReleaseLogObjectsStage StageName = "release_log_objects"
	BuildsStage            StageName = "builds"
	BuildLogsStage         StageName = "build_logs"
	BuildLogObjectsStage   StageName = "build_log_objects"
	BuildVersionsStage     StageName = "build_versions"
	CallbackStage          StageName = "callback"
	CompletedStage         StageName = "completed"
)

type StageName string

type Stage interface {
	Name() StageName
	Success() Step
	Failure() Step
	Execute(ctx context.Context, task *Task) ([]Change, error)
}

type Execution func(ctx context.Context, task *Task) ([]Change, error)

type stage struct {
	name    StageName
	success Step
	failure Step
	execute Execution
}

func (s *stage) Name() StageName {
	return s.name
}

func (s *stage) Success() Step {
	return s.success
}

func (s *stage) Failure() Step {
	return s.failure
}

func (s *stage) Execute(ctx context.Context, task *Task) ([]Change, error) {
	changes, err := s.execute(ctx, task)
	task.LastStep = s.Success()
	if err != nil {
		task.LastStep = s.Failure()
		errorDetails := err.Error()
		task.ErrorDetails = &errorDetails
	}
	return changes, err
}

func Releases(fn Execution) Stage {
	return &stage{
		name:    ReleasesStage,
		success: StepReleasesDone,
		failure: StepReleasesFailed,
		execute: fn,
	}
}

func ReleaseLogs(fn Execution) Stage {
	return &stage{
		name:    ReleaseLogsStage,
		success: StepReleaseLogsDone,
		failure: StepReleaseLogsFailed,
		execute: fn,
	}
}

func ReleaseLogObjects(fn Execution) Stage {
	return &stage{
		name:    ReleaseLogObjectsStage,
		success: StepReleaseLogObjectsDone,
		failure: StepReleaseLogObjectsFailed,
		execute: fn,
	}
}

func Builds(fn Execution) Stage {
	return &stage{
		name:    BuildsStage,
		success: StepBuildsDone,
		failure: StepBuildsFailed,
		execute: fn,
	}
}

func BuildLogs(fn Execution) Stage {
	return &stage{
		name:    BuildLogsStage,
		success: StepBuildLogsDone,
		failure: StepBuildLogsFailed,
		execute: fn,
	}
}

func BuildLogObjects(fn Execution) Stage {
	return &stage{
		name:    BuildLogObjectsStage,
		success: StepBuildLogObjectsDone,
		failure: StepBuildLogObjectsFailed,
		execute: fn,
	}
}

func BuildVersions(fn Execution) Stage {
	return &stage{
		name:    BuildVersionsStage,
		success: StepBuildVersionsDone,
		failure: StepBuildVersionsFailed,
		execute: fn,
	}
}

func Callback() Stage {
	return &stage{
		name:    CallbackStage,
		success: StepCallbackDone,
		failure: StepCallbackFailed,
		execute: func(ctx context.Context, task *Task) ([]Change, error) {
			if task.CallbackURL == nil {
				builds := -1
				releases := -1
				if task.Changes[BuildsStage] != nil {
					builds = len(task.Changes[BuildsStage])
				}
				if task.Changes[ReleasesStage] != nil {
					releases = len(task.Changes[ReleasesStage])
				}
				payload, err := json.Marshal(CallbackPayload{
					ID:           task.ID,
					Status:       task.Status.String(),
					LastStep:     task.LastStep.String(),
					Builds:       builds,
					Releases:     releases,
					ErrorDetails: task.ErrorDetails,
				})
				if err != nil {
					return nil, fmt.Errorf("failed to marshal migration callback payload: %w", err)
				}
				resp, err := http.Post(*task.CallbackURL, "application/json", bytes.NewBuffer(payload))
				if err != nil {
					return nil, fmt.Errorf("failed to post migration callback: %w", err)
				}
				if resp.StatusCode <= http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
					return nil, fmt.Errorf("migration callback returned invalid status code %d", resp.StatusCode)
				}
			}
			return nil, nil
		},
	}
}

func Completed() Stage {
	return &stage{
		name:    CompletedStage,
		success: -1,
		failure: -1,
		execute: func(ctx context.Context, task *Task) ([]Change, error) {
			task.Status = StatusCompleted
			return nil, nil
		},
	}
}
