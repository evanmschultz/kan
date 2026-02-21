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

type fakeService struct {
	projects []domain.Project
	columns  map[string][]domain.Column
	tasks    map[string][]domain.Task
	err      error
}

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

func (f *fakeService) ListProjects(context.Context, bool) ([]domain.Project, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]domain.Project, len(f.projects))
	copy(out, f.projects)
	return out, nil
}

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
	m = applyMsg(t, m, tea.MouseClickMsg{X: 2, Y: 6, Button: tea.MouseLeft})
	if m.projectPickerIndex != 1 {
		t.Fatalf("expected click to target second project, got %d", m.projectPickerIndex)
	}
}

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

func loadReadyModel(t *testing.T, m Model) Model {
	t.Helper()
	return applyMsg(t, applyCmd(t, m, m.Init()), tea.WindowSizeMsg{Width: 120, Height: 40})
}

func applyMsg(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	updated, cmd := m.Update(msg)
	out, ok := updated.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", updated)
	}
	return applyCmd(t, out, cmd)
}

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

func keyRune(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}
