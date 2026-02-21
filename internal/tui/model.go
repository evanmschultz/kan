package tui

import (
	"context"
	"fmt"
	"image/color"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
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
	SearchTaskMatches(context.Context, app.SearchTasksFilter) ([]app.TaskMatch, error)
	CreateProjectWithMetadata(context.Context, app.CreateProjectInput) (domain.Project, error)
	UpdateProject(context.Context, app.UpdateProjectInput) (domain.Project, error)
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
	modeDuePicker
	modeProjectPicker
	modeTaskInfo
	modeAddProject
	modeEditProject
	modeSearchResults
	modeCommandPalette
	modeQuickActions
)

var taskFormFields = []string{"title", "description", "priority", "due", "labels"}

var priorityOptions = []domain.Priority{
	domain.PriorityLow,
	domain.PriorityMedium,
	domain.PriorityHigh,
}

type duePickerOption struct {
	Label string
	Value string
}

var quickActionOptions = []string{
	"Task Info",
	"Edit Task",
	"Move Left",
	"Move Right",
	"Archive Task",
	"Hard Delete",
}

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

	mode          inputMode
	input         string
	searchQuery   string
	searchApplied bool
	showArchived  bool

	searchInput        textinput.Model
	searchStateInput   textinput.Model
	commandInput       textinput.Model
	searchFocus        int
	searchCrossProject bool
	searchStates       []string
	searchMatches      []app.TaskMatch
	searchResultIndex  int
	quickActionIndex   int

	formInputs  []textinput.Model
	formFocus   int
	priorityIdx int
	duePicker   int
	pickerBack  inputMode

	projectPickerIndex int
	projectFormInputs  []textinput.Model
	projectFormFocus   int
	editingProjectID   string
	editingTaskID      string
	pendingProjectID   string
	pendingFocusTaskID string

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
	err       error
	status    string
	reload    bool
	projectID string
}

type searchResultsMsg struct {
	matches []app.TaskMatch
	err     error
}

func NewModel(svc Service, opts ...Option) Model {
	h := help.New()
	h.ShowAll = false
	searchInput := textinput.New()
	searchInput.Prompt = "query: "
	searchInput.Placeholder = "title, description, labels"
	searchInput.CharLimit = 120
	searchStateInput := textinput.New()
	searchStateInput.Prompt = "states: "
	searchStateInput.Placeholder = "todo,progress,done,archived"
	searchStateInput.CharLimit = 120
	commandInput := textinput.New()
	commandInput.Prompt = ": "
	commandInput.Placeholder = "new-project | edit-project | search-all | clear-search | help | quit"
	commandInput.CharLimit = 120
	m := Model{
		svc:               svc,
		status:            "loading...",
		help:              h,
		keys:              newKeyMap(),
		taskFields:        DefaultTaskFieldConfig(),
		defaultDeleteMode: app.DeleteModeArchive,
		searchInput:       searchInput,
		searchStateInput:  searchStateInput,
		commandInput:      commandInput,
		searchStates:      []string{"todo", "progress", "done"},
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
		if m.pendingProjectID != "" {
			for idx, project := range m.projects {
				if project.ID == m.pendingProjectID {
					m.selectedProject = idx
					break
				}
			}
			m.pendingProjectID = ""
		}
		m.clampSelections()
		if m.pendingFocusTaskID != "" {
			m.focusTaskByID(m.pendingFocusTaskID)
			m.pendingFocusTaskID = ""
		}
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
		if msg.projectID != "" {
			m.pendingProjectID = msg.projectID
		}
		if msg.reload {
			return m, m.loadData
		}
		return m, nil

	case searchResultsMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.searchMatches = msg.matches
		m.searchResultIndex = clamp(m.searchResultIndex, 0, len(m.searchMatches)-1)
		if len(m.searchMatches) > 0 {
			m.mode = modeSearchResults
			m.status = fmt.Sprintf("%d matches", len(m.searchMatches))
		} else {
			m.mode = modeNone
			m.status = "no matches"
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
		v.AltScreen = true
		return v
	}
	if !m.ready {
		v := tea.NewView("loading...")
		v.MouseMode = tea.MouseModeCellMotion
		v.AltScreen = true
		return v
	}
	if len(m.projects) == 0 {
		v := tea.NewView("no projects yet\n\npress q to quit\n")
		v.MouseMode = tea.MouseModeCellMotion
		v.AltScreen = true
		return v
	}

	project := m.projects[clamp(m.selectedProject, 0, len(m.projects)-1)]
	accent := lipgloss.Color("62")
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	helpStyle := lipgloss.NewStyle().Foreground(muted)
	statusStyle := lipgloss.NewStyle().Foreground(dim)

	header := titleStyle.Render("kan") + "  " + project.Name
	header += statusStyle.Render("  [" + m.modeLabel() + "]")
	if m.searchApplied && m.searchQuery != "" {
		header += statusStyle.Render("  search: " + m.searchQuery)
	}
	if m.searchApplied && m.searchCrossProject {
		header += statusStyle.Render("  scope: all-projects")
	}
	if m.showArchived {
		header += statusStyle.Render("  showing archived")
	}
	tabs := m.renderProjectTabs(accent, dim)
	boardWidth := m.width

	columnViews := make([]string, 0, len(m.columns))
	colWidth := m.columnWidthFor(boardWidth)
	colHeight := m.columnHeight()
	baseColStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(dim).
		Padding(1, 2).
		MarginRight(1).
		Width(colWidth).
		Height(colHeight)
	selColStyle := baseColStyle.Copy().BorderForeground(accent)
	normColStyle := baseColStyle.Copy()
	colTitle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	archivedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	selectedTaskStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
	itemSubStyle := lipgloss.NewStyle().Foreground(muted)

	for colIdx, column := range m.columns {
		colTasks := m.tasksForColumn(column.ID)
		lines := []string{colTitle.Render(fmt.Sprintf("%s (%d)", column.Name, len(colTasks)))}
		if len(colTasks) == 0 {
			lines = append(lines, archivedStyle.Render("(empty)"))
		} else {
			for taskIdx, task := range colTasks {
				cursor := "  "
				if colIdx == m.selectedColumn && taskIdx == m.selectedTask {
					cursor = "> "
				}
				title := cursor + "│ " + truncate(task.Title, max(1, colWidth-8))
				sub := m.taskListSecondary(task)
				if sub != "" {
					sub = truncate(sub, max(1, colWidth-8))
				}
				if task.ArchivedAt != nil {
					title = archivedStyle.Render(title)
					if sub != "" {
						sub = archivedStyle.Render(sub)
					}
				} else if colIdx == m.selectedColumn && taskIdx == m.selectedTask {
					title = selectedTaskStyle.Render(title)
				}
				lines = append(lines, title)
				if sub != "" {
					lines = append(lines, "  │ "+itemSubStyle.Render(sub))
				}
				if taskIdx < len(colTasks)-1 {
					lines = append(lines, "")
				}
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
	overlay := m.renderModeOverlay(accent, muted, dim, helpStyle, m.width-8)
	if m.help.ShowAll {
		overlay = m.renderHelpOverlay(accent, muted, dim, helpStyle, m.width-8)
	}

	mainArea := body
	infoLine := m.renderInfoLine(project, muted)

	sections := []string{header}
	if tabs != "" {
		sections = append(sections, tabs)
	}
	sections = append(sections, "", mainArea)
	if infoLine != "" {
		sections = append(sections, infoLine)
	}
	if strings.TrimSpace(m.status) != "" && m.status != "ready" {
		sections = append(sections, statusStyle.Render(m.status))
	}
	content := strings.Join(sections, "\n")

	helpBubble := m.help
	helpBubble.ShowAll = false
	helpBubble.SetWidth(max(0, m.width-2))
	helpLine := lipgloss.NewStyle().
		Foreground(muted).
		BorderTop(true).
		BorderForeground(dim).
		Padding(0, 1).
		Width(max(0, m.width)).
		Render(helpBubble.View(m.keys))

	contentHeight := lipgloss.Height(content)
	if m.height > 0 {
		helpHeight := lipgloss.Height(helpLine)
		contentHeight = max(0, m.height-helpHeight)
		content = fitLines(content, contentHeight)
	}

	fullContent := content + "\n" + helpLine
	if overlay != "" {
		overlayHeight := lipgloss.Height(fullContent)
		if m.height > 0 {
			overlayHeight = m.height
		}
		fullContent = overlayOnContent(fullContent, overlay, max(1, m.width), max(1, overlayHeight))
	}

	view := tea.NewView(fullContent)
	view.MouseMode = tea.MouseModeCellMotion
	view.AltScreen = true
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
	searchFilterActive := m.searchApplied
	if searchFilterActive {
		matches, searchErr := m.svc.SearchTaskMatches(context.Background(), app.SearchTasksFilter{
			ProjectID:       projectID,
			Query:           m.searchQuery,
			CrossProject:    m.searchCrossProject,
			IncludeArchived: m.showArchived,
			States:          append([]string(nil), m.searchStates...),
		})
		if searchErr != nil {
			return loadedMsg{err: searchErr}
		}
		tasks = make([]domain.Task, 0, len(matches))
		for _, match := range matches {
			if match.Project.ID == projectID {
				tasks = append(tasks, match.Task)
			}
		}
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

func (m Model) loadSearchMatches() tea.Msg {
	projectID, _ := m.currentProjectID()
	matches, err := m.svc.SearchTaskMatches(context.Background(), app.SearchTasksFilter{
		ProjectID:       projectID,
		Query:           m.searchQuery,
		CrossProject:    m.searchCrossProject,
		IncludeArchived: m.showArchived,
		States:          append([]string(nil), m.searchStates...),
	})
	if err != nil {
		return searchResultsMsg{err: err}
	}
	return searchResultsMsg{matches: matches}
}

func newModalInput(prompt, placeholder, value string, limit int) textinput.Model {
	in := textinput.New()
	in.Prompt = prompt
	in.Placeholder = placeholder
	in.CharLimit = limit
	if value != "" {
		in.SetValue(value)
	}
	return in
}

func (m *Model) startSearchMode() tea.Cmd {
	m.mode = modeSearch
	m.input = ""
	m.searchInput.SetValue(m.searchQuery)
	m.searchInput.CursorEnd()
	m.searchStateInput.SetValue(strings.Join(m.searchStates, ","))
	m.searchStateInput.CursorEnd()
	m.searchFocus = 0
	m.status = "search"
	return m.searchInput.Focus()
}

func (m *Model) startCommandPalette() tea.Cmd {
	m.mode = modeCommandPalette
	m.commandInput.SetValue("")
	m.commandInput.CursorEnd()
	m.status = "command palette"
	return m.commandInput.Focus()
}

func (m *Model) startQuickActions() tea.Cmd {
	if _, ok := m.selectedTaskInCurrentColumn(); !ok {
		m.status = "no task selected"
		return nil
	}
	m.mode = modeQuickActions
	m.quickActionIndex = 0
	m.status = "quick actions"
	return nil
}

func (m *Model) startProjectForm(project *domain.Project) tea.Cmd {
	m.projectFormFocus = 0
	m.projectFormInputs = []textinput.Model{
		newModalInput("", "project name", "", 120),
		newModalInput("", "short description", "", 240),
		newModalInput("", "owner/team", "", 120),
		newModalInput("", "icon text", "", 24),
		newModalInput("", "accent color (e.g. 62)", "", 32),
		newModalInput("", "https://...", "", 200),
		newModalInput("", "csv tags", "", 200),
	}
	m.editingProjectID = ""
	if project != nil {
		m.mode = modeEditProject
		m.status = "edit project"
		m.editingProjectID = project.ID
		m.projectFormInputs[0].SetValue(project.Name)
		m.projectFormInputs[1].SetValue(project.Description)
		m.projectFormInputs[2].SetValue(project.Metadata.Owner)
		m.projectFormInputs[3].SetValue(project.Metadata.Icon)
		m.projectFormInputs[4].SetValue(project.Metadata.Color)
		m.projectFormInputs[5].SetValue(project.Metadata.Homepage)
		if len(project.Metadata.Tags) > 0 {
			m.projectFormInputs[6].SetValue(strings.Join(project.Metadata.Tags, ","))
		}
	} else {
		m.mode = modeAddProject
		m.status = "new project"
	}
	return m.focusProjectFormField(0)
}

func (m *Model) startTaskForm(task *domain.Task) tea.Cmd {
	m.formFocus = 0
	m.priorityIdx = 1
	m.duePicker = 0
	m.pickerBack = modeNone
	m.input = ""
	m.formInputs = []textinput.Model{
		newModalInput("", "task title (required)", "", 120),
		newModalInput("", "short description", "", 240),
		newModalInput("", "low | medium | high", "", 16),
		newModalInput("", "YYYY-MM-DD or -", "", 16),
		newModalInput("", "csv labels", "", 160),
	}
	m.formInputs[2].SetValue(string(priorityOptions[m.priorityIdx]))
	if task != nil {
		m.formInputs[0].SetValue(task.Title)
		m.formInputs[1].SetValue(task.Description)
		m.priorityIdx = priorityIndex(task.Priority)
		m.formInputs[2].SetValue(string(priorityOptions[m.priorityIdx]))
		if task.DueAt != nil {
			m.formInputs[3].SetValue(task.DueAt.UTC().Format("2006-01-02"))
		}
		if len(task.Labels) > 0 {
			m.formInputs[4].SetValue(strings.Join(task.Labels, ","))
		}
		m.mode = modeEditTask
		m.editingTaskID = task.ID
		m.status = "edit task"
	} else {
		m.formInputs[2].Placeholder = "medium"
		m.formInputs[3].Placeholder = "-"
		m.formInputs[4].Placeholder = "-"
		m.mode = modeAddTask
		m.editingTaskID = ""
		m.status = "new task"
	}
	return m.focusTaskFormField(0)
}

func (m *Model) focusTaskFormField(idx int) tea.Cmd {
	if len(m.formInputs) == 0 {
		return nil
	}
	idx = clamp(idx, 0, len(m.formInputs)-1)
	m.formFocus = idx
	for i := range m.formInputs {
		m.formInputs[i].Blur()
	}
	if idx == 2 {
		return nil
	}
	return m.formInputs[idx].Focus()
}

func (m *Model) focusProjectFormField(idx int) tea.Cmd {
	if len(m.projectFormInputs) == 0 {
		return nil
	}
	idx = clamp(idx, 0, len(m.projectFormInputs)-1)
	m.projectFormFocus = idx
	for i := range m.projectFormInputs {
		m.projectFormInputs[i].Blur()
	}
	return m.projectFormInputs[idx].Focus()
}

func (m Model) taskFormValues() map[string]string {
	out := map[string]string{}
	for i, key := range taskFormFields {
		if i >= len(m.formInputs) {
			break
		}
		out[key] = strings.TrimSpace(m.formInputs[i].Value())
	}
	return out
}

var projectFormFields = []string{"name", "description", "owner", "icon", "color", "homepage", "tags"}

func (m Model) projectFormValues() map[string]string {
	out := map[string]string{}
	for idx, key := range projectFormFields {
		if idx >= len(m.projectFormInputs) {
			break
		}
		out[key] = strings.TrimSpace(m.projectFormInputs[idx].Value())
	}
	return out
}

func parseDueInput(raw string, current *time.Time) (*time.Time, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return current, nil
	}
	if text == "-" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", text)
	if err != nil {
		return nil, fmt.Errorf("due date must be YYYY-MM-DD or -")
	}
	ts := parsed.UTC()
	return &ts, nil
}

func parseLabelsInput(raw string, current []string) []string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return current
	}
	if text == "-" {
		return nil
	}
	rawLabels := strings.Split(text, ",")
	out := make([]string, 0, len(rawLabels))
	for _, label := range rawLabels {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		out = append(out, label)
	}
	return out
}

func parseStateFilters(raw string, fallback []string) []string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return append([]string(nil), fallback...)
	}
	parts := strings.Split(text, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		state := strings.TrimSpace(strings.ToLower(part))
		if state == "" {
			continue
		}
		if _, ok := seen[state]; ok {
			continue
		}
		seen[state] = struct{}{}
		out = append(out, state)
	}
	return out
}

func priorityIndex(priority domain.Priority) int {
	for i, p := range priorityOptions {
		if p == priority {
			return i
		}
	}
	return 1
}

func (m *Model) cyclePriority(delta int) {
	if len(priorityOptions) == 0 {
		return
	}
	m.priorityIdx += delta
	if m.priorityIdx < 0 {
		m.priorityIdx = len(priorityOptions) - 1
	}
	if m.priorityIdx >= len(priorityOptions) {
		m.priorityIdx = 0
	}
	if len(m.formInputs) > 2 {
		m.formInputs[2].SetValue(string(priorityOptions[m.priorityIdx]))
	}
}

func (m *Model) startDuePicker() {
	m.pickerBack = m.mode
	m.mode = modeDuePicker
	m.duePicker = 0
}

func (m *Model) duePickerOptions() []duePickerOption {
	now := time.Now().UTC()
	today := now.Format("2006-01-02")
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")
	nextWeek := now.AddDate(0, 0, 7).Format("2006-01-02")
	inTwoWeeks := now.AddDate(0, 0, 14).Format("2006-01-02")
	return []duePickerOption{
		{Label: "No due date", Value: "-"},
		{Label: "Today (" + today + ")", Value: today},
		{Label: "Tomorrow (" + tomorrow + ")", Value: tomorrow},
		{Label: "Next week (" + nextWeek + ")", Value: nextWeek},
		{Label: "In two weeks (" + inTwoWeeks + ")", Value: inTwoWeeks},
	}
}

func (m Model) labelSuggestions(maxLabels int) []string {
	if maxLabels <= 0 {
		maxLabels = 5
	}
	projectID, ok := m.currentProjectID()
	if !ok {
		return nil
	}
	counts := map[string]int{}
	for _, task := range m.tasks {
		if task.ProjectID != projectID {
			continue
		}
		for _, label := range task.Labels {
			label = strings.TrimSpace(label)
			if label == "" {
				continue
			}
			counts[label]++
		}
	}
	if len(counts) == 0 {
		return nil
	}
	type pair struct {
		label string
		count int
	}
	pairs := make([]pair, 0, len(counts))
	for label, count := range counts {
		pairs = append(pairs, pair{label: label, count: count})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count == pairs[j].count {
			return pairs[i].label < pairs[j].label
		}
		return pairs[i].count > pairs[j].count
	})
	out := make([]string, 0, min(maxLabels, len(pairs)))
	for idx := range pairs {
		if idx >= maxLabels {
			break
		}
		out = append(out, pairs[idx].label)
	}
	return out
}

func (m Model) handleNormalModeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.toggleHelp):
		m.help.ShowAll = !m.help.ShowAll
		if m.help.ShowAll {
			m.status = "help"
		} else {
			m.status = "ready"
		}
		return m, nil
	case msg.String() == "esc":
		if m.help.ShowAll {
			m.help.ShowAll = false
			m.status = "ready"
			return m, nil
		}
		if m.searchApplied || m.searchQuery != "" {
			m.searchApplied = false
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
		m.help.ShowAll = false
		return m, m.startTaskForm(nil)
	case key.Matches(msg, m.keys.newProject):
		m.help.ShowAll = false
		return m, m.startProjectForm(nil)
	case key.Matches(msg, m.keys.taskInfo):
		if _, ok := m.selectedTaskInCurrentColumn(); !ok {
			m.status = "no task selected"
			return m, nil
		}
		m.help.ShowAll = false
		m.mode = modeTaskInfo
		m.status = "task info"
		return m, nil
	case key.Matches(msg, m.keys.search):
		m.help.ShowAll = false
		return m, m.startSearchMode()
	case key.Matches(msg, m.keys.commandPalette):
		m.help.ShowAll = false
		return m, m.startCommandPalette()
	case key.Matches(msg, m.keys.quickActions):
		m.help.ShowAll = false
		return m, m.startQuickActions()
	case key.Matches(msg, m.keys.editTask):
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		m.help.ShowAll = false
		return m, m.startTaskForm(&task)
	case key.Matches(msg, m.keys.editProject):
		if len(m.projects) == 0 {
			m.status = "no project selected"
			return m, nil
		}
		m.help.ShowAll = false
		project := m.projects[clamp(m.selectedProject, 0, len(m.projects)-1)]
		return m, m.startProjectForm(&project)
	case key.Matches(msg, m.keys.projects):
		if len(m.projects) > 0 {
			m.help.ShowAll = false
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
	if m.mode == modeTaskInfo {
		switch msg.String() {
		case "esc", "i":
			m.mode = modeNone
			m.status = "ready"
			return m, nil
		case "e":
			task, ok := m.selectedTaskInCurrentColumn()
			if !ok {
				m.status = "no task selected"
				return m, nil
			}
			return m, m.startTaskForm(&task)
		default:
			return m, nil
		}
	}

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

	if m.mode == modeSearch {
		switch {
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			m.mode = modeNone
			m.searchInput.Blur()
			m.searchStateInput.Blur()
			m.status = "cancelled"
			return m, nil
		case msg.Code == tea.KeyTab || msg.String() == "tab":
			if m.searchFocus == 0 {
				m.searchFocus = 1
				m.searchInput.Blur()
				return m, m.searchStateInput.Focus()
			}
			m.searchFocus = 0
			m.searchStateInput.Blur()
			return m, m.searchInput.Focus()
		case msg.String() == "ctrl+p":
			m.searchCrossProject = !m.searchCrossProject
			return m, nil
		case msg.String() == "ctrl+a":
			m.showArchived = !m.showArchived
			return m, nil
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			text := strings.TrimSpace(m.searchInput.Value())
			states := parseStateFilters(m.searchStateInput.Value(), m.searchStates)
			m.mode = modeNone
			m.searchInput.Blur()
			m.searchStateInput.Blur()
			m.searchQuery = text
			m.searchStates = states
			m.searchApplied = true
			m.status = "search updated"
			m.selectedTask = 0
			if m.searchCrossProject {
				return m, m.loadSearchMatches
			}
			return m, m.loadData
		default:
			var cmd tea.Cmd
			if m.searchFocus == 0 {
				m.searchInput, cmd = m.searchInput.Update(msg)
			} else {
				m.searchStateInput, cmd = m.searchStateInput.Update(msg)
			}
			return m, cmd
		}
	}

	if m.mode == modeSearchResults {
		switch msg.String() {
		case "esc":
			m.mode = modeNone
			m.status = "ready"
			return m, nil
		case "j", "down":
			if m.searchResultIndex < len(m.searchMatches)-1 {
				m.searchResultIndex++
			}
			return m, nil
		case "k", "up":
			if m.searchResultIndex > 0 {
				m.searchResultIndex--
			}
			return m, nil
		case "enter":
			if len(m.searchMatches) == 0 {
				m.mode = modeNone
				m.status = "no matches"
				return m, nil
			}
			match := m.searchMatches[clamp(m.searchResultIndex, 0, len(m.searchMatches)-1)]
			for idx, project := range m.projects {
				if project.ID == match.Project.ID {
					m.selectedProject = idx
					break
				}
			}
			m.pendingFocusTaskID = match.Task.ID
			m.mode = modeNone
			m.status = "jumped to match"
			return m, m.loadData
		default:
			return m, nil
		}
	}

	if m.mode == modeCommandPalette {
		switch {
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			m.mode = modeNone
			m.commandInput.Blur()
			m.status = "cancelled"
			return m, nil
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			cmd := strings.TrimSpace(strings.ToLower(m.commandInput.Value()))
			m.mode = modeNone
			m.commandInput.Blur()
			return m.executeCommandPalette(cmd)
		default:
			var cmd tea.Cmd
			m.commandInput, cmd = m.commandInput.Update(msg)
			return m, cmd
		}
	}

	if m.mode == modeQuickActions {
		switch msg.String() {
		case "esc":
			m.mode = modeNone
			m.status = "cancelled"
			return m, nil
		case "j", "down":
			if m.quickActionIndex < len(quickActionOptions)-1 {
				m.quickActionIndex++
			}
			return m, nil
		case "k", "up":
			if m.quickActionIndex > 0 {
				m.quickActionIndex--
			}
			return m, nil
		case "enter":
			m.mode = modeNone
			return m.applyQuickAction()
		default:
			return m, nil
		}
	}

	if m.mode == modeDuePicker {
		options := m.duePickerOptions()
		switch msg.String() {
		case "esc":
			m.mode = m.pickerBack
			m.pickerBack = modeNone
			m.status = "due picker cancelled"
			return m, m.focusTaskFormField(3)
		case "j", "down":
			if m.duePicker < len(options)-1 {
				m.duePicker++
			}
			return m, nil
		case "k", "up":
			if m.duePicker > 0 {
				m.duePicker--
			}
			return m, nil
		case "enter":
			if len(options) == 0 || len(m.formInputs) <= 3 {
				m.mode = m.pickerBack
				m.pickerBack = modeNone
				return m, m.focusTaskFormField(3)
			}
			choice := options[clamp(m.duePicker, 0, len(options)-1)]
			m.formInputs[3].SetValue(choice.Value)
			m.mode = m.pickerBack
			m.pickerBack = modeNone
			m.status = "due updated"
			return m, m.focusTaskFormField(3)
		default:
			return m, nil
		}
	}

	if m.mode == modeAddTask || m.mode == modeEditTask {
		switch {
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			m.mode = modeNone
			m.formInputs = nil
			m.formFocus = 0
			m.editingTaskID = ""
			m.status = "cancelled"
			return m, nil
		case msg.Code == tea.KeyTab || msg.String() == "tab" || msg.String() == "ctrl+i" || msg.String() == "down":
			return m, m.focusTaskFormField(m.formFocus + 1)
		case msg.String() == "shift+tab" || msg.String() == "backtab" || msg.String() == "up":
			return m, m.focusTaskFormField(m.formFocus - 1)
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			return m.submitInputMode()
		default:
			if m.formFocus == 2 {
				switch msg.String() {
				case "h", "left":
					m.cyclePriority(-1)
					return m, nil
				case "l", "right":
					m.cyclePriority(1)
					return m, nil
				}
				return m, nil
			}
			if m.formFocus == 3 && (msg.String() == "ctrl+d" || msg.String() == "D") {
				m.startDuePicker()
				m.status = "due picker"
				return m, nil
			}
			if len(m.formInputs) == 0 {
				return m, nil
			}
			var cmd tea.Cmd
			m.formInputs[m.formFocus], cmd = m.formInputs[m.formFocus].Update(msg)
			return m, cmd
		}
	}

	if m.mode == modeAddProject || m.mode == modeEditProject {
		switch {
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			m.mode = modeNone
			m.projectFormInputs = nil
			m.projectFormFocus = 0
			m.editingProjectID = ""
			m.status = "cancelled"
			return m, nil
		case msg.Code == tea.KeyTab || msg.String() == "tab" || msg.String() == "ctrl+i" || msg.String() == "down":
			return m, m.focusProjectFormField(m.projectFormFocus + 1)
		case msg.String() == "shift+tab" || msg.String() == "backtab" || msg.String() == "up":
			return m, m.focusProjectFormField(m.projectFormFocus - 1)
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			return m.submitInputMode()
		default:
			if len(m.projectFormInputs) == 0 {
				return m, nil
			}
			var cmd tea.Cmd
			m.projectFormInputs[m.projectFormFocus], cmd = m.projectFormInputs[m.projectFormFocus].Update(msg)
			return m, cmd
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
	switch m.mode {
	case modeAddTask:
		if text := strings.TrimSpace(m.input); text != "" {
			vals := m.taskFormValues()
			if vals["title"] == "" {
				vals["title"] = text
			}
		}
		vals := m.taskFormValues()
		title := vals["title"]
		if title == "" {
			m.mode = modeNone
			m.formInputs = nil
			m.input = ""
			m.status = "title required"
			return m, nil
		}
		priority := domain.Priority(strings.ToLower(vals["priority"]))
		if priority == "" {
			priority = domain.PriorityMedium
		}
		switch priority {
		case domain.PriorityLow, domain.PriorityMedium, domain.PriorityHigh:
		default:
			m.status = "priority must be low|medium|high"
			return m, nil
		}
		dueAt, err := parseDueInput(vals["due"], nil)
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		labels := parseLabelsInput(vals["labels"], nil)

		m.mode = modeNone
		m.formInputs = nil
		return m.createTask(app.CreateTaskInput{
			Title:       title,
			Description: vals["description"],
			Priority:    priority,
			DueAt:       dueAt,
			Labels:      labels,
		})
	case modeSearch:
		text := strings.TrimSpace(m.searchInput.Value())
		states := parseStateFilters(m.searchStateInput.Value(), m.searchStates)
		m.mode = modeNone
		m.searchInput.Blur()
		m.searchStateInput.Blur()
		m.searchQuery = text
		m.searchStates = states
		m.searchApplied = true
		m.selectedTask = 0
		m.status = "search updated"
		if m.searchCrossProject {
			return m, m.loadSearchMatches
		}
		return m, m.loadData
	case modeRenameTask:
		text := strings.TrimSpace(m.input)
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
		vals := m.taskFormValues()
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

		if text := strings.TrimSpace(m.input); text != "" {
			in, err := parseTaskEditInput(text, task)
			if err != nil {
				m.status = "invalid edit format: " + err.Error()
				return m, nil
			}
			m.mode = modeNone
			m.formInputs = nil
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
		}

		title := vals["title"]
		if title == "" {
			title = task.Title
		}
		description := vals["description"]
		if description == "" {
			description = task.Description
		}

		priority := domain.Priority(strings.ToLower(vals["priority"]))
		if priority == "" {
			priority = task.Priority
		}
		switch priority {
		case domain.PriorityLow, domain.PriorityMedium, domain.PriorityHigh:
		default:
			m.status = "priority must be low|medium|high"
			return m, nil
		}

		dueAt, err := parseDueInput(vals["due"], task.DueAt)
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		labels := parseLabelsInput(vals["labels"], task.Labels)

		m.mode = modeNone
		m.formInputs = nil
		m.editingTaskID = ""
		in := app.UpdateTaskInput{
			TaskID:      taskID,
			Title:       title,
			Description: description,
			Priority:    priority,
			DueAt:       dueAt,
			Labels:      labels,
		}
		return m, func() tea.Msg {
			_, updateErr := m.svc.UpdateTask(context.Background(), in)
			if updateErr != nil {
				return actionMsg{err: updateErr}
			}
			return actionMsg{status: "task updated", reload: true}
		}
	case modeAddProject, modeEditProject:
		isAdd := m.mode == modeAddProject
		vals := m.projectFormValues()
		name := vals["name"]
		if name == "" {
			m.status = "project name required"
			return m, nil
		}
		metadata := domain.ProjectMetadata{
			Owner:    vals["owner"],
			Icon:     vals["icon"],
			Color:    vals["color"],
			Homepage: vals["homepage"],
			Tags:     parseLabelsInput(vals["tags"], nil),
		}
		description := vals["description"]
		projectID := m.editingProjectID
		m.mode = modeNone
		m.projectFormInputs = nil
		m.projectFormFocus = 0
		m.editingProjectID = ""
		if isAdd || projectID == "" {
			return m, func() tea.Msg {
				project, err := m.svc.CreateProjectWithMetadata(context.Background(), app.CreateProjectInput{
					Name:        name,
					Description: description,
					Metadata:    metadata,
				})
				if err != nil {
					return actionMsg{err: err}
				}
				return actionMsg{status: "project created", reload: true, projectID: project.ID}
			}
		}
		return m, func() tea.Msg {
			project, err := m.svc.UpdateProject(context.Background(), app.UpdateProjectInput{
				ProjectID:   projectID,
				Name:        name,
				Description: description,
				Metadata:    metadata,
			})
			if err != nil {
				return actionMsg{err: err}
			}
			return actionMsg{status: "project updated", reload: true, projectID: project.ID}
		}
	default:
		return m, nil
	}
}

func (m Model) executeCommandPalette(command string) (tea.Model, tea.Cmd) {
	switch command {
	case "":
		m.status = "no command"
		return m, nil
	case "new-project", "project-new":
		return m, m.startProjectForm(nil)
	case "edit-project", "project-edit":
		if len(m.projects) == 0 {
			m.status = "no project selected"
			return m, nil
		}
		project := m.projects[clamp(m.selectedProject, 0, len(m.projects)-1)]
		return m, m.startProjectForm(&project)
	case "search":
		return m, m.startSearchMode()
	case "search-all":
		m.searchCrossProject = true
		m.status = "search scope set to all projects"
		return m, nil
	case "search-project":
		m.searchCrossProject = false
		m.status = "search scope set to current project"
		return m, nil
	case "clear-search":
		m.searchQuery = ""
		m.searchApplied = false
		m.status = "search cleared"
		return m, m.loadData
	case "toggle-archived":
		m.showArchived = !m.showArchived
		m.selectedTask = 0
		if m.showArchived {
			m.status = "showing archived tasks"
		} else {
			m.status = "hiding archived tasks"
		}
		return m, m.loadData
	case "help":
		m.help.ShowAll = true
		m.status = "help"
		return m, nil
	case "quit", "exit":
		return m, tea.Quit
	default:
		m.status = "unknown command: " + command
		return m, nil
	}
}

func (m Model) applyQuickAction() (tea.Model, tea.Cmd) {
	switch clamp(m.quickActionIndex, 0, len(quickActionOptions)-1) {
	case 0:
		if _, ok := m.selectedTaskInCurrentColumn(); !ok {
			m.status = "no task selected"
			return m, nil
		}
		m.mode = modeTaskInfo
		m.status = "task info"
		return m, nil
	case 1:
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		return m, m.startTaskForm(&task)
	case 2:
		return m.moveSelectedTask(-1)
	case 3:
		return m.moveSelectedTask(1)
	case 4:
		return m.deleteSelectedTask(app.DeleteModeArchive)
	case 5:
		return m.deleteSelectedTask(app.DeleteModeHard)
	default:
		return m, nil
	}
}

func (m Model) createTask(in app.CreateTaskInput) (tea.Model, tea.Cmd) {
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
	in.ProjectID = projectID
	in.ColumnID = columnID
	return m, func() tea.Msg {
		_, err := m.svc.CreateTask(context.Background(), in)
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
	if m.help.ShowAll {
		return m, nil
	}
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
	if m.help.ShowAll {
		return m, nil
	}
	if m.mode == modeProjectPicker {
		overlayTop := m.boardTop()
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
	colWidth := m.columnWidth() + 5 // border + padding approximation for mouse hit testing
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
		tasks := m.currentColumnTasks()
		if len(tasks) > 0 {
			row := relativeY - 2
			m.selectedTask = clamp(m.taskIndexAtRow(tasks, row), 0, len(tasks)-1)
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

func (m *Model) focusTaskByID(taskID string) {
	if strings.TrimSpace(taskID) == "" {
		return
	}
	var targetColIdx = -1
	for idx, column := range m.columns {
		tasks := m.tasksForColumn(column.ID)
		for taskIdx, task := range tasks {
			if task.ID == taskID {
				targetColIdx = idx
				m.selectedColumn = idx
				m.selectedTask = taskIdx
				break
			}
		}
		if targetColIdx >= 0 {
			break
		}
	}
	if targetColIdx >= 0 {
		m.clampSelections()
	}
}

func (m Model) taskByID(taskID string) (domain.Task, bool) {
	for _, task := range m.tasks {
		if task.ID == taskID {
			return task, true
		}
	}
	return domain.Task{}, false
}

func (m Model) renderProjectTabs(accent, dim color.Color) string {
	if len(m.projects) <= 1 {
		return ""
	}
	active := lipgloss.NewStyle().Bold(true).Foreground(accent)
	inactive := lipgloss.NewStyle().Foreground(dim)

	parts := make([]string, 0, len(m.projects))
	for idx, p := range m.projects {
		label := p.Name
		if idx == m.selectedProject {
			parts = append(parts, active.Render("["+label+"]"))
		} else {
			parts = append(parts, inactive.Render(label))
		}
	}
	return strings.Join(parts, "  ")
}

func (m Model) renderOverviewPanel(project domain.Project, accent, muted, dim color.Color) string {
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(accent).Render("Overview"),
		lipgloss.NewStyle().Foreground(muted).Render("project: " + project.Name),
		lipgloss.NewStyle().Foreground(muted).Render(fmt.Sprintf("tasks: %d", len(m.tasks))),
	}

	task, ok := m.selectedTaskInCurrentColumn()
	if ok {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(accent).Render("Selection"))
		lines = append(lines, task.Title)
		if meta := m.cardMeta(task); meta != "" {
			lines = append(lines, lipgloss.NewStyle().Foreground(muted).Render(meta))
		}
		if m.taskFields.ShowDescription {
			desc := strings.TrimSpace(task.Description)
			if desc == "" {
				desc = "-"
			}
			lines = append(lines, lipgloss.NewStyle().Foreground(muted).Render("description: "+desc))
		}
	} else {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(muted).Render("no task selected"))
		lines = append(lines, lipgloss.NewStyle().Foreground(muted).Render("tip: press n to add a task"))
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.HiddenBorder()).
		BorderForeground(dim).
		Padding(0, 2)
	return style.Render(strings.Join(lines, "\n"))
}

func (m Model) renderInfoLine(project domain.Project, muted color.Color) string {
	task, ok := m.selectedTaskInCurrentColumn()
	selected := "none"
	if ok {
		selected = truncate(task.Title, 36)
	}
	return lipgloss.NewStyle().Foreground(muted).Render(
		fmt.Sprintf("project: %s • tasks: %d • selected: %s", project.Name, len(m.tasks), selected),
	)
}

func (m Model) renderHelpOverlay(accent, muted, dim color.Color, _ lipgloss.Style, maxWidth int) string {
	width := clamp(maxWidth, 56, 100)
	if width <= 0 {
		width = 72
	}
	hb := m.help
	hb.ShowAll = true
	hb.SetWidth(width - 4)

	title := lipgloss.NewStyle().Bold(true).Foreground(accent).Render("KAN Help")
	subtitle := lipgloss.NewStyle().Foreground(muted).Render("Fang-style command reference")
	workflow := []string{
		lipgloss.NewStyle().Bold(true).Foreground(accent).Render("Workflows"),
		"1. n add task  •  i/enter view task  •  e edit task",
		"2. [ ] move task across states  •  d/a/D delete modes",
		"3. N new project  •  M edit project  •  p switch project",
		"4. / search (tab fields, ctrl+p scope, ctrl+a archived)",
		"5. : command palette  •  . quick actions",
		"6. task form: h/l priority picker  •  ctrl+d/D due picker",
	}
	lines := []string{
		title,
		subtitle,
		"",
		hb.View(m.keys),
		"",
		lipgloss.NewStyle().Foreground(muted).Render(strings.Join(workflow, "\n")),
		lipgloss.NewStyle().Foreground(muted).Render("press ? or esc to close"),
	}
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(dim).
		Padding(0, 1)
	if maxWidth > 0 {
		style = style.Width(width)
	}
	return style.Render(strings.Join(lines, "\n"))
}

func (m Model) taskListSecondary(task domain.Task) string {
	if m.taskFields.ShowDescription {
		if desc := strings.TrimSpace(task.Description); desc != "" {
			return desc
		}
	}
	if meta := m.cardMeta(task); meta != "" {
		return meta
	}
	return ""
}

func (m Model) taskIndexAtRow(tasks []domain.Task, row int) int {
	if len(tasks) == 0 {
		return 0
	}
	if row <= 0 {
		return 0
	}
	current := 0
	for idx, task := range tasks {
		start := current
		span := 1
		if m.taskListSecondary(task) != "" {
			span++
		}
		if idx < len(tasks)-1 {
			span++
		}
		end := start + span - 1
		if row >= start && row <= end {
			return idx
		}
		current += span
	}
	return len(tasks) - 1
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

func (m Model) renderModeOverlay(accent, muted, dim color.Color, helpStyle lipgloss.Style, maxWidth int) string {
	switch m.mode {
	case modeTaskInfo:
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			return ""
		}
		boxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			boxStyle = boxStyle.Width(clamp(maxWidth, 24, 76))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		due := "-"
		if task.DueAt != nil {
			due = task.DueAt.UTC().Format("2006-01-02")
		}
		labels := "-"
		if len(task.Labels) > 0 {
			labels = strings.Join(task.Labels, ", ")
		}
		lines := []string{
			titleStyle.Render("Task Info"),
			task.Title,
			hintStyle.Render("priority: " + string(task.Priority) + " • due: " + due),
			hintStyle.Render("labels: " + labels),
		}
		if desc := strings.TrimSpace(task.Description); desc != "" {
			lines = append(lines, "", desc)
		}
		lines = append(lines, "", hintStyle.Render("e edit • esc close"))
		return boxStyle.Render(strings.Join(lines, "\n"))

	case modeDuePicker:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 36, 72))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		lines := []string{titleStyle.Render("Due Date")}
		options := m.duePickerOptions()
		for idx, option := range options {
			cursor := "  "
			if idx == m.duePicker {
				cursor = "> "
			}
			lines = append(lines, cursor+option.Label)
		}
		lines = append(lines, hintStyle.Render("j/k choose • enter apply • esc cancel"))
		return style.Render(strings.Join(lines, "\n"))

	case modeProjectPicker:
		pickerStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			pickerStyle = pickerStyle.Width(clamp(maxWidth, 24, 56))
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

	case modeSearchResults:
		resultsStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			resultsStyle = resultsStyle.Width(clamp(maxWidth, 36, 96))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		lines := []string{titleStyle.Render("Search Results")}
		if len(m.searchMatches) == 0 {
			lines = append(lines, hintStyle.Render("(empty)"))
		} else {
			for idx, match := range m.searchMatches {
				cursor := "  "
				if idx == m.searchResultIndex {
					cursor = "> "
				}
				row := fmt.Sprintf("%s%s • %s • %s", cursor, match.Project.Name, match.StateID, truncate(match.Task.Title, 48))
				lines = append(lines, row)
			}
		}
		lines = append(lines, hintStyle.Render("j/k navigate • enter open • esc close"))
		return resultsStyle.Render(strings.Join(lines, "\n"))

	case modeCommandPalette:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 36, 96))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		in := m.commandInput
		in.SetWidth(max(18, maxWidth-20))
		lines := []string{
			titleStyle.Render("Command Palette"),
			in.View(),
			hintStyle.Render("new-project | edit-project | search-all | clear-search | help | quit"),
			hintStyle.Render("enter run • esc cancel"),
		}
		return style.Render(strings.Join(lines, "\n"))

	case modeQuickActions:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 28, 64))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		lines := []string{titleStyle.Render("Quick Actions")}
		for idx, action := range quickActionOptions {
			cursor := "  "
			if idx == m.quickActionIndex {
				cursor = "> "
			}
			lines = append(lines, fmt.Sprintf("%s%s", cursor, action))
		}
		lines = append(lines, hintStyle.Render("j/k navigate • enter run • esc close"))
		return style.Render(strings.Join(lines, "\n"))

	case modeAddTask, modeSearch, modeRenameTask, modeEditTask, modeAddProject, modeEditProject:
		boxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			boxStyle = boxStyle.Width(clamp(maxWidth, 24, 96))
		}

		title := "Input"
		hint := "enter save • esc cancel • tab next field"
		switch m.mode {
		case modeAddTask:
			title = "New Task"
		case modeSearch:
			title = "Search"
			hint = "enter apply • tab next field • ctrl+p all/current • ctrl+a archived"
		case modeRenameTask:
			title = "Rename Task"
		case modeEditTask:
			title = "Edit Task"
			hint = "enter save • esc cancel • tab next field"
		case modeAddProject:
			title = "New Project"
			hint = "enter save • esc cancel • tab next field"
		case modeEditProject:
			title = "Edit Project"
			hint = "enter save • esc cancel • tab next field"
		}

		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		lines := []string{
			titleStyle.Render(title),
		}

		switch m.mode {
		case modeSearch:
			queryInput := m.searchInput
			queryInput.SetWidth(max(18, maxWidth-20))
			stateInput := m.searchStateInput
			stateInput.SetWidth(max(18, maxWidth-20))
			scope := "current project"
			if m.searchCrossProject {
				scope = "all projects"
			}
			labelStyle := lipgloss.NewStyle().Foreground(muted)
			if m.searchFocus == 0 {
				labelStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
			}
			lines = append(lines, labelStyle.Render("query:")+" "+queryInput.View())
			labelStyle = lipgloss.NewStyle().Foreground(muted)
			if m.searchFocus == 1 {
				labelStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
			}
			lines = append(lines, labelStyle.Render("states:")+" "+stateInput.View())
			lines = append(lines, hintStyle.Render("scope: "+scope))
			if m.showArchived {
				lines = append(lines, hintStyle.Render("archived: included"))
			} else {
				lines = append(lines, hintStyle.Render("archived: hidden"))
			}
		case modeAddTask, modeEditTask:
			fieldWidth := max(18, maxWidth-28)
			for i, in := range m.formInputs {
				label := fmt.Sprintf("%d.", i+1)
				if i < len(taskFormFields) {
					label = taskFormFields[i]
				}
				labelStyle := lipgloss.NewStyle().Foreground(muted)
				if i == m.formFocus {
					labelStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
				}
				if i == 2 {
					lines = append(lines, labelStyle.Render(fmt.Sprintf("%-12s", label+":"))+" "+m.renderPriorityPicker(accent, muted))
					continue
				}
				in.SetWidth(fieldWidth)
				lines = append(lines, labelStyle.Render(fmt.Sprintf("%-12s", label+":"))+" "+in.View())
			}
			if m.formFocus == 3 {
				lines = append(lines, hintStyle.Render("ctrl+d or D open due-date picker"))
			}
			if suggestions := m.labelSuggestions(5); len(suggestions) > 0 {
				lines = append(lines, hintStyle.Render("suggested labels: "+strings.Join(suggestions, ", ")))
			}
			if m.mode == modeEditTask {
				lines = append(lines, hintStyle.Render("blank values keep current task value"))
			}
		case modeAddProject, modeEditProject:
			fieldWidth := max(18, maxWidth-28)
			for i, in := range m.projectFormInputs {
				label := fmt.Sprintf("%d.", i+1)
				if i < len(projectFormFields) {
					label = projectFormFields[i]
				}
				labelStyle := lipgloss.NewStyle().Foreground(muted)
				if i == m.projectFormFocus {
					labelStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
				}
				in.SetWidth(fieldWidth)
				lines = append(lines, labelStyle.Render(fmt.Sprintf("%-12s", label+":"))+" "+in.View())
			}
		default:
			lines = append(lines, m.input)
		}

		lines = append(lines, hintStyle.Render(hint))
		return boxStyle.Render(strings.Join(lines, "\n"))
	default:
		return ""
	}
}

func (m Model) renderPriorityPicker(accent, muted color.Color) string {
	parts := make([]string, 0, len(priorityOptions))
	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	baseStyle := lipgloss.NewStyle().Foreground(muted)
	for i, p := range priorityOptions {
		label := string(p)
		if i == m.priorityIdx {
			label = activeStyle.Render("[" + label + "]")
		} else {
			label = baseStyle.Render(label)
		}
		parts = append(parts, label)
	}
	return strings.Join(parts, "  ")
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
	case modeDuePicker:
		return "due-picker"
	case modeProjectPicker:
		return "project-picker"
	case modeTaskInfo:
		return "task-info"
	case modeAddProject:
		return "add-project"
	case modeEditProject:
		return "edit-project"
	case modeSearchResults:
		return "search-results"
	case modeCommandPalette:
		return "command"
	case modeQuickActions:
		return "actions"
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
	case modeDuePicker:
		return "due picker: j/k select, enter apply, esc cancel"
	case modeProjectPicker:
		return "project picker: j/k select, enter choose, esc cancel"
	case modeTaskInfo:
		return "task info: e edit, esc close"
	case modeAddProject:
		return "new project: enter save, esc cancel"
	case modeEditProject:
		return "edit project: enter save, esc cancel"
	case modeSearchResults:
		return "search results: j/k select, enter jump, esc close"
	case modeCommandPalette:
		return "command palette: enter run, esc cancel"
	case modeQuickActions:
		return "quick actions: j/k select, enter run, esc close"
	default:
		return ""
	}
}

func (m Model) columnWidth() int {
	return m.columnWidthFor(m.width)
}

func (m Model) columnWidthFor(boardWidth int) int {
	if len(m.columns) == 0 {
		return 24
	}
	w := 28
	if boardWidth > 0 {
		// Per-column overhead: left/right border (2), horizontal padding (4), margin-right (1)
		const colOverhead = 7
		usable := boardWidth - len(m.columns)*colOverhead
		candidate := usable / len(m.columns)
		if candidate > 0 {
			w = candidate
		}
	}
	if w < 24 {
		return 24
	}
	if w > 42 {
		return 42
	}
	return w
}

func (m Model) columnHeight() int {
	headerLines := 3
	if len(m.projects) > 1 {
		headerLines++
	}
	footerLines := 4
	h := m.height - headerLines - footerLines
	if h < 14 {
		return 14
	}
	return h
}

func (m Model) boardTop() int {
	// mouse coordinates from tea are 1-based
	// header + optional tabs + spacer
	top := 3
	if len(m.projects) > 1 {
		top++
	}
	return top
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func fitLines(content string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	lines := strings.Split(content, "\n")
	switch {
	case len(lines) > maxLines:
		if maxLines == 1 {
			lines = []string{"…"}
		} else {
			lines = append(lines[:maxLines-1], "…")
		}
	case len(lines) < maxLines:
		padding := make([]string, maxLines-len(lines))
		lines = append(lines, padding...)
	}
	return strings.Join(lines, "\n")
}

func overlayOnContent(base, overlay string, width, height int) string {
	if width <= 0 || height <= 0 {
		if strings.TrimSpace(overlay) == "" {
			return base
		}
		return overlay + "\n\n" + base
	}

	base = fitLines(base, height)
	canvas := lipgloss.NewCanvas(width, height)
	baseLayer := lipgloss.NewLayer(base).X(0).Y(0).Z(0)
	centeredOverlay := lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		overlay,
	)
	overlayLayer := lipgloss.NewLayer(centeredOverlay).X(0).Y(0).Z(10)

	canvas.Compose(baseLayer)
	canvas.Compose(overlayLayer)
	return canvas.Render()
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
