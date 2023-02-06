package migration

import (
	"fmt"
	"time"
)

type Task struct {
	ID           string
	Status       Status
	LastStep     Step
	FromSource   string
	FromOwner    string
	FromName     string
	ToSource     string
	ToOwner      string
	ToName       string
	CallbackURL  *string
	ErrorDetails *string
	QueuedAt     time.Time
	UpdatedAt    time.Time
	Changes      map[StageName][]Change
}

func (m Task) FromFQN() string {
	return fmt.Sprintf("%s/%s/%s", m.FromSource, m.FromOwner, m.FromName)
}

func (m Task) ToFQN() string {
	return fmt.Sprintf("%s/%s/%s", m.ToSource, m.ToOwner, m.ToName)
}
