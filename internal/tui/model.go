package tui

import (
	"context"
	"fmt"
	"image/color"
	"strings"
	"time"
	"unicode/utf8"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/evanschultz/kan/internal/app"
	"github.com/evanschultz/kan/internal/domain"
)

type Service interface {
	ListProjects(context.Context, bool) ([]domain.Project, error)
	ListColumns(context.Context, string, bool) ([]domain.Column, error)
	ListTasks(context.Context, string, bool) ([]domain.Task, error)
	SearchTasks(context.Context, string, string, bool) ([]domain.Task, error)
	CreateTask(context.Context, app.CreateTaskInput) (domain.Task, error)
	UpdateTask(context.Context, app.UpdateTaskInput) (domain.Task, error)
	MoveTask(context.Context, string, string, int) (domain.Task, error)
	DeleteTask(context.Context, string, app.DeleteMode) error
	RestoreTask(context.Context, string) (domain.Task, error)
	RenameTask(context.Context, string, string) (domain.Task, error)
}

type inputMode int

const (
	modeNone inputMode = iota
	modeAddTask
	modeSearch
	modeRenameTask
	modeEditTask
	modeProjectPicker
)

type Model struct {
	svc Service

	ready  bool
	width  int
	height int
	err    error

	status string

	help help.Model
	keys keyMap

	taskFields        TaskFieldConfig
	defaultDeleteMode app.DeleteMode

	projects        []domain.Project
	selectedProject int
	columns         []domain.Column
	tasks           []domain.Task
	selectedColumn  int
	selectedTask    int

	mode         inputMode
	input        string
	searchQuery  string
	showArchived bool

	projectPickerIndex int
	editingTaskID      string

	lastArchivedTaskID string
}

type loadedMsg struct {
	projects        []domain.Project
	selectedProject int
	columns         []domain.Column
	tasks           []domain.Task
	err             error
}

type actionMsg struct {
	err    error
	status string
	reload bool
}

func NewModel(svc Service, opts ...Option) Model {
	h := help.New()
	h.ShowAll = false
	m := Model{
		svc:               svc,
		status:            "loading...",
		help:              h,
		keys:              newKeyMap(),
		taskFields:        DefaultTaskFieldConfig(),
		defaultDeleteMode: app.DeleteModeArchive,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&m)
		}
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return m.loadData
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ready = true
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case loadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.projects = msg.projects
		m.selectedProject = msg.selectedProject
		m.columns = msg.columns
		m.tasks = msg.tasks
		m.clampSelections()
		if m.status == "" || m.status == "loading..." {
			m.status = "ready"
		}
		return m, nil

	case actionMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		if msg.status != "" {
			m.status = msg.status
		}
		if msg.reload {
			return m, m.loadData
		}
		return m, nil

	case tea.KeyPressMsg:
		if m.mode != modeNone {
			return m.handleInputModeKey(msg)
		}
		return m.handleNormalModeKey(msg)

	case tea.MouseWheelMsg:
		return m.handleMouseWheel(msg)

	case tea.MouseClickMsg:
		return m.handleMouseClick(msg)

	default:
		return m, nil
	}
}

func (m Model) View() tea.View {
	if m.err != nil {
		v := tea.NewView("error: " + m.err.Error() + "\n\npress r to retry • q quit\n")
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}
	if !m.ready {
		v := tea.NewView("loading...")
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}
	if len(m.projects) == 0 {
		v := tea.NewView("no projects yet\n\npress q to quit\n")
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	project := m.projects[clamp(m.selectedProject, 0, len(m.projects)-1)]
	accent := lipgloss.Color("212")
	muted := lipgloss.Color("245")
	dim := lipgloss.Color("241")

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	helpStyle := lipgloss.NewStyle().Foreground(muted)
	statusStyle := lipgloss.NewStyle().Foreground(dim)

	header := titleStyle.Render("kan") + "  " + project.Name
	header += statusStyle.Render("  [" + m.modeLabel() + "]")
	if m.searchQuery != "" {
		header += statusStyle.Render("  search: " + m.searchQuery)
	}
	if m.showArchived {
		header += statusStyle.Render("  showing archived")
	}

	columnViews := make([]string, 0, len(m.columns))
	colWidth := m.columnWidth()
	baseColStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(colWidth)
	selColStyle := baseColStyle.Copy().BorderForeground(accent)
	normColStyle := baseColStyle.Copy().BorderForeground(dim)
	colTitle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	archivedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	selectedTaskStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("230"))

	for colIdx, column := range m.columns {
		lines := []string{colTitle.Render(column.Name)}
		colTasks := m.tasksForColumn(column.ID)
		if len(colTasks) == 0 {
			lines = append(lines, archivedStyle.Render("(empty)"))
		} else {
			for taskIdx, task := range colTasks {
				cursor := "  "
				if colIdx == m.selectedColumn && taskIdx == m.selectedTask {
					cursor = "> "
				}
				meta := m.cardMeta(task)
				titleMax := colWidth - 6
				if meta != "" {
					titleMax -= utf8.RuneCountInString(meta) + 1
				}
				line := cursor + truncate(task.Title, titleMax)
				if meta != "" {
					line += " " + meta
				}
				if task.ArchivedAt != nil {
					line += " (archived)"
					line = archivedStyle.Render(line)
				} else if colIdx == m.selectedColumn && taskIdx == m.selectedTask {
					line = selectedTaskStyle.Render(line)
				}
				lines = append(lines, line)
			}
		}

		content := strings.Join(lines, "\n")
		if colIdx == m.selectedColumn {
			columnViews = append(columnViews, selColStyle.Render(content))
		} else {
			columnViews = append(columnViews, normColStyle.Render(content))
		}
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top, columnViews...)
	content := header
	if m.mode == modeProjectPicker {
		if overlay := m.renderModeOverlay(accent, muted, dim, helpStyle); overlay != "" {
			content += "\n\n" + overlay
		}
	}
	content += "\n\n" + body
	if details := m.renderTaskDetails(accent, muted, dim); details != "" {
		content += "\n\n" + details
	}
	content += "\n" + statusStyle.Render(m.status)
	if m.mode != modeProjectPicker {
		if overlay := m.renderModeOverlay(accent, muted, dim, helpStyle); overlay != "" {
			content += "\n\n" + overlay
		}
	}

	helpBubble := m.help
	helpBubble.SetWidth(max(0, m.width-2))
	helpLine := lipgloss.NewStyle().
		Foreground(muted).
		BorderTop(true).
		BorderForeground(dim).
		Padding(0, 1).
		Width(max(0, m.width)).
		Render(helpBubble.View(m.keys))

	if m.height > 0 {
		contentHeight := lipgloss.Height(content)
		helpHeight := lipgloss.Height(helpLine)
		if contentHeight+helpHeight < m.height {
			content += strings.Repeat("\n", m.height-contentHeight-helpHeight)
		}
	}

	view := tea.NewView(content + "\n" + helpLine)
	view.MouseMode = tea.MouseModeCellMotion
	return view
}

func (m Model) loadData() tea.Msg {
	projects, err := m.svc.ListProjects(context.Background(), false)
	if err != nil {
		return loadedMsg{err: err}
	}
	if len(projects) == 0 {
		return loadedMsg{projects: projects}
	}

	projectIdx := clamp(m.selectedProject, 0, len(projects)-1)
	projectID := projects[projectIdx].ID
	columns, err := m.svc.ListColumns(context.Background(), projectID, false)
	if err != nil {
		return loadedMsg{err: err}
	}

	var tasks []domain.Task
	if strings.TrimSpace(m.searchQuery) != "" {
		tasks, err = m.svc.SearchTasks(context.Background(), projectID, m.searchQuery, m.showArchived)
	} else {
		tasks, err = m.svc.ListTasks(context.Background(), projectID, m.showArchived)
	}
	if err != nil {
		return loadedMsg{err: err}
	}

	return loadedMsg{
		projects:        projects,
		selectedProject: projectIdx,
		columns:         columns,
		tasks:           tasks,
	}
}

func (m Model) handleNormalModeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.toggleHelp):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	case msg.String() == "esc":
		if m.searchQuery != "" {
			m.searchQuery = ""
			m.status = "search cleared"
			return m, m.loadData
		}
		return m, nil
	case key.Matches(msg, m.keys.reload):
		m.status = "reloading..."
		return m, m.loadData
	case key.Matches(msg, m.keys.moveLeft):
		if m.selectedColumn > 0 {
			m.selectedColumn--
			m.selectedTask = 0
		}
		return m, nil
	case key.Matches(msg, m.keys.moveRight):
		if m.selectedColumn < len(m.columns)-1 {
			m.selectedColumn++
			m.selectedTask = 0
		}
		return m, nil
	case key.Matches(msg, m.keys.moveDown):
		tasks := m.currentColumnTasks()
		if len(tasks) > 0 && m.selectedTask < len(tasks)-1 {
			m.selectedTask++
		}
		return m, nil
	case key.Matches(msg, m.keys.moveUp):
		if m.selectedTask > 0 {
			m.selectedTask--
		}
		return m, nil
	case key.Matches(msg, m.keys.addTask):
		m.mode = modeAddTask
		m.input = ""
		m.status = "new task"
		return m, nil
	case key.Matches(msg, m.keys.search):
		m.mode = modeSearch
		m.input = m.searchQuery
		m.status = "search mode"
		return m, nil
	case key.Matches(msg, m.keys.editTask):
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		m.mode = modeEditTask
		m.editingTaskID = task.ID
		m.input = formatTaskEditInput(task)
		m.status = "edit task"
		return m, nil
	case key.Matches(msg, m.keys.projects):
		if len(m.projects) > 0 {
			m.mode = modeProjectPicker
			m.projectPickerIndex = m.selectedProject
			m.status = "project picker"
			return m, nil
		}
		return m, nil
	case key.Matches(msg, m.keys.moveTaskLeft):
		return m.moveSelectedTask(-1)
	case key.Matches(msg, m.keys.moveTaskRight):
		return m.moveSelectedTask(1)
	case key.Matches(msg, m.keys.deleteTask):
		return m.deleteSelectedTask(m.defaultDeleteMode)
	case key.Matches(msg, m.keys.archiveTask):
		return m.deleteSelectedTask(app.DeleteModeArchive)
	case key.Matches(msg, m.keys.hardDeleteTask):
		return m.deleteSelectedTask(app.DeleteModeHard)
	case key.Matches(msg, m.keys.restoreTask):
		return m.restoreTask()
	case key.Matches(msg, m.keys.toggleArchived):
		m.showArchived = !m.showArchived
		if m.showArchived {
			m.status = "showing archived tasks"
		} else {
			m.status = "hiding archived tasks"
		}
		m.selectedTask = 0
		return m, m.loadData
	default:
		return m, nil
	}
}

func (m Model) handleInputModeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.mode == modeProjectPicker {
		switch msg.String() {
		case "esc":
			m.mode = modeNone
			m.status = "cancelled"
			return m, nil
		case "j", "down":
			if m.projectPickerIndex < len(m.projects)-1 {
				m.projectPickerIndex++
			}
			return m, nil
		case "k", "up":
			if m.projectPickerIndex > 0 {
				m.projectPickerIndex--
			}
			return m, nil
		case "enter":
			if len(m.projects) == 0 {
				m.mode = modeNone
				return m, nil
			}
			m.selectedProject = clamp(m.projectPickerIndex, 0, len(m.projects)-1)
			m.selectedColumn = 0
			m.selectedTask = 0
			m.mode = modeNone
			m.status = "project switched"
			return m, m.loadData
		default:
			return m, nil
		}
	}

	switch msg.String() {
	case "esc":
		m.mode = modeNone
		m.input = ""
		m.editingTaskID = ""
		m.status = "cancelled"
		return m, nil
	case "backspace":
		if m.input != "" {
			_, size := utf8.DecodeLastRuneInString(m.input)
			m.input = m.input[:len(m.input)-size]
		}
		return m, nil
	case "enter":
		return m.submitInputMode()
	default:
		if msg.Text != "" {
			m.input += msg.Text
		}
		return m, nil
	}
}

func (m Model) submitInputMode() (tea.Model, tea.Cmd) {
	text := strings.TrimSpace(m.input)
	switch m.mode {
	case modeAddTask:
		m.mode = modeNone
		m.input = ""
		if text == "" {
			m.status = "title required"
			return m, nil
		}
		return m.createTask(text)
	case modeSearch:
		m.mode = modeNone
		m.searchQuery = text
		m.input = ""
		m.selectedTask = 0
		m.status = "search updated"
		return m, m.loadData
	case modeRenameTask:
		m.mode = modeNone
		m.input = ""
		if text == "" {
			m.status = "title required"
			return m, nil
		}
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		taskID := task.ID
		return m, func() tea.Msg {
			_, err := m.svc.RenameTask(context.Background(), taskID, text)
			if err != nil {
				return actionMsg{err: err}
			}
			return actionMsg{status: "task renamed", reload: true}
		}
	case modeEditTask:
		taskID := m.editingTaskID
		if taskID == "" {
			task, ok := m.selectedTaskInCurrentColumn()
			if !ok {
				m.status = "no task selected"
				return m, nil
			}
			taskID = task.ID
		}
		task, ok := m.taskByID(taskID)
		if !ok {
			m.status = "task not found"
			return m, nil
		}

		in, err := parseTaskEditInput(text, task)
		if err != nil {
			m.status = "invalid edit format: " + err.Error()
			return m, nil
		}

		m.mode = modeNone
		m.input = ""
		m.editingTaskID = ""
		in.TaskID = taskID
		return m, func() tea.Msg {
			_, updateErr := m.svc.UpdateTask(context.Background(), in)
			if updateErr != nil {
				return actionMsg{err: updateErr}
			}
			return actionMsg{status: "task updated", reload: true}
		}
	default:
		return m, nil
	}
}

func (m Model) createTask(title string) (tea.Model, tea.Cmd) {
	projectID, ok := m.currentProjectID()
	if !ok {
		m.status = "no active project"
		return m, nil
	}
	columnID, ok := m.currentColumnID()
	if !ok {
		m.status = "no active column"
		return m, nil
	}
	return m, func() tea.Msg {
		_, err := m.svc.CreateTask(context.Background(), app.CreateTaskInput{
			ProjectID: projectID,
			ColumnID:  columnID,
			Title:     title,
			Priority:  domain.PriorityMedium,
		})
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{status: "task created", reload: true}
	}
}

func (m Model) moveSelectedTask(delta int) (tea.Model, tea.Cmd) {
	task, ok := m.selectedTaskInCurrentColumn()
	if !ok {
		m.status = "no task selected"
		return m, nil
	}
	targetCol := m.selectedColumn + delta
	if targetCol < 0 || targetCol >= len(m.columns) {
		return m, nil
	}
	targetColumnID := m.columns[targetCol].ID
	targetPos := len(m.tasksForColumn(targetColumnID))
	m.selectedColumn = targetCol
	m.selectedTask = targetPos
	taskID := task.ID
	return m, func() tea.Msg {
		_, err := m.svc.MoveTask(context.Background(), taskID, targetColumnID, targetPos)
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{status: "task moved", reload: true}
	}
}

func (m Model) deleteSelectedTask(mode app.DeleteMode) (tea.Model, tea.Cmd) {
	task, ok := m.selectedTaskInCurrentColumn()
	if !ok {
		m.status = "no task selected"
		return m, nil
	}
	taskID := task.ID
	if mode == app.DeleteModeArchive {
		m.lastArchivedTaskID = taskID
	}
	return m, func() tea.Msg {
		if err := m.svc.DeleteTask(context.Background(), taskID, mode); err != nil {
			return actionMsg{err: err}
		}
		if mode == app.DeleteModeHard {
			return actionMsg{status: "task deleted", reload: true}
		}
		return actionMsg{status: "task archived", reload: true}
	}
}

func (m Model) restoreTask() (tea.Model, tea.Cmd) {
	taskID := m.lastArchivedTaskID
	if taskID == "" {
		task, ok := m.selectedTaskInCurrentColumn()
		if ok && task.ArchivedAt != nil {
			taskID = task.ID
		}
	}
	if taskID == "" {
		m.status = "nothing to restore"
		return m, nil
	}

	return m, func() tea.Msg {
		_, err := m.svc.RestoreTask(context.Background(), taskID)
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{status: "task restored", reload: true}
	}
}

func (m Model) handleMouseWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	if m.mode == modeProjectPicker {
		switch msg.Button {
		case tea.MouseWheelUp:
			if m.projectPickerIndex > 0 {
				m.projectPickerIndex--
			}
		case tea.MouseWheelDown:
			if m.projectPickerIndex < len(m.projects)-1 {
				m.projectPickerIndex++
			}
		}
		return m, nil
	}
	if m.mode != modeNone {
		return m, nil
	}

	tasks := m.currentColumnTasks()
	if len(tasks) == 0 {
		return m, nil
	}

	switch msg.Button {
	case tea.MouseWheelUp:
		if m.selectedTask > 0 {
			m.selectedTask--
		}
	case tea.MouseWheelDown:
		if m.selectedTask < len(tasks)-1 {
			m.selectedTask++
		}
	}
	return m, nil
}

func (m Model) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if m.mode == modeProjectPicker {
		overlayTop := 3                    // after header/help/blank
		relative := msg.Y - overlayTop - 1 // inside border, first row is title
		if relative >= 1 {
			idx := relative - 1
			if idx >= 0 && idx < len(m.projects) {
				m.projectPickerIndex = idx
			}
		}
		return m, nil
	}
	if m.mode != modeNone {
		return m, nil
	}

	if len(m.columns) == 0 {
		return m, nil
	}
	colWidth := m.columnWidth() + 4 // account for border/padding
	gap := 0
	for idx := range m.columns {
		start := idx * (colWidth + gap)
		end := start + colWidth
		if msg.X >= start && msg.X < end {
			m.selectedColumn = idx
			break
		}
	}

	relativeY := msg.Y - m.boardTop()
	if relativeY >= 2 {
		idx := relativeY - 2
		tasks := m.currentColumnTasks()
		if idx >= 0 && len(tasks) > 0 {
			m.selectedTask = clamp(idx, 0, len(tasks)-1)
		}
	}
	m.clampSelections()
	return m, nil
}

func (m *Model) clampSelections() {
	if len(m.projects) == 0 {
		m.selectedProject = 0
		m.selectedColumn = 0
		m.selectedTask = 0
		return
	}
	m.selectedProject = clamp(m.selectedProject, 0, len(m.projects)-1)

	if len(m.columns) == 0 {
		m.selectedColumn = 0
		m.selectedTask = 0
		return
	}
	m.selectedColumn = clamp(m.selectedColumn, 0, len(m.columns)-1)
	colTasks := m.currentColumnTasks()
	if len(colTasks) == 0 {
		m.selectedTask = 0
		return
	}
	m.selectedTask = clamp(m.selectedTask, 0, len(colTasks)-1)
}

func (m Model) currentProjectID() (string, bool) {
	if len(m.projects) == 0 {
		return "", false
	}
	idx := clamp(m.selectedProject, 0, len(m.projects)-1)
	return m.projects[idx].ID, true
}

func (m Model) currentColumnID() (string, bool) {
	if len(m.columns) == 0 {
		return "", false
	}
	idx := clamp(m.selectedColumn, 0, len(m.columns)-1)
	return m.columns[idx].ID, true
}

func (m Model) currentColumnTasks() []domain.Task {
	columnID, ok := m.currentColumnID()
	if !ok {
		return nil
	}
	return m.tasksForColumn(columnID)
}

func (m Model) tasksForColumn(columnID string) []domain.Task {
	out := make([]domain.Task, 0)
	for _, task := range m.tasks {
		if task.ColumnID != columnID {
			continue
		}
		out = append(out, task)
	}
	return out
}

func (m Model) selectedTaskInCurrentColumn() (domain.Task, bool) {
	tasks := m.currentColumnTasks()
	if len(tasks) == 0 {
		return domain.Task{}, false
	}
	idx := clamp(m.selectedTask, 0, len(tasks)-1)
	return tasks[idx], true
}

func (m Model) taskByID(taskID string) (domain.Task, bool) {
	for _, task := range m.tasks {
		if task.ID == taskID {
			return task, true
		}
	}
	return domain.Task{}, false
}

func (m Model) cardMeta(task domain.Task) string {
	parts := make([]string, 0, 3)
	if m.taskFields.ShowPriority {
		parts = append(parts, string(task.Priority))
	}
	if m.taskFields.ShowDueDate && task.DueAt != nil {
		parts = append(parts, task.DueAt.UTC().Format("01-02"))
	}
	if m.taskFields.ShowLabels && len(task.Labels) > 0 {
		parts = append(parts, summarizeLabels(task.Labels, 2))
	}
	if len(parts) == 0 {
		return ""
	}
	return "[" + strings.Join(parts, "|") + "]"
}

func (m Model) renderTaskDetails(accent, muted, dim color.Color) string {
	task, ok := m.selectedTaskInCurrentColumn()
	if !ok {
		return ""
	}

	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(accent).Render("Task Details"),
		task.Title,
	}

	meta := make([]string, 0, 3)
	if m.taskFields.ShowPriority {
		meta = append(meta, "priority: "+string(task.Priority))
	}
	if m.taskFields.ShowDueDate {
		due := "-"
		if task.DueAt != nil {
			due = task.DueAt.UTC().Format("2006-01-02")
		}
		meta = append(meta, "due: "+due)
	}
	if m.taskFields.ShowLabels {
		labels := "-"
		if len(task.Labels) > 0 {
			labels = strings.Join(task.Labels, ", ")
		}
		meta = append(meta, "labels: "+labels)
	}
	if len(meta) > 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(muted).Render(strings.Join(meta, "  ")))
	}

	if m.taskFields.ShowDescription {
		if desc := strings.TrimSpace(task.Description); desc != "" {
			lines = append(lines, desc)
		} else {
			lines = append(lines, lipgloss.NewStyle().Foreground(muted).Render("description: -"))
		}
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(dim).
		Padding(0, 1)
	if m.width > 0 {
		style = style.Width(max(24, m.width-2))
	}
	return style.Render(strings.Join(lines, "\n"))
}

func (m Model) renderModeOverlay(accent, muted, dim color.Color, helpStyle lipgloss.Style) string {
	switch m.mode {
	case modeProjectPicker:
		pickerStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if m.width > 0 {
			pickerStyle = pickerStyle.Width(clamp(m.width-8, 28, 56))
		}

		title := lipgloss.NewStyle().Bold(true).Foreground(accent).Render("Projects")
		lines := []string{title}
		for idx, p := range m.projects {
			cursor := "  "
			if idx == m.projectPickerIndex {
				cursor = "> "
			}
			lines = append(lines, cursor+p.Name)
		}
		lines = append(lines, helpStyle.Render("j/k or wheel • enter choose • esc cancel"))
		return pickerStyle.Render(strings.Join(lines, "\n"))

	case modeAddTask, modeSearch, modeRenameTask, modeEditTask:
		boxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if m.width > 0 {
			boxStyle = boxStyle.Width(clamp(m.width-8, 48, 96))
		}

		title := "Input"
		hint := "enter save • esc cancel"
		switch m.mode {
		case modeAddTask:
			title = "New Task"
		case modeSearch:
			title = "Search"
			hint = "enter apply • esc cancel"
		case modeRenameTask:
			title = "Rename Task"
		case modeEditTask:
			title = "Edit Task"
			hint = "enter save • esc cancel • format: title|description|priority|due|labels"
		}

		value := m.input
		if value == "" {
			value = " "
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("230"))
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		return boxStyle.Render(strings.Join([]string{
			titleStyle.Render(title),
			valueStyle.Render(value),
			hintStyle.Render(hint),
		}, "\n"))
	default:
		return ""
	}
}

func formatTaskEditInput(task domain.Task) string {
	due := "-"
	if task.DueAt != nil {
		due = task.DueAt.UTC().Format("2006-01-02")
	}
	labels := "-"
	if len(task.Labels) > 0 {
		labels = strings.Join(task.Labels, ",")
	}
	return strings.Join([]string{
		task.Title,
		task.Description,
		string(task.Priority),
		due,
		labels,
	}, " | ")
}

func parseTaskEditInput(raw string, current domain.Task) (app.UpdateTaskInput, error) {
	parts := strings.Split(raw, "|")
	for len(parts) < 5 {
		parts = append(parts, "")
	}
	if len(parts) > 5 {
		return app.UpdateTaskInput{}, fmt.Errorf("expected 5 fields")
	}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	title := parts[0]
	if title == "" {
		title = current.Title
	}

	description := parts[1]
	if description == "" {
		description = current.Description
	}

	priority := domain.Priority(parts[2])
	if priority == "" {
		priority = current.Priority
	}
	switch priority {
	case domain.PriorityLow, domain.PriorityMedium, domain.PriorityHigh:
	default:
		return app.UpdateTaskInput{}, fmt.Errorf("priority must be low|medium|high")
	}

	dueAt := current.DueAt
	if parts[3] == "-" {
		dueAt = nil
	} else if parts[3] != "" {
		parsed, err := time.Parse("2006-01-02", parts[3])
		if err != nil {
			return app.UpdateTaskInput{}, fmt.Errorf("due date must be YYYY-MM-DD or -")
		}
		ts := parsed.UTC()
		dueAt = &ts
	}

	labels := current.Labels
	if parts[4] == "-" {
		labels = nil
	} else if parts[4] != "" {
		rawLabels := strings.Split(parts[4], ",")
		parsedLabels := make([]string, 0, len(rawLabels))
		for _, label := range rawLabels {
			label = strings.TrimSpace(label)
			if label == "" {
				continue
			}
			parsedLabels = append(parsedLabels, label)
		}
		labels = parsedLabels
	}

	return app.UpdateTaskInput{
		Title:       title,
		Description: description,
		Priority:    priority,
		DueAt:       dueAt,
		Labels:      labels,
	}, nil
}

func (m Model) modeLabel() string {
	switch m.mode {
	case modeAddTask:
		return "add-task"
	case modeSearch:
		return "search"
	case modeRenameTask:
		return "rename"
	case modeEditTask:
		return "edit-task"
	case modeProjectPicker:
		return "project-picker"
	default:
		return "normal"
	}
}

func (m Model) modePrompt() string {
	switch m.mode {
	case modeAddTask:
		return "new task title: " + m.input + " (enter save, esc cancel)"
	case modeSearch:
		return "search query: " + m.input + " (enter apply, esc cancel)"
	case modeRenameTask:
		return "rename task: " + m.input + " (enter save, esc cancel)"
	case modeEditTask:
		return "edit task: " + m.input + " (title | description | priority(low|medium|high) | due(YYYY-MM-DD or -) | labels(csv))"
	case modeProjectPicker:
		return "project picker: j/k select, enter choose, esc cancel"
	default:
		return ""
	}
}

func (m Model) columnWidth() int {
	if len(m.columns) == 0 {
		return 24
	}
	w := 28
	if m.width > 0 {
		candidate := (m.width - (len(m.columns) - 1)) / len(m.columns)
		if candidate > 0 {
			w = candidate - 4
		}
	}
	if w < 18 {
		return 18
	}
	if w > 42 {
		return 42
	}
	return w
}

func (m Model) boardTop() int {
	// header + spacer
	return 3
}

func clamp(v, minV, maxV int) int {
	if maxV < minV {
		return minV
	}
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	rs := []rune(s)
	if len(rs) <= max {
		return s
	}
	if max <= 1 {
		return string(rs[:max])
	}
	return string(rs[:max-1]) + "…"
}

func summarizeLabels(labels []string, maxLabels int) string {
	if len(labels) == 0 {
		return ""
	}
	if maxLabels <= 0 {
		maxLabels = 1
	}
	visible := labels
	extra := 0
	if len(labels) > maxLabels {
		visible = labels[:maxLabels]
		extra = len(labels) - maxLabels
	}
	joined := "#" + strings.Join(visible, ",#")
	if extra > 0 {
		joined += fmt.Sprintf("+%d", extra)
	}
	return joined
}
