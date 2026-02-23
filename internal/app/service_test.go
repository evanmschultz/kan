package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/evanschultz/kan/internal/domain"
)

// fakeRepo represents fake repo data used by this package.
type fakeRepo struct {
	projects     map[string]domain.Project
	columns      map[string]domain.Column
	tasks        map[string]domain.Task
	changeEvents map[string][]domain.ChangeEvent
}

// newFakeRepo constructs fake repo.
func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		projects:     map[string]domain.Project{},
		columns:      map[string]domain.Column{},
		tasks:        map[string]domain.Task{},
		changeEvents: map[string][]domain.ChangeEvent{},
	}
}

// CreateProject creates project.
func (f *fakeRepo) CreateProject(_ context.Context, p domain.Project) error {
	f.projects[p.ID] = p
	return nil
}

// UpdateProject updates state for the requested operation.
func (f *fakeRepo) UpdateProject(_ context.Context, p domain.Project) error {
	f.projects[p.ID] = p
	return nil
}

// GetProject returns project.
func (f *fakeRepo) GetProject(_ context.Context, id string) (domain.Project, error) {
	p, ok := f.projects[id]
	if !ok {
		return domain.Project{}, ErrNotFound
	}
	return p, nil
}

// ListProjects lists projects.
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

// CreateColumn creates column.
func (f *fakeRepo) CreateColumn(_ context.Context, c domain.Column) error {
	f.columns[c.ID] = c
	return nil
}

// UpdateColumn updates state for the requested operation.
func (f *fakeRepo) UpdateColumn(_ context.Context, c domain.Column) error {
	f.columns[c.ID] = c
	return nil
}

// ListColumns lists columns.
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

// CreateTask creates task.
func (f *fakeRepo) CreateTask(_ context.Context, t domain.Task) error {
	f.tasks[t.ID] = t
	return nil
}

// UpdateTask updates state for the requested operation.
func (f *fakeRepo) UpdateTask(_ context.Context, t domain.Task) error {
	if _, ok := f.tasks[t.ID]; !ok {
		return ErrNotFound
	}
	f.tasks[t.ID] = t
	return nil
}

// GetTask returns task.
func (f *fakeRepo) GetTask(_ context.Context, id string) (domain.Task, error) {
	t, ok := f.tasks[id]
	if !ok {
		return domain.Task{}, ErrNotFound
	}
	return t, nil
}

// ListTasks lists tasks.
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

// DeleteTask deletes task.
func (f *fakeRepo) DeleteTask(_ context.Context, id string) error {
	if _, ok := f.tasks[id]; !ok {
		return ErrNotFound
	}
	delete(f.tasks, id)
	return nil
}

// ListProjectChangeEvents lists change events.
func (f *fakeRepo) ListProjectChangeEvents(_ context.Context, projectID string, limit int) ([]domain.ChangeEvent, error) {
	events := append([]domain.ChangeEvent(nil), f.changeEvents[projectID]...)
	if limit <= 0 || limit >= len(events) {
		return events, nil
	}
	return events[:limit], nil
}

// TestEnsureDefaultProject verifies behavior for the covered scenario.
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

// TestCreateTaskMoveSearchAndDeleteModes verifies behavior for the covered scenario.
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

	search, err := svc.SearchTaskMatches(context.Background(), SearchTasksFilter{
		ProjectID: project.ID,
		Query:     "parser",
	})
	if err != nil {
		t.Fatalf("SearchTaskMatches() error = %v", err)
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

// TestDeleteTaskModeValidation verifies behavior for the covered scenario.
func TestDeleteTaskModeValidation(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, func() string { return "x" }, time.Now, ServiceConfig{})
	err := svc.DeleteTask(context.Background(), "task-1", DeleteMode("invalid"))
	if err != ErrInvalidDeleteMode {
		t.Fatalf("expected ErrInvalidDeleteMode, got %v", err)
	}
}

// TestRenameTask verifies behavior for the covered scenario.
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

// TestUpdateTask verifies behavior for the covered scenario.
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

// TestListAndSortHelpers verifies behavior for the covered scenario.
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

	allWithEmptyQuery, err := svc.SearchTaskMatches(context.Background(), SearchTasksFilter{
		ProjectID: p.ID,
		Query:     " ",
	})
	if err != nil {
		t.Fatalf("SearchTaskMatches(empty) error = %v", err)
	}
	if len(allWithEmptyQuery) != 3 {
		t.Fatalf("expected 3 results for empty query, got %d", len(allWithEmptyQuery))
	}
}

// TestSearchTaskMatchesAcrossProjectsAndStates verifies behavior for the covered scenario.
func TestSearchTaskMatchesAcrossProjectsAndStates(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p1, _ := domain.NewProject("p1", "Inbox", "", now)
	p2, _ := domain.NewProject("p2", "Client", "", now)
	repo.projects[p1.ID] = p1
	repo.projects[p2.ID] = p2

	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p1.ID, "In Progress", 1, 0, now)
	c3, _ := domain.NewColumn("c3", p2.ID, "In Progress", 0, 0, now)
	repo.columns[c1.ID] = c1
	repo.columns[c2.ID] = c2
	repo.columns[c3.ID] = c3

	t1, _ := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   p1.ID,
		ColumnID:    c1.ID,
		Position:    0,
		Title:       "Roadmap draft",
		Description: "planning",
		Priority:    domain.PriorityMedium,
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:          "t2",
		ProjectID:   p1.ID,
		ColumnID:    c2.ID,
		Position:    0,
		Title:       "Implement parser",
		Description: "roadmap parser",
		Priority:    domain.PriorityHigh,
	}, now)
	t3, _ := domain.NewTask(domain.TaskInput{
		ID:          "t3",
		ProjectID:   p2.ID,
		ColumnID:    c3.ID,
		Position:    0,
		Title:       "Client sync",
		Description: "roadmap review",
		Priority:    domain.PriorityLow,
	}, now)
	t3.Archive(now.Add(time.Minute))
	repo.tasks[t1.ID] = t1
	repo.tasks[t2.ID] = t2
	repo.tasks[t3.ID] = t3

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})

	matches, err := svc.SearchTaskMatches(context.Background(), SearchTasksFilter{
		CrossProject:    true,
		IncludeArchived: false,
		States:          []string{"progress"},
		Query:           "parser",
	})
	if err != nil {
		t.Fatalf("SearchTaskMatches() error = %v", err)
	}
	if len(matches) != 1 || matches[0].Task.ID != "t2" || matches[0].StateID != "progress" {
		t.Fatalf("unexpected active matches %#v", matches)
	}

	matches, err = svc.SearchTaskMatches(context.Background(), SearchTasksFilter{
		CrossProject:    true,
		IncludeArchived: true,
		States:          []string{"archived"},
		Query:           "roadmap",
	})
	if err != nil {
		t.Fatalf("SearchTaskMatches(archived) error = %v", err)
	}
	if len(matches) != 1 || matches[0].Task.ID != "t3" || matches[0].StateID != "archived" {
		t.Fatalf("unexpected archived matches %#v", matches)
	}
}

// TestEnsureDefaultProjectAlreadyExists verifies behavior for the covered scenario.
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

// TestCreateProjectWithMetadataAndAutoColumns verifies behavior for the covered scenario.
func TestCreateProjectWithMetadataAndAutoColumns(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	ids := []string{"p1", "c1", "c2"}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time { return now }, ServiceConfig{
		AutoCreateProjectColumns: true,
		StateTemplates: []StateTemplate{
			{ID: "todo", Name: "To Do", Position: 0},
			{ID: "doing", Name: "Doing", Position: 1},
		},
	})

	project, err := svc.CreateProjectWithMetadata(context.Background(), CreateProjectInput{
		Name:        "Roadmap",
		Description: "Q2 plan",
		Metadata: domain.ProjectMetadata{
			Owner: "Evan",
			Tags:  []string{"Roadmap", "roadmap"},
		},
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}
	if project.Metadata.Owner != "Evan" || len(project.Metadata.Tags) != 1 {
		t.Fatalf("unexpected project metadata %#v", project.Metadata)
	}
	columns, err := svc.ListColumns(context.Background(), project.ID, false)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	if len(columns) != 2 {
		t.Fatalf("expected 2 auto-created columns, got %d", len(columns))
	}
	if columns[0].Name != "To Do" || columns[1].Name != "Doing" {
		t.Fatalf("unexpected column names %#v", columns)
	}
}

// TestUpdateProject verifies behavior for the covered scenario.
func TestUpdateProject(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "old desc", now)
	repo.projects[project.ID] = project

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	updated, err := svc.UpdateProject(context.Background(), UpdateProjectInput{
		ProjectID:   project.ID,
		Name:        "Platform",
		Description: "new desc",
		Metadata: domain.ProjectMetadata{
			Owner: "team-kan",
			Tags:  []string{"go", "Go"},
		},
	})
	if err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}
	if updated.Name != "Platform" || updated.Description != "new desc" {
		t.Fatalf("unexpected updated project %#v", updated)
	}
	if updated.Metadata.Owner != "team-kan" || len(updated.Metadata.Tags) != 1 || updated.Metadata.Tags[0] != "go" {
		t.Fatalf("unexpected metadata %#v", updated.Metadata)
	}
}

// TestStateTemplateSanitization verifies behavior for the covered scenario.
func TestStateTemplateSanitization(t *testing.T) {
	got := sanitizeStateTemplates([]StateTemplate{
		{ID: "", Name: " To Do ", Position: 3},
		{ID: "todo", Name: "Duplicate", Position: 1},
		{ID: "", Name: "In Progress", Position: 2, WIPLimit: -1},
		{ID: "", Name: " ", Position: 4},
	})
	if len(got) != 2 {
		t.Fatalf("expected 2 sanitized states, got %#v", got)
	}
	if got[0].ID != "progress" || got[1].ID != "todo" {
		t.Fatalf("unexpected sanitized IDs %#v", got)
	}
	if got[0].WIPLimit != 0 {
		t.Fatalf("expected clamped wip limit, got %d", got[0].WIPLimit)
	}
}

// failingRepo represents failing repo data used by this package.
type failingRepo struct {
	*fakeRepo
	err error
}

// ListProjects lists projects.
func (f failingRepo) ListProjects(context.Context, bool) ([]domain.Project, error) {
	return nil, f.err
}

// TestEnsureDefaultProjectErrorPropagation verifies behavior for the covered scenario.
func TestEnsureDefaultProjectErrorPropagation(t *testing.T) {
	expected := errors.New("boom")
	svc := NewService(failingRepo{fakeRepo: newFakeRepo(), err: expected}, nil, time.Now, ServiceConfig{})
	_, err := svc.EnsureDefaultProject(context.Background())
	if !errors.Is(err, expected) {
		t.Fatalf("expected wrapped error %v, got %v", expected, err)
	}
}

// TestMoveTaskBlocksWhenStartCriteriaUnmet verifies behavior for the covered scenario.
func TestMoveTaskBlocksWhenStartCriteriaUnmet(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	repo.projects[project.ID] = project
	todo, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	progress, _ := domain.NewColumn("c2", project.ID, "In Progress", 1, 0, now)
	repo.columns[todo.ID] = todo
	repo.columns[progress.ID] = progress

	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  todo.ID,
		Position:  0,
		Title:     "blocked",
		Priority:  domain.PriorityMedium,
		Metadata: domain.TaskMetadata{
			CompletionContract: domain.CompletionContract{
				StartCriteria: []domain.ChecklistItem{{ID: "s1", Text: "design reviewed", Done: false}},
			},
		},
	}, now)
	repo.tasks[task.ID] = task

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})
	_, err := svc.MoveTask(context.Background(), task.ID, progress.ID, 0)
	if err == nil || !errors.Is(err, domain.ErrTransitionBlocked) {
		t.Fatalf("expected ErrTransitionBlocked, got %v", err)
	}
}

// TestMoveTaskAllowsDoneWhenContractsSatisfied verifies behavior for the covered scenario.
func TestMoveTaskAllowsDoneWhenContractsSatisfied(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	repo.projects[project.ID] = project
	progress, _ := domain.NewColumn("c2", project.ID, "In Progress", 1, 0, now)
	done, _ := domain.NewColumn("c3", project.ID, "Done", 2, 0, now)
	repo.columns[progress.ID] = progress
	repo.columns[done.ID] = done

	parent, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-parent",
		ProjectID:      project.ID,
		ColumnID:       progress.ID,
		Position:       0,
		Title:          "parent",
		Priority:       domain.PriorityHigh,
		LifecycleState: domain.StateProgress,
		Metadata: domain.TaskMetadata{
			CompletionContract: domain.CompletionContract{
				CompletionCriteria: []domain.ChecklistItem{{ID: "c1", Text: "tests green", Done: true}},
				CompletionChecklist: []domain.ChecklistItem{
					{ID: "k1", Text: "docs updated", Done: true},
				},
				Policy: domain.CompletionPolicy{RequireChildrenDone: true},
			},
		},
	}, now)
	child, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-child",
		ProjectID:      project.ID,
		ParentID:       parent.ID,
		ColumnID:       done.ID,
		Position:       0,
		Title:          "child",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateDone,
	}, now)
	repo.tasks[parent.ID] = parent
	repo.tasks[child.ID] = child

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	moved, err := svc.MoveTask(context.Background(), parent.ID, done.ID, 0)
	if err != nil {
		t.Fatalf("MoveTask() error = %v", err)
	}
	if moved.LifecycleState != domain.StateDone {
		t.Fatalf("expected done lifecycle state, got %q", moved.LifecycleState)
	}
	if moved.CompletedAt == nil {
		t.Fatal("expected completed_at to be set")
	}
}

// TestMoveTaskBlocksDoneWhenAnySubtaskIncomplete verifies behavior for the covered scenario.
func TestMoveTaskBlocksDoneWhenAnySubtaskIncomplete(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	repo.projects[project.ID] = project
	progress, _ := domain.NewColumn("c2", project.ID, "In Progress", 1, 0, now)
	done, _ := domain.NewColumn("c3", project.ID, "Done", 2, 0, now)
	repo.columns[progress.ID] = progress
	repo.columns[done.ID] = done

	parent, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-parent",
		ProjectID:      project.ID,
		ColumnID:       progress.ID,
		Position:       0,
		Title:          "parent",
		Priority:       domain.PriorityHigh,
		LifecycleState: domain.StateProgress,
	}, now)
	child, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-child",
		ProjectID:      project.ID,
		ParentID:       parent.ID,
		ColumnID:       progress.ID,
		Position:       1,
		Title:          "child",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateProgress,
	}, now)
	repo.tasks[parent.ID] = parent
	repo.tasks[child.ID] = child

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	_, err := svc.MoveTask(context.Background(), parent.ID, done.ID, 0)
	if err == nil || !errors.Is(err, domain.ErrTransitionBlocked) {
		t.Fatalf("expected ErrTransitionBlocked, got %v", err)
	}
	if !strings.Contains(err.Error(), "subtasks must be done") {
		t.Fatalf("expected incomplete subtask reason, got %v", err)
	}
}

// TestReparentTaskAndListChildTasks verifies behavior for the covered scenario.
func TestReparentTaskAndListChildTasks(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	repo.projects[project.ID] = project
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	repo.columns[column.ID] = column
	parent, _ := domain.NewTask(domain.TaskInput{
		ID:        "parent",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "parent",
		Priority:  domain.PriorityMedium,
	}, now)
	child, _ := domain.NewTask(domain.TaskInput{
		ID:        "child",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  1,
		Title:     "child",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[parent.ID] = parent
	repo.tasks[child.ID] = child

	svc := NewService(repo, nil, func() time.Time { return now.Add(2 * time.Minute) }, ServiceConfig{})
	updated, err := svc.ReparentTask(context.Background(), child.ID, parent.ID)
	if err != nil {
		t.Fatalf("ReparentTask() error = %v", err)
	}
	if updated.ParentID != parent.ID {
		t.Fatalf("expected parent id %q, got %q", parent.ID, updated.ParentID)
	}
	children, err := svc.ListChildTasks(context.Background(), project.ID, parent.ID, false)
	if err != nil {
		t.Fatalf("ListChildTasks() error = %v", err)
	}
	if len(children) != 1 || children[0].ID != child.ID {
		t.Fatalf("unexpected child list %#v", children)
	}
}

// TestGetProjectDependencyRollup verifies behavior for the covered scenario.
func TestGetProjectDependencyRollup(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	repo.projects[project.ID] = project
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	repo.columns[column.ID] = column

	readyDep, _ := domain.NewTask(domain.TaskInput{
		ID:             "dep-ready",
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Position:       0,
		Title:          "ready dep",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateDone,
	}, now)
	openDep, _ := domain.NewTask(domain.TaskInput{
		ID:             "dep-open",
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Position:       1,
		Title:          "open dep",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateProgress,
	}, now)
	blocked, _ := domain.NewTask(domain.TaskInput{
		ID:        "blocked",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  2,
		Title:     "blocked",
		Priority:  domain.PriorityMedium,
		Metadata: domain.TaskMetadata{
			DependsOn:     []string{"dep-ready", "dep-open", "dep-missing"},
			BlockedBy:     []string{"dep-open"},
			BlockedReason: "waiting on review",
		},
	}, now)

	repo.tasks[readyDep.ID] = readyDep
	repo.tasks[openDep.ID] = openDep
	repo.tasks[blocked.ID] = blocked

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})
	rollup, err := svc.GetProjectDependencyRollup(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("GetProjectDependencyRollup() error = %v", err)
	}
	if rollup.TotalItems != 3 {
		t.Fatalf("expected 3 total items, got %d", rollup.TotalItems)
	}
	if rollup.ItemsWithDependencies != 1 || rollup.DependencyEdges != 3 {
		t.Fatalf("unexpected dependency counts %#v", rollup)
	}
	if rollup.BlockedItems != 1 || rollup.BlockedByEdges != 1 {
		t.Fatalf("unexpected blocked counts %#v", rollup)
	}
	if rollup.UnresolvedDependencyEdges != 2 {
		t.Fatalf("expected 2 unresolved dependencies, got %d", rollup.UnresolvedDependencyEdges)
	}
}

// TestListProjectChangeEvents verifies behavior for the covered scenario.
func TestListProjectChangeEvents(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	repo.projects[project.ID] = project
	repo.changeEvents[project.ID] = []domain.ChangeEvent{
		{ID: 3, ProjectID: project.ID, WorkItemID: "t1", Operation: domain.ChangeOperationUpdate},
		{ID: 2, ProjectID: project.ID, WorkItemID: "t1", Operation: domain.ChangeOperationCreate},
	}

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})
	events, err := svc.ListProjectChangeEvents(context.Background(), project.ID, 1)
	if err != nil {
		t.Fatalf("ListProjectChangeEvents() error = %v", err)
	}
	if len(events) != 1 || events[0].Operation != domain.ChangeOperationUpdate {
		t.Fatalf("unexpected events %#v", events)
	}
}
