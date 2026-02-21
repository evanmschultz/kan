package domain

import (
	"slices"
	"strings"
	"time"
)

// Priority represents priority data used by this package.
type Priority string

// PriorityLow and related constants define package defaults.
const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

// validPriorities stores a package-level helper value.
var validPriorities = []Priority{PriorityLow, PriorityMedium, PriorityHigh}

// Task represents task data used by this package.
type Task struct {
	ID          string
	ProjectID   string
	ColumnID    string
	Position    int
	Title       string
	Description string
	Priority    Priority
	DueAt       *time.Time
	Labels      []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ArchivedAt  *time.Time
}

// TaskInput holds input values for task operations.
type TaskInput struct {
	ID          string
	ProjectID   string
	ColumnID    string
	Position    int
	Title       string
	Description string
	Priority    Priority
	DueAt       *time.Time
	Labels      []string
}

// NewTask constructs a new value for this package.
func NewTask(in TaskInput, now time.Time) (Task, error) {
	in.ID = strings.TrimSpace(in.ID)
	in.ProjectID = strings.TrimSpace(in.ProjectID)
	in.ColumnID = strings.TrimSpace(in.ColumnID)
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)

	if in.ID == "" {
		return Task{}, ErrInvalidID
	}
	if in.ProjectID == "" {
		return Task{}, ErrInvalidID
	}
	if in.ColumnID == "" {
		return Task{}, ErrInvalidColumnID
	}
	if in.Title == "" {
		return Task{}, ErrInvalidTitle
	}
	if in.Position < 0 {
		return Task{}, ErrInvalidPosition
	}

	if in.Priority == "" {
		in.Priority = PriorityMedium
	}
	if !slices.Contains(validPriorities, in.Priority) {
		return Task{}, ErrInvalidPriority
	}

	labels := normalizeLabels(in.Labels)

	return Task{
		ID:          in.ID,
		ProjectID:   in.ProjectID,
		ColumnID:    in.ColumnID,
		Position:    in.Position,
		Title:       in.Title,
		Description: in.Description,
		Priority:    in.Priority,
		DueAt:       normalizeDueAt(in.DueAt),
		Labels:      labels,
		CreatedAt:   now.UTC(),
		UpdatedAt:   now.UTC(),
	}, nil
}

// Move moves the requested operation.
func (t *Task) Move(columnID string, position int, now time.Time) error {
	columnID = strings.TrimSpace(columnID)
	if columnID == "" {
		return ErrInvalidColumnID
	}
	if position < 0 {
		return ErrInvalidPosition
	}
	t.ColumnID = columnID
	t.Position = position
	t.UpdatedAt = now.UTC()
	return nil
}

// UpdateDetails updates state for the requested operation.
func (t *Task) UpdateDetails(title, description string, priority Priority, dueAt *time.Time, labels []string, now time.Time) error {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if title == "" {
		return ErrInvalidTitle
	}
	if !slices.Contains(validPriorities, priority) {
		return ErrInvalidPriority
	}
	t.Title = title
	t.Description = description
	t.Priority = priority
	t.DueAt = normalizeDueAt(dueAt)
	t.Labels = normalizeLabels(labels)
	t.UpdatedAt = now.UTC()
	return nil
}

// Archive archives the requested operation.
func (t *Task) Archive(now time.Time) {
	ts := now.UTC()
	t.ArchivedAt = &ts
	t.UpdatedAt = ts
}

// Restore restores the requested operation.
func (t *Task) Restore(now time.Time) {
	t.ArchivedAt = nil
	t.UpdatedAt = now.UTC()
}

// normalizeDueAt normalizes due at.
func normalizeDueAt(dueAt *time.Time) *time.Time {
	if dueAt == nil {
		return nil
	}
	ts := dueAt.UTC().Truncate(time.Second)
	return &ts
}

// normalizeLabels normalizes labels.
func normalizeLabels(labels []string) []string {
	out := make([]string, 0, len(labels))
	seen := map[string]struct{}{}
	for _, raw := range labels {
		label := strings.ToLower(strings.TrimSpace(raw))
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		out = append(out, label)
	}
	slices.Sort(out)
	return out
}
