package domain

import (
	"strings"
	"time"
)

// Column represents column data used by this package.
type Column struct {
	ID         string
	ProjectID  string
	Name       string
	WIPLimit   int
	Position   int
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ArchivedAt *time.Time
}

// NewColumn constructs a new value for this package.
func NewColumn(id, projectID, name string, position, wipLimit int, now time.Time) (Column, error) {
	id = strings.TrimSpace(id)
	projectID = strings.TrimSpace(projectID)
	name = strings.TrimSpace(name)
	if id == "" {
		return Column{}, ErrInvalidID
	}
	if projectID == "" {
		return Column{}, ErrInvalidID
	}
	if name == "" {
		return Column{}, ErrInvalidName
	}
	if position < 0 {
		return Column{}, ErrInvalidPosition
	}
	if wipLimit < 0 {
		return Column{}, ErrInvalidPosition
	}

	return Column{
		ID:        id,
		ProjectID: projectID,
		Name:      name,
		WIPLimit:  wipLimit,
		Position:  position,
		CreatedAt: now.UTC(),
		UpdatedAt: now.UTC(),
	}, nil
}

// Rename renames the requested operation.
func (c *Column) Rename(name string, now time.Time) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidName
	}
	c.Name = name
	c.UpdatedAt = now.UTC()
	return nil
}

// SetPosition handles set position.
func (c *Column) SetPosition(position int, now time.Time) error {
	if position < 0 {
		return ErrInvalidPosition
	}
	c.Position = position
	c.UpdatedAt = now.UTC()
	return nil
}

// Archive archives the requested operation.
func (c *Column) Archive(now time.Time) {
	ts := now.UTC()
	c.ArchivedAt = &ts
	c.UpdatedAt = ts
}

// Restore restores the requested operation.
func (c *Column) Restore(now time.Time) {
	c.ArchivedAt = nil
	c.UpdatedAt = now.UTC()
}
