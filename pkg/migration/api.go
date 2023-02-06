package migration

import "time"

type Repository struct {
	Source string `json:"source,omitempty"`
	Owner  string `json:"owner,omitempty"`
	Name   string `json:"name,omitempty"`
}

type Request struct {
	From        Repository `json:"from"`
	To          Repository `json:"to"`
	Step        Step       `json:"step,omitempty"`
	CallbackURL string     `json:"callbackURL,omitempty"`
	Restart     bool       `json:"restart,omitempty"`
}

type CallbackPayload struct {
	ID           string        `json:"id"`
	Status       string        `json:"status"`
	LastStep     string        `json:"lastStep"`
	Builds       int           `json:"builds"`
	Releases     int           `json:"releases"`
	Duration     time.Duration `json:"duration"`
	ErrorDetails *string       `json:"errorDetails,omitempty"`
}

func (m Request) ToTask(requestID string) *Task {
	url := &m.CallbackURL
	if m.CallbackURL == "" {
		url = nil
	}
	status := Status(-1)
	if m.Restart {
		// if restart is true, set status to queued to restart/ resume migration
		status = StatusQueued
	}
	return &Task{
		ID:          requestID,
		Status:      status,
		LastStep:    m.Step,
		FromSource:  m.From.Source,
		FromOwner:   m.From.Owner,
		FromName:    m.From.Name,
		ToSource:    m.To.Source,
		ToOwner:     m.To.Owner,
		ToName:      m.To.Name,
		CallbackURL: url,
	}
}
