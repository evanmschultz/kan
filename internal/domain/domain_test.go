package domain

import (
	"testing"
	"time"
)

func TestNewProjectAndSlug(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, err := NewProject("p1", "  My Big Project!  ", " desc ", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if p.Slug != "my-big-project" {
		t.Fatalf("unexpected slug %q", p.Slug)
	}
	if p.Name != "My Big Project!" {
		t.Fatalf("unexpected name %q", p.Name)
	}
}

func TestNewProjectValidation(t *testing.T) {
	now := time.Now()
	if _, err := NewProject("", "ok", "", now); err != ErrInvalidID {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
	if _, err := NewProject("id", "   ", "", now); err != ErrInvalidName {
		t.Fatalf("expected ErrInvalidName, got %v", err)
	}
}

func TestProjectArchiveRestore(t *testing.T) {
	now := time.Now()
	p, err := NewProject("p1", "test", "", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	later := now.Add(time.Minute)
	p.Archive(later)
	if p.ArchivedAt == nil {
		t.Fatal("expected archived_at to be set")
	}
	p.Restore(later.Add(time.Minute))
	if p.ArchivedAt != nil {
		t.Fatal("expected archived_at to be nil")
	}
}

func TestNewColumnValidation(t *testing.T) {
	now := time.Now()
	_, err := NewColumn("c1", "p1", "todo", -1, 0, now)
	if err != ErrInvalidPosition {
		t.Fatalf("expected ErrInvalidPosition, got %v", err)
	}
	_, err = NewColumn("c1", "p1", "todo", 0, -1, now)
	if err != ErrInvalidPosition {
		t.Fatalf("expected ErrInvalidPosition, got %v", err)
	}
}

func TestColumnMutations(t *testing.T) {
	now := time.Now()
	c, err := NewColumn("c1", "p1", "todo", 0, 5, now)
	if err != nil {
		t.Fatalf("NewColumn() error = %v", err)
	}
	if err := c.Rename("  done ", now.Add(time.Minute)); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if c.Name != "done" {
		t.Fatalf("unexpected column name %q", c.Name)
	}
	if err := c.SetPosition(3, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("SetPosition() error = %v", err)
	}
	if c.Position != 3 {
		t.Fatalf("unexpected position %d", c.Position)
	}
}

func TestNewTaskDefaultsAndLabels(t *testing.T) {
	now := time.Now()
	due := now.Add(24 * time.Hour)
	task, err := NewTask(TaskInput{
		ID:        "t1",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "  Ship feature ",
		DueAt:     &due,
		Labels:    []string{"Backend", "backend", "  ", "Urgent"},
	}, now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	if task.Priority != PriorityMedium {
		t.Fatalf("expected default medium, got %q", task.Priority)
	}
	if task.Title != "Ship feature" {
		t.Fatalf("unexpected title %q", task.Title)
	}
	if len(task.Labels) != 2 || task.Labels[0] != "backend" || task.Labels[1] != "urgent" {
		t.Fatalf("unexpected labels %#v", task.Labels)
	}
}

func TestNewTaskValidation(t *testing.T) {
	now := time.Now()
	_, err := NewTask(TaskInput{
		ID:        "t1",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "x",
		Priority:  Priority("bad"),
	}, now)
	if err != ErrInvalidPriority {
		t.Fatalf("expected ErrInvalidPriority, got %v", err)
	}
}

func TestTaskMoveUpdateArchiveRestore(t *testing.T) {
	now := time.Now()
	task, err := NewTask(TaskInput{
		ID:        "t1",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "x",
		Priority:  PriorityLow,
	}, now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}

	if err := task.Move("c2", 2, now.Add(time.Minute)); err != nil {
		t.Fatalf("Move() error = %v", err)
	}
	if task.ColumnID != "c2" || task.Position != 2 {
		t.Fatalf("unexpected move state: %#v", task)
	}

	due := now.Add(2 * time.Hour)
	err = task.UpdateDetails("new", "desc", PriorityHigh, &due, []string{"A", "a", "B"}, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("UpdateDetails() error = %v", err)
	}
	if task.Title != "new" || task.Priority != PriorityHigh {
		t.Fatalf("unexpected task update state %#v", task)
	}
	task.Archive(now.Add(3 * time.Minute))
	if task.ArchivedAt == nil {
		t.Fatal("expected archived_at to be set")
	}
	task.Restore(now.Add(4 * time.Minute))
	if task.ArchivedAt != nil {
		t.Fatal("expected archived_at nil")
	}
}
