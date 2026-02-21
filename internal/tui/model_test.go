package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/evanschultz/kan/internal/app"
	"github.com/evanschultz/kan/internal/domain"
)

// fakeService represents fake service data used by this package.
type fakeService struct {
	projects []domain.Project
	columns  map[string][]domain.Column
	tasks    map[string][]domain.Task
	err      error
}

// newFakeService constructs fake service.
func newFakeService(projects []domain.Project, columns []domain.Column, tasks []domain.Task) *fakeService {
	colByProject := map[string][]domain.Column{}
	for _, c := range columns {
		colByProject[c.ProjectID] = append(colByProject[c.ProjectID], c)
	}
	taskByProject := map[string][]domain.Task{}
	for _, t := range tasks {
		taskByProject[t.ProjectID] = append(taskByProject[t.ProjectID], t)
	}
	return &fakeService{
		projects: projects,
		columns:  colByProject,
		tasks:    taskByProject,
	}
}

// ListProjects lists projects.
func (f *fakeService) ListProjects(context.Context, bool) ([]domain.Project, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]domain.Project, len(f.projects))
	copy(out, f.projects)
	return out, nil
}

// ListColumns lists columns.
func (f *fakeService) ListColumns(_ context.Context, projectID string, includeArchived bool) ([]domain.Column, error) {
	if f.err != nil {
		return nil, f.err
	}
	cols := f.columns[projectID]
	out := make([]domain.Column, 0, len(cols))
	for _, c := range cols {
		if !includeArchived && c.ArchivedAt != nil {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

// ListTasks lists tasks.
func (f *fakeService) ListTasks(_ context.Context, projectID string, includeArchived bool) ([]domain.Task, error) {
	if f.err != nil {
		return nil, f.err
	}
	tasks := f.tasks[projectID]
	out := make([]domain.Task, 0, len(tasks))
	for _, task := range tasks {
		if !includeArchived && task.ArchivedAt != nil {
			continue
		}
		out = append(out, task)
	}
	return out, nil
}

// SearchTasks handles search tasks.
func (f *fakeService) SearchTasks(ctx context.Context, projectID, query string, includeArchived bool) ([]domain.Task, error) {
	tasks, err := f.ListTasks(ctx, projectID, includeArchived)
	if err != nil {
		return nil, err
	}
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return tasks, nil
	}
	out := make([]domain.Task, 0)
	for _, task := range tasks {
		if strings.Contains(strings.ToLower(task.Title), query) || strings.Contains(strings.ToLower(task.Description), query) {
			out = append(out, task)
			continue
		}
		for _, label := range task.Labels {
			if strings.Contains(strings.ToLower(label), query) {
				out = append(out, task)
				break
			}
		}
	}
	return out, nil
}

// SearchTaskMatches handles search task matches.
func (f *fakeService) SearchTaskMatches(ctx context.Context, in app.SearchTasksFilter) ([]app.TaskMatch, error) {
	query := strings.ToLower(strings.TrimSpace(in.Query))
	stateSet := map[string]struct{}{}
	for _, state := range in.States {
		state = strings.ToLower(strings.TrimSpace(state))
		if state == "" {
			continue
		}
		stateSet[state] = struct{}{}
	}
	allowAllStates := len(stateSet) == 0
	out := make([]app.TaskMatch, 0)

	projectIDs := make([]string, 0)
	if in.CrossProject {
		for _, p := range f.projects {
			if !in.IncludeArchived && p.ArchivedAt != nil {
				continue
			}
			projectIDs = append(projectIDs, p.ID)
		}
	} else {
		projectIDs = append(projectIDs, in.ProjectID)
	}

	for _, projectID := range projectIDs {
		project, ok := f.projectByID(projectID)
		if !ok {
			continue
		}
		for _, task := range f.tasks[projectID] {
			stateID := "todo"
			columnName := ""
			for _, c := range f.columns[projectID] {
				if c.ID == task.ColumnID {
					columnName = strings.ToLower(strings.ReplaceAll(c.Name, " ", "-"))
					break
				}
			}
			if columnName != "" {
				switch columnName {
				case "to-do", "todo":
					stateID = "todo"
				case "in-progress", "progress", "doing":
					stateID = "progress"
				default:
					stateID = columnName
				}
			}
			if task.ArchivedAt != nil {
				if !in.IncludeArchived {
					continue
				}
				stateID = "archived"
			}
			if !allowAllStates {
				if _, ok := stateSet[stateID]; !ok {
					continue
				}
			}
			if query != "" {
				matched := strings.Contains(strings.ToLower(task.Title), query) || strings.Contains(strings.ToLower(task.Description), query)
				if !matched {
					for _, label := range task.Labels {
						if strings.Contains(strings.ToLower(label), query) {
							matched = true
							break
						}
					}
				}
				if !matched {
					continue
				}
			}
			out = append(out, app.TaskMatch{
				Project: project,
				Task:    task,
				StateID: stateID,
			})
		}
	}
	return out, nil
}

// CreateProjectWithMetadata creates project with metadata.
func (f *fakeService) CreateProjectWithMetadata(_ context.Context, in app.CreateProjectInput) (domain.Project, error) {
	project, err := domain.NewProject("p-new", in.Name, in.Description, time.Now().UTC())
	if err != nil {
		return domain.Project{}, err
	}
	if err := project.UpdateDetails(project.Name, project.Description, in.Metadata, time.Now().UTC()); err != nil {
		return domain.Project{}, err
	}
	f.projects = append(f.projects, project)
	if _, ok := f.columns[project.ID]; !ok {
		now := time.Now().UTC()
		c1, _ := domain.NewColumn("c-new-1", project.ID, "To Do", 0, 0, now)
		c2, _ := domain.NewColumn("c-new-2", project.ID, "In Progress", 1, 0, now)
		c3, _ := domain.NewColumn("c-new-3", project.ID, "Done", 2, 0, now)
		f.columns[project.ID] = []domain.Column{c1, c2, c3}
	}
	if _, ok := f.tasks[project.ID]; !ok {
		f.tasks[project.ID] = []domain.Task{}
	}
	return project, nil
}

// UpdateProject updates state for the requested operation.
func (f *fakeService) UpdateProject(_ context.Context, in app.UpdateProjectInput) (domain.Project, error) {
	for idx := range f.projects {
		if f.projects[idx].ID != in.ProjectID {
			continue
		}
		if err := f.projects[idx].UpdateDetails(in.Name, in.Description, in.Metadata, time.Now().UTC()); err != nil {
			return domain.Project{}, err
		}
		return f.projects[idx], nil
	}
	return domain.Project{}, app.ErrNotFound
}

// CreateTask creates task.
func (f *fakeService) CreateTask(_ context.Context, in app.CreateTaskInput) (domain.Task, error) {
	pos := 0
	for _, t := range f.tasks[in.ProjectID] {
		if t.ColumnID == in.ColumnID && t.Position >= pos {
			pos = t.Position + 1
		}
	}
	task, err := domain.NewTask(domain.TaskInput{
		ID:          "t-new",
		ProjectID:   in.ProjectID,
		ColumnID:    in.ColumnID,
		Position:    pos,
		Title:       in.Title,
		Description: in.Description,
		Priority:    in.Priority,
		DueAt:       in.DueAt,
		Labels:      in.Labels,
	}, time.Now().UTC())
	if err != nil {
		return domain.Task{}, err
	}
	f.tasks[in.ProjectID] = append(f.tasks[in.ProjectID], task)
	return task, nil
}

// UpdateTask updates state for the requested operation.
func (f *fakeService) UpdateTask(_ context.Context, in app.UpdateTaskInput) (domain.Task, error) {
	for projectID := range f.tasks {
		for idx := range f.tasks[projectID] {
			if f.tasks[projectID][idx].ID != in.TaskID {
				continue
			}
			f.tasks[projectID][idx].Title = strings.TrimSpace(in.Title)
			f.tasks[projectID][idx].Description = strings.TrimSpace(in.Description)
			f.tasks[projectID][idx].Priority = in.Priority
			f.tasks[projectID][idx].DueAt = in.DueAt
			f.tasks[projectID][idx].Labels = in.Labels
			return f.tasks[projectID][idx], nil
		}
	}
	return domain.Task{}, app.ErrNotFound
}

// MoveTask moves task.
func (f *fakeService) MoveTask(_ context.Context, taskID, toColumnID string, position int) (domain.Task, error) {
	for projectID := range f.tasks {
		for idx := range f.tasks[projectID] {
			if f.tasks[projectID][idx].ID == taskID {
				f.tasks[projectID][idx].ColumnID = toColumnID
				f.tasks[projectID][idx].Position = position
				return f.tasks[projectID][idx], nil
			}
		}
	}
	return domain.Task{}, app.ErrNotFound
}

// DeleteTask deletes task.
func (f *fakeService) DeleteTask(_ context.Context, taskID string, mode app.DeleteMode) error {
	for projectID := range f.tasks {
		for idx := range f.tasks[projectID] {
			task := f.tasks[projectID][idx]
			if task.ID != taskID {
				continue
			}
			switch mode {
			case app.DeleteModeArchive:
				now := time.Now().UTC()
				f.tasks[projectID][idx].ArchivedAt = &now
				return nil
			case app.DeleteModeHard:
				f.tasks[projectID] = append(f.tasks[projectID][:idx], f.tasks[projectID][idx+1:]...)
				return nil
			default:
				return app.ErrInvalidDeleteMode
			}
		}
	}
	return app.ErrNotFound
}

// RestoreTask restores task.
func (f *fakeService) RestoreTask(_ context.Context, taskID string) (domain.Task, error) {
	for projectID := range f.tasks {
		for idx := range f.tasks[projectID] {
			if f.tasks[projectID][idx].ID == taskID {
				f.tasks[projectID][idx].ArchivedAt = nil
				return f.tasks[projectID][idx], nil
			}
		}
	}
	return domain.Task{}, app.ErrNotFound
}

// RenameTask renames task.
func (f *fakeService) RenameTask(_ context.Context, taskID, title string) (domain.Task, error) {
	for projectID := range f.tasks {
		for idx := range f.tasks[projectID] {
			if f.tasks[projectID][idx].ID == taskID {
				f.tasks[projectID][idx].Title = strings.TrimSpace(title)
				return f.tasks[projectID][idx], nil
			}
		}
	}
	return domain.Task{}, app.ErrNotFound
}

// projectByID returns project by id.
func (f *fakeService) projectByID(projectID string) (domain.Project, bool) {
	for _, project := range f.projects {
		if project.ID == projectID {
			return project, true
		}
	}
	return domain.Project{}, false
}

// TestModelLoadAndNavigation verifies behavior for the covered scenario.
func TestModelLoadAndNavigation(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p.ID, "Done", 1, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Ship",
		Priority:  domain.PriorityMedium,
	}, now)

	svc := newFakeService([]domain.Project{p}, []domain.Column{c1, c2}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc))

	if len(m.projects) != 1 || len(m.columns) != 2 || len(m.tasks) != 1 {
		t.Fatalf("unexpected loaded model: %#v", m)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyRight})
	if m.selectedColumn != 1 {
		t.Fatalf("expected selectedColumn=1, got %d", m.selectedColumn)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyLeft})
	if m.selectedColumn != 0 {
		t.Fatalf("expected selectedColumn=0, got %d", m.selectedColumn)
	}
}

// TestModelQuickAddMoveArchiveRestoreDelete verifies behavior for the covered scenario.
func TestModelQuickAddMoveArchiveRestoreDelete(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p.ID, "Done", 1, 0, now)
	existing, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Existing",
		Priority:  domain.PriorityLow,
	}, now)

	svc := newFakeService([]domain.Project{p}, []domain.Column{c1, c2}, []domain.Task{existing})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune('n'))
	m = applyMsg(t, m, keyRune('N'))
	m = applyMsg(t, m, keyRune('e'))
	m = applyMsg(t, m, keyRune('w'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(svc.tasks[p.ID]) != 2 {
		t.Fatalf("expected 2 tasks after quick add, got %d", len(svc.tasks[p.ID]))
	}

	m = applyMsg(t, m, keyRune(']'))
	if svc.tasks[p.ID][0].ColumnID != c2.ID {
		// existing task is selected first and should move.
		t.Fatalf("expected selected task to move to column %q", c2.ID)
	}

	m = applyMsg(t, m, keyRune('d'))
	if svc.tasks[p.ID][0].ArchivedAt == nil {
		t.Fatal("expected selected task archived")
	}
	m = applyMsg(t, m, keyRune('u'))
	if svc.tasks[p.ID][0].ArchivedAt != nil {
		t.Fatal("expected selected task restored")
	}

	m = applyMsg(t, m, keyRune('D'))
	if len(svc.tasks[p.ID]) != 1 {
		t.Fatalf("expected hard delete to remove task, got %d tasks", len(svc.tasks[p.ID]))
	}
}

// TestModelProjectSwitchAndSearch verifies behavior for the covered scenario.
func TestModelProjectSwitchAndSearch(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p1, _ := domain.NewProject("p1", "A", "", now)
	p2, _ := domain.NewProject("p2", "B", "", now)
	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p2.ID, "To Do", 0, 0, now)
	t1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p1.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Alpha task",
		Priority:  domain.PriorityLow,
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p2.ID,
		ColumnID:  c2.ID,
		Position:  0,
		Title:     "Beta task",
		Priority:  domain.PriorityLow,
	}, now)

	svc := newFakeService([]domain.Project{p1, p2}, []domain.Column{c1, c2}, []domain.Task{t1, t2})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune('p'))
	if m.mode != modeProjectPicker {
		t.Fatalf("expected project picker mode, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.selectedProject != 1 {
		t.Fatalf("expected selectedProject=1 after picker choose, got %d", m.selectedProject)
	}

	m = applyMsg(t, m, keyRune('/'))
	m = applyMsg(t, m, keyRune('B'))
	m = applyMsg(t, m, keyRune('e'))
	m = applyMsg(t, m, keyRune('t'))
	m = applyMsg(t, m, keyRune('a'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(m.tasks) != 1 || !strings.Contains(m.tasks[0].Title, "Beta") {
		t.Fatalf("expected filtered tasks to include only beta, got %#v", m.tasks)
	}
}

// TestModelCrossProjectSearchResultsAndJump verifies behavior for the covered scenario.
func TestModelCrossProjectSearchResultsAndJump(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p1, _ := domain.NewProject("p1", "Inbox", "", now)
	p2, _ := domain.NewProject("p2", "Client", "", now)
	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p2.ID, "To Do", 0, 0, now)
	t1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p1.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Local task",
		Priority:  domain.PriorityLow,
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p2.ID,
		ColumnID:  c2.ID,
		Position:  0,
		Title:     "Client roadmap",
		Priority:  domain.PriorityLow,
	}, now)

	svc := newFakeService([]domain.Project{p1, p2}, []domain.Column{c1, c2}, []domain.Task{t1, t2})
	m := loadReadyModel(t, NewModel(svc))
	m = applyMsg(t, m, keyRune('/'))
	m.searchCrossProject = true
	m = applyMsg(t, m, keyRune('c'))
	m = applyMsg(t, m, keyRune('l'))
	m = applyMsg(t, m, keyRune('i'))
	m = applyMsg(t, m, keyRune('e'))
	m = applyMsg(t, m, keyRune('n'))
	m = applyMsg(t, m, keyRune('t'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeSearchResults {
		t.Fatalf("expected search results mode, got %v", m.mode)
	}
	if len(m.searchMatches) == 0 || m.searchMatches[0].Task.ID != "t2" {
		t.Fatalf("expected cross-project match for t2, got %#v", m.searchMatches)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.selectedProject != 1 {
		t.Fatalf("expected jump to second project, got %d", m.selectedProject)
	}
	if task, ok := m.selectedTaskInCurrentColumn(); !ok || task.ID != "t2" {
		t.Fatalf("expected selected task t2 after jump, got %#v ok=%t", task, ok)
	}
}

// TestModelAddAndEditProject verifies behavior for the covered scenario.
func TestModelAddAndEditProject(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, nil)
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune('N'))
	if m.mode != modeAddProject {
		t.Fatalf("expected add project mode, got %v", m.mode)
	}
	m = applyMsg(t, m, keyRune('R'))
	m = applyMsg(t, m, keyRune('o'))
	m = applyMsg(t, m, keyRune('a'))
	m = applyMsg(t, m, keyRune('d'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(m.projects) < 2 {
		t.Fatalf("expected project created, got %#v", m.projects)
	}
	if m.selectedProject != len(m.projects)-1 {
		t.Fatalf("expected selection on new project, got %d", m.selectedProject)
	}

	m = applyMsg(t, m, keyRune('M'))
	if m.mode != modeEditProject {
		t.Fatalf("expected edit project mode, got %v", m.mode)
	}
	m.projectFormInputs[0].SetValue("Renamed")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if got := m.projects[m.selectedProject].Name; got != "Renamed" {
		t.Fatalf("expected project renamed, got %q", got)
	}
}

// TestModelCommandPaletteAndQuickActions verifies behavior for the covered scenario.
func TestModelCommandPaletteAndQuickActions(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune(':'))
	if m.mode != modeCommandPalette {
		t.Fatalf("expected command palette mode, got %v", m.mode)
	}
	m = applyMsg(t, m, keyRune('s'))
	m = applyMsg(t, m, keyRune('e'))
	m = applyMsg(t, m, keyRune('a'))
	m = applyMsg(t, m, keyRune('r'))
	m = applyMsg(t, m, keyRune('c'))
	m = applyMsg(t, m, keyRune('h'))
	m = applyMsg(t, m, keyRune('-'))
	m = applyMsg(t, m, keyRune('a'))
	m = applyMsg(t, m, keyRune('l'))
	m = applyMsg(t, m, keyRune('l'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !m.searchCrossProject {
		t.Fatal("expected search-all command to enable cross-project scope")
	}

	m = applyMsg(t, m, keyRune('.'))
	if m.mode != modeQuickActions {
		t.Fatalf("expected quick actions mode, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeTaskInfo {
		t.Fatalf("expected quick action enter to open task info, got %v", m.mode)
	}

	m.mode = modeNone
	m = applyMsg(t, m, keyRune(':'))
	m = applyMsg(t, m, keyRune('x'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !strings.Contains(m.status, "unknown command") {
		t.Fatalf("expected unknown command status, got %q", m.status)
	}
}

// TestModelMouseWheelAndClick verifies behavior for the covered scenario.
func TestModelMouseWheelAndClick(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p.ID, "Done", 1, 0, now)
	t1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "One",
		Priority:  domain.PriorityLow,
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  1,
		Title:     "Two",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c1, c2}, []domain.Task{t1, t2})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	if m.selectedTask != 1 {
		t.Fatalf("expected selectedTask=1 after wheel down, got %d", m.selectedTask)
	}

	clickX := m.columnWidth() + 5
	clickY := m.boardTop() + 2
	m = applyMsg(t, m, tea.MouseClickMsg{X: clickX, Y: clickY, Button: tea.MouseLeft})
	if m.selectedColumn != 1 {
		t.Fatalf("expected selectedColumn=1 after mouse click, got %d", m.selectedColumn)
	}
}

// TestModelQuitKey verifies behavior for the covered scenario.
func TestModelQuitKey(t *testing.T) {
	m := NewModel(newFakeService(nil, nil, nil))
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if updated == nil {
		t.Fatal("expected model return value")
	}
	if cmd == nil {
		t.Fatal("expected quit cmd")
	}
}

// TestModelViewStatesAndPrompts verifies behavior for the covered scenario.
func TestModelViewStatesAndPrompts(t *testing.T) {
	m := NewModel(newFakeService(nil, nil, nil))
	v := m.View()
	if v.Content == nil || v.MouseMode != tea.MouseModeCellMotion {
		t.Fatal("expected loading view with mouse enabled")
	}

	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, nil)
	m = loadReadyModel(t, NewModel(svc))
	m.mode = modeAddTask
	m.input = "abc"
	if !strings.Contains(m.modePrompt(), "new task title") {
		t.Fatal("expected add mode prompt")
	}

	m.err = context.DeadlineExceeded
	v = m.View()
	if v.Content == nil {
		t.Fatal("expected error view content")
	}
}

// TestModelInputModePaths verifies behavior for the covered scenario.
func TestModelInputModePaths(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune('n'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	if m.mode != modeAddTask {
		t.Fatalf("expected add mode, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeNone {
		t.Fatalf("expected modeNone after escape, got %v", m.mode)
	}

	m = applyMsg(t, m, keyRune('n'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter}) // empty submit
	if !strings.Contains(m.status, "title required") {
		t.Fatalf("expected title required status, got %q", m.status)
	}

	m = applyMsg(t, m, keyRune('/'))
	m = applyMsg(t, m, keyRune('T'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.searchQuery != "T" {
		t.Fatalf("expected search query set, got %q", m.searchQuery)
	}

	m = applyMsg(t, m, keyRune('e'))
	m.input = "Task 2 | expanded details | high | 2026-03-01 | alpha,beta"
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !strings.Contains(svc.tasks[p.ID][0].Title, "Task 2") {
		t.Fatalf("expected edited title, got %q", svc.tasks[p.ID][0].Title)
	}
	if svc.tasks[p.ID][0].Priority != domain.PriorityHigh || len(svc.tasks[p.ID][0].Labels) != 2 {
		t.Fatalf("expected full-field update, got %#v", svc.tasks[p.ID][0])
	}
}

// TestModelNormalModeExtraBranches verifies behavior for the covered scenario.
func TestModelNormalModeExtraBranches(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p1, _ := domain.NewProject("p1", "A", "", now)
	p2, _ := domain.NewProject("p2", "B", "", now)
	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p2.ID, "To Do", 0, 0, now)
	t1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p1.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Alpha",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p1, p2}, []domain.Column{c1, c2}, []domain.Task{t1})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune('t'))
	if !m.showArchived {
		t.Fatal("expected showArchived enabled")
	}
	m = applyMsg(t, m, keyRune('t'))
	if m.showArchived {
		t.Fatal("expected showArchived disabled")
	}

	m = applyMsg(t, m, keyRune('u'))
	if !strings.Contains(m.status, "nothing to restore") {
		t.Fatalf("expected restore status, got %q", m.status)
	}

	m.searchQuery = "x"
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.searchQuery != "" {
		t.Fatalf("expected search cleared, got %q", m.searchQuery)
	}

	m = applyMsg(t, m, keyRune('P'))
	if m.selectedProject != 0 {
		t.Fatalf("expected selection unchanged in picker-open path, got %d", m.selectedProject)
	}
	if m.mode != modeProjectPicker {
		t.Fatalf("expected project picker mode, got %v", m.mode)
	}

	// out of range move left should no-op
	m.mode = modeNone
	m.selectedColumn = 0
	m = applyMsg(t, m, keyRune('['))
	if m.selectedColumn != 0 {
		t.Fatalf("expected no-op move left, got %d", m.selectedColumn)
	}
}

// TestHelpersCoverage verifies behavior for the covered scenario.
func TestHelpersCoverage(t *testing.T) {
	if clamp(5, 0, 1) != 1 {
		t.Fatal("clamp upper bound failed")
	}
	if clamp(-1, 0, 1) != 0 {
		t.Fatal("clamp lower bound failed")
	}
	if clamp(0, 2, 1) != 2 {
		t.Fatal("clamp invalid range failed")
	}
	if truncate("abc", 0) != "" {
		t.Fatal("truncate max 0 failed")
	}
	if truncate("abc", 1) != "a" {
		t.Fatal("truncate max 1 failed")
	}
	if truncate("abcdef", 3) != "ab…" {
		t.Fatal("truncate ellipsis failed")
	}
	if summarizeLabels([]string{"a", "b", "c"}, 2) != "#a,#b+1" {
		t.Fatalf("unexpected label summary %q", summarizeLabels([]string{"a", "b", "c"}, 2))
	}

	m := Model{}
	if m.modeLabel() != "normal" {
		t.Fatalf("mode label mismatch: %q", m.modeLabel())
	}
	m.mode = modeAddTask
	if !strings.Contains(m.modePrompt(), "new task title") {
		t.Fatal("expected add mode prompt")
	}
	m.mode = modeSearch
	if !strings.Contains(m.modePrompt(), "search query") {
		t.Fatal("expected search mode prompt")
	}
	m.mode = modeRenameTask
	if !strings.Contains(m.modePrompt(), "rename task") {
		t.Fatal("expected rename mode prompt")
	}
	m.mode = modeEditTask
	if !strings.Contains(m.modePrompt(), "title | description") {
		t.Fatal("expected edit mode prompt")
	}
	m.mode = modeProjectPicker
	if !strings.Contains(m.modePrompt(), "project picker") {
		t.Fatal("expected picker mode prompt")
	}
	m.mode = modeTaskInfo
	if !strings.Contains(m.modePrompt(), "task info") {
		t.Fatal("expected task info mode prompt")
	}
	m.mode = modeAddProject
	if !strings.Contains(m.modePrompt(), "new project") {
		t.Fatal("expected add project mode prompt")
	}
	m.mode = modeEditProject
	if !strings.Contains(m.modePrompt(), "edit project") {
		t.Fatal("expected edit project mode prompt")
	}
	m.mode = modeSearchResults
	if !strings.Contains(m.modePrompt(), "search results") {
		t.Fatal("expected search results mode prompt")
	}
	m.mode = modeCommandPalette
	if !strings.Contains(m.modePrompt(), "command palette") {
		t.Fatal("expected command palette mode prompt")
	}
	m.mode = modeQuickActions
	if !strings.Contains(m.modePrompt(), "quick actions") {
		t.Fatal("expected quick actions mode prompt")
	}

	m.columns = []domain.Column{{ID: "c1"}}
	m.width = 10
	if m.columnWidth() < 18 {
		t.Fatal("expected minimum width")
	}
	m.width = 300
	if m.columnWidth() > 42 {
		t.Fatal("expected maximum width")
	}
}

// TestTaskEditParsing verifies behavior for the covered scenario.
func TestTaskEditParsing(t *testing.T) {
	now := time.Date(2026, 2, 21, 0, 0, 0, 0, time.UTC)
	current, _ := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   "p1",
		ColumnID:    "c1",
		Position:    0,
		Title:       "old",
		Description: "desc",
		Priority:    domain.PriorityMedium,
		DueAt:       &now,
		Labels:      []string{"x"},
	}, now)

	input, err := parseTaskEditInput("new | details | high | 2026-03-01 | a,b", current)
	if err != nil {
		t.Fatalf("parseTaskEditInput() error = %v", err)
	}
	if input.Title != "new" || input.Priority != domain.PriorityHigh {
		t.Fatalf("unexpected parsed input %#v", input)
	}
	if input.DueAt == nil || input.DueAt.Format("2006-01-02") != "2026-03-01" {
		t.Fatalf("unexpected parsed due date %#v", input.DueAt)
	}

	_, err = parseTaskEditInput("x | y | urgent | - | -", current)
	if err == nil {
		t.Fatal("expected invalid priority error")
	}
	_, err = parseTaskEditInput("x | y | low | 03/01/2026 | -", current)
	if err == nil {
		t.Fatal("expected invalid date error")
	}

	if !strings.Contains(formatTaskEditInput(current), "old") {
		t.Fatal("expected formatter to include title")
	}
}

// TestProjectPickerMouseAndWheel verifies behavior for the covered scenario.
func TestProjectPickerMouseAndWheel(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p1, _ := domain.NewProject("p1", "A", "", now)
	p2, _ := domain.NewProject("p2", "B", "", now)
	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p2.ID, "To Do", 0, 0, now)
	svc := newFakeService([]domain.Project{p1, p2}, []domain.Column{c1, c2}, nil)
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune('p'))
	m = applyMsg(t, m, tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	if m.projectPickerIndex != 1 {
		t.Fatalf("expected wheel to move picker, got %d", m.projectPickerIndex)
	}
	m = applyMsg(t, m, tea.MouseClickMsg{X: 2, Y: 7, Button: tea.MouseLeft})
	if m.projectPickerIndex != 1 {
		t.Fatalf("expected click to target second project, got %d", m.projectPickerIndex)
	}
}

// TestTaskFieldConfigAffectsRendering verifies behavior for the covered scenario.
func TestTaskFieldConfigAffectsRendering(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	due := now.Add(24 * time.Hour)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   p.ID,
		ColumnID:    c.ID,
		Position:    0,
		Title:       "Task",
		Description: "detailed notes",
		Priority:    domain.PriorityHigh,
		DueAt:       &due,
		Labels:      []string{"one", "two", "three"},
	}, now)

	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	mDefault := loadReadyModel(t, NewModel(svc))
	meta := mDefault.cardMeta(task)
	if !strings.Contains(meta, "high") || !strings.Contains(meta, "#one,#three+1") {
		t.Fatalf("expected default card meta with priority and labels, got %q", meta)
	}

	mHidden := loadReadyModel(t, NewModel(svc, WithTaskFieldConfig(TaskFieldConfig{
		ShowPriority:    false,
		ShowDueDate:     false,
		ShowLabels:      false,
		ShowDescription: false,
	})))
	if mHidden.cardMeta(task) != "" {
		t.Fatalf("expected empty card meta when all card fields hidden, got %q", mHidden.cardMeta(task))
	}
	details := mHidden.renderTaskDetails(lipgloss.Color("212"), lipgloss.Color("245"), lipgloss.Color("241"))
	if strings.Contains(details, "priority:") || strings.Contains(details, "due:") || strings.Contains(details, "labels:") {
		t.Fatalf("expected details metadata hidden, got %q", details)
	}
	if strings.Contains(details, "detailed notes") {
		t.Fatalf("expected description hidden, got %q", details)
	}
}

// TestDeleteUsesConfiguredDefaultMode verifies behavior for the covered scenario.
func TestDeleteUsesConfiguredDefaultMode(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	t1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "One",
		Priority:  domain.PriorityLow,
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  1,
		Title:     "Two",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{t1, t2})

	m := loadReadyModel(t, NewModel(svc, WithDefaultDeleteMode(app.DeleteModeHard)))
	m = applyMsg(t, m, keyRune('d'))
	if len(svc.tasks[p.ID]) != 1 {
		t.Fatalf("expected default delete mode hard to remove selected task, got %d", len(svc.tasks[p.ID]))
	}
}

// TestParseDueAndLabelsInput verifies behavior for the covered scenario.
func TestParseDueAndLabelsInput(t *testing.T) {
	now := time.Date(2026, 2, 21, 0, 0, 0, 0, time.UTC)

	gotDue, err := parseDueInput("", &now)
	if err != nil {
		t.Fatalf("parseDueInput empty unexpected error: %v", err)
	}
	if gotDue == nil || !gotDue.Equal(now) {
		t.Fatalf("expected current due date to be preserved, got %#v", gotDue)
	}

	gotDue, err = parseDueInput("-", &now)
	if err != nil {
		t.Fatalf("parseDueInput dash unexpected error: %v", err)
	}
	if gotDue != nil {
		t.Fatalf("expected due date cleared, got %#v", gotDue)
	}

	gotDue, err = parseDueInput("2026-03-01", nil)
	if err != nil {
		t.Fatalf("parseDueInput valid unexpected error: %v", err)
	}
	if gotDue == nil || gotDue.Format("2006-01-02") != "2026-03-01" {
		t.Fatalf("expected parsed due date, got %#v", gotDue)
	}

	if _, err = parseDueInput("03/01/2026", nil); err == nil {
		t.Fatal("expected parseDueInput invalid format error")
	}

	currentLabels := []string{"one"}
	if got := parseLabelsInput("", currentLabels); len(got) != 1 || got[0] != "one" {
		t.Fatalf("expected current labels preserved, got %#v", got)
	}
	if got := parseLabelsInput("-", currentLabels); got != nil {
		t.Fatalf("expected labels cleared with -, got %#v", got)
	}
	if got := parseLabelsInput("a, b, , c", nil); len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Fatalf("expected parsed labels, got %#v", got)
	}

	if got := parseStateFilters("", []string{"todo"}); len(got) != 1 || got[0] != "todo" {
		t.Fatalf("expected state fallback, got %#v", got)
	}
	if got := parseStateFilters("todo, progress, todo", nil); len(got) != 2 || got[1] != "progress" {
		t.Fatalf("expected deduped state filters, got %#v", got)
	}
}

// TestRenderModeOverlayAndIndexHelpers verifies behavior for the covered scenario.
func TestRenderModeOverlayAndIndexHelpers(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p.ID, "Done", 1, 0, now)
	t1, _ := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   p.ID,
		ColumnID:    c1.ID,
		Position:    0,
		Title:       "First",
		Description: "desc one",
		Priority:    domain.PriorityLow,
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  1,
		Title:     "Second",
		Priority:  domain.PriorityHigh,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c1, c2}, []domain.Task{t1, t2})
	m := loadReadyModel(t, NewModel(svc))

	accent := lipgloss.Color("62")
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")
	helpStyle := lipgloss.NewStyle().Foreground(muted)

	projectPicker := m
	projectPicker.mode = modeProjectPicker
	projectPicker.projectPickerIndex = 0
	if out := projectPicker.renderModeOverlay(accent, muted, dim, helpStyle, 80); !strings.Contains(out, "Projects") {
		t.Fatalf("expected project picker overlay, got %q", out)
	}

	addMode := m
	_ = addMode.startTaskForm(nil)
	if out := addMode.renderModeOverlay(accent, muted, dim, helpStyle, 80); !strings.Contains(out, "New Task") || !strings.Contains(out, "title:") {
		t.Fatalf("expected add-task overlay with fields, got %q", out)
	}
	if out := addMode.renderModeOverlay(accent, muted, dim, helpStyle, 80); strings.Contains(out, "fields: title") {
		t.Fatalf("expected simplified modal hints without repeated fields legend, got %q", out)
	}

	task, ok := m.selectedTaskInCurrentColumn()
	if !ok {
		t.Fatal("expected selected task")
	}
	editMode := m
	_ = editMode.startTaskForm(&task)
	if out := editMode.renderModeOverlay(accent, muted, dim, helpStyle, 80); !strings.Contains(out, "Edit Task") {
		t.Fatalf("expected edit overlay, got %q", out)
	}

	searchMode := m
	_ = searchMode.startSearchMode()
	if out := searchMode.renderModeOverlay(accent, muted, dim, helpStyle, 80); !strings.Contains(out, "Search") {
		t.Fatalf("expected search overlay, got %q", out)
	}
	searchMode.mode = modeSearchResults
	searchMode.searchMatches = []app.TaskMatch{{Project: p, Task: t1, StateID: "todo"}}
	if out := searchMode.renderModeOverlay(accent, muted, dim, helpStyle, 80); !strings.Contains(out, "Search Results") {
		t.Fatalf("expected search-results overlay, got %q", out)
	}

	renameMode := m
	renameMode.mode = modeRenameTask
	renameMode.input = "rename me"
	if out := renameMode.renderModeOverlay(accent, muted, dim, helpStyle, 80); !strings.Contains(out, "Rename Task") {
		t.Fatalf("expected rename overlay, got %q", out)
	}
	infoMode := m
	infoMode.mode = modeTaskInfo
	if out := infoMode.renderModeOverlay(accent, muted, dim, helpStyle, 80); !strings.Contains(out, "Task Info") {
		t.Fatalf("expected task info overlay, got %q", out)
	}

	projectMode := m
	_ = projectMode.startProjectForm(nil)
	if out := projectMode.renderModeOverlay(accent, muted, dim, helpStyle, 80); !strings.Contains(out, "New Project") {
		t.Fatalf("expected project overlay, got %q", out)
	}

	commandMode := m
	_ = commandMode.startCommandPalette()
	if out := commandMode.renderModeOverlay(accent, muted, dim, helpStyle, 80); !strings.Contains(out, "Command Palette") {
		t.Fatalf("expected command palette overlay, got %q", out)
	}

	actionMode := m
	_ = actionMode.startQuickActions()
	if out := actionMode.renderModeOverlay(accent, muted, dim, helpStyle, 80); !strings.Contains(out, "Quick Actions") {
		t.Fatalf("expected quick actions overlay, got %q", out)
	}

	tasks := m.currentColumnTasks()
	if idx := m.taskIndexAtRow(tasks, 0); idx != 0 {
		t.Fatalf("expected row 0 => task 0, got %d", idx)
	}
	if idx := m.taskIndexAtRow(tasks, 3); idx != 1 {
		t.Fatalf("expected row 3 => task 1, got %d", idx)
	}
	if idx := m.taskIndexAtRow(tasks, 99); idx != 1 {
		t.Fatalf("expected large row => last task, got %d", idx)
	}

	panelWithSelection := m.renderOverviewPanel(p, accent, muted, dim)
	if !strings.Contains(panelWithSelection, "Selection") {
		t.Fatalf("expected overview panel selection section, got %q", panelWithSelection)
	}
	noneSelected := m
	noneSelected.selectedColumn = 1
	panelWithoutSelection := noneSelected.renderOverviewPanel(p, accent, muted, dim)
	if !strings.Contains(panelWithoutSelection, "no task selected") {
		t.Fatalf("expected overview panel no-selection hint, got %q", panelWithoutSelection)
	}
}

// TestModelFormValidationPaths verifies behavior for the covered scenario.
func TestModelFormValidationPaths(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	existing, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Existing",
		Priority:  domain.PriorityLow,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{existing})
	m := loadReadyModel(t, NewModel(svc))

	// Add mode: invalid priority branch.
	m = applyMsg(t, m, keyRune('n'))
	m.formInputs[0].SetValue("Draft roadmap")
	m.formInputs[2].SetValue("urgent")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !strings.Contains(m.status, "priority must be low|medium|high") {
		t.Fatalf("expected invalid priority status, got %q", m.status)
	}

	// Add mode: invalid due date branch.
	m.formInputs[2].SetValue("high")
	m.formInputs[3].SetValue("03/01/2026")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !strings.Contains(m.status, "due date must be YYYY-MM-DD or -") {
		t.Fatalf("expected invalid due status, got %q", m.status)
	}

	// Add mode: success path.
	m.formInputs[3].SetValue("2026-03-01")
	m.formInputs[4].SetValue("planning,kan")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(svc.tasks[p.ID]) != 2 {
		t.Fatalf("expected create task success, got %d tasks", len(svc.tasks[p.ID]))
	}

	// Edit mode: invalid priority branch.
	m.selectedTask = 0
	m = applyMsg(t, m, keyRune('e'))
	m.formInputs[2].SetValue("invalid")
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !strings.Contains(m.status, "priority must be low|medium|high") {
		t.Fatalf("expected invalid edit priority status, got %q", m.status)
	}
}

// TestTaskInfoModeAndPriorityPicker verifies behavior for the covered scenario.
func TestTaskInfoModeAndPriorityPicker(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{task})
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeTaskInfo {
		t.Fatalf("expected enter to open task info mode, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeNone {
		t.Fatalf("expected esc to close task info mode, got %v", m.mode)
	}

	m = applyMsg(t, m, keyRune('i'))
	if m.mode != modeTaskInfo {
		t.Fatalf("expected task info mode, got %v", m.mode)
	}
	m = applyMsg(t, m, keyRune('e'))
	if m.mode != modeEditTask {
		t.Fatalf("expected edit mode from task info, got %v", m.mode)
	}

	m.formFocus = 2
	before := m.formInputs[2].Value()
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyRight})
	after := m.formInputs[2].Value()
	if before == after {
		t.Fatalf("expected priority picker value to change, still %q", after)
	}
	changed := m.formInputs[2].Value()
	m = applyMsg(t, m, keyRune('x'))
	if m.formInputs[2].Value() != changed {
		t.Fatalf("expected typing ignored on priority picker, got %q", m.formInputs[2].Value())
	}
}

// TestTaskFormDuePickerFlow verifies behavior for the covered scenario.
func TestTaskFormDuePickerFlow(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, nil)
	m := loadReadyModel(t, NewModel(svc))

	m = applyMsg(t, m, keyRune('n'))
	if m.mode != modeAddTask {
		t.Fatalf("expected add task mode, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.formFocus != 3 {
		t.Fatalf("expected due field focus, got %d", m.formFocus)
	}

	m = applyMsg(t, m, keyRune('D'))
	if m.mode != modeDuePicker {
		t.Fatalf("expected due picker mode, got %v", m.mode)
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeAddTask {
		t.Fatalf("expected return to add task mode, got %v", m.mode)
	}
	if got := strings.TrimSpace(m.formInputs[3].Value()); got != "-" {
		t.Fatalf("expected due field to be '-', got %q", got)
	}

	m = applyMsg(t, m, keyRune('D'))
	m = applyMsg(t, m, keyRune('j'))
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	due := strings.TrimSpace(m.formInputs[3].Value())
	if len(due) != 10 || strings.Count(due, "-") != 2 {
		t.Fatalf("expected YYYY-MM-DD due value, got %q", due)
	}
}

// TestTaskFormLabelSuggestions verifies behavior for the covered scenario.
func TestTaskFormLabelSuggestions(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	t1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  0,
		Title:     "Task 1",
		Priority:  domain.PriorityMedium,
		Labels:    []string{"planning", "kan"},
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p.ID,
		ColumnID:  c.ID,
		Position:  1,
		Title:     "Task 2",
		Priority:  domain.PriorityMedium,
		Labels:    []string{"kan", "roadmap"},
	}, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c}, []domain.Task{t1, t2})
	m := loadReadyModel(t, NewModel(svc))
	m = applyMsg(t, m, keyRune('n'))
	m = applyCmd(t, m, m.focusTaskFormField(4))
	accent := lipgloss.Color("62")
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")
	helpStyle := lipgloss.NewStyle().Foreground(muted)
	out := m.renderModeOverlay(accent, muted, dim, helpStyle, 96)
	if !strings.Contains(out, "suggested labels:") {
		t.Fatalf("expected suggested labels hint, got %q", out)
	}
	if !strings.Contains(out, "kan") {
		t.Fatalf("expected label suggestions to include 'kan', got %q", out)
	}
}

// loadReadyModel loads required data for the current operation.
func loadReadyModel(t *testing.T, m Model) Model {
	t.Helper()
	return applyMsg(t, applyCmd(t, m, m.Init()), tea.WindowSizeMsg{Width: 120, Height: 40})
}

// applyMsg applies msg.
func applyMsg(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	updated, cmd := m.Update(msg)
	out, ok := updated.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", updated)
	}
	return applyCmd(t, out, cmd)
}

// applyCmd applies cmd.
func applyCmd(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	out := m
	currentCmd := cmd
	for i := 0; i < 6 && currentCmd != nil; i++ {
		msg := currentCmd()
		updated, nextCmd := out.Update(msg)
		casted, ok := updated.(Model)
		if !ok {
			t.Fatalf("expected Model, got %T", updated)
		}
		out = casted
		currentCmd = nextCmd
	}
	return out
}

// keyRune handles key rune.
func keyRune(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}
