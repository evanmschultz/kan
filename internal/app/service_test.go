package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/evanschultz/kan/internal/domain"
)

type fakeRepo struct {
	projects map[string]domain.Project
	columns  map[string]domain.Column
	tasks    map[string]domain.Task
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		projects: map[string]domain.Project{},
		columns:  map[string]domain.Column{},
		tasks:    map[string]domain.Task{},
	}
}

func (f *fakeRepo) CreateProject(_ context.Context, p domain.Project) error {
	f.projects[p.ID] = p
	return nil
}

func (f *fakeRepo) UpdateProject(_ context.Context, p domain.Project) error {
	f.projects[p.ID] = p
	return nil
}

func (f *fakeRepo) GetProject(_ context.Context, id string) (domain.Project, error) {
	p, ok := f.projects[id]
	if !ok {
		return domain.Project{}, ErrNotFound
	}
	return p, nil
}

func (f *fakeRepo) ListProjects(_ context.Context, includeArchived bool) ([]domain.Project, error) {
	out := make([]domain.Project, 0, len(f.projects))
	for _, p := range f.projects {
		if !includeArchived && p.ArchivedAt != nil {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func (f *fakeRepo) CreateColumn(_ context.Context, c domain.Column) error {
	f.columns[c.ID] = c
	return nil
}

func (f *fakeRepo) UpdateColumn(_ context.Context, c domain.Column) error {
	f.columns[c.ID] = c
	return nil
}

func (f *fakeRepo) ListColumns(_ context.Context, projectID string, includeArchived bool) ([]domain.Column, error) {
	out := make([]domain.Column, 0, len(f.columns))
	for _, c := range f.columns {
		if c.ProjectID != projectID {
			continue
		}
		if !includeArchived && c.ArchivedAt != nil {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

func (f *fakeRepo) CreateTask(_ context.Context, t domain.Task) error {
	f.tasks[t.ID] = t
	return nil
}

func (f *fakeRepo) UpdateTask(_ context.Context, t domain.Task) error {
	if _, ok := f.tasks[t.ID]; !ok {
		return ErrNotFound
	}
	f.tasks[t.ID] = t
	return nil
}

func (f *fakeRepo) GetTask(_ context.Context, id string) (domain.Task, error) {
	t, ok := f.tasks[id]
	if !ok {
		return domain.Task{}, ErrNotFound
	}
	return t, nil
}

func (f *fakeRepo) ListTasks(_ context.Context, projectID string, includeArchived bool) ([]domain.Task, error) {
	out := make([]domain.Task, 0, len(f.tasks))
	for _, t := range f.tasks {
		if t.ProjectID != projectID {
			continue
		}
		if !includeArchived && t.ArchivedAt != nil {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func (f *fakeRepo) DeleteTask(_ context.Context, id string) error {
	if _, ok := f.tasks[id]; !ok {
		return ErrNotFound
	}
	delete(f.tasks, id)
	return nil
}

func TestEnsureDefaultProject(t *testing.T) {
	repo := newFakeRepo()
	idCounter := 0
	svc := NewService(repo, func() string {
		idCounter++
		return "id-" + string(rune('0'+idCounter))
	}, func() time.Time {
		return time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.EnsureDefaultProject(context.Background())
	if err != nil {
		t.Fatalf("EnsureDefaultProject() error = %v", err)
	}
	if project.Name != "Inbox" {
		t.Fatalf("unexpected project name %q", project.Name)
	}
	columns, err := svc.ListColumns(context.Background(), project.ID, false)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	if len(columns) != 3 {
		t.Fatalf("expected 3 default columns, got %d", len(columns))
	}
}

func TestCreateTaskMoveSearchAndDeleteModes(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	ids := []string{"p1", "c1", "c2", "t1"}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return now
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Project", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	col1, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	col2, err := svc.CreateColumn(context.Background(), project.ID, "Done", 1, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	task, err := svc.CreateTask(context.Background(), CreateTaskInput{
		ProjectID:   project.ID,
		ColumnID:    col1.ID,
		Title:       "Fix parser",
		Description: "Add tests for parser",
		Priority:    domain.PriorityHigh,
		Labels:      []string{"parser"},
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if task.Position != 0 {
		t.Fatalf("unexpected task position %d", task.Position)
	}

	task, err = svc.MoveTask(context.Background(), task.ID, col2.ID, 1)
	if err != nil {
		t.Fatalf("MoveTask() error = %v", err)
	}
	if task.ColumnID != col2.ID || task.Position != 1 {
		t.Fatalf("unexpected moved task %#v", task)
	}

	search, err := svc.SearchTasks(context.Background(), project.ID, "parser", false)
	if err != nil {
		t.Fatalf("SearchTasks() error = %v", err)
	}
	if len(search) != 1 {
		t.Fatalf("expected 1 search result, got %d", len(search))
	}

	if err := svc.DeleteTask(context.Background(), task.ID, ""); err != nil {
		t.Fatalf("DeleteTask(archive default) error = %v", err)
	}
	tAfterArchive, err := repo.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if tAfterArchive.ArchivedAt == nil {
		t.Fatal("expected task to be archived")
	}

	restored, err := svc.RestoreTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("RestoreTask() error = %v", err)
	}
	if restored.ArchivedAt != nil {
		t.Fatal("expected task to be restored")
	}

	if err := svc.DeleteTask(context.Background(), task.ID, DeleteModeHard); err != nil {
		t.Fatalf("DeleteTask(hard) error = %v", err)
	}
	if _, err := repo.GetTask(context.Background(), task.ID); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteTaskModeValidation(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, func() string { return "x" }, time.Now, ServiceConfig{})
	err := svc.DeleteTask(context.Background(), "task-1", DeleteMode("invalid"))
	if err != ErrInvalidDeleteMode {
		t.Fatalf("expected ErrInvalidDeleteMode, got %v", err)
	}
}

func TestRenameTask(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "old",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[task.ID] = task

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	updated, err := svc.RenameTask(context.Background(), task.ID, "new title")
	if err != nil {
		t.Fatalf("RenameTask() error = %v", err)
	}
	if updated.Title != "new title" {
		t.Fatalf("unexpected title %q", updated.Title)
	}
}

func TestUpdateTask(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "old",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[task.ID] = task

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	due := now.Add(24 * time.Hour)
	updated, err := svc.UpdateTask(context.Background(), UpdateTaskInput{
		TaskID:      task.ID,
		Title:       "new title",
		Description: "details",
		Priority:    domain.PriorityHigh,
		DueAt:       &due,
		Labels:      []string{"frontend", "backend"},
	})
	if err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	if updated.Title != "new title" || updated.Description != "details" || updated.Priority != domain.PriorityHigh {
		t.Fatalf("unexpected updated task %#v", updated)
	}
	if updated.DueAt == nil || len(updated.Labels) != 2 {
		t.Fatalf("expected due date and labels, got %#v", updated)
	}
}

func TestListAndSortHelpers(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Project", "", now)
	repo.projects[p.ID] = p
	c1, _ := domain.NewColumn("c1", p.ID, "First", 5, 0, now)
	c2, _ := domain.NewColumn("c2", p.ID, "Second", 1, 0, now)
	repo.columns[c1.ID] = c1
	repo.columns[c2.ID] = c2

	t1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  2,
		Title:     "later",
		Priority:  domain.PriorityLow,
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  1,
		Title:     "earlier",
		Priority:  domain.PriorityLow,
	}, now)
	t3, _ := domain.NewTask(domain.TaskInput{
		ID:        "t3",
		ProjectID: p.ID,
		ColumnID:  c2.ID,
		Position:  0,
		Title:     "other column",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[t1.ID] = t1
	repo.tasks[t2.ID] = t2
	repo.tasks[t3.ID] = t3

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})
	projects, err := svc.ListProjects(context.Background(), false)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}

	columns, err := svc.ListColumns(context.Background(), p.ID, false)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	if columns[0].ID != c2.ID {
		t.Fatalf("expected column c2 first after sort, got %q", columns[0].ID)
	}

	tasks, err := svc.ListTasks(context.Background(), p.ID, false)
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
	if tasks[0].ID != t2.ID || tasks[1].ID != t1.ID || tasks[2].ID != t3.ID {
		t.Fatalf("unexpected task order: %#v", tasks)
	}

	allWithEmptyQuery, err := svc.SearchTasks(context.Background(), p.ID, " ", false)
	if err != nil {
		t.Fatalf("SearchTasks(empty) error = %v", err)
	}
	if len(allWithEmptyQuery) != 3 {
		t.Fatalf("expected 3 results for empty query, got %d", len(allWithEmptyQuery))
	}
}

func TestEnsureDefaultProjectAlreadyExists(t *testing.T) {
	repo := newFakeRepo()
	now := time.Now()
	p, _ := domain.NewProject("p1", "Existing", "", now)
	repo.projects[p.ID] = p

	svc := NewService(repo, func() string { return "new-id" }, func() time.Time { return now }, ServiceConfig{})
	got, err := svc.EnsureDefaultProject(context.Background())
	if err != nil {
		t.Fatalf("EnsureDefaultProject() error = %v", err)
	}
	if got.ID != p.ID {
		t.Fatalf("expected existing project id %q, got %q", p.ID, got.ID)
	}
	if len(repo.columns) != 0 {
		t.Fatalf("expected no default columns to be inserted, got %d", len(repo.columns))
	}
}

type failingRepo struct {
	*fakeRepo
	err error
}

func (f failingRepo) ListProjects(context.Context, bool) ([]domain.Project, error) {
	return nil, f.err
}

func TestEnsureDefaultProjectErrorPropagation(t *testing.T) {
	expected := errors.New("boom")
	svc := NewService(failingRepo{fakeRepo: newFakeRepo(), err: expected}, nil, time.Now, ServiceConfig{})
	_, err := svc.EnsureDefaultProject(context.Background())
	if !errors.Is(err, expected) {
		t.Fatalf("expected wrapped error %v, got %v", expected, err)
	}
}
