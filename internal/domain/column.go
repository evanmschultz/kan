package domain

import (
	"strings"
	"time"
)

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

func (c *Column) Rename(name string, now time.Time) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidName
	}
	c.Name = name
	c.UpdatedAt = now.UTC()
	return nil
}

func (c *Column) SetPosition(position int, now time.Time) error {
	if position < 0 {
		return ErrInvalidPosition
	}
	c.Position = position
	c.UpdatedAt = now.UTC()
	return nil
}

func (c *Column) Archive(now time.Time) {
	ts := now.UTC()
	c.ArchivedAt = &ts
	c.UpdatedAt = ts
}

func (c *Column) Restore(now time.Time) {
	c.ArchivedAt = nil
	c.UpdatedAt = now.UTC()
}
