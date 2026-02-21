package domain

import (
	"strings"
	"time"
)

type Project struct {
	ID          string
	Slug        string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ArchivedAt  *time.Time
}

func NewProject(id, name, description string, now time.Time) (Project, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	if id == "" {
		return Project{}, ErrInvalidID
	}
	if name == "" {
		return Project{}, ErrInvalidName
	}

	slug := normalizeSlug(name)

	return Project{
		ID:          id,
		Slug:        slug,
		Name:        name,
		Description: strings.TrimSpace(description),
		CreatedAt:   now.UTC(),
		UpdatedAt:   now.UTC(),
	}, nil
}

func (p *Project) Rename(name string, now time.Time) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidName
	}
	p.Name = name
	p.Slug = normalizeSlug(name)
	p.UpdatedAt = now.UTC()
	return nil
}

func (p *Project) Archive(now time.Time) {
	ts := now.UTC()
	p.ArchivedAt = &ts
	p.UpdatedAt = ts
}

func (p *Project) Restore(now time.Time) {
	p.ArchivedAt = nil
	p.UpdatedAt = now.UTC()
}

func normalizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}

	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			prevDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	return out
}
