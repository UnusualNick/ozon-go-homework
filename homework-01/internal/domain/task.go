package domain

import (
	"fmt"
	"strings"
	"time"
)

type Status string

const (
	StatusPlanned    Status = "planned"
	StatusInProgress Status = "in_progress"
	StatusCanceled   Status = "canceled"
	StatusDone       Status = "done"
)

func (s Status) Valid() bool {
	switch s {
	case StatusPlanned, StatusInProgress, StatusCanceled, StatusDone:
		return true
	default:
		return false
	}
}

func ParseStatus(value string) (Status, error) {
	status := Status(strings.ToLower(strings.TrimSpace(value)))
	if !status.Valid() {
		return "", fmt.Errorf("unknown status %q (allowed: planned, in_progress, canceled, done)", value)
	}
	return status, nil
}

type Task struct {
	ID       int64     `json:"id"`
	Title    string    `json:"title"`
	Status   Status    `json:"status"`
	Deadline time.Time `json:"deadline"`
}

func (t Task) Validate() error {
	if t.ID < 0 {
		return fmt.Errorf("id must not be negative")
	}
	if strings.TrimSpace(t.Title) == "" {
		return fmt.Errorf("title must not be empty")
	}
	if !t.Status.Valid() {
		return fmt.Errorf("invalid status %q", t.Status)
	}
	if t.Deadline.IsZero() {
		return fmt.Errorf("deadline must not be empty")
	}
	return nil
}
