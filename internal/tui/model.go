package tui

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"slices"
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

// Service represents service data used by this package.
type Service interface {
	ListProjects(context.Context, bool) ([]domain.Project, error)
	ListColumns(context.Context, string, bool) ([]domain.Column, error)
	ListTasks(context.Context, string, bool) ([]domain.Task, error)
	ListProjectChangeEvents(context.Context, string, int) ([]domain.ChangeEvent, error)
	GetProjectDependencyRollup(context.Context, string) (domain.DependencyRollup, error)
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

// inputMode represents a selectable mode.
type inputMode int

// modeNone and related constants define package defaults.
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
	modeConfirmAction
	modeActivityLog
	modeResourcePicker
	modeLabelPicker
	modePathsRoots
	modeLabelsConfig
)

// taskFormFields stores task-form field keys in display/update order.
var taskFormFields = []string{"title", "description", "priority", "due", "labels", "depends_on", "blocked_by", "blocked_reason"}

// task-form field indexes used throughout keyboard/update logic.
const (
	taskFieldTitle = iota
	taskFieldDescription
	taskFieldPriority
	taskFieldDue
	taskFieldLabels
	taskFieldDependsOn
	taskFieldBlockedBy
	taskFieldBlockedReason
)

// project-form field indexes used for focused form actions.
const (
	projectFieldName = iota
	projectFieldDescription
	projectFieldOwner
	projectFieldIcon
	projectFieldColor
	projectFieldHomepage
	projectFieldTags
	projectFieldRootPath
)

// activity log limits used by modal rendering and retention.
const (
	activityLogMaxItems   = 200
	activityLogViewWindow = 14
)

// priorityOptions stores a package-level helper value.
var priorityOptions = []domain.Priority{
	domain.PriorityLow,
	domain.PriorityMedium,
	domain.PriorityHigh,
}

// duePickerOption defines a functional option for model configuration.
type duePickerOption struct {
	Label string
	Value string
}

// quickActionSpec defines one quick-action command and label.
type quickActionSpec struct {
	ID    string
	Label string
}

// quickActionItem defines one rendered quick-action entry with availability metadata.
type quickActionItem struct {
	ID             string
	Label          string
	Enabled        bool
	DisabledReason string
}

// quickActionSpecs stores the canonical quick-action ordering.
var quickActionSpecs = []quickActionSpec{
	{ID: "task-info", Label: "Task Info"},
	{ID: "edit-task", Label: "Edit Task"},
	{ID: "move-left", Label: "Move Left"},
	{ID: "move-right", Label: "Move Right"},
	{ID: "archive-task", Label: "Archive Task"},
	{ID: "hard-delete", Label: "Hard Delete"},
	{ID: "toggle-selection", Label: "Toggle Selection"},
	{ID: "clear-selection", Label: "Clear Selection"},
	{ID: "bulk-move-left", Label: "Bulk Move Left"},
	{ID: "bulk-move-right", Label: "Bulk Move Right"},
	{ID: "bulk-archive", Label: "Bulk Archive"},
	{ID: "bulk-hard-delete", Label: "Bulk Hard Delete"},
	{ID: "undo", Label: "Undo"},
	{ID: "redo", Label: "Redo"},
	{ID: "activity-log", Label: "Activity Log"},
}

// canonicalSearchStates stores canonical searchable lifecycle states.
var canonicalSearchStatesOrdered = []string{"todo", "progress", "done"}

// canonicalSearchStateLabels stores display labels for canonical lifecycle states.
var canonicalSearchStateLabels = map[string]string{
	"todo":     "To Do",
	"progress": "In Progress",
	"done":     "Done",
}

// commandPaletteItem describes one command-palette command.
type commandPaletteItem struct {
	Command     string
	Aliases     []string
	Description string
}

// resourcePickerEntry describes one filesystem candidate in the resource picker.
type resourcePickerEntry struct {
	Name  string
	Path  string
	IsDir bool
}

// labelPickerItem describes one inherited label suggestion and its source.
type labelPickerItem struct {
	Label  string
	Source string
}

// labelInheritanceSources groups inherited labels by source precedence.
type labelInheritanceSources struct {
	Global  []string
	Project []string
	Phase   []string
}

// confirmAction describes a pending confirmation action.
type confirmAction struct {
	Kind    string
	Task    domain.Task
	TaskIDs []string
	Mode    app.DeleteMode
	Label   string
}

// activityEntry describes one recorded user action for the in-app activity log.
type activityEntry struct {
	At      time.Time
	Summary string
	Target  string
}

// historyStepKind identifies one reversible operation in a mutation set.
type historyStepKind string

// history step kinds used for undo/redo.
const (
	historyStepMove       historyStepKind = "move"
	historyStepArchive    historyStepKind = "archive"
	historyStepRestore    historyStepKind = "restore"
	historyStepHardDelete historyStepKind = "hard-delete"
)

// historyStep describes one mutation required to replay or reverse a change.
type historyStep struct {
	Kind         historyStepKind
	TaskID       string
	FromColumnID string
	FromPosition int
	ToColumnID   string
	ToPosition   int
}

// historyActionSet describes one logical user mutation for undo/redo.
type historyActionSet struct {
	ID       int
	Label    string
	Summary  string
	Target   string
	Steps    []historyStep
	Undoable bool
	At       time.Time
}

// Model represents model data used by this package.
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

	searchInput                 textinput.Model
	commandInput                textinput.Model
	pathsRootInput              textinput.Model
	searchFocus                 int
	searchStateCursor           int
	searchCrossProject          bool
	searchDefaultCrossProject   bool
	searchDefaultIncludeArchive bool
	searchStates                []string
	searchDefaultStates         []string
	searchMatches               []app.TaskMatch
	searchResultIndex           int
	quickActionIndex            int
	commandMatches              []commandPaletteItem
	commandIndex                int

	formInputs  []textinput.Model
	formFocus   int
	priorityIdx int
	duePicker   int
	pickerBack  inputMode
	// taskFormResourceRefs stages resource refs while creating or editing a task.
	taskFormResourceRefs []domain.ResourceRef

	projectPickerIndex int
	projectFormInputs  []textinput.Model
	projectFormFocus   int
	labelsConfigInputs []textinput.Model
	labelsConfigFocus  int
	labelsConfigSlug   string
	editingProjectID   string
	editingTaskID      string
	taskInfoTaskID     string
	taskInfoSubtaskIdx int
	taskFormParentID   string
	taskFormKind       domain.WorkKind
	pendingProjectID   string
	pendingFocusTaskID string

	lastArchivedTaskID string

	confirmDelete     bool
	confirmArchive    bool
	confirmHardDelete bool
	confirmRestore    bool
	pendingConfirm    confirmAction
	confirmChoice     int

	boardGroupBy    string
	showWIPWarnings bool
	dueSoonWindows  []time.Duration
	showDueSummary  bool
	projectRoots    map[string]string
	defaultRootDir  string

	projectionRootTaskID string

	selectedTaskIDs  map[string]struct{}
	activityLog      []activityEntry
	undoStack        []historyActionSet
	redoStack        []historyActionSet
	nextHistoryID    int
	dependencyRollup domain.DependencyRollup

	resourcePickerBack   inputMode
	resourcePickerTaskID string
	resourcePickerRoot   string
	resourcePickerDir    string
	resourcePickerIndex  int
	resourcePickerItems  []resourcePickerEntry
	resourcePickerFilter textinput.Model

	labelPickerBack  inputMode
	labelPickerIndex int
	labelPickerItems []labelPickerItem

	allowedLabelGlobal   []string
	allowedLabelProject  map[string][]string
	enforceAllowedLabels bool

	reloadConfig    ReloadConfigFunc
	saveProjectRoot SaveProjectRootFunc
	saveLabels      SaveLabelsConfigFunc
}

// loadedMsg carries message data through update handling.
type loadedMsg struct {
	projects        []domain.Project
	selectedProject int
	columns         []domain.Column
	tasks           []domain.Task
	rollup          domain.DependencyRollup
	err             error
}

// resourcePickerLoadedMsg carries resource picker directory entries.
type resourcePickerLoadedMsg struct {
	root    string
	current string
	entries []resourcePickerEntry
	err     error
}

// actionMsg carries message data through update handling.
type actionMsg struct {
	err          error
	status       string
	reload       bool
	projectID    string
	focusTaskID  string
	clearSelect  bool
	clearTaskIDs []string
	historyPush  *historyActionSet
	historyUndo  *historyActionSet
	historyRedo  *historyActionSet
	activityItem *activityEntry
}

// searchResultsMsg carries message data through update handling.
type searchResultsMsg struct {
	matches []app.TaskMatch
	err     error
}

// activityLogLoadedMsg carries persisted activity entries for the active project.
type activityLogLoadedMsg struct {
	entries []activityEntry
	err     error
}

// configReloadedMsg carries runtime settings loaded through the reload callback.
type configReloadedMsg struct {
	config RuntimeConfig
	err    error
}

// projectRootSavedMsg carries one persisted project-root mapping update.
type projectRootSavedMsg struct {
	projectSlug string
	rootPath    string
	err         error
}

// NewModel constructs a new value for this package.
func NewModel(svc Service, opts ...Option) Model {
	h := help.New()
	h.ShowAll = false
	searchInput := textinput.New()
	searchInput.Prompt = ""
	searchInput.Placeholder = "title, description, labels"
	searchInput.CharLimit = 120
	commandInput := textinput.New()
	commandInput.Prompt = ": "
	commandInput.Placeholder = "type to filter commands"
	commandInput.CharLimit = 120
	pathsRootInput := textinput.New()
	pathsRootInput.Prompt = "root: "
	pathsRootInput.Placeholder = "absolute path (empty clears mapping)"
	pathsRootInput.CharLimit = 512
	resourcePickerFilter := textinput.New()
	resourcePickerFilter.Prompt = "filter: "
	resourcePickerFilter.Placeholder = "type to fuzzy-filter files/dirs"
	resourcePickerFilter.CharLimit = 120
	m := Model{
		svc:                  svc,
		status:               "loading...",
		help:                 h,
		keys:                 newKeyMap(),
		taskFields:           DefaultTaskFieldConfig(),
		defaultDeleteMode:    app.DeleteModeArchive,
		searchInput:          searchInput,
		commandInput:         commandInput,
		pathsRootInput:       pathsRootInput,
		resourcePickerFilter: resourcePickerFilter,
		searchStates:         []string{"todo", "progress", "done"},
		searchDefaultStates:  []string{"todo", "progress", "done"},
		boardGroupBy:         "none",
		showWIPWarnings:      true,
		dueSoonWindows:       []time.Duration{24 * time.Hour, time.Hour},
		showDueSummary:       true,
		selectedTaskIDs:      map[string]struct{}{},
		activityLog:          []activityEntry{},
		confirmDelete:        true,
		confirmArchive:       true,
		confirmHardDelete:    true,
		confirmRestore:       false,
		taskFormKind:         domain.WorkKindTask,
		allowedLabelProject:  map[string][]string{},
		projectRoots:         map[string]string{},
	}
	if cwd, err := os.Getwd(); err == nil {
		m.defaultRootDir = cwd
	} else {
		m.defaultRootDir = "."
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&m)
		}
	}
	return m
}

// Init handles init.
func (m Model) Init() tea.Cmd {
	return m.loadData
}

// Update updates state for the requested operation.
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
		m.dependencyRollup = msg.rollup
		if len(m.projects) == 0 {
			m.selectedProject = 0
			m.selectedColumn = 0
			m.selectedTask = 0
			m.columns = nil
			m.tasks = nil
			if m.mode == modeNone {
				m.status = "create your first project"
				return m, m.startProjectForm(nil)
			}
			return m, nil
		}
		if m.pendingProjectID != "" {
			for idx, project := range m.projects {
				if project.ID == m.pendingProjectID {
					m.selectedProject = idx
					break
				}
			}
			m.pendingProjectID = ""
		}
		if m.projectionRootTaskID != "" {
			if _, ok := m.taskByID(m.projectionRootTaskID); !ok {
				m.projectionRootTaskID = ""
				m.status = "focus cleared (parent not found)"
			}
		}
		m.clampSelections()
		m.retainSelectionForLoadedTasks()
		if m.pendingFocusTaskID != "" {
			m.focusTaskByID(m.pendingFocusTaskID)
			m.pendingFocusTaskID = ""
		}
		if m.status == "" || m.status == "loading..." {
			m.status = "ready"
		}
		return m, nil

	case resourcePickerLoadedMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
			return m, nil
		}
		m.resourcePickerRoot = msg.root
		m.resourcePickerDir = msg.current
		m.resourcePickerItems = msg.entries
		m.resourcePickerIndex = 0
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
		if msg.focusTaskID != "" {
			m.pendingFocusTaskID = msg.focusTaskID
		}
		if msg.clearSelect {
			m.clearSelection()
		}
		if len(msg.clearTaskIDs) > 0 {
			m.unselectTasks(msg.clearTaskIDs)
		}
		if msg.historyPush != nil {
			m.pushUndoHistory(*msg.historyPush)
		}
		if msg.historyUndo != nil {
			m.applyUndoTransition(*msg.historyUndo)
		}
		if msg.historyRedo != nil {
			m.applyRedoTransition(*msg.historyRedo)
		}
		if msg.activityItem != nil {
			m.appendActivity(*msg.activityItem)
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

	case activityLogLoadedMsg:
		if msg.err != nil {
			// Keep the app usable when persisted activity fetch fails; fall back to current in-memory log.
			if m.mode == modeActivityLog {
				m.status = "activity log unavailable: " + msg.err.Error()
			}
			return m, nil
		}
		m.activityLog = append([]activityEntry(nil), msg.entries...)
		if m.mode == modeActivityLog {
			m.status = "activity log"
		}
		return m, nil

	case configReloadedMsg:
		if msg.err != nil {
			m.status = "reload config failed: " + msg.err.Error()
			return m, nil
		}
		m.applyRuntimeConfig(msg.config)
		m.status = "config reloaded"
		return m, m.loadData

	case projectRootSavedMsg:
		if msg.err != nil {
			m.status = "save root failed: " + msg.err.Error()
			return m, nil
		}
		if m.projectRoots == nil {
			m.projectRoots = map[string]string{}
		}
		if msg.rootPath == "" {
			delete(m.projectRoots, msg.projectSlug)
			m.status = "project root cleared"
			return m, nil
		}
		m.projectRoots[msg.projectSlug] = msg.rootPath
		m.status = "project root saved"
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

// View handles view.
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
		accent := lipgloss.Color("62")
		muted := lipgloss.Color("241")
		dim := lipgloss.Color("239")
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
		helpStyle := lipgloss.NewStyle().Foreground(muted)
		statusStyle := lipgloss.NewStyle().Foreground(dim)
		sections := []string{
			titleStyle.Render("kan"),
			"",
			"No projects yet.",
			"Press N to create your first project.",
			"Press q to quit.",
		}
		if strings.TrimSpace(m.status) != "" && m.status != "ready" {
			sections = append(sections, "", statusStyle.Render(m.status))
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
		if overlay := m.renderModeOverlay(accent, muted, dim, helpStyle, m.width-8); overlay != "" {
			overlayHeight := lipgloss.Height(fullContent)
			if m.height > 0 {
				overlayHeight = m.height
			}
			fullContent = overlayOnContent(fullContent, overlay, max(1, m.width), max(1, overlayHeight))
		}
		v := tea.NewView(fullContent)
		v.MouseMode = tea.MouseModeCellMotion
		v.AltScreen = true
		return v
	}

	project := m.projects[clamp(m.selectedProject, 0, len(m.projects)-1)]
	accent := projectAccentColor(project)
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
	if m.boardGroupBy != "none" {
		header += statusStyle.Render("  grouped: " + m.boardGroupBy)
	}
	if breadcrumb := m.projectionBreadcrumb(); breadcrumb != "" {
		header += statusStyle.Render("  focus: " + truncate(breadcrumb, 48))
	}
	if count := len(m.selectedTaskIDs); count > 0 {
		header += statusStyle.Render(fmt.Sprintf("  selected: %d", count))
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
		Width(colWidth)
	selColStyle := baseColStyle.Copy().BorderForeground(accent)
	normColStyle := baseColStyle.Copy()
	colTitle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	archivedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	selectedTaskStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	selectedMultiTaskStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true).Underline(true)
	multiSelectedTaskStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("237")).Bold(true)
	itemSubStyle := lipgloss.NewStyle().Foreground(muted)
	groupStyle := lipgloss.NewStyle().Bold(true).Foreground(muted)
	warningStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203"))

	for colIdx, column := range m.columns {
		colTasks := m.boardTasksForColumn(column.ID)
		parentByID := map[string]string{}
		for _, task := range colTasks {
			parentByID[task.ID] = task.ParentID
		}
		activeCount := 0
		for _, task := range colTasks {
			if task.ArchivedAt == nil {
				activeCount++
			}
		}

		colHeader := fmt.Sprintf("%s (%d)", column.Name, len(colTasks))
		if column.WIPLimit > 0 {
			colHeader = fmt.Sprintf("%s (%d/%d)", column.Name, activeCount, column.WIPLimit)
		}
		headerLines := []string{colTitle.Render(colHeader)}
		if m.showWIPWarnings && column.WIPLimit > 0 && activeCount > column.WIPLimit {
			headerLines = append(headerLines, warningStyle.Render(fmt.Sprintf("WIP limit exceeded: %d/%d", activeCount, column.WIPLimit)))
		}

		taskLines := make([]string, 0, max(1, len(colTasks)*3))
		selectedStart := -1
		selectedEnd := -1

		if len(colTasks) == 0 {
			taskLines = append(taskLines, archivedStyle.Render("(empty)"))
		} else {
			prevGroup := ""
			for taskIdx, task := range colTasks {
				if m.boardGroupBy != "none" {
					groupLabel := m.groupLabelForTask(task)
					if taskIdx == 0 || groupLabel != prevGroup {
						if taskIdx > 0 {
							taskLines = append(taskLines, "")
						}
						taskLines = append(taskLines, groupStyle.Render(groupLabel))
						prevGroup = groupLabel
					}
				}
				selected := colIdx == m.selectedColumn && taskIdx == m.selectedTask
				multiSelected := m.isTaskSelected(task.ID)

				prefix := "   "
				switch {
				case selected && multiSelected:
					prefix = "│* "
				case selected:
					prefix = "│  "
				case multiSelected:
					prefix = " * "
				}
				depth := taskDepth(task.ID, parentByID, 0)
				indent := strings.Repeat("  ", min(depth, 4))
				title := prefix + indent + truncate(task.Title, max(1, colWidth-(10+2*min(depth, 4))))
				sub := m.taskListSecondary(task)
				if sub != "" {
					sub = indent + truncate(sub, max(1, colWidth-(10+2*min(depth, 4))))
				}
				if task.ArchivedAt != nil {
					title = archivedStyle.Render(title)
					if sub != "" {
						sub = archivedStyle.Render(sub)
					}
				} else {
					switch {
					case selected && multiSelected:
						title = selectedMultiTaskStyle.Render(title)
					case selected:
						title = selectedTaskStyle.Render(title)
					case multiSelected:
						title = multiSelectedTaskStyle.Render(title)
					}
				}

				rowStart := len(taskLines)
				taskLines = append(taskLines, title)
				if sub != "" {
					subPrefix := "   "
					switch {
					case selected && multiSelected:
						subPrefix = "│* "
					case selected:
						subPrefix = "│  "
					case multiSelected:
						subPrefix = " * "
					}
					taskLines = append(taskLines, subPrefix+itemSubStyle.Render(sub))
				}
				if taskIdx < len(colTasks)-1 {
					taskLines = append(taskLines, "")
				}
				if selected {
					selectedStart = rowStart
					selectedEnd = len(taskLines) - 1
				}
			}
		}

		innerHeight := max(1, colHeight-4)
		taskWindowHeight := max(1, innerHeight-len(headerLines))
		scrollTop := 0
		if colIdx == m.selectedColumn && selectedStart >= 0 {
			if selectedEnd >= scrollTop+taskWindowHeight {
				scrollTop = selectedEnd - taskWindowHeight + 1
			}
			if selectedStart < scrollTop {
				scrollTop = selectedStart
			}
		}
		maxScrollTop := max(0, len(taskLines)-taskWindowHeight)
		scrollTop = clamp(scrollTop, 0, maxScrollTop)
		if len(taskLines) > taskWindowHeight {
			taskLines = taskLines[scrollTop : scrollTop+taskWindowHeight]
		}
		if len(taskLines) < taskWindowHeight {
			taskLines = append(taskLines, make([]string, taskWindowHeight-len(taskLines))...)
		}

		lines := append(append([]string{}, headerLines...), taskLines...)
		content := fitLines(strings.Join(lines, "\n"), innerHeight)
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
	sections = append(sections, statusStyle.Render(m.dependencyRollupSummary()))
	if strings.TrimSpace(m.projectionRootTaskID) != "" {
		sections = append(sections, statusStyle.Render(fmt.Sprintf("subtree focus active • %s full board", m.keys.clearFocus.Help().Key)))
	}
	if count := len(m.selectedTaskIDs); count > 0 {
		sections = append(sections, statusStyle.Render(fmt.Sprintf("%d tasks selected • %s toggle • esc clear", count, m.keys.multiSelect.Help().Key)))
	}
	if m.showDueSummary {
		overdue, dueSoon := m.dueCounts(time.Now().UTC())
		sections = append(sections, statusStyle.Render(fmt.Sprintf("%d overdue * %d due soon", overdue, dueSoon)))
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

// loadData loads required data for the current operation.
func (m Model) loadData() tea.Msg {
	projects, err := m.svc.ListProjects(context.Background(), false)
	if err != nil {
		return loadedMsg{err: err}
	}
	if len(projects) == 0 {
		return loadedMsg{projects: projects}
	}

	projectIdx := clamp(m.selectedProject, 0, len(projects)-1)
	if pendingProjectID := strings.TrimSpace(m.pendingProjectID); pendingProjectID != "" {
		for idx, project := range projects {
			if project.ID == pendingProjectID {
				projectIdx = idx
				break
			}
		}
	}
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
	rollup, err := m.svc.GetProjectDependencyRollup(context.Background(), projectID)
	if err != nil {
		return loadedMsg{err: err}
	}

	return loadedMsg{
		projects:        projects,
		selectedProject: projectIdx,
		columns:         columns,
		tasks:           tasks,
		rollup:          rollup,
	}
}

// loadSearchMatches loads required data for the current operation.
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

// loadActivityLog loads persisted project activity entries for modal rendering.
func (m Model) loadActivityLog() tea.Msg {
	projectID, ok := m.currentProjectID()
	if !ok {
		return activityLogLoadedMsg{entries: nil}
	}
	events, err := m.svc.ListProjectChangeEvents(context.Background(), projectID, activityLogMaxItems)
	if err != nil {
		return activityLogLoadedMsg{err: err}
	}
	return activityLogLoadedMsg{entries: mapChangeEventsToActivityEntries(events)}
}

// openActivityLog enters activity-log mode and triggers persisted activity fetch.
func (m *Model) openActivityLog() tea.Cmd {
	m.mode = modeActivityLog
	m.status = "activity log"
	return m.loadActivityLog
}

// mapChangeEventsToActivityEntries converts newest-first persisted events into modal rows.
func mapChangeEventsToActivityEntries(events []domain.ChangeEvent) []activityEntry {
	if len(events) == 0 {
		return []activityEntry{}
	}
	entries := make([]activityEntry, 0, len(events))
	// Repository events are newest-first; modal rendering expects chronological order.
	for idx := len(events) - 1; idx >= 0; idx-- {
		entries = append(entries, mapChangeEventToActivityEntry(events[idx]))
	}
	if len(entries) > activityLogMaxItems {
		entries = append([]activityEntry(nil), entries[len(entries)-activityLogMaxItems:]...)
	}
	return entries
}

// mapChangeEventToActivityEntry derives a compact activity row from one persisted event.
func mapChangeEventToActivityEntry(event domain.ChangeEvent) activityEntry {
	summary := "update task"
	switch event.Operation {
	case domain.ChangeOperationCreate:
		summary = "create task"
	case domain.ChangeOperationUpdate:
		summary = "update task"
	case domain.ChangeOperationMove:
		summary = "move task"
	case domain.ChangeOperationArchive:
		summary = "archive task"
	case domain.ChangeOperationRestore:
		summary = "restore task"
	case domain.ChangeOperationDelete:
		summary = "delete task"
	}
	target := strings.TrimSpace(event.Metadata["title"])
	if target == "" {
		target = strings.TrimSpace(event.WorkItemID)
	}
	if target == "" {
		target = "-"
	}
	return activityEntry{
		At:      event.OccurredAt.UTC(),
		Summary: summary,
		Target:  target,
	}
}

// newModalInput constructs modal input.
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

// startSearchMode starts search mode.
func (m *Model) startSearchMode() tea.Cmd {
	m.mode = modeSearch
	m.input = ""
	m.searchStates = canonicalSearchStates(m.searchStates)
	m.searchInput.SetValue(m.searchQuery)
	m.searchInput.CursorEnd()
	m.searchFocus = 0
	m.searchStateCursor = 0
	m.status = "search"
	return m.searchInput.Focus()
}

// startCommandPalette starts command palette.
func (m *Model) startCommandPalette() tea.Cmd {
	m.mode = modeCommandPalette
	m.commandInput.SetValue("")
	m.commandInput.CursorEnd()
	m.commandMatches = m.filteredCommandItems("")
	m.commandIndex = 0
	m.status = "command palette"
	return m.commandInput.Focus()
}

// startPathsRootsMode opens the modal used to edit one current-project root mapping.
func (m *Model) startPathsRootsMode() tea.Cmd {
	project, ok := m.currentProject()
	if !ok {
		m.status = "no project selected"
		return nil
	}
	slug := strings.TrimSpace(strings.ToLower(project.Slug))
	if slug == "" {
		m.status = "project slug is empty"
		return nil
	}
	m.mode = modePathsRoots
	m.pathsRootInput.SetValue(strings.TrimSpace(m.projectRoots[slug]))
	m.pathsRootInput.CursorEnd()
	m.status = "paths/roots"
	return m.pathsRootInput.Focus()
}

// startQuickActions starts quick actions.
func (m *Model) startQuickActions() tea.Cmd {
	m.mode = modeQuickActions
	actions := m.quickActions()
	m.quickActionIndex = 0
	for idx, action := range actions {
		if action.Enabled {
			m.quickActionIndex = idx
			break
		}
	}
	m.status = "quick actions"
	return nil
}

// startProjectForm starts project form.
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
		newModalInput("", "project root path (optional)", "", 512),
	}
	m.editingProjectID = ""
	if project != nil {
		m.mode = modeEditProject
		m.status = "edit project"
		m.editingProjectID = project.ID
		m.projectFormInputs[projectFieldName].SetValue(project.Name)
		m.projectFormInputs[projectFieldDescription].SetValue(project.Description)
		m.projectFormInputs[projectFieldOwner].SetValue(project.Metadata.Owner)
		m.projectFormInputs[projectFieldIcon].SetValue(project.Metadata.Icon)
		m.projectFormInputs[projectFieldColor].SetValue(project.Metadata.Color)
		m.projectFormInputs[projectFieldHomepage].SetValue(project.Metadata.Homepage)
		if len(project.Metadata.Tags) > 0 {
			m.projectFormInputs[projectFieldTags].SetValue(strings.Join(project.Metadata.Tags, ","))
		}
		if slug := strings.TrimSpace(strings.ToLower(project.Slug)); slug != "" {
			m.projectFormInputs[projectFieldRootPath].SetValue(strings.TrimSpace(m.projectRoots[slug]))
		}
	} else {
		m.mode = modeAddProject
		m.status = "new project"
	}
	return m.focusProjectFormField(0)
}

// startTaskForm starts task form.
func (m *Model) startTaskForm(task *domain.Task) tea.Cmd {
	m.formFocus = 0
	m.priorityIdx = 1
	m.duePicker = 0
	m.pickerBack = modeNone
	m.input = ""
	m.taskFormParentID = ""
	m.taskFormKind = domain.WorkKindTask
	m.taskFormResourceRefs = nil
	m.formInputs = []textinput.Model{
		newModalInput("", "task title (required)", "", 120),
		newModalInput("", "short description", "", 240),
		newModalInput("", "low | medium | high", "", 16),
		newModalInput("", "YYYY-MM-DD[THH:MM] or -", "", 32),
		newModalInput("", "csv labels", "", 160),
		newModalInput("", "csv task ids", "", 240),
		newModalInput("", "csv task ids", "", 240),
		newModalInput("", "why blocked? (optional)", "", 240),
	}
	labelsIdx := taskFieldLabels
	m.formInputs[labelsIdx].ShowSuggestions = true
	m.formInputs[taskFieldPriority].SetValue(string(priorityOptions[m.priorityIdx]))
	if task != nil {
		m.taskFormParentID = task.ParentID
		m.taskFormKind = task.Kind
		m.formInputs[taskFieldTitle].SetValue(task.Title)
		m.formInputs[taskFieldDescription].SetValue(task.Description)
		m.priorityIdx = priorityIndex(task.Priority)
		m.formInputs[taskFieldPriority].SetValue(string(priorityOptions[m.priorityIdx]))
		if task.DueAt != nil {
			m.formInputs[taskFieldDue].SetValue(formatDueValue(task.DueAt))
		}
		if len(task.Labels) > 0 {
			m.formInputs[taskFieldLabels].SetValue(strings.Join(task.Labels, ","))
		}
		if len(task.Metadata.DependsOn) > 0 {
			m.formInputs[taskFieldDependsOn].SetValue(strings.Join(task.Metadata.DependsOn, ","))
		}
		if len(task.Metadata.BlockedBy) > 0 {
			m.formInputs[taskFieldBlockedBy].SetValue(strings.Join(task.Metadata.BlockedBy, ","))
		}
		if blockedReason := strings.TrimSpace(task.Metadata.BlockedReason); blockedReason != "" {
			m.formInputs[taskFieldBlockedReason].SetValue(blockedReason)
		}
		m.taskFormResourceRefs = append([]domain.ResourceRef(nil), task.Metadata.ResourceRefs...)
		m.mode = modeEditTask
		m.editingTaskID = task.ID
		m.status = "edit task"
	} else {
		m.formInputs[taskFieldPriority].Placeholder = "medium"
		m.formInputs[taskFieldDue].Placeholder = "-"
		m.formInputs[taskFieldLabels].Placeholder = "-"
		m.mode = modeAddTask
		m.editingTaskID = ""
		m.status = "new task"
	}
	m.refreshTaskFormLabelSuggestions()
	return m.focusTaskFormField(0)
}

// startSubtaskForm opens the task form preconfigured for a child item.
func (m *Model) startSubtaskForm(parent domain.Task) tea.Cmd {
	cmd := m.startTaskForm(nil)
	m.taskFormParentID = parent.ID
	m.taskFormKind = domain.WorkKindSubtask
	m.refreshTaskFormLabelSuggestions()
	m.status = "new subtask for " + parent.Title
	return cmd
}

// focusTaskFormField focuses task form field.
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

// focusProjectFormField focuses project form field.
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

// startLabelsConfigForm opens a modal for editing global + current-project label defaults.
func (m *Model) startLabelsConfigForm() tea.Cmd {
	project, ok := m.currentProject()
	if !ok {
		m.status = "no project selected"
		return nil
	}
	slug := strings.TrimSpace(strings.ToLower(project.Slug))
	if slug == "" {
		m.status = "project slug is empty"
		return nil
	}
	m.labelsConfigSlug = slug
	m.labelsConfigFocus = 0
	m.labelsConfigInputs = []textinput.Model{
		newModalInput("", "global labels csv", "", 240),
		newModalInput("", "project labels csv", "", 240),
	}
	if len(m.allowedLabelGlobal) > 0 {
		m.labelsConfigInputs[0].SetValue(strings.Join(m.allowedLabelGlobal, ","))
	}
	if labels := m.allowedLabelProject[slug]; len(labels) > 0 {
		m.labelsConfigInputs[1].SetValue(strings.Join(labels, ","))
	}
	m.mode = modeLabelsConfig
	m.status = "edit labels config"
	return m.focusLabelsConfigField(0)
}

// focusLabelsConfigField focuses one labels-config input.
func (m *Model) focusLabelsConfigField(idx int) tea.Cmd {
	if len(m.labelsConfigInputs) == 0 {
		return nil
	}
	idx = clamp(idx, 0, len(m.labelsConfigInputs)-1)
	m.labelsConfigFocus = idx
	for i := range m.labelsConfigInputs {
		m.labelsConfigInputs[i].Blur()
	}
	return m.labelsConfigInputs[idx].Focus()
}

// taskFormValues returns task form values.
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

// allowedLabelsForSelectedProject returns merged global + project-scoped allowed labels.
func (m Model) allowedLabelsForSelectedProject() []string {
	out := make([]string, 0)
	seen := map[string]struct{}{}
	appendUnique := func(labels []string) {
		for _, raw := range labels {
			label := strings.TrimSpace(strings.ToLower(raw))
			if label == "" {
				continue
			}
			if _, ok := seen[label]; ok {
				continue
			}
			seen[label] = struct{}{}
			out = append(out, label)
		}
	}
	appendUnique(m.allowedLabelGlobal)
	if project, ok := m.currentProject(); ok {
		appendUnique(m.allowedLabelProject[strings.TrimSpace(strings.ToLower(project.Slug))])
	}
	sort.Strings(out)
	return out
}

// projectFormFields stores a package-level helper value.
var projectFormFields = []string{"name", "description", "owner", "icon", "color", "homepage", "tags", "root_path"}

// projectFormValues returns project form values.
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

// parseDueInput parses input into a normalized form.
func parseDueInput(raw string, current *time.Time) (*time.Time, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return current, nil
	}
	if text == "-" {
		return nil, nil
	}
	layouts := []string{
		"2006-01-02",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
		time.RFC3339,
	}
	var parsed time.Time
	var err error
	for _, layout := range layouts {
		parsed, err = time.Parse(layout, text)
		if err == nil {
			ts := parsed.UTC()
			return &ts, nil
		}
	}
	return nil, fmt.Errorf("due date must be YYYY-MM-DD, YYYY-MM-DDTHH:MM, RFC3339, or -")
}

// dueWarning returns a warning message for due input values.
func dueWarning(raw string, now time.Time) string {
	parsed, err := parseDueInput(raw, nil)
	if err != nil || parsed == nil {
		return ""
	}
	if parsed.Before(now.UTC()) {
		return "warning: due datetime is in the past"
	}
	return ""
}

// formatDueValue formats due datetime values for compact UI display and editing.
func formatDueValue(dueAt *time.Time) string {
	if dueAt == nil {
		return "-"
	}
	due := dueAt.UTC()
	if due.Hour() == 0 && due.Minute() == 0 {
		return due.Format("2006-01-02")
	}
	return due.Format("2006-01-02 15:04")
}

// parseLabelsInput parses input into a normalized form.
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

// parseTaskRefIDsInput parses dependency reference ids from comma-separated task-id input.
func parseTaskRefIDsInput(raw string, current []string) []string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return append([]string(nil), current...)
	}
	if text == "-" {
		return nil
	}
	parts := strings.Split(text, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		id := strings.TrimSpace(part)
		if id == "" {
			continue
		}
		key := strings.ToLower(id)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, id)
	}
	return out
}

// buildTaskMetadataFromForm overlays dependency/resource task metadata fields from form values.
func (m Model) buildTaskMetadataFromForm(vals map[string]string, current domain.TaskMetadata) domain.TaskMetadata {
	meta := current
	meta.DependsOn = parseTaskRefIDsInput(vals["depends_on"], current.DependsOn)
	meta.BlockedBy = parseTaskRefIDsInput(vals["blocked_by"], current.BlockedBy)
	blockedReason := strings.TrimSpace(vals["blocked_reason"])
	switch blockedReason {
	case "":
		// Keep current metadata when field is untouched.
	case "-":
		meta.BlockedReason = ""
	default:
		meta.BlockedReason = blockedReason
	}
	meta.ResourceRefs = append([]domain.ResourceRef(nil), m.taskFormResourceRefs...)
	return meta
}

// validateAllowedLabels enforces label allowlists when configured.
func (m Model) validateAllowedLabels(labels []string) error {
	if !m.enforceAllowedLabels || len(labels) == 0 {
		return nil
	}
	allowed := m.allowedLabelsForSelectedProject()
	if len(allowed) == 0 {
		return fmt.Errorf("no labels configured for current project; disable labels.enforce_allowed to allow free-form labels")
	}
	allowedSet := map[string]struct{}{}
	for _, label := range allowed {
		allowedSet[strings.TrimSpace(strings.ToLower(label))] = struct{}{}
	}
	disallowed := make([]string, 0)
	for _, raw := range labels {
		label := strings.TrimSpace(strings.ToLower(raw))
		if label == "" {
			continue
		}
		if _, ok := allowedSet[label]; ok {
			continue
		}
		disallowed = append(disallowed, label)
	}
	if len(disallowed) == 0 {
		return nil
	}
	sort.Strings(disallowed)
	return fmt.Errorf("labels not allowed: %s", strings.Join(disallowed, ", "))
}

// canonicalSearchStates normalizes configured and user-selected search states.
func canonicalSearchStates(states []string) []string {
	out := make([]string, 0, len(canonicalSearchStatesOrdered))
	seen := map[string]struct{}{}
	for _, raw := range states {
		state := strings.TrimSpace(strings.ToLower(raw))
		if state == "" {
			continue
		}
		if !slices.Contains(canonicalSearchStatesOrdered, state) {
			continue
		}
		if _, ok := seen[state]; ok {
			continue
		}
		seen[state] = struct{}{}
		out = append(out, state)
	}
	if len(out) == 0 {
		return append([]string(nil), canonicalSearchStatesOrdered...)
	}
	return out
}

// toggleSearchState toggles one canonical search state.
func (m *Model) toggleSearchState(state string) {
	state = strings.TrimSpace(strings.ToLower(state))
	if state == "" {
		return
	}
	states := canonicalSearchStates(m.searchStates)
	next := make([]string, 0, len(states))
	found := false
	for _, item := range states {
		if item == state {
			found = true
			continue
		}
		next = append(next, item)
	}
	if !found {
		next = append(next, state)
	}
	m.searchStates = canonicalSearchStates(next)
}

// isSearchStateEnabled reports whether a search state is currently enabled.
func (m Model) isSearchStateEnabled(state string) bool {
	state = strings.TrimSpace(strings.ToLower(state))
	for _, item := range m.searchStates {
		if strings.TrimSpace(strings.ToLower(item)) == state {
			return true
		}
	}
	return false
}

// wrapIndex wraps an index by delta for a bounded collection.
func wrapIndex(current int, delta int, total int) int {
	if total <= 0 {
		return 0
	}
	next := current + delta
	for next < 0 {
		next += total
	}
	for next >= total {
		next -= total
	}
	return next
}

// windowBounds returns an inclusive-exclusive list window that keeps selected visible.
func windowBounds(total, selected, windowSize int) (int, int) {
	if total <= 0 || windowSize <= 0 {
		return 0, 0
	}
	if total <= windowSize {
		return 0, total
	}
	selected = clamp(selected, 0, total-1)
	half := windowSize / 2
	start := selected - half
	if start < 0 {
		start = 0
	}
	end := start + windowSize
	if end > total {
		end = total
		start = max(0, end-windowSize)
	}
	return start, end
}

// applySearchFilter applies current search values and returns the follow-up command.
func (m *Model) applySearchFilter() tea.Cmd {
	m.mode = modeNone
	m.searchInput.Blur()
	m.searchQuery = strings.TrimSpace(m.searchInput.Value())
	m.searchStates = canonicalSearchStates(m.searchStates)
	m.searchApplied = true
	m.selectedTask = 0
	m.status = "search updated"
	if m.searchCrossProject {
		return m.loadSearchMatches
	}
	return m.loadData
}

// clearSearchQuery clears only the search query.
func (m *Model) clearSearchQuery() tea.Cmd {
	m.searchQuery = ""
	m.searchInput.SetValue("")
	m.searchApplied = true
	m.status = "query cleared"
	if m.searchCrossProject {
		return m.loadSearchMatches
	}
	return m.loadData
}

// resetSearchFilters resets query and filters back to defaults.
func (m *Model) resetSearchFilters() tea.Cmd {
	m.searchQuery = ""
	m.searchInput.SetValue("")
	m.searchCrossProject = m.searchDefaultCrossProject
	m.showArchived = m.searchDefaultIncludeArchive
	m.searchStates = canonicalSearchStates(m.searchDefaultStates)
	m.searchApplied = false
	m.status = "filters reset"
	return m.loadData
}

// applyRuntimeConfig applies runtime-updateable settings from a reload callback.
func (m *Model) applyRuntimeConfig(cfg RuntimeConfig) {
	WithRuntimeConfig(cfg)(m)
	m.refreshTaskFormLabelSuggestions()
}

// reloadRuntimeConfigCmd reloads runtime settings through the configured callback.
func (m Model) reloadRuntimeConfigCmd() tea.Cmd {
	if m.reloadConfig == nil {
		return func() tea.Msg {
			return configReloadedMsg{err: fmt.Errorf("config reload callback is unavailable")}
		}
	}
	return func() tea.Msg {
		cfg, err := m.reloadConfig()
		if err != nil {
			return configReloadedMsg{err: err}
		}
		return configReloadedMsg{config: cfg}
	}
}

// submitPathsRoots validates and persists a current-project root mapping change.
func (m Model) submitPathsRoots() (tea.Model, tea.Cmd) {
	project, ok := m.currentProject()
	if !ok {
		m.status = "no project selected"
		return m, nil
	}
	slug := strings.TrimSpace(strings.ToLower(project.Slug))
	if slug == "" {
		m.status = "project slug is empty"
		return m, nil
	}
	rootPath, err := normalizeProjectRootPathInput(m.pathsRootInput.Value())
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	if m.saveProjectRoot == nil {
		m.status = "save root failed: callback unavailable"
		return m, nil
	}
	m.mode = modeNone
	m.pathsRootInput.Blur()
	m.status = "saving root..."
	return m, m.saveProjectRootCmd(slug, rootPath)
}

// normalizeProjectRootPathInput validates and normalizes an optional project root path value.
func normalizeProjectRootPathInput(raw string) (string, error) {
	rootPath := strings.TrimSpace(raw)
	if rootPath == "" {
		return "", nil
	}
	absPath, err := filepath.Abs(rootPath)
	if err == nil {
		rootPath = absPath
	}
	info, err := os.Stat(rootPath)
	if err != nil {
		return "", fmt.Errorf("root path not found: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("root path must be a directory")
	}
	return rootPath, nil
}

// saveProjectRootCmd persists one project-root mapping through the callback surface.
func (m Model) saveProjectRootCmd(projectSlug, rootPath string) tea.Cmd {
	return func() tea.Msg {
		if err := m.saveProjectRoot(projectSlug, rootPath); err != nil {
			return projectRootSavedMsg{err: err}
		}
		return projectRootSavedMsg{
			projectSlug: projectSlug,
			rootPath:    rootPath,
		}
	}
}

// commandPaletteItems returns all known command-palette items.
func commandPaletteItems() []commandPaletteItem {
	return []commandPaletteItem{
		{Command: "new-task", Aliases: []string{"task-new"}, Description: "create a new task"},
		{Command: "new-subtask", Aliases: []string{"task-subtask"}, Description: "create subtask for selected item"},
		{Command: "edit-task", Aliases: []string{"task-edit"}, Description: "edit selected task"},
		{Command: "new-project", Aliases: []string{"project-new"}, Description: "create a new project"},
		{Command: "edit-project", Aliases: []string{"project-edit"}, Description: "edit selected project"},
		{Command: "search", Aliases: []string{}, Description: "open search modal"},
		{Command: "search-all", Aliases: []string{}, Description: "set search scope to all projects"},
		{Command: "search-project", Aliases: []string{}, Description: "set search scope to current project"},
		{Command: "clear-query", Aliases: []string{"clear-search-query"}, Description: "clear search text only"},
		{Command: "reset-filters", Aliases: []string{"clear-search"}, Description: "reset query + states + scope + archived"},
		{Command: "toggle-archived", Aliases: []string{}, Description: "toggle archived visibility"},
		{Command: "focus-subtree", Aliases: []string{"zoom-task"}, Description: "show selected task subtree only"},
		{Command: "focus-clear", Aliases: []string{"zoom-reset"}, Description: "return to full board view"},
		{Command: "toggle-select", Aliases: []string{"select-task"}, Description: "toggle selected task in multi-select"},
		{Command: "clear-selection", Aliases: []string{"selection-clear"}, Description: "clear all selected tasks"},
		{Command: "bulk-move-left", Aliases: []string{"move-left-selected"}, Description: "move selected tasks to previous column"},
		{Command: "bulk-move-right", Aliases: []string{"move-right-selected"}, Description: "move selected tasks to next column"},
		{Command: "bulk-archive", Aliases: []string{"archive-selected"}, Description: "archive selected tasks"},
		{Command: "bulk-delete", Aliases: []string{"delete-selected"}, Description: "hard delete selected tasks"},
		{Command: "undo", Aliases: []string{}, Description: "undo last mutation"},
		{Command: "redo", Aliases: []string{}, Description: "redo last undone mutation"},
		{Command: "reload-config", Aliases: []string{"config-reload", "reload"}, Description: "reload runtime config from disk"},
		{Command: "paths-roots", Aliases: []string{"roots", "project-root"}, Description: "edit current project root mapping"},
		{Command: "labels-config", Aliases: []string{"labels", "edit-labels"}, Description: "edit global + project labels defaults"},
		{Command: "activity-log", Aliases: []string{"log"}, Description: "open recent activity modal"},
		{Command: "help", Aliases: []string{}, Description: "open help modal"},
		{Command: "quit", Aliases: []string{"exit"}, Description: "quit kan"},
	}
}

// filteredCommandItems returns command items filtered by query.
func (m Model) filteredCommandItems(raw string) []commandPaletteItem {
	query := strings.TrimSpace(strings.ToLower(raw))
	items := commandPaletteItems()
	if query == "" {
		return items
	}
	type scoredItem struct {
		item  commandPaletteItem
		score int
	}
	scored := make([]scoredItem, 0, len(items))
	for _, item := range items {
		score, ok := scoreCommandPaletteItem(query, item)
		if !ok {
			continue
		}
		scored = append(scored, scoredItem{item: item, score: score})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].item.Command < scored[j].item.Command
	})
	out := make([]commandPaletteItem, 0, len(scored))
	for _, item := range scored {
		out = append(out, item.item)
	}
	return out
}

// scoreCommandPaletteItem ranks one command-palette item for a fuzzy query.
func scoreCommandPaletteItem(query string, item commandPaletteItem) (int, bool) {
	score := -1
	ok := false
	if v, match := bestFuzzyScore(query, item.Command); match {
		score = max(score, v+200)
		ok = true
	}
	if len(item.Aliases) > 0 {
		if v, match := bestFuzzyScore(query, item.Aliases...); match {
			score = max(score, v+160)
			ok = true
		}
	}
	if v, match := bestFuzzyScore(query, item.Description); match {
		score = max(score, v+80)
		ok = true
	}
	return score, ok
}

// bestFuzzyScore returns the best fuzzy score across candidate strings.
func bestFuzzyScore(query string, candidates ...string) (int, bool) {
	best := 0
	ok := false
	for _, candidate := range candidates {
		score, match := fuzzyScore(query, candidate)
		if !match {
			continue
		}
		if !ok || score > best {
			best = score
		}
		ok = true
	}
	return best, ok
}

// fuzzyScore returns a deterministic fuzzy score where higher is better.
func fuzzyScore(query, candidate string) (int, bool) {
	query = strings.TrimSpace(strings.ToLower(query))
	candidate = strings.TrimSpace(strings.ToLower(candidate))
	if query == "" {
		return 0, true
	}
	if candidate == "" {
		return 0, false
	}

	// Strongly prefer exact/prefix/contains matches before subsequence scoring.
	if query == candidate {
		return 6000, true
	}
	if strings.HasPrefix(candidate, query) {
		return 5000 - len(candidate), true
	}
	if idx := strings.Index(candidate, query); idx >= 0 {
		return 4200 - idx, true
	}

	q := []rune(query)
	c := []rune(candidate)
	qi := 0
	score := 3000
	last := -1
	for ci, r := range c {
		if qi >= len(q) {
			break
		}
		if r != q[qi] {
			continue
		}
		if last < 0 {
			score -= ci
		} else {
			gap := ci - last - 1
			score -= gap * 3
		}
		last = ci
		qi++
	}
	if qi != len(q) {
		return 0, false
	}
	score -= len(c) - len(q)
	return score, true
}

// commandToExecute returns the selected command from the palette state.
func (m Model) commandToExecute() string {
	if len(m.commandMatches) > 0 {
		idx := clamp(m.commandIndex, 0, len(m.commandMatches)-1)
		return m.commandMatches[idx].Command
	}
	return strings.TrimSpace(strings.ToLower(m.commandInput.Value()))
}

// priorityIndex handles priority index.
func priorityIndex(priority domain.Priority) int {
	for i, p := range priorityOptions {
		if p == priority {
			return i
		}
	}
	return 1
}

// cyclePriority handles cycle priority.
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

// startDuePicker starts due picker.
func (m *Model) startDuePicker() {
	m.pickerBack = m.mode
	m.mode = modeDuePicker
	m.duePicker = 0
}

// duePickerOptions handles due picker options.
func (m *Model) duePickerOptions() []duePickerOption {
	now := time.Now().UTC()
	today := now.Format("2006-01-02")
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")
	nextWeek := now.AddDate(0, 0, 7).Format("2006-01-02")
	inTwoWeeks := now.AddDate(0, 0, 14).Format("2006-01-02")
	todayEnd := time.Date(now.Year(), now.Month(), now.Day(), 17, 0, 0, 0, time.UTC).Format("2006-01-02 15:04")
	tomorrowStart := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, time.UTC).AddDate(0, 0, 1).Format("2006-01-02 15:04")
	return []duePickerOption{
		{Label: "No due date", Value: "-"},
		{Label: "Today (" + today + ")", Value: today},
		{Label: "Today 17:00 UTC (" + todayEnd + ")", Value: todayEnd},
		{Label: "Tomorrow (" + tomorrow + ")", Value: tomorrow},
		{Label: "Tomorrow 09:00 UTC (" + tomorrowStart + ")", Value: tomorrowStart},
		{Label: "Next week (" + nextWeek + ")", Value: nextWeek},
		{Label: "In two weeks (" + inTwoWeeks + ")", Value: inTwoWeeks},
	}
}

// startLabelPicker opens a modal picker with inherited label suggestions.
func (m *Model) startLabelPicker() tea.Cmd {
	m.labelPickerBack = m.mode
	m.mode = modeLabelPicker
	m.labelPickerItems = m.taskFormLabelPickerItems()
	m.labelPickerIndex = 0
	if len(m.labelPickerItems) == 0 {
		m.status = "no inherited labels available"
	} else {
		m.status = "label inheritance picker"
	}
	return nil
}

// refreshTaskFormLabelSuggestions refreshes task-form label suggestions from inherited sources.
func (m *Model) refreshTaskFormLabelSuggestions() {
	if len(m.formInputs) <= taskFieldLabels {
		return
	}
	suggestions := mergeUniqueLabels(
		mergeLabelSources(m.taskFormLabelSources()),
		m.labelSuggestions(24),
	)
	m.formInputs[taskFieldLabels].SetSuggestions(suggestions)
}

// mergeUniqueLabels returns normalized labels preserving first-seen order across source slices.
func mergeUniqueLabels(groups ...[]string) []string {
	out := make([]string, 0)
	seen := map[string]struct{}{}
	for _, group := range groups {
		for _, raw := range group {
			label := strings.TrimSpace(strings.ToLower(raw))
			if label == "" {
				continue
			}
			if _, ok := seen[label]; ok {
				continue
			}
			seen[label] = struct{}{}
			out = append(out, label)
		}
	}
	return out
}

// taskFormLabelSources resolves label inheritance sources for the active task form context.
func (m Model) taskFormLabelSources() labelInheritanceSources {
	task, ok := m.selectedTaskForLabelInheritance()
	if !ok {
		return m.labelSourcesForTask(domain.Task{})
	}
	return m.labelSourcesForTask(task)
}

// labelSourcesForTask resolves inherited labels for one task or taskless project context.
func (m Model) labelSourcesForTask(task domain.Task) labelInheritanceSources {
	sources := labelInheritanceSources{
		Global: normalizeConfigLabels(m.allowedLabelGlobal),
	}
	if project, ok := m.currentProject(); ok {
		projectSlug := strings.TrimSpace(strings.ToLower(project.Slug))
		sources.Project = normalizeConfigLabels(m.allowedLabelProject[projectSlug])
	}
	if strings.TrimSpace(task.ID) != "" {
		sources.Phase = m.labelsFromPhaseAncestors(task)
	}
	return sources
}

// selectedTaskForLabelInheritance picks the best task context for inherited label sources.
func (m Model) selectedTaskForLabelInheritance() (domain.Task, bool) {
	if strings.TrimSpace(m.editingTaskID) != "" {
		if task, ok := m.taskByID(m.editingTaskID); ok {
			return task, true
		}
	}
	if strings.TrimSpace(m.taskFormParentID) != "" {
		if task, ok := m.taskByID(m.taskFormParentID); ok {
			return task, true
		}
	}
	return m.selectedTaskInCurrentColumn()
}

// labelsFromPhaseAncestors collects inherited labels from phase ancestors in parent-chain order.
func (m Model) labelsFromPhaseAncestors(task domain.Task) []string {
	out := make([]string, 0)
	seenLabels := map[string]struct{}{}
	visited := map[string]struct{}{}
	current := task
	for strings.TrimSpace(current.ID) != "" {
		if _, seen := visited[current.ID]; seen {
			break
		}
		visited[current.ID] = struct{}{}
		if current.Kind == domain.WorkKindPhase {
			for _, rawLabel := range current.Labels {
				label := strings.TrimSpace(strings.ToLower(rawLabel))
				if label == "" {
					continue
				}
				if _, ok := seenLabels[label]; ok {
					continue
				}
				seenLabels[label] = struct{}{}
				out = append(out, label)
			}
		}
		parentID := strings.TrimSpace(current.ParentID)
		if parentID == "" {
			break
		}
		parent, ok := m.taskByID(parentID)
		if !ok {
			break
		}
		current = parent
	}
	return out
}

// taskFormLabelPickerItems builds source-tagged inherited labels for modal selection.
func (m Model) taskFormLabelPickerItems() []labelPickerItem {
	sources := m.taskFormLabelSources()
	out := make([]labelPickerItem, 0, len(sources.Global)+len(sources.Project)+len(sources.Phase))
	appendItems := func(source string, labels []string) {
		for _, label := range labels {
			out = append(out, labelPickerItem{Label: label, Source: source})
		}
	}
	appendItems("global", sources.Global)
	appendItems("project", sources.Project)
	appendItems("phase", sources.Phase)
	return out
}

// appendTaskFormLabel appends one normalized label to the form without duplicating entries.
func (m *Model) appendTaskFormLabel(label string) {
	if len(m.formInputs) <= taskFieldLabels {
		return
	}
	label = strings.TrimSpace(strings.ToLower(label))
	if label == "" {
		return
	}
	current := parseLabelsInput(m.formInputs[taskFieldLabels].Value(), nil)
	for _, existing := range current {
		if strings.EqualFold(strings.TrimSpace(existing), label) {
			return
		}
	}
	current = append(current, label)
	m.formInputs[taskFieldLabels].SetValue(strings.Join(current, ","))
}

// acceptCurrentLabelSuggestion applies the active autocomplete suggestion into the labels field.
func (m *Model) acceptCurrentLabelSuggestion() bool {
	if len(m.formInputs) <= taskFieldLabels {
		return false
	}
	suggestion := strings.TrimSpace(strings.ToLower(m.formInputs[taskFieldLabels].CurrentSuggestion()))
	if suggestion == "" {
		matches := m.formInputs[taskFieldLabels].MatchedSuggestions()
		if len(matches) == 0 {
			return false
		}
		suggestion = strings.TrimSpace(strings.ToLower(matches[0]))
	}
	if suggestion == "" {
		return false
	}

	raw := strings.TrimSpace(m.formInputs[taskFieldLabels].Value())
	if raw == "" || raw == "-" {
		m.formInputs[taskFieldLabels].SetValue(suggestion)
		m.formInputs[taskFieldLabels].CursorEnd()
		return true
	}

	parts := strings.Split(raw, ",")
	labels := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for idx, part := range parts {
		label := strings.TrimSpace(strings.ToLower(part))
		if idx == len(parts)-1 {
			label = suggestion
		}
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		labels = append(labels, label)
	}
	if len(labels) == 0 {
		labels = append(labels, suggestion)
	}
	m.formInputs[taskFieldLabels].SetValue(strings.Join(labels, ","))
	m.formInputs[taskFieldLabels].CursorEnd()
	return true
}

// startResourcePicker opens filesystem resource selection for a task.
func (m *Model) startResourcePicker(taskID string, back inputMode) tea.Cmd {
	taskID = strings.TrimSpace(taskID)
	root := m.resourcePickerRootForCurrentProject()
	m.mode = modeResourcePicker
	m.resourcePickerBack = back
	m.resourcePickerTaskID = taskID
	m.resourcePickerRoot = root
	m.resourcePickerDir = root
	m.resourcePickerIndex = 0
	m.resourcePickerItems = nil
	m.resourcePickerFilter.SetValue("")
	m.resourcePickerFilter.CursorEnd()
	m.resourcePickerFilter.Focus()
	m.status = "resource picker"
	return m.openResourcePickerDir(root)
}

// openResourcePickerDir loads one directory within the picker root.
func (m Model) openResourcePickerDir(dir string) tea.Cmd {
	root := strings.TrimSpace(m.resourcePickerRoot)
	if root == "" {
		root = m.resourcePickerRootForCurrentProject()
	}
	return func() tea.Msg {
		entries, current, err := listResourcePickerEntries(root, dir)
		if err != nil {
			return resourcePickerLoadedMsg{err: fmt.Errorf("resource picker: %w", err)}
		}
		return resourcePickerLoadedMsg{
			root:    root,
			current: current,
			entries: entries,
		}
	}
}

// openResourcePickerParent opens the current picker directory parent.
func (m Model) openResourcePickerParent() tea.Cmd {
	current := strings.TrimSpace(m.resourcePickerDir)
	if current == "" {
		current = m.resourcePickerRoot
	}
	parent := filepath.Dir(current)
	if parent == "." || parent == "" {
		parent = m.resourcePickerRoot
	}
	return m.openResourcePickerDir(parent)
}

// selectedResourcePickerEntry returns the currently highlighted resource picker entry.
func (m Model) selectedResourcePickerEntry() (resourcePickerEntry, bool) {
	items := m.visibleResourcePickerItems()
	if len(items) == 0 {
		return resourcePickerEntry{}, false
	}
	idx := clamp(m.resourcePickerIndex, 0, len(items)-1)
	return items[idx], true
}

// visibleResourcePickerItems returns resource picker entries after applying fuzzy filter text.
func (m Model) visibleResourcePickerItems() []resourcePickerEntry {
	if len(m.resourcePickerItems) == 0 {
		return nil
	}
	query := strings.TrimSpace(m.resourcePickerFilter.Value())
	if query == "" {
		return append([]resourcePickerEntry(nil), m.resourcePickerItems...)
	}

	type scoredEntry struct {
		entry resourcePickerEntry
		score int
	}
	scored := make([]scoredEntry, 0, len(m.resourcePickerItems))
	for _, entry := range m.resourcePickerItems {
		score, ok := bestFuzzyScore(query, entry.Name, filepath.ToSlash(entry.Path))
		if !ok {
			continue
		}
		if entry.IsDir {
			score += 8
		}
		if entry.Name == ".." {
			score -= 100
		}
		scored = append(scored, scoredEntry{entry: entry, score: score})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if scored[i].entry.IsDir != scored[j].entry.IsDir {
			return scored[i].entry.IsDir
		}
		return strings.ToLower(scored[i].entry.Name) < strings.ToLower(scored[j].entry.Name)
	})
	out := make([]resourcePickerEntry, 0, len(scored))
	for _, item := range scored {
		out = append(out, item.entry)
	}
	return out
}

// attachSelectedResourceEntry attaches the currently selected resource entry to the target task.
func (m *Model) attachSelectedResourceEntry() tea.Cmd {
	entry, ok := m.selectedResourcePickerEntry()
	if !ok {
		// Empty directories still allow attaching the current folder as context.
		entry = resourcePickerEntry{
			Name:  filepath.Base(m.resourcePickerDir),
			Path:  m.resourcePickerDir,
			IsDir: true,
		}
	}
	back := m.resourcePickerBack
	m.mode = back
	m.resourcePickerFilter.Blur()
	m.resourcePickerFilter.SetValue("")
	m.resourcePickerIndex = 0

	// Task form attachment flow stages refs for create/edit submit.
	if back == modeAddTask || back == modeEditTask {
		ref := buildResourceRef(strings.TrimSpace(m.resourcePickerRoot), entry.Path, entry.IsDir)
		refs, added := appendResourceRefIfMissing(m.taskFormResourceRefs, ref)
		if !added {
			m.status = "resource already staged"
			return m.focusTaskFormField(m.formFocus)
		}
		m.taskFormResourceRefs = refs
		m.status = "resource staged"
		return m.focusTaskFormField(m.formFocus)
	}

	// Project root picker flow writes selected directory back to form/input.
	if back == modeAddProject || back == modeEditProject || back == modePathsRoots {
		selectedDir := entry.Path
		if !entry.IsDir {
			selectedDir = filepath.Dir(selectedDir)
		}
		normalized, err := normalizeProjectRootPathInput(selectedDir)
		if err != nil {
			m.status = err.Error()
			return nil
		}
		if back == modePathsRoots {
			m.pathsRootInput.SetValue(normalized)
			m.pathsRootInput.CursorEnd()
			m.status = "root path selected"
			return m.pathsRootInput.Focus()
		}
		if len(m.projectFormInputs) > projectFieldRootPath {
			m.projectFormInputs[projectFieldRootPath].SetValue(normalized)
			m.projectFormInputs[projectFieldRootPath].CursorEnd()
			m.projectFormFocus = projectFieldRootPath
			m.status = "root path selected"
			return m.focusProjectFormField(projectFieldRootPath)
		}
		m.status = "root path selected"
		return nil
	}

	// Existing task-info path persists immediately to task metadata.
	m.status = "attaching resource..."
	return m.attachResourceEntry(entry.Path, entry.IsDir)
}

// attachResourceEntry persists one filesystem reference through task metadata update.
func (m Model) attachResourceEntry(path string, isDir bool) tea.Cmd {
	taskID := strings.TrimSpace(m.resourcePickerTaskID)
	root := strings.TrimSpace(m.resourcePickerRoot)
	return func() tea.Msg {
		task, ok := m.taskByID(taskID)
		if !ok {
			return actionMsg{status: "resource attach failed: task not found"}
		}
		ref := buildResourceRef(root, path, isDir)
		refs, added := appendResourceRefIfMissing(task.Metadata.ResourceRefs, ref)
		if !added {
			return actionMsg{status: "resource already attached"}
		}
		meta := task.Metadata
		meta.ResourceRefs = refs
		_, err := m.svc.UpdateTask(context.Background(), app.UpdateTaskInput{
			TaskID:      task.ID,
			Title:       task.Title,
			Description: task.Description,
			Priority:    task.Priority,
			DueAt:       task.DueAt,
			Labels:      append([]string(nil), task.Labels...),
			Metadata:    &meta,
		})
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{
			status:      "resource attached",
			reload:      true,
			focusTaskID: task.ID,
		}
	}
}

// resourcePickerRootForCurrentProject returns configured project root or cwd fallback.
func (m Model) resourcePickerRootForCurrentProject() string {
	if project, ok := m.currentProject(); ok {
		slug := strings.TrimSpace(strings.ToLower(project.Slug))
		if root := strings.TrimSpace(m.projectRoots[slug]); root != "" {
			if abs, err := filepath.Abs(root); err == nil {
				return abs
			}
			return root
		}
	}
	if strings.TrimSpace(m.defaultRootDir) != "" {
		if abs, err := filepath.Abs(m.defaultRootDir); err == nil {
			return abs
		}
		return m.defaultRootDir
	}
	return "."
}

// summarizeTaskRefs renders dependency IDs with known task titles when available.
func (m Model) summarizeTaskRefs(ids []string, maxItems int) string {
	items := uniqueTrimmed(ids)
	if len(items) == 0 {
		return "-"
	}
	if maxItems <= 0 {
		maxItems = 4
	}
	visible := items
	extra := 0
	if len(items) > maxItems {
		visible = items[:maxItems]
		extra = len(items) - maxItems
	}
	parts := make([]string, 0, len(visible))
	for _, id := range visible {
		label := id
		if task, ok := m.taskByID(id); ok && strings.TrimSpace(task.Title) != "" {
			label = fmt.Sprintf("%s(%s)", id, truncate(task.Title, 22))
		}
		parts = append(parts, label)
	}
	joined := strings.Join(parts, ", ")
	if extra > 0 {
		joined += fmt.Sprintf(" +%d", extra)
	}
	return joined
}

// uniqueTrimmed trims and deduplicates text values while preserving order.
func uniqueTrimmed(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

// formatLabelSource renders one inherited label source for modal hints.
func formatLabelSource(source string, labels []string) string {
	if len(labels) == 0 {
		return source + ": -"
	}
	return source + ": " + strings.Join(labels, ", ")
}

// mergeLabelSources merges inherited label sources using global -> project -> phase precedence.
func mergeLabelSources(sources labelInheritanceSources) []string {
	out := make([]string, 0, len(sources.Global)+len(sources.Project)+len(sources.Phase))
	seen := map[string]struct{}{}
	appendUnique := func(values []string) {
		for _, raw := range values {
			label := strings.TrimSpace(strings.ToLower(raw))
			if label == "" {
				continue
			}
			if _, ok := seen[label]; ok {
				continue
			}
			seen[label] = struct{}{}
			out = append(out, label)
		}
	}
	appendUnique(sources.Global)
	appendUnique(sources.Project)
	appendUnique(sources.Phase)
	return out
}

// normalizeConfigLabels trims and deduplicates config-provided label lists.
func normalizeConfigLabels(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, raw := range values {
		label := strings.TrimSpace(strings.ToLower(raw))
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		out = append(out, label)
	}
	return out
}

// listResourcePickerEntries loads picker entries and keeps the current directory within root bounds.
func listResourcePickerEntries(root, dir string) ([]resourcePickerEntry, string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		root = "."
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, "", err
	}
	dirAbs := strings.TrimSpace(dir)
	if dirAbs == "" {
		dirAbs = rootAbs
	}
	dirAbs, err = filepath.Abs(dirAbs)
	if err != nil {
		return nil, "", err
	}
	if rel, relErr := filepath.Rel(rootAbs, dirAbs); relErr != nil || strings.HasPrefix(rel, "..") {
		dirAbs = rootAbs
	}
	items, err := os.ReadDir(dirAbs)
	if err != nil {
		return nil, "", err
	}
	entries := make([]resourcePickerEntry, 0, len(items)+1)
	if dirAbs != rootAbs {
		parent := filepath.Dir(dirAbs)
		if rel, relErr := filepath.Rel(rootAbs, parent); relErr != nil || strings.HasPrefix(rel, "..") {
			parent = rootAbs
		}
		entries = append(entries, resourcePickerEntry{
			Name:  "..",
			Path:  parent,
			IsDir: true,
		})
	}
	for _, item := range items {
		entries = append(entries, resourcePickerEntry{
			Name:  item.Name(),
			Path:  filepath.Join(dirAbs, item.Name()),
			IsDir: item.IsDir(),
		})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return entries, dirAbs, nil
}

// buildResourceRef builds a normalized local-file or local-directory resource reference.
func buildResourceRef(root, path string, isDir bool) domain.ResourceRef {
	path = strings.TrimSpace(path)
	if path == "" {
		path = root
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	resourceType := domain.ResourceTypeLocalFile
	if isDir {
		resourceType = domain.ResourceTypeLocalDir
	}
	now := time.Now().UTC()
	ref := domain.ResourceRef{
		ResourceType:   resourceType,
		Location:       filepath.ToSlash(path),
		PathMode:       domain.PathModeAbsolute,
		Title:          filepath.Base(path),
		LastVerifiedAt: &now,
	}
	root = strings.TrimSpace(root)
	if root != "" {
		if absRoot, err := filepath.Abs(root); err == nil {
			if rel, relErr := filepath.Rel(absRoot, path); relErr == nil && !strings.HasPrefix(rel, "..") {
				ref.Location = filepath.ToSlash(rel)
				ref.PathMode = domain.PathModeRelative
				ref.BaseAlias = "project_root"
			}
		}
	}
	return ref
}

// appendResourceRefIfMissing appends a resource ref unless an equivalent ref already exists.
func appendResourceRefIfMissing(in []domain.ResourceRef, candidate domain.ResourceRef) ([]domain.ResourceRef, bool) {
	candidateLocation := strings.TrimSpace(strings.ToLower(candidate.Location))
	for _, existing := range in {
		existingLocation := strings.TrimSpace(strings.ToLower(existing.Location))
		if existing.ResourceType == candidate.ResourceType &&
			existing.PathMode == candidate.PathMode &&
			existingLocation == candidateLocation {
			return in, false
		}
	}
	return append(append([]domain.ResourceRef(nil), in...), candidate), true
}

// labelSuggestions handles label suggestions.
func (m Model) labelSuggestions(maxLabels int) []string {
	if maxLabels <= 0 {
		maxLabels = 5
	}
	projectID, ok := m.currentProjectID()
	if !ok {
		return nil
	}
	counts := map[string]int{}
	for _, allowed := range m.allowedLabelsForSelectedProject() {
		counts[allowed] += 1000
	}
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

// handleNormalModeKey handles normal mode key.
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
		if count := m.clearSelection(); count > 0 {
			m.status = fmt.Sprintf("cleared %d selected tasks", count)
			m.appendActivity(activityEntry{
				At:      time.Now().UTC(),
				Summary: "clear selection",
				Target:  fmt.Sprintf("%d tasks", count),
			})
			return m, nil
		}
		if strings.TrimSpace(m.projectionRootTaskID) != "" {
			m.projectionRootTaskID = ""
			m.status = "full board view"
			return m, nil
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
	case key.Matches(msg, m.keys.multiSelect):
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		if m.toggleTaskSelection(task.ID) {
			m.status = fmt.Sprintf("selected %q (%d total)", truncate(task.Title, 28), len(m.selectedTaskIDs))
			m.appendActivity(activityEntry{
				At:      time.Now().UTC(),
				Summary: "select task",
				Target:  task.Title,
			})
		} else {
			m.status = fmt.Sprintf("unselected %q (%d total)", truncate(task.Title, 28), len(m.selectedTaskIDs))
			m.appendActivity(activityEntry{
				At:      time.Now().UTC(),
				Summary: "unselect task",
				Target:  task.Title,
			})
		}
		return m, nil
	case key.Matches(msg, m.keys.activityLog):
		return m, m.openActivityLog()
	case key.Matches(msg, m.keys.undo):
		return m.undoLastMutation()
	case key.Matches(msg, m.keys.redo):
		return m.redoLastMutation()
	case key.Matches(msg, m.keys.addTask):
		m.help.ShowAll = false
		return m, m.startTaskForm(nil)
	case key.Matches(msg, m.keys.addSubtask):
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		m.help.ShowAll = false
		return m, m.startSubtaskForm(task)
	case key.Matches(msg, m.keys.newProject):
		m.help.ShowAll = false
		return m, m.startProjectForm(nil)
	case key.Matches(msg, m.keys.taskInfo):
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		m.help.ShowAll = false
		m.mode = modeTaskInfo
		m.taskInfoTaskID = task.ID
		m.taskInfoSubtaskIdx = 0
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
	case key.Matches(msg, m.keys.focusSubtree):
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		m.projectionRootTaskID = task.ID
		m.focusTaskByID(task.ID)
		m.status = "focused subtree"
		return m, nil
	case key.Matches(msg, m.keys.clearFocus):
		if m.projectionRootTaskID == "" {
			m.status = "full board already visible"
			return m, nil
		}
		m.projectionRootTaskID = ""
		m.status = "full board view"
		return m, nil
	case key.Matches(msg, m.keys.moveTaskLeft):
		if len(m.selectedTaskIDs) > 0 {
			return m.moveSelectedTasks(-1)
		}
		return m.moveSelectedTask(-1)
	case key.Matches(msg, m.keys.moveTaskRight):
		if len(m.selectedTaskIDs) > 0 {
			return m.moveSelectedTasks(1)
		}
		return m.moveSelectedTask(1)
	case key.Matches(msg, m.keys.deleteTask):
		return m.confirmDeleteAction(m.defaultDeleteMode, m.confirmDelete, "delete task")
	case key.Matches(msg, m.keys.archiveTask):
		return m.confirmDeleteAction(app.DeleteModeArchive, m.confirmArchive, "archive task")
	case key.Matches(msg, m.keys.hardDeleteTask):
		return m.confirmDeleteAction(app.DeleteModeHard, m.confirmHardDelete, "hard delete task")
	case key.Matches(msg, m.keys.restoreTask):
		return m.confirmRestoreAction()
	case key.Matches(msg, m.keys.toggleArchived):
		m.showArchived = !m.showArchived
		if m.showArchived {
			m.status = "showing archived tasks"
		} else {
			m.status = "hiding archived tasks"
		}
		m.selectedTask = 0
		m.clearSelection()
		return m, m.loadData
	default:
		return m, nil
	}
}

// handleInputModeKey handles input mode key.
func (m Model) handleInputModeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.mode == modeActivityLog {
		switch {
		case msg.String() == "esc" || key.Matches(msg, m.keys.activityLog):
			m.mode = modeNone
			m.status = "ready"
			return m, nil
		case key.Matches(msg, m.keys.undo):
			return m.undoLastMutation()
		case key.Matches(msg, m.keys.redo):
			return m.redoLastMutation()
		default:
			return m, nil
		}
	}

	if m.mode == modeTaskInfo {
		task, ok := m.taskInfoTask()
		if !ok {
			m.mode = modeNone
			m.taskInfoTaskID = ""
			m.taskInfoSubtaskIdx = 0
			m.status = "task info unavailable"
			return m, nil
		}
		subtasks := m.subtasksForParent(task.ID)
		switch msg.String() {
		case "esc":
			if m.stepBackTaskInfo(task) {
				return m, nil
			}
			m.mode = modeNone
			m.taskInfoTaskID = ""
			m.taskInfoSubtaskIdx = 0
			m.status = "ready"
			return m, nil
		case "i":
			m.mode = modeNone
			m.taskInfoTaskID = ""
			m.taskInfoSubtaskIdx = 0
			m.status = "ready"
			return m, nil
		case "j", "down":
			if len(subtasks) > 0 && m.taskInfoSubtaskIdx < len(subtasks)-1 {
				m.taskInfoSubtaskIdx++
			}
			return m, nil
		case "k", "up":
			if m.taskInfoSubtaskIdx > 0 {
				m.taskInfoSubtaskIdx--
			}
			return m, nil
		case "enter":
			if len(subtasks) == 0 {
				return m, nil
			}
			subtask := subtasks[clamp(m.taskInfoSubtaskIdx, 0, len(subtasks)-1)]
			m.taskInfoTaskID = subtask.ID
			m.taskInfoSubtaskIdx = 0
			m.status = "subtask info"
			return m, nil
		case "backspace", "h", "left":
			parentID := strings.TrimSpace(task.ParentID)
			if parentID == "" {
				return m, nil
			}
			if _, ok := m.taskByID(parentID); !ok {
				return m, nil
			}
			m.taskInfoTaskID = parentID
			m.taskInfoSubtaskIdx = 0
			m.status = "parent task info"
			return m, nil
		case "e":
			return m, m.startTaskForm(&task)
		case "s":
			return m, m.startSubtaskForm(task)
		case "b":
			_ = m.startTaskForm(&task)
			m.status = "edit dependencies"
			return m, m.focusTaskFormField(taskFieldDependsOn)
		case "r", "R":
			return m, m.startResourcePicker(task.ID, modeTaskInfo)
		case "[":
			return m.moveTaskIDs([]string{task.ID}, -1, "move task", task.Title, false)
		case "]":
			return m.moveTaskIDs([]string{task.ID}, 1, "move task", task.Title, false)
		case "f":
			m.projectionRootTaskID = task.ID
			m.mode = modeNone
			m.taskInfoTaskID = ""
			m.taskInfoSubtaskIdx = 0
			m.status = "focused subtree"
			return m, nil
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
			m.status = "cancelled"
			return m, nil
		case msg.Code == tea.KeyTab || msg.String() == "tab" || msg.String() == "ctrl+i":
			m.searchFocus = wrapIndex(m.searchFocus, 1, 5)
			if m.searchFocus == 0 {
				return m, m.searchInput.Focus()
			}
			m.searchInput.Blur()
			return m, nil
		case msg.String() == "shift+tab" || msg.String() == "backtab":
			m.searchFocus = wrapIndex(m.searchFocus, -1, 5)
			if m.searchFocus == 0 {
				return m, m.searchInput.Focus()
			}
			m.searchInput.Blur()
			return m, nil
		case msg.String() == "ctrl+p":
			m.searchCrossProject = !m.searchCrossProject
			return m, nil
		case msg.String() == "ctrl+a":
			m.showArchived = !m.showArchived
			return m, nil
		case msg.String() == "ctrl+u":
			return m, m.clearSearchQuery()
		case msg.String() == "ctrl+r":
			return m, m.resetSearchFilters()
		case (msg.String() == "h" || msg.String() == "left") && m.searchFocus != 0:
			switch m.searchFocus {
			case 1:
				m.searchStateCursor = wrapIndex(m.searchStateCursor, -1, len(canonicalSearchStatesOrdered))
			case 2:
				m.searchCrossProject = !m.searchCrossProject
			case 3:
				m.showArchived = !m.showArchived
			}
			return m, nil
		case (msg.String() == "l" || msg.String() == "right") && m.searchFocus != 0:
			switch m.searchFocus {
			case 1:
				m.searchStateCursor = wrapIndex(m.searchStateCursor, 1, len(canonicalSearchStatesOrdered))
			case 2:
				m.searchCrossProject = !m.searchCrossProject
			case 3:
				m.showArchived = !m.showArchived
			}
			return m, nil
		case (msg.String() == " " || msg.String() == "space") && m.searchFocus != 0:
			switch m.searchFocus {
			case 1:
				if len(canonicalSearchStatesOrdered) > 0 {
					idx := clamp(m.searchStateCursor, 0, len(canonicalSearchStatesOrdered)-1)
					m.toggleSearchState(canonicalSearchStatesOrdered[idx])
				}
			case 2:
				m.searchCrossProject = !m.searchCrossProject
			case 3:
				m.showArchived = !m.showArchived
			}
			return m, nil
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			switch m.searchFocus {
			case 1:
				if len(canonicalSearchStatesOrdered) > 0 {
					idx := clamp(m.searchStateCursor, 0, len(canonicalSearchStatesOrdered)-1)
					m.toggleSearchState(canonicalSearchStatesOrdered[idx])
				}
				return m, nil
			case 2:
				m.searchCrossProject = !m.searchCrossProject
				return m, nil
			case 3:
				m.showArchived = !m.showArchived
				return m, nil
			default:
				return m, m.applySearchFilter()
			}
		default:
			if m.searchFocus == 0 {
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				m.searchQuery = strings.TrimSpace(m.searchInput.Value())
				return m, cmd
			} else {
				return m, nil
			}
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
		case msg.Code == tea.KeyTab || msg.String() == "tab":
			if len(m.commandMatches) == 0 {
				return m, nil
			}
			m.commandInput.SetValue(m.commandMatches[0].Command)
			m.commandInput.CursorEnd()
			m.commandMatches = m.filteredCommandItems(m.commandInput.Value())
			m.commandIndex = 0
			return m, nil
		case msg.String() == "j" || msg.String() == "down":
			if len(m.commandMatches) > 0 && m.commandIndex < len(m.commandMatches)-1 {
				m.commandIndex++
			}
			return m, nil
		case msg.String() == "k" || msg.String() == "up":
			if m.commandIndex > 0 {
				m.commandIndex--
			}
			return m, nil
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			cmd := m.commandToExecute()
			m.mode = modeNone
			m.commandInput.Blur()
			return m.executeCommandPalette(cmd)
		default:
			var cmd tea.Cmd
			m.commandInput, cmd = m.commandInput.Update(msg)
			m.commandMatches = m.filteredCommandItems(m.commandInput.Value())
			m.commandIndex = clamp(m.commandIndex, 0, len(m.commandMatches)-1)
			return m, cmd
		}
	}

	if m.mode == modeConfirmAction {
		switch msg.String() {
		case "esc", "n":
			m.mode = modeNone
			m.pendingConfirm = confirmAction{}
			m.status = "cancelled"
			return m, nil
		case "h", "left", "l", "right":
			if m.confirmChoice == 0 {
				m.confirmChoice = 1
			} else {
				m.confirmChoice = 0
			}
			return m, nil
		case "y":
			m.confirmChoice = 0
			m.mode = modeNone
			action := m.pendingConfirm
			m.pendingConfirm = confirmAction{}
			m.status = "applying action..."
			return m.applyConfirmedAction(action)
		case "enter":
			if m.confirmChoice == 1 {
				m.mode = modeNone
				m.pendingConfirm = confirmAction{}
				m.status = "cancelled"
				return m, nil
			}
			m.mode = modeNone
			action := m.pendingConfirm
			m.pendingConfirm = confirmAction{}
			m.status = "applying action..."
			return m.applyConfirmedAction(action)
		default:
			return m, nil
		}
	}

	if m.mode == modeQuickActions {
		actions := m.quickActions()
		switch msg.String() {
		case "esc":
			m.mode = modeNone
			m.status = "cancelled"
			return m, nil
		case "j", "down":
			if m.quickActionIndex < len(actions)-1 {
				m.quickActionIndex++
			}
			return m, nil
		case "k", "up":
			if m.quickActionIndex > 0 {
				m.quickActionIndex--
			}
			return m, nil
		case "enter":
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
			return m, m.focusTaskFormField(taskFieldDue)
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
			if len(options) == 0 || len(m.formInputs) <= taskFieldDue {
				m.mode = m.pickerBack
				m.pickerBack = modeNone
				return m, m.focusTaskFormField(taskFieldDue)
			}
			choice := options[clamp(m.duePicker, 0, len(options)-1)]
			m.formInputs[taskFieldDue].SetValue(choice.Value)
			m.mode = m.pickerBack
			m.pickerBack = modeNone
			m.status = "due updated"
			return m, m.focusTaskFormField(taskFieldDue)
		default:
			return m, nil
		}
	}

	if m.mode == modeResourcePicker {
		items := m.visibleResourcePickerItems()
		switch msg.String() {
		case "esc":
			m.mode = m.resourcePickerBack
			m.resourcePickerFilter.Blur()
			m.resourcePickerFilter.SetValue("")
			m.status = "resource picker cancelled"
			return m, nil
		case "j", "down":
			if m.resourcePickerIndex < len(items)-1 {
				m.resourcePickerIndex++
			}
			return m, nil
		case "k", "up":
			if m.resourcePickerIndex > 0 {
				m.resourcePickerIndex--
			}
			return m, nil
		case "h", "left":
			return m, m.openResourcePickerParent()
		case "backspace":
			if strings.TrimSpace(m.resourcePickerFilter.Value()) == "" {
				return m, m.openResourcePickerParent()
			}
			var cmd tea.Cmd
			m.resourcePickerFilter, cmd = m.resourcePickerFilter.Update(msg)
			m.resourcePickerIndex = 0
			return m, cmd
		case "ctrl+u":
			m.resourcePickerFilter.SetValue("")
			m.resourcePickerFilter.CursorEnd()
			m.resourcePickerIndex = 0
			return m, nil
		case "l", "right":
			entry, ok := m.selectedResourcePickerEntry()
			if !ok || !entry.IsDir {
				return m, nil
			}
			return m, m.openResourcePickerDir(entry.Path)
		case "a":
			return m, m.attachSelectedResourceEntry()
		case "enter":
			entry, ok := m.selectedResourcePickerEntry()
			if !ok {
				return m, nil
			}
			if entry.IsDir {
				return m, m.openResourcePickerDir(entry.Path)
			}
			return m, m.attachSelectedResourceEntry()
		default:
			var cmd tea.Cmd
			before := m.resourcePickerFilter.Value()
			m.resourcePickerFilter, cmd = m.resourcePickerFilter.Update(msg)
			if m.resourcePickerFilter.Value() != before {
				m.resourcePickerIndex = 0
			}
			return m, cmd
		}
	}

	if m.mode == modeLabelPicker {
		switch msg.String() {
		case "esc":
			m.mode = m.labelPickerBack
			m.status = "label picker cancelled"
			if m.mode == modeAddTask || m.mode == modeEditTask {
				return m, m.focusTaskFormField(taskFieldLabels)
			}
			return m, nil
		case "j", "down":
			if m.labelPickerIndex < len(m.labelPickerItems)-1 {
				m.labelPickerIndex++
			}
			return m, nil
		case "k", "up":
			if m.labelPickerIndex > 0 {
				m.labelPickerIndex--
			}
			return m, nil
		case "enter":
			if len(m.labelPickerItems) == 0 || len(m.formInputs) <= taskFieldLabels {
				m.mode = m.labelPickerBack
				return m, m.focusTaskFormField(taskFieldLabels)
			}
			item := m.labelPickerItems[clamp(m.labelPickerIndex, 0, len(m.labelPickerItems)-1)]
			m.appendTaskFormLabel(item.Label)
			m.mode = m.labelPickerBack
			m.status = "label added from inheritance"
			return m, m.focusTaskFormField(taskFieldLabels)
		default:
			return m, nil
		}
	}

	if m.mode == modePathsRoots {
		switch {
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			m.mode = modeNone
			m.pathsRootInput.Blur()
			m.status = "paths/roots cancelled"
			return m, nil
		case msg.String() == "ctrl+r":
			return m, m.startResourcePicker("", modePathsRoots)
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			return m.submitPathsRoots()
		default:
			var cmd tea.Cmd
			m.pathsRootInput, cmd = m.pathsRootInput.Update(msg)
			return m, cmd
		}
	}

	if m.mode == modeAddTask || m.mode == modeEditTask {
		switch {
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			m.mode = modeNone
			m.formInputs = nil
			m.formFocus = 0
			m.editingTaskID = ""
			m.taskFormParentID = ""
			m.taskFormKind = domain.WorkKindTask
			m.taskFormResourceRefs = nil
			m.status = "cancelled"
			return m, nil
		case msg.Code == tea.KeyTab || msg.String() == "tab" || msg.String() == "ctrl+i" || msg.String() == "down":
			return m, m.focusTaskFormField(m.formFocus + 1)
		case msg.String() == "shift+tab" || msg.String() == "backtab" || msg.String() == "up":
			return m, m.focusTaskFormField(m.formFocus - 1)
		case msg.String() == "ctrl+l":
			if m.formFocus == taskFieldLabels {
				return m, m.startLabelPicker()
			}
			return m, nil
		case isCtrlY(msg):
			if m.formFocus == taskFieldLabels {
				if m.acceptCurrentLabelSuggestion() {
					m.status = "accepted label suggestion"
				} else {
					m.status = "no label suggestion"
				}
			}
			return m, nil
		case msg.String() == "ctrl+r":
			back := m.mode
			taskID := ""
			if back == modeEditTask {
				taskID = strings.TrimSpace(m.editingTaskID)
				if taskID == "" {
					task, ok := m.selectedTaskInCurrentColumn()
					if !ok {
						m.status = "no task selected"
						return m, nil
					}
					taskID = task.ID
				}
			}
			return m, m.startResourcePicker(taskID, back)
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			return m.submitInputMode()
		default:
			if m.formFocus == taskFieldPriority {
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
			if m.formFocus == taskFieldDue && (msg.String() == "ctrl+d" || msg.String() == "D") {
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
		case msg.String() == "ctrl+r":
			if m.projectFormFocus == projectFieldRootPath {
				return m, m.startResourcePicker("", m.mode)
			}
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

	if m.mode == modeLabelsConfig {
		switch {
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			m.mode = modeNone
			m.labelsConfigInputs = nil
			m.labelsConfigFocus = 0
			m.labelsConfigSlug = ""
			m.status = "cancelled"
			return m, nil
		case msg.Code == tea.KeyTab || msg.String() == "tab" || msg.String() == "ctrl+i" || msg.String() == "down":
			return m, m.focusLabelsConfigField(m.labelsConfigFocus + 1)
		case msg.String() == "shift+tab" || msg.String() == "backtab" || msg.String() == "up":
			return m, m.focusLabelsConfigField(m.labelsConfigFocus - 1)
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			return m.submitInputMode()
		default:
			if len(m.labelsConfigInputs) == 0 {
				return m, nil
			}
			var cmd tea.Cmd
			m.labelsConfigInputs[m.labelsConfigFocus], cmd = m.labelsConfigInputs[m.labelsConfigFocus].Update(msg)
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

// isCtrlY reports whether a keypress represents the Ctrl+Y autocomplete shortcut.
func isCtrlY(msg tea.KeyPressMsg) bool {
	if msg.String() == "ctrl+y" {
		return true
	}
	if (msg.Mod & tea.ModCtrl) == 0 {
		return false
	}
	if msg.Code == 'y' || msg.Code == 'Y' {
		return true
	}
	return strings.EqualFold(msg.Text, "y")
}

// submitInputMode submits input mode.
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
		if err := m.validateAllowedLabels(labels); err != nil {
			m.status = err.Error()
			return m, nil
		}
		metadata := m.buildTaskMetadataFromForm(vals, domain.TaskMetadata{})
		parentID := m.taskFormParentID
		kind := m.taskFormKind

		m.mode = modeNone
		m.formInputs = nil
		m.taskFormParentID = ""
		m.taskFormKind = domain.WorkKindTask
		m.taskFormResourceRefs = nil
		return m.createTask(app.CreateTaskInput{
			ParentID:    parentID,
			Kind:        kind,
			Title:       title,
			Description: vals["description"],
			Priority:    priority,
			DueAt:       dueAt,
			Labels:      labels,
			Metadata:    metadata,
		})
	case modeSearch:
		return m, m.applySearchFilter()
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
			m.taskFormResourceRefs = nil
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
		if err := m.validateAllowedLabels(labels); err != nil {
			m.status = err.Error()
			return m, nil
		}
		metadata := m.buildTaskMetadataFromForm(vals, task.Metadata)

		m.mode = modeNone
		m.formInputs = nil
		m.editingTaskID = ""
		m.taskFormParentID = ""
		m.taskFormKind = domain.WorkKindTask
		m.taskFormResourceRefs = nil
		in := app.UpdateTaskInput{
			TaskID:      taskID,
			Title:       title,
			Description: description,
			Priority:    priority,
			DueAt:       dueAt,
			Labels:      labels,
			Metadata:    &metadata,
		}
		return m, func() tea.Msg {
			_, updateErr := m.svc.UpdateTask(context.Background(), in)
			if updateErr != nil {
				return actionMsg{err: updateErr}
			}
			return actionMsg{status: "task updated", reload: true}
		}
	case modeLabelsConfig:
		if len(m.labelsConfigInputs) < 2 {
			m.status = "labels config unavailable"
			return m, nil
		}
		slug := strings.TrimSpace(strings.ToLower(m.labelsConfigSlug))
		if slug == "" {
			m.status = "project slug is empty"
			return m, nil
		}
		if m.saveLabels == nil {
			m.status = "save labels failed: callback unavailable"
			return m, nil
		}
		globalLabels := normalizeConfigLabels(parseLabelsInput(m.labelsConfigInputs[0].Value(), nil))
		projectLabels := normalizeConfigLabels(parseLabelsInput(m.labelsConfigInputs[1].Value(), nil))

		m.allowedLabelGlobal = append([]string(nil), globalLabels...)
		if len(projectLabels) == 0 {
			delete(m.allowedLabelProject, slug)
		} else {
			m.allowedLabelProject[slug] = append([]string(nil), projectLabels...)
		}
		m.refreshTaskFormLabelSuggestions()
		m.mode = modeNone
		m.labelsConfigInputs = nil
		m.labelsConfigFocus = 0
		m.labelsConfigSlug = ""
		return m, func() tea.Msg {
			if err := m.saveLabels(slug, globalLabels, projectLabels); err != nil {
				return actionMsg{err: err}
			}
			return actionMsg{status: "labels config saved"}
		}
	case modeAddProject, modeEditProject:
		isAdd := m.mode == modeAddProject
		vals := m.projectFormValues()
		name := vals["name"]
		if name == "" {
			m.status = "project name required"
			return m, nil
		}
		rootPath, err := normalizeProjectRootPathInput(vals["root_path"])
		if err != nil {
			m.status = err.Error()
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
				if m.saveProjectRoot != nil {
					if err := m.saveProjectRoot(project.Slug, rootPath); err != nil {
						return actionMsg{err: err}
					}
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
			if m.saveProjectRoot != nil {
				if err := m.saveProjectRoot(project.Slug, rootPath); err != nil {
					return actionMsg{err: err}
				}
			}
			return actionMsg{status: "project updated", reload: true, projectID: project.ID}
		}
	default:
		return m, nil
	}
}

// executeCommandPalette executes command palette.
func (m Model) executeCommandPalette(command string) (tea.Model, tea.Cmd) {
	switch command {
	case "":
		m.status = "no command"
		return m, nil
	case "new-task", "task-new":
		return m, m.startTaskForm(nil)
	case "new-subtask", "task-subtask":
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		return m, m.startSubtaskForm(task)
	case "edit-task", "task-edit":
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		return m, m.startTaskForm(&task)
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
		return m, m.startSearchMode()
	case "search-project":
		m.searchCrossProject = false
		return m, m.startSearchMode()
	case "clear-query", "clear-search-query":
		return m, m.clearSearchQuery()
	case "reset-filters", "clear-search":
		return m, m.resetSearchFilters()
	case "toggle-archived":
		m.showArchived = !m.showArchived
		m.selectedTask = 0
		m.clearSelection()
		if m.showArchived {
			m.status = "showing archived tasks"
		} else {
			m.status = "hiding archived tasks"
		}
		return m, m.loadData
	case "focus-subtree", "zoom-task":
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		m.projectionRootTaskID = task.ID
		m.focusTaskByID(task.ID)
		m.status = "focused subtree"
		return m, nil
	case "focus-clear", "zoom-reset":
		if m.projectionRootTaskID == "" {
			m.status = "full board already visible"
			return m, nil
		}
		m.projectionRootTaskID = ""
		m.status = "full board view"
		return m, nil
	case "toggle-select", "select-task":
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		if m.toggleTaskSelection(task.ID) {
			m.status = fmt.Sprintf("selected %q (%d total)", truncate(task.Title, 28), len(m.selectedTaskIDs))
			m.appendActivity(activityEntry{
				At:      time.Now().UTC(),
				Summary: "select task",
				Target:  task.Title,
			})
		} else {
			m.status = fmt.Sprintf("unselected %q (%d total)", truncate(task.Title, 28), len(m.selectedTaskIDs))
			m.appendActivity(activityEntry{
				At:      time.Now().UTC(),
				Summary: "unselect task",
				Target:  task.Title,
			})
		}
		return m, nil
	case "clear-selection", "selection-clear":
		count := m.clearSelection()
		if count == 0 {
			m.status = "selection already empty"
			return m, nil
		}
		m.status = fmt.Sprintf("cleared %d selected tasks", count)
		m.appendActivity(activityEntry{
			At:      time.Now().UTC(),
			Summary: "clear selection",
			Target:  fmt.Sprintf("%d tasks", count),
		})
		return m, nil
	case "bulk-move-left", "move-left-selected":
		return m.moveSelectedTasks(-1)
	case "bulk-move-right", "move-right-selected":
		return m.moveSelectedTasks(1)
	case "bulk-archive", "archive-selected":
		return m.confirmBulkDeleteAction(app.DeleteModeArchive, m.confirmArchive, "archive selected")
	case "bulk-delete", "delete-selected":
		return m.confirmBulkDeleteAction(app.DeleteModeHard, m.confirmHardDelete, "hard delete selected")
	case "undo":
		return m.undoLastMutation()
	case "redo":
		return m.redoLastMutation()
	case "reload-config", "config-reload", "reload":
		m.status = "reloading config..."
		return m, m.reloadRuntimeConfigCmd()
	case "paths-roots", "roots", "project-root":
		return m, m.startPathsRootsMode()
	case "labels-config", "labels", "edit-labels":
		return m, m.startLabelsConfigForm()
	case "activity-log", "log":
		return m, m.openActivityLog()
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

// quickActions returns state-aware quick actions with enabled entries first.
func (m Model) quickActions() []quickActionItem {
	_, hasTask := m.selectedTaskInCurrentColumn()
	hasSelection := len(m.selectedTaskIDs) > 0
	enabled := make([]quickActionItem, 0, len(quickActionSpecs))
	disabled := make([]quickActionItem, 0, len(quickActionSpecs))
	for _, spec := range quickActionSpecs {
		available, reason := m.quickActionAvailability(spec.ID, hasTask, hasSelection)
		item := quickActionItem{
			ID:             spec.ID,
			Label:          spec.Label,
			Enabled:        available,
			DisabledReason: reason,
		}
		if item.Enabled {
			enabled = append(enabled, item)
			continue
		}
		disabled = append(disabled, item)
	}
	return append(enabled, disabled...)
}

// quickActionAvailability returns whether one quick action can run in the current state.
func (m Model) quickActionAvailability(actionID string, hasTask bool, hasSelection bool) (bool, string) {
	switch actionID {
	case "task-info", "edit-task", "archive-task", "hard-delete", "toggle-selection":
		if !hasTask {
			return false, "no task selected"
		}
		return true, ""
	case "move-left":
		if !hasTask {
			return false, "no task selected"
		}
		if m.selectedColumn <= 0 {
			return false, "already at first column"
		}
		return true, ""
	case "move-right":
		if !hasTask {
			return false, "no task selected"
		}
		if m.selectedColumn >= len(m.columns)-1 {
			return false, "already at last column"
		}
		return true, ""
	case "clear-selection":
		if !hasSelection {
			return false, "selection already empty"
		}
		return true, ""
	case "bulk-move-left":
		if !hasSelection {
			return false, "no tasks selected"
		}
		if len(m.buildMoveSteps(m.sortedSelectedTaskIDs(), -1)) == 0 {
			return false, "no movable tasks selected"
		}
		return true, ""
	case "bulk-move-right":
		if !hasSelection {
			return false, "no tasks selected"
		}
		if len(m.buildMoveSteps(m.sortedSelectedTaskIDs(), 1)) == 0 {
			return false, "no movable tasks selected"
		}
		return true, ""
	case "bulk-archive", "bulk-hard-delete":
		if !hasSelection {
			return false, "no tasks selected"
		}
		return true, ""
	case "undo":
		if len(m.undoStack) == 0 {
			return false, "nothing to undo"
		}
		return true, ""
	case "redo":
		if len(m.redoStack) == 0 {
			return false, "nothing to redo"
		}
		return true, ""
	case "activity-log":
		return true, ""
	default:
		return false, "unknown action"
	}
}

// applyQuickAction applies the currently focused quick action when available.
func (m Model) applyQuickAction() (tea.Model, tea.Cmd) {
	actions := m.quickActions()
	if len(actions) == 0 {
		m.status = "no quick actions"
		return m, nil
	}
	idx := clamp(m.quickActionIndex, 0, len(actions)-1)
	action := actions[idx]
	if !action.Enabled {
		reason := strings.TrimSpace(action.DisabledReason)
		if reason == "" {
			reason = "unavailable"
		}
		m.status = strings.ToLower(action.Label) + " unavailable: " + reason
		return m, nil
	}

	m.mode = modeNone
	switch action.ID {
	case "task-info":
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		m.mode = modeTaskInfo
		m.taskInfoTaskID = task.ID
		m.taskInfoSubtaskIdx = 0
		m.status = "task info"
		return m, nil
	case "edit-task":
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		return m, m.startTaskForm(&task)
	case "move-left":
		return m.moveSelectedTask(-1)
	case "move-right":
		return m.moveSelectedTask(1)
	case "archive-task":
		return m.confirmDeleteAction(app.DeleteModeArchive, m.confirmArchive, "archive task")
	case "hard-delete":
		return m.confirmDeleteAction(app.DeleteModeHard, m.confirmHardDelete, "hard delete task")
	case "toggle-selection":
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		if m.toggleTaskSelection(task.ID) {
			m.status = fmt.Sprintf("selected %q (%d total)", truncate(task.Title, 28), len(m.selectedTaskIDs))
		} else {
			m.status = fmt.Sprintf("unselected %q (%d total)", truncate(task.Title, 28), len(m.selectedTaskIDs))
		}
		return m, nil
	case "clear-selection":
		count := m.clearSelection()
		if count == 0 {
			m.status = "selection already empty"
			return m, nil
		}
		m.status = fmt.Sprintf("cleared %d selected tasks", count)
		return m, nil
	case "bulk-move-left":
		return m.moveSelectedTasks(-1)
	case "bulk-move-right":
		return m.moveSelectedTasks(1)
	case "bulk-archive":
		return m.confirmBulkDeleteAction(app.DeleteModeArchive, m.confirmArchive, "archive selected")
	case "bulk-hard-delete":
		return m.confirmBulkDeleteAction(app.DeleteModeHard, m.confirmHardDelete, "hard delete selected")
	case "undo":
		return m.undoLastMutation()
	case "redo":
		return m.redoLastMutation()
	case "activity-log":
		return m, m.openActivityLog()
	default:
		m.status = "unknown quick action"
		return m, nil
	}
}

// createTask creates task.
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
		task, err := m.svc.CreateTask(context.Background(), in)
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{status: "task created", reload: true, focusTaskID: task.ID}
	}
}

// moveSelectedTask moves the currently focused task one column left/right.
func (m Model) moveSelectedTask(delta int) (tea.Model, tea.Cmd) {
	task, ok := m.selectedTaskInCurrentColumn()
	if !ok {
		m.status = "no task selected"
		return m, nil
	}
	return m.moveTaskIDs([]string{task.ID}, delta, "move task", task.Title, false)
}

// moveSelectedTasks moves every selected task one column left/right.
func (m Model) moveSelectedTasks(delta int) (tea.Model, tea.Cmd) {
	taskIDs := m.sortedSelectedTaskIDs()
	if len(taskIDs) == 0 {
		m.status = "no tasks selected"
		return m, nil
	}
	label := "bulk move right"
	if delta < 0 {
		label = "bulk move left"
	}
	return m.moveTaskIDs(taskIDs, delta, label, fmt.Sprintf("%d tasks", len(taskIDs)), true)
}

// moveTaskIDs moves the provided task ids and records undo/redo history.
func (m Model) moveTaskIDs(taskIDs []string, delta int, label, target string, bulk bool) (tea.Model, tea.Cmd) {
	steps := m.buildMoveSteps(taskIDs, delta)
	if len(steps) == 0 {
		m.status = "no movable tasks selected"
		return m, nil
	}
	direction := "right"
	if delta < 0 {
		direction = "left"
	}
	status := "task moved"
	if bulk {
		status = fmt.Sprintf("moved %d tasks %s", len(steps), direction)
	}
	focusTaskID := steps[0].TaskID
	if bulk {
		focusTaskID = ""
	}
	history := historyActionSet{
		Label:    label,
		Summary:  status,
		Target:   target,
		Steps:    append([]historyStep(nil), steps...),
		Undoable: true,
		At:       time.Now().UTC(),
	}
	activity := activityEntry{
		At:      history.At,
		Summary: label,
		Target:  target,
	}
	return m, func() tea.Msg {
		for _, step := range steps {
			if _, err := m.svc.MoveTask(context.Background(), step.TaskID, step.ToColumnID, step.ToPosition); err != nil {
				return actionMsg{err: err}
			}
		}
		return actionMsg{
			status:       status,
			reload:       true,
			focusTaskID:  focusTaskID,
			historyPush:  &history,
			activityItem: &activity,
		}
	}
}

// deleteSelectedTask deletes or archives the currently focused task.
func (m Model) deleteSelectedTask(mode app.DeleteMode) (tea.Model, tea.Cmd) {
	task, ok := m.selectedTaskInCurrentColumn()
	if !ok {
		m.status = "no task selected"
		return m, nil
	}
	return m.deleteTaskIDs([]string{task.ID}, mode)
}

// deleteTaskIDs archives/deletes task ids and records undo metadata when possible.
func (m Model) deleteTaskIDs(taskIDs []string, mode app.DeleteMode) (tea.Model, tea.Cmd) {
	ids := m.normalizeKnownTaskIDs(taskIDs)
	if len(ids) == 0 {
		m.status = "no tasks selected"
		return m, nil
	}
	undoable := mode != app.DeleteModeHard
	label := "archive task"
	if mode == app.DeleteModeHard {
		label = "hard delete task"
	}
	if len(ids) > 1 {
		if mode == app.DeleteModeHard {
			label = "bulk hard delete"
		} else {
			label = "bulk archive"
		}
	}
	status := "task archived"
	if mode == app.DeleteModeHard {
		status = "task deleted"
	}
	if len(ids) > 1 {
		if mode == app.DeleteModeHard {
			status = fmt.Sprintf("deleted %d tasks", len(ids))
		} else {
			status = fmt.Sprintf("archived %d tasks", len(ids))
		}
	}

	steps := make([]historyStep, 0, len(ids))
	for _, taskID := range ids {
		step := historyStep{TaskID: taskID}
		if mode == app.DeleteModeHard {
			step.Kind = historyStepHardDelete
		} else {
			step.Kind = historyStepArchive
		}
		steps = append(steps, step)
	}
	history := historyActionSet{
		Label:    label,
		Summary:  status,
		Target:   fmt.Sprintf("%d tasks", len(ids)),
		Steps:    steps,
		Undoable: undoable,
		At:       time.Now().UTC(),
	}
	activity := activityEntry{
		At:      history.At,
		Summary: label,
		Target:  fmt.Sprintf("%d tasks", len(ids)),
	}
	if mode == app.DeleteModeArchive {
		m.lastArchivedTaskID = ids[len(ids)-1]
	}
	return m, func() tea.Msg {
		for _, taskID := range ids {
			if err := m.svc.DeleteTask(context.Background(), taskID, mode); err != nil {
				return actionMsg{err: err}
			}
		}
		return actionMsg{
			status:       status,
			reload:       true,
			clearTaskIDs: ids,
			historyPush:  &history,
			activityItem: &activity,
		}
	}
}

// confirmDeleteAction opens a confirmation modal when configured, or executes directly.
func (m Model) confirmDeleteAction(mode app.DeleteMode, needsConfirm bool, label string) (tea.Model, tea.Cmd) {
	task, ok := m.selectedTaskInCurrentColumn()
	if !ok {
		m.status = "no task selected"
		return m, nil
	}
	label = strings.TrimSpace(label)
	if label == "" {
		label = "delete task"
	}
	if !needsConfirm {
		return m.deleteTaskIDs([]string{task.ID}, mode)
	}
	m.mode = modeConfirmAction
	m.pendingConfirm = confirmAction{
		Kind:    "delete",
		Task:    task,
		TaskIDs: []string{task.ID},
		Mode:    mode,
		Label:   label,
	}
	m.confirmChoice = 1
	m.status = "confirm action"
	return m, nil
}

// confirmBulkDeleteAction confirms and applies bulk archive/hard-delete operations.
func (m Model) confirmBulkDeleteAction(mode app.DeleteMode, needsConfirm bool, label string) (tea.Model, tea.Cmd) {
	taskIDs := m.sortedSelectedTaskIDs()
	if len(taskIDs) == 0 {
		m.status = "no tasks selected"
		return m, nil
	}
	if !needsConfirm {
		return m.deleteTaskIDs(taskIDs, mode)
	}
	task, _ := m.taskByID(taskIDs[0])
	m.mode = modeConfirmAction
	m.pendingConfirm = confirmAction{
		Kind:    "delete",
		Task:    task,
		TaskIDs: taskIDs,
		Mode:    mode,
		Label:   label,
	}
	m.confirmChoice = 1
	m.status = "confirm action"
	return m, nil
}

// restoreTask restores the most-recent archived task or selected archived task.
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
	return m.restoreTaskIDs([]string{taskID}, "task restored", "restore task")
}

// restoreTaskIDs restores tasks and records undo history.
func (m Model) restoreTaskIDs(taskIDs []string, status, label string) (tea.Model, tea.Cmd) {
	ids := make([]string, 0, len(taskIDs))
	for _, taskID := range taskIDs {
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			continue
		}
		ids = append(ids, taskID)
	}
	if len(ids) == 0 {
		m.status = "nothing to restore"
		return m, nil
	}
	steps := make([]historyStep, 0, len(ids))
	for _, taskID := range ids {
		steps = append(steps, historyStep{
			Kind:   historyStepRestore,
			TaskID: taskID,
		})
	}
	history := historyActionSet{
		Label:    label,
		Summary:  status,
		Target:   fmt.Sprintf("%d tasks", len(ids)),
		Steps:    steps,
		Undoable: true,
		At:       time.Now().UTC(),
	}
	activity := activityEntry{
		At:      history.At,
		Summary: label,
		Target:  fmt.Sprintf("%d tasks", len(ids)),
	}
	return m, func() tea.Msg {
		for _, taskID := range ids {
			if _, err := m.svc.RestoreTask(context.Background(), taskID); err != nil {
				return actionMsg{err: err}
			}
		}
		return actionMsg{
			status:       status,
			reload:       true,
			historyPush:  &history,
			activityItem: &activity,
		}
	}
}

// confirmRestoreAction opens restore confirmation when configured, or executes directly.
func (m Model) confirmRestoreAction() (tea.Model, tea.Cmd) {
	task, ok := m.selectedTaskInCurrentColumn()
	if ok && task.ArchivedAt == nil {
		ok = false
	}
	if !m.confirmRestore || !ok {
		return m.restoreTask()
	}
	m.mode = modeConfirmAction
	m.pendingConfirm = confirmAction{
		Kind:    "restore",
		Task:    task,
		TaskIDs: []string{task.ID},
		Mode:    app.DeleteModeArchive,
		Label:   "restore task",
	}
	m.confirmChoice = 1
	m.status = "confirm action"
	return m, nil
}

// applyConfirmedAction executes a previously confirmed action.
func (m Model) applyConfirmedAction(action confirmAction) (tea.Model, tea.Cmd) {
	switch action.Kind {
	case "delete":
		taskIDs := action.TaskIDs
		if len(taskIDs) == 0 && strings.TrimSpace(action.Task.ID) != "" {
			taskIDs = []string{action.Task.ID}
		}
		return m.deleteTaskIDs(taskIDs, action.Mode)
	case "restore":
		taskIDs := action.TaskIDs
		if len(taskIDs) == 0 && strings.TrimSpace(action.Task.ID) != "" {
			taskIDs = []string{action.Task.ID}
		}
		return m.restoreTaskIDs(taskIDs, "task restored", "restore task")
	default:
		m.status = "unknown confirm action"
		return m, nil
	}
}

// handleMouseWheel handles mouse wheel.
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

// handleMouseClick handles mouse click.
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

// clampSelections clamps selections.
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

// retainSelectionForLoadedTasks drops selected task ids that are no longer loaded.
func (m *Model) retainSelectionForLoadedTasks() {
	if len(m.selectedTaskIDs) == 0 {
		return
	}
	known := map[string]struct{}{}
	for _, task := range m.tasks {
		known[task.ID] = struct{}{}
	}
	for taskID := range m.selectedTaskIDs {
		if _, ok := known[taskID]; !ok {
			delete(m.selectedTaskIDs, taskID)
		}
	}
}

// isTaskSelected reports whether a task id is currently in the multi-select set.
func (m Model) isTaskSelected(taskID string) bool {
	_, ok := m.selectedTaskIDs[strings.TrimSpace(taskID)]
	return ok
}

// toggleTaskSelection adds/removes a task id from the current selection.
func (m *Model) toggleTaskSelection(taskID string) bool {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return false
	}
	if m.selectedTaskIDs == nil {
		m.selectedTaskIDs = map[string]struct{}{}
	}
	if _, ok := m.selectedTaskIDs[taskID]; ok {
		delete(m.selectedTaskIDs, taskID)
		return false
	}
	m.selectedTaskIDs[taskID] = struct{}{}
	return true
}

// clearSelection clears all selected task ids and returns the previous count.
func (m *Model) clearSelection() int {
	count := len(m.selectedTaskIDs)
	if count == 0 {
		return 0
	}
	m.selectedTaskIDs = map[string]struct{}{}
	return count
}

// unselectTasks removes provided task ids from multi-select state.
func (m *Model) unselectTasks(taskIDs []string) int {
	if len(m.selectedTaskIDs) == 0 {
		return 0
	}
	removed := 0
	for _, taskID := range taskIDs {
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			continue
		}
		if _, ok := m.selectedTaskIDs[taskID]; !ok {
			continue
		}
		delete(m.selectedTaskIDs, taskID)
		removed++
	}
	return removed
}

// sortedSelectedTaskIDs returns selected ids in board display order.
func (m Model) sortedSelectedTaskIDs() []string {
	if len(m.selectedTaskIDs) == 0 {
		return nil
	}
	taskIDs := make([]string, 0, len(m.selectedTaskIDs))
	for taskID := range m.selectedTaskIDs {
		taskIDs = append(taskIDs, taskID)
	}
	return m.normalizeKnownTaskIDs(taskIDs)
}

// normalizeKnownTaskIDs returns deduplicated task ids in deterministic board order.
func (m Model) normalizeKnownTaskIDs(taskIDs []string) []string {
	if len(taskIDs) == 0 {
		return nil
	}
	needed := map[string]struct{}{}
	for _, taskID := range taskIDs {
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			continue
		}
		needed[taskID] = struct{}{}
	}
	if len(needed) == 0 {
		return nil
	}
	out := make([]string, 0, len(needed))
	seen := map[string]struct{}{}
	for _, column := range m.columns {
		for _, task := range m.tasksForColumn(column.ID) {
			if _, ok := needed[task.ID]; !ok {
				continue
			}
			if _, ok := seen[task.ID]; ok {
				continue
			}
			seen[task.ID] = struct{}{}
			out = append(out, task.ID)
		}
	}
	for _, taskID := range taskIDs {
		if _, ok := seen[taskID]; ok {
			continue
		}
		if _, ok := m.taskByID(taskID); !ok {
			continue
		}
		seen[taskID] = struct{}{}
		out = append(out, taskID)
	}
	return out
}

// appendActivity appends one item to the in-app activity log with bounded retention.
func (m *Model) appendActivity(entry activityEntry) {
	if strings.TrimSpace(entry.Summary) == "" {
		return
	}
	if entry.At.IsZero() {
		entry.At = time.Now().UTC()
	}
	if strings.TrimSpace(entry.Target) == "" {
		entry.Target = "-"
	}
	m.activityLog = append(m.activityLog, entry)
	if len(m.activityLog) > activityLogMaxItems {
		m.activityLog = append([]activityEntry(nil), m.activityLog[len(m.activityLog)-activityLogMaxItems:]...)
	}
}

// pushUndoHistory records one user mutation and clears redo history.
func (m *Model) pushUndoHistory(set historyActionSet) {
	if len(set.Steps) == 0 {
		return
	}
	m.nextHistoryID++
	set.ID = m.nextHistoryID
	if set.At.IsZero() {
		set.At = time.Now().UTC()
	}
	m.undoStack = append(m.undoStack, set)
	const maxItems = 100
	if len(m.undoStack) > maxItems {
		m.undoStack = append([]historyActionSet(nil), m.undoStack[len(m.undoStack)-maxItems:]...)
	}
	m.redoStack = nil
}

// applyUndoTransition shifts one action from undo stack to redo stack after success.
func (m *Model) applyUndoTransition(set historyActionSet) {
	if len(m.undoStack) > 0 {
		m.undoStack = m.undoStack[:len(m.undoStack)-1]
	}
	m.redoStack = append(m.redoStack, set)
}

// applyRedoTransition shifts one action from redo stack back to undo stack after success.
func (m *Model) applyRedoTransition(set historyActionSet) {
	if len(m.redoStack) > 0 {
		m.redoStack = m.redoStack[:len(m.redoStack)-1]
	}
	m.undoStack = append(m.undoStack, set)
}

// undoLastMutation reverses the most recent undoable mutation set.
func (m Model) undoLastMutation() (tea.Model, tea.Cmd) {
	if len(m.undoStack) == 0 {
		m.status = "nothing to undo"
		return m, nil
	}
	set := m.undoStack[len(m.undoStack)-1]
	if !set.Undoable {
		m.undoStack = m.undoStack[:len(m.undoStack)-1]
		m.status = "last action cannot be undone"
		m.appendActivity(activityEntry{
			At:      time.Now().UTC(),
			Summary: "undo unavailable",
			Target:  set.Label,
		})
		return m, nil
	}
	return m, m.executeHistorySet(set, true)
}

// redoLastMutation reapplies the most recently undone mutation set.
func (m Model) redoLastMutation() (tea.Model, tea.Cmd) {
	if len(m.redoStack) == 0 {
		m.status = "nothing to redo"
		return m, nil
	}
	set := m.redoStack[len(m.redoStack)-1]
	return m, m.executeHistorySet(set, false)
}

// executeHistorySet applies one history action set in either undo or redo direction.
func (m Model) executeHistorySet(set historyActionSet, undo bool) tea.Cmd {
	steps := append([]historyStep(nil), set.Steps...)
	if undo {
		slices.Reverse(steps)
	}
	return func() tea.Msg {
		clearIDs := make([]string, 0, len(steps))
		for _, step := range steps {
			switch step.Kind {
			case historyStepMove:
				columnID := step.ToColumnID
				position := step.ToPosition
				if undo {
					columnID = step.FromColumnID
					position = step.FromPosition
				}
				if _, err := m.svc.MoveTask(context.Background(), step.TaskID, columnID, position); err != nil {
					return actionMsg{err: err}
				}
			case historyStepArchive:
				if undo {
					if _, err := m.svc.RestoreTask(context.Background(), step.TaskID); err != nil {
						return actionMsg{err: err}
					}
				} else {
					if err := m.svc.DeleteTask(context.Background(), step.TaskID, app.DeleteModeArchive); err != nil {
						return actionMsg{err: err}
					}
					clearIDs = append(clearIDs, step.TaskID)
				}
			case historyStepRestore:
				if undo {
					if err := m.svc.DeleteTask(context.Background(), step.TaskID, app.DeleteModeArchive); err != nil {
						return actionMsg{err: err}
					}
					clearIDs = append(clearIDs, step.TaskID)
				} else {
					if _, err := m.svc.RestoreTask(context.Background(), step.TaskID); err != nil {
						return actionMsg{err: err}
					}
				}
			case historyStepHardDelete:
				if undo {
					return actionMsg{status: "undo failed: hard delete cannot be restored"}
				}
				if err := m.svc.DeleteTask(context.Background(), step.TaskID, app.DeleteModeHard); err != nil {
					return actionMsg{err: err}
				}
				clearIDs = append(clearIDs, step.TaskID)
			}
		}
		status := "redo complete"
		activitySummary := "redo"
		msg := actionMsg{
			reload:       true,
			clearTaskIDs: clearIDs,
			historyRedo:  &set,
		}
		if undo {
			status = "undo complete"
			activitySummary = "undo"
			msg.historyRedo = nil
			msg.historyUndo = &set
		}
		msg.status = fmt.Sprintf("%s: %s", status, set.Label)
		msg.activityItem = &activityEntry{
			At:      time.Now().UTC(),
			Summary: activitySummary,
			Target:  set.Label,
		}
		return msg
	}
}

// buildMoveSteps computes move history steps for task ids with deterministic ordering.
func (m Model) buildMoveSteps(taskIDs []string, delta int) []historyStep {
	if delta == 0 {
		return nil
	}
	ids := m.normalizeKnownTaskIDs(taskIDs)
	if len(ids) == 0 {
		return nil
	}
	colIndexByID := map[string]int{}
	for idx, column := range m.columns {
		colIndexByID[column.ID] = idx
	}
	steps := make([]historyStep, 0, len(ids))
	for _, taskID := range ids {
		task, ok := m.taskByID(taskID)
		if !ok {
			continue
		}
		fromColIdx, ok := colIndexByID[task.ColumnID]
		if !ok {
			continue
		}
		toColIdx := fromColIdx + delta
		if toColIdx < 0 || toColIdx >= len(m.columns) {
			continue
		}
		steps = append(steps, historyStep{
			Kind:         historyStepMove,
			TaskID:       task.ID,
			FromColumnID: task.ColumnID,
			FromPosition: task.Position,
			ToColumnID:   m.columns[toColIdx].ID,
		})
	}
	if len(steps) == 0 {
		return nil
	}
	sort.SliceStable(steps, func(i, j int) bool {
		iTask, _ := m.taskByID(steps[i].TaskID)
		jTask, _ := m.taskByID(steps[j].TaskID)
		if iTask.ColumnID == jTask.ColumnID {
			if iTask.Position == jTask.Position {
				return iTask.ID < jTask.ID
			}
			return iTask.Position < jTask.Position
		}
		return colIndexByID[iTask.ColumnID] < colIndexByID[jTask.ColumnID]
	})

	targetPosByColumn := map[string]int{}
	for _, step := range steps {
		if _, ok := targetPosByColumn[step.ToColumnID]; ok {
			continue
		}
		targetPosByColumn[step.ToColumnID] = len(m.tasksForColumn(step.ToColumnID))
	}
	for idx := range steps {
		steps[idx].ToPosition = targetPosByColumn[steps[idx].ToColumnID]
		targetPosByColumn[steps[idx].ToColumnID]++
	}
	return steps
}

// groupLabelForTask returns the swimlane/group label for a task under current settings.
func (m Model) groupLabelForTask(task domain.Task) string {
	switch normalizeBoardGroupBy(m.boardGroupBy) {
	case "priority":
		switch task.Priority {
		case domain.PriorityHigh:
			return "Priority: High"
		case domain.PriorityMedium:
			return "Priority: Medium"
		case domain.PriorityLow:
			return "Priority: Low"
		default:
			return "Priority: Unknown"
		}
	case "state":
		switch strings.ToLower(strings.TrimSpace(string(task.LifecycleState))) {
		case "todo":
			return "State: To Do"
		case "progress":
			return "State: In Progress"
		case "done":
			return "State: Done"
		case "archived":
			return "State: Archived"
		default:
			return "State: Unknown"
		}
	default:
		return "Tasks"
	}
}

// currentProjectID returns current project id.
func (m Model) currentProjectID() (string, bool) {
	if len(m.projects) == 0 {
		return "", false
	}
	idx := clamp(m.selectedProject, 0, len(m.projects)-1)
	return m.projects[idx].ID, true
}

// currentColumnID returns current column id.
func (m Model) currentColumnID() (string, bool) {
	if len(m.columns) == 0 {
		return "", false
	}
	idx := clamp(m.selectedColumn, 0, len(m.columns)-1)
	return m.columns[idx].ID, true
}

// currentProject returns the currently selected project.
func (m Model) currentProject() (domain.Project, bool) {
	if len(m.projects) == 0 {
		return domain.Project{}, false
	}
	idx := clamp(m.selectedProject, 0, len(m.projects)-1)
	return m.projects[idx], true
}

// currentColumnTasks returns current column tasks.
func (m Model) currentColumnTasks() []domain.Task {
	columnID, ok := m.currentColumnID()
	if !ok {
		return nil
	}
	return m.boardTasksForColumn(columnID)
}

// boardTasksForColumn returns only board-visible tasks for a column.
func (m Model) boardTasksForColumn(columnID string) []domain.Task {
	columnTasks := m.tasksForColumn(columnID)
	if len(columnTasks) == 0 {
		return nil
	}
	out := make([]domain.Task, 0, len(columnTasks))
	for _, task := range columnTasks {
		if task.Kind == domain.WorkKindSubtask {
			continue
		}
		out = append(out, task)
	}
	return out
}

// tasksForColumn handles tasks for column.
func (m Model) tasksForColumn(columnID string) []domain.Task {
	out := make([]domain.Task, 0)
	projected := m.projectedTaskSet()
	for _, task := range m.tasks {
		if task.ColumnID != columnID {
			continue
		}
		if len(projected) > 0 {
			if _, ok := projected[task.ID]; !ok {
				continue
			}
		}
		out = append(out, task)
	}
	ordered := orderTasksByHierarchy(out)
	groupBy := normalizeBoardGroupBy(m.boardGroupBy)
	if groupBy != "none" {
		sort.SliceStable(ordered, func(i, j int) bool {
			iRank := taskGroupRank(ordered[i], groupBy)
			jRank := taskGroupRank(ordered[j], groupBy)
			if iRank == jRank {
				return false
			}
			return iRank < jRank
		})
	}
	return ordered
}

// projectedTaskSet returns every task ID visible in focused subtree mode.
func (m Model) projectedTaskSet() map[string]struct{} {
	rootID := strings.TrimSpace(m.projectionRootTaskID)
	if rootID == "" {
		return nil
	}
	if _, ok := m.taskByID(rootID); !ok {
		return nil
	}
	childrenByParent := map[string][]string{}
	for _, task := range m.tasks {
		parentID := strings.TrimSpace(task.ParentID)
		if parentID == "" {
			continue
		}
		childrenByParent[parentID] = append(childrenByParent[parentID], task.ID)
	}
	visible := map[string]struct{}{}
	stack := []string{rootID}
	for len(stack) > 0 {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]
		if _, seen := visible[current]; seen {
			continue
		}
		visible[current] = struct{}{}
		// Depth-first traversal keeps the projection bounded to the selected root descendants.
		for _, childID := range childrenByParent[current] {
			stack = append(stack, childID)
		}
	}
	return visible
}

// projectionBreadcrumb returns the active subtree breadcrumb path.
func (m Model) projectionBreadcrumb() string {
	rootID := strings.TrimSpace(m.projectionRootTaskID)
	if rootID == "" {
		return ""
	}
	root, ok := m.taskByID(rootID)
	if !ok {
		return ""
	}
	path := []string{root.Title}
	visited := map[string]struct{}{root.ID: {}}
	parentID := strings.TrimSpace(root.ParentID)
	for parentID != "" {
		if _, seen := visited[parentID]; seen {
			break
		}
		parent, found := m.taskByID(parentID)
		if !found {
			break
		}
		visited[parentID] = struct{}{}
		path = append(path, parent.Title)
		parentID = strings.TrimSpace(parent.ParentID)
	}
	slices.Reverse(path)
	return strings.Join(path, " / ")
}

// dependencyRollupSummary returns compact project dependency totals for board rendering.
func (m Model) dependencyRollupSummary() string {
	rollup := m.dependencyRollup
	return fmt.Sprintf(
		"deps: total %d • blocked %d • unresolved %d • edges %d",
		rollup.TotalItems,
		rollup.BlockedItems,
		rollup.UnresolvedDependencyEdges,
		rollup.DependencyEdges,
	)
}

// taskGroupRank returns deterministic ordering rank for configured board grouping.
func taskGroupRank(task domain.Task, groupBy string) int {
	switch normalizeBoardGroupBy(groupBy) {
	case "priority":
		switch task.Priority {
		case domain.PriorityHigh:
			return 0
		case domain.PriorityMedium:
			return 1
		case domain.PriorityLow:
			return 2
		default:
			return 3
		}
	case "state":
		switch strings.ToLower(strings.TrimSpace(string(task.LifecycleState))) {
		case "todo":
			return 0
		case "progress":
			return 1
		case "done":
			return 2
		case "archived":
			return 3
		default:
			return 4
		}
	default:
		return 0
	}
}

// orderTasksByHierarchy renders parent items before their descendants.
func orderTasksByHierarchy(tasks []domain.Task) []domain.Task {
	if len(tasks) <= 1 {
		return tasks
	}
	childrenByParent := map[string][]domain.Task{}
	byID := map[string]domain.Task{}
	roots := make([]domain.Task, 0)
	for _, task := range tasks {
		byID[task.ID] = task
	}
	for _, task := range tasks {
		parentID := strings.TrimSpace(task.ParentID)
		if parentID == "" {
			roots = append(roots, task)
			continue
		}
		if _, ok := byID[parentID]; !ok {
			roots = append(roots, task)
			continue
		}
		childrenByParent[parentID] = append(childrenByParent[parentID], task)
	}
	sortTaskSlice(roots)
	for parentID := range childrenByParent {
		children := childrenByParent[parentID]
		sortTaskSlice(children)
		childrenByParent[parentID] = children
	}
	ordered := make([]domain.Task, 0, len(tasks))
	visited := map[string]struct{}{}
	var visit func(domain.Task)
	visit = func(task domain.Task) {
		if _, ok := visited[task.ID]; ok {
			return
		}
		visited[task.ID] = struct{}{}
		ordered = append(ordered, task)
		for _, child := range childrenByParent[task.ID] {
			visit(child)
		}
	}
	for _, root := range roots {
		visit(root)
	}
	for _, task := range tasks {
		if _, ok := visited[task.ID]; ok {
			continue
		}
		visit(task)
	}
	return ordered
}

// sortTaskSlice orders tasks by creation time (oldest-first) with deterministic fallbacks.
func sortTaskSlice(tasks []domain.Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		iCreated := tasks[i].CreatedAt
		jCreated := tasks[j].CreatedAt
		if !iCreated.IsZero() && !jCreated.IsZero() && !iCreated.Equal(jCreated) {
			return iCreated.Before(jCreated)
		}
		if tasks[i].Position != tasks[j].Position {
			return tasks[i].Position < tasks[j].Position
		}
		return tasks[i].ID < tasks[j].ID
	})
}

// taskDepth returns nesting depth for a task id with cycle protection.
func taskDepth(taskID string, parentByID map[string]string, depth int) int {
	if depth > 32 {
		return depth
	}
	parentID, ok := parentByID[taskID]
	if !ok || strings.TrimSpace(parentID) == "" {
		return depth
	}
	if _, exists := parentByID[parentID]; !exists {
		return depth + 1
	}
	return taskDepth(parentID, parentByID, depth+1)
}

// selectedTaskInCurrentColumn returns selected task in current column.
func (m Model) selectedTaskInCurrentColumn() (domain.Task, bool) {
	tasks := m.currentColumnTasks()
	if len(tasks) == 0 {
		return domain.Task{}, false
	}
	idx := clamp(m.selectedTask, 0, len(tasks)-1)
	return tasks[idx], true
}

// focusTaskByID focuses task by id.
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

// taskByID returns task by id.
func (m Model) taskByID(taskID string) (domain.Task, bool) {
	for _, task := range m.tasks {
		if task.ID == taskID {
			return task, true
		}
	}
	return domain.Task{}, false
}

// renderProjectTabs renders output for the current model state.
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

// projectAccentColor returns the project-specific accent color or the default accent.
func projectAccentColor(project domain.Project) color.Color {
	value := strings.TrimSpace(project.Metadata.Color)
	if value == "" {
		return lipgloss.Color("62")
	}
	return lipgloss.Color(value)
}

// renderOverviewPanel renders output for the current model state.
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

// renderInfoLine renders output for the current model state.
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

// renderHelpOverlay renders output for the current model state.
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
		"1. n add task  •  s add subtask  •  i/enter view task  •  e edit task",
		"2. space toggle select  •  . quick actions / : command palette for bulk actions",
		"3. [ ] move task across states  •  d/a/D/u actions use confirmations",
		"4. f focus selected subtree  •  F return full board  •  breadcrumb shows active focus",
		"5. z undo  •  Z redo  •  g activity log",
		"6. N new project  •  M edit project  •  p switch project",
		"7. / search: query -> states -> scope -> archived -> apply",
		"8. search hotkeys: ctrl+u clear query • ctrl+r reset filters",
		"9. task form: h/l priority  •  ctrl+d due picker  •  ctrl+l labels  •  ctrl+r resources",
		"10. task info: b edit dependencies  •  r attach file/dir from project root (or cwd fallback)",
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

// taskListSecondary returns task list secondary.
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

// taskIndexAtRow returns task index at row.
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

// cardMeta handles card meta.
func (m Model) cardMeta(task domain.Task) string {
	parts := make([]string, 0, 4)
	if m.taskFields.ShowPriority {
		parts = append(parts, string(task.Priority))
	}
	if task.Kind != domain.WorkKindSubtask {
		done, total := m.subtaskProgress(task.ID)
		if total > 0 {
			parts = append(parts, fmt.Sprintf("%d/%d", done, total))
		}
	}
	if m.taskFields.ShowDueDate && task.DueAt != nil {
		dueLabel := task.DueAt.UTC().Format("01-02")
		if task.DueAt.UTC().Before(time.Now().UTC()) {
			dueLabel = "!" + dueLabel
		}
		parts = append(parts, dueLabel)
	}
	if m.taskFields.ShowLabels && len(task.Labels) > 0 {
		parts = append(parts, summarizeLabels(task.Labels, 2))
	}
	if len(parts) == 0 {
		return ""
	}
	return "[" + strings.Join(parts, "|") + "]"
}

// taskDueWarning reports due warning text for one task in board/info contexts.
func (m Model) taskDueWarning(task domain.Task, now time.Time) string {
	if task.ArchivedAt != nil || task.DueAt == nil {
		return ""
	}
	now = now.UTC()
	due := task.DueAt.UTC()
	if due.Before(now) {
		return "warning: overdue"
	}
	maxWindow := time.Duration(0)
	for _, window := range m.dueSoonWindows {
		if window > maxWindow {
			maxWindow = window
		}
	}
	if maxWindow > 0 && due.Sub(now) <= maxWindow {
		return "warning: due soon"
	}
	return ""
}

// taskInfoTask resolves the task currently shown in the task-info modal.
func (m Model) taskInfoTask() (domain.Task, bool) {
	taskID := strings.TrimSpace(m.taskInfoTaskID)
	if taskID == "" {
		return m.selectedTaskInCurrentColumn()
	}
	return m.taskByID(taskID)
}

// stepBackTaskInfo moves task-info focus to the parent task when available.
func (m *Model) stepBackTaskInfo(task domain.Task) bool {
	parentID := strings.TrimSpace(task.ParentID)
	if parentID == "" {
		return false
	}
	if _, ok := m.taskByID(parentID); !ok {
		return false
	}
	m.taskInfoTaskID = parentID
	m.taskInfoSubtaskIdx = 0
	// Keep the cursor aligned to the child we navigated from when it remains visible.
	for idx, child := range m.subtasksForParent(parentID) {
		if child.ID == task.ID {
			m.taskInfoSubtaskIdx = idx
			break
		}
	}
	m.status = "parent task info"
	return true
}

// subtasksForParent returns direct subtask children for a parent task.
func (m Model) subtasksForParent(parentID string) []domain.Task {
	parentID = strings.TrimSpace(parentID)
	if parentID == "" {
		return nil
	}
	out := make([]domain.Task, 0)
	for _, task := range m.tasks {
		if strings.TrimSpace(task.ParentID) != parentID {
			continue
		}
		if task.Kind != domain.WorkKindSubtask {
			continue
		}
		if !m.showArchived && task.ArchivedAt != nil {
			continue
		}
		out = append(out, task)
	}
	sortTaskSlice(out)
	return out
}

// subtaskProgress returns completed/total direct subtasks for a parent task.
func (m Model) subtaskProgress(parentID string) (int, int) {
	subtasks := m.subtasksForParent(parentID)
	if len(subtasks) == 0 {
		return 0, 0
	}
	done := 0
	for _, task := range subtasks {
		if task.LifecycleState == domain.StateDone {
			done++
		}
	}
	return done, len(subtasks)
}

// dueCounts returns overdue and due-soon counts for loaded tasks.
func (m Model) dueCounts(now time.Time) (int, int) {
	if len(m.tasks) == 0 {
		return 0, 0
	}
	overdue := 0
	dueSoon := 0
	windows := append([]time.Duration(nil), m.dueSoonWindows...)
	sort.Slice(windows, func(i, j int) bool { return windows[i] < windows[j] })
	maxWindow := time.Duration(0)
	if len(windows) > 0 {
		maxWindow = windows[len(windows)-1]
	}
	for _, task := range m.tasks {
		if task.ArchivedAt != nil || task.DueAt == nil {
			continue
		}
		due := task.DueAt.UTC()
		if due.Before(now) {
			overdue++
			continue
		}
		if maxWindow > 0 && due.Sub(now) <= maxWindow {
			dueSoon++
		}
	}
	return overdue, dueSoon
}

// renderTaskDetails renders output for the current model state.
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
			due = formatDueValue(task.DueAt)
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

// renderModeOverlay renders output for the current model state.
func (m Model) renderModeOverlay(accent, muted, dim color.Color, helpStyle lipgloss.Style, maxWidth int) string {
	switch m.mode {
	case modeActivityLog:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 44, 96))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		lines := []string{titleStyle.Render("Activity Log")}
		if len(m.activityLog) == 0 {
			lines = append(lines, hintStyle.Render("(no activity yet)"))
		} else {
			rendered := 0
			for idx := len(m.activityLog) - 1; idx >= 0; idx-- {
				entry := m.activityLog[idx]
				lines = append(lines, fmt.Sprintf("%s  %s • %s", formatActivityTimestamp(entry.At), entry.Summary, truncate(entry.Target, 42)))
				rendered++
				if rendered >= activityLogViewWindow {
					break
				}
			}
		}
		lines = append(lines, hintStyle.Render("esc close • undo/redo available"))
		return style.Render(strings.Join(lines, "\n"))

	case modeTaskInfo:
		task, ok := m.taskInfoTask()
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
			due = formatDueValue(task.DueAt)
		}
		labels := "-"
		if len(task.Labels) > 0 {
			labels = strings.Join(task.Labels, ", ")
		}
		lines := []string{
			titleStyle.Render("Task Info"),
			task.Title,
			hintStyle.Render("kind: " + string(task.Kind) + " • state: " + string(task.LifecycleState)),
			hintStyle.Render("priority: " + string(task.Priority) + " • due: " + due),
			hintStyle.Render("labels: " + labels),
		}
		if warning := m.taskDueWarning(task, time.Now().UTC()); warning != "" {
			lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203")).Render(warning))
		}
		subtasks := m.subtasksForParent(task.ID)
		if len(subtasks) > 0 {
			lines = append(lines, "")
			done, total := m.subtaskProgress(task.ID)
			lines = append(lines, hintStyle.Render(fmt.Sprintf("subtasks (%d/%d done)", done, total)))
			subtaskIdx := clamp(m.taskInfoSubtaskIdx, 0, len(subtasks)-1)
			for idx, subtask := range subtasks {
				prefix := "  "
				if idx == subtaskIdx {
					prefix = "│ "
				}
				line := prefix + truncate(subtask.Title, 42)
				meta := m.cardMeta(subtask)
				if meta != "" {
					line += " " + meta
				}
				if subtask.LifecycleState == domain.StateDone {
					line = lipgloss.NewStyle().Foreground(muted).Render(line)
				}
				lines = append(lines, line)
			}
			lines = append(lines, hintStyle.Render("j/k choose • enter open subtask • backspace parent"))
		}
		inherited := m.labelSourcesForTask(task)
		lines = append(lines, "")
		lines = append(lines, hintStyle.Render("effective labels (global/project/phase fallback)"))
		lines = append(lines, hintStyle.Render(formatLabelSource("global", inherited.Global)))
		lines = append(lines, hintStyle.Render(formatLabelSource("project", inherited.Project)))
		lines = append(lines, hintStyle.Render(formatLabelSource("phase", inherited.Phase)))

		dependsOn := uniqueTrimmed(task.Metadata.DependsOn)
		blockedBy := uniqueTrimmed(task.Metadata.BlockedBy)
		blockedReason := strings.TrimSpace(task.Metadata.BlockedReason)
		lines = append(lines, "")
		lines = append(lines, hintStyle.Render("dependencies"))
		lines = append(lines, hintStyle.Render("depends_on: "+m.summarizeTaskRefs(dependsOn, 4)))
		lines = append(lines, hintStyle.Render("blocked_by: "+m.summarizeTaskRefs(blockedBy, 4)))
		if blockedReason == "" {
			blockedReason = "-"
		}
		lines = append(lines, hintStyle.Render("blocked_reason: "+blockedReason))

		lines = append(lines, "")
		lines = append(lines, hintStyle.Render("resources"))
		if len(task.Metadata.ResourceRefs) == 0 {
			lines = append(lines, hintStyle.Render("(none)"))
		} else {
			for idx, ref := range task.Metadata.ResourceRefs {
				if idx >= 4 {
					lines = append(lines, hintStyle.Render(fmt.Sprintf("+%d more", len(task.Metadata.ResourceRefs)-idx)))
					break
				}
				location := strings.TrimSpace(ref.Location)
				if ref.PathMode == domain.PathModeRelative && strings.TrimSpace(ref.BaseAlias) != "" {
					location = strings.TrimSpace(ref.BaseAlias) + ":" + location
				}
				lines = append(lines, hintStyle.Render(fmt.Sprintf("%s %s", ref.ResourceType, truncate(location, 48))))
			}
		}
		if strings.TrimSpace(task.ParentID) != "" {
			lines = append(lines, hintStyle.Render("parent: "+task.ParentID))
		}
		if objective := strings.TrimSpace(task.Metadata.Objective); objective != "" {
			lines = append(lines, "", hintStyle.Render("objective"), objective)
		}
		if len(task.Metadata.CompletionContract.CompletionCriteria) > 0 {
			unmet := 0
			for _, item := range task.Metadata.CompletionContract.CompletionCriteria {
				if strings.TrimSpace(item.Text) == "" {
					continue
				}
				if !item.Done {
					unmet++
				}
			}
			if unmet > 0 {
				lines = append(lines, hintStyle.Render(fmt.Sprintf("completion: %d unmet checks", unmet)))
			}
		}
		if desc := strings.TrimSpace(task.Description); desc != "" {
			lines = append(lines, "", desc)
		}
		lines = append(lines, "", hintStyle.Render("e edit • s subtask • [/] move • b dependencies • r attach resource • f focus subtree • esc back/close"))
		return boxStyle.Render(strings.Join(lines, "\n"))

	case modeResourcePicker:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 44, 108))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		currentPath := strings.TrimSpace(m.resourcePickerDir)
		if currentPath == "" {
			currentPath = m.resourcePickerRoot
		}
		displayPath := "."
		if rel, err := filepath.Rel(m.resourcePickerRoot, currentPath); err == nil {
			displayPath = filepath.ToSlash(rel)
		}
		title := "Attach Resource"
		if m.resourcePickerBack == modeAddProject || m.resourcePickerBack == modeEditProject || m.resourcePickerBack == modePathsRoots {
			title = "Pick Project Root"
		}
		filterInput := m.resourcePickerFilter
		filterInput.SetWidth(max(20, min(72, maxWidth-18)))
		lines := []string{
			titleStyle.Render(title),
			hintStyle.Render("root: " + truncate(m.resourcePickerRoot, 72)),
			hintStyle.Render("path: " + displayPath),
			hintStyle.Render("filter: ") + filterInput.View(),
		}
		items := m.visibleResourcePickerItems()
		if len(items) == 0 {
			lines = append(lines, hintStyle.Render("(empty directory)"))
			lines = append(lines, hintStyle.Render("press a to choose current directory"))
		} else {
			for idx, entry := range items {
				cursor := "  "
				if idx == m.resourcePickerIndex {
					cursor = "> "
				}
				name := entry.Name
				if entry.IsDir {
					name += "/"
				}
				lines = append(lines, cursor+name)
				if idx >= 13 {
					lines = append(lines, hintStyle.Render(fmt.Sprintf("+%d more entries", len(items)-idx-1)))
					break
				}
			}
		}
		if m.resourcePickerBack == modeAddProject || m.resourcePickerBack == modeEditProject || m.resourcePickerBack == modePathsRoots {
			lines = append(lines, hintStyle.Render("enter open dir (or choose file parent) • a choose dir • h parent • ctrl+u clear filter • esc close"))
		} else {
			lines = append(lines, hintStyle.Render("enter open dir • enter/a attach file • a attach dir • h parent • ctrl+u clear filter • esc close"))
		}
		return style.Render(strings.Join(lines, "\n"))

	case modeLabelPicker:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 38, 88))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		lines := []string{
			titleStyle.Render("Inherited Labels"),
			hintStyle.Render("global/project/phase fallback"),
		}
		if len(m.labelPickerItems) == 0 {
			lines = append(lines, hintStyle.Render("(no inherited labels)"))
		} else {
			for idx, item := range m.labelPickerItems {
				cursor := "  "
				if idx == m.labelPickerIndex {
					cursor = "> "
				}
				lines = append(lines, fmt.Sprintf("%s%s (%s)", cursor, item.Label, item.Source))
				if idx >= 11 {
					lines = append(lines, hintStyle.Render(fmt.Sprintf("+%d more labels", len(m.labelPickerItems)-idx-1)))
					break
				}
			}
		}
		lines = append(lines, hintStyle.Render("j/k navigate • enter add label • esc close"))
		return style.Render(strings.Join(lines, "\n"))

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
		}
		if len(m.commandMatches) == 0 {
			lines = append(lines, hintStyle.Render("(no matching commands)"))
		} else {
			const commandWindowSize = 9
			start, end := windowBounds(len(m.commandMatches), m.commandIndex, commandWindowSize)
			for idx := start; idx < end; idx++ {
				item := m.commandMatches[idx]
				prefix := "  "
				if idx == m.commandIndex {
					prefix = "› "
				}
				alias := ""
				if len(item.Aliases) > 0 {
					alias = " (" + strings.Join(item.Aliases, ", ") + ")"
				}
				lines = append(lines, fmt.Sprintf("%s%s%s — %s", prefix, item.Command, alias, item.Description))
			}
			if len(m.commandMatches) > commandWindowSize {
				lines = append(lines, hintStyle.Render(fmt.Sprintf("showing %d-%d of %d", start+1, end, len(m.commandMatches))))
			}
		}
		lines = append(lines, hintStyle.Render("enter run • tab autocomplete • j/k move • esc cancel"))
		if m.searchApplied {
			lines = append(lines, hintStyle.Render("search hints: clear-query • reset-filters • search-all • search-project"))
		}
		return style.Render(strings.Join(lines, "\n"))

	case modePathsRoots:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 42, 100))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		projectLabel := "(none)"
		if project, ok := m.currentProject(); ok {
			projectLabel = project.Name
			if slug := strings.TrimSpace(strings.ToLower(project.Slug)); slug != "" {
				projectLabel += " (" + slug + ")"
			}
		}
		in := m.pathsRootInput
		in.SetWidth(max(20, maxWidth-24))
		lines := []string{
			titleStyle.Render("Paths / Roots"),
			hintStyle.Render("project: " + projectLabel),
			in.View(),
			hintStyle.Render("enter save • esc cancel • ctrl+r browse dirs • empty value clears mapping"),
		}
		return style.Render(strings.Join(lines, "\n"))

	case modeConfirmAction:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 36, 88))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		taskTitle := strings.TrimSpace(m.pendingConfirm.Task.Title)
		if len(m.pendingConfirm.TaskIDs) > 1 {
			taskTitle = fmt.Sprintf("%d selected tasks", len(m.pendingConfirm.TaskIDs))
		}
		if taskTitle == "" {
			taskTitle = "(unknown task)"
		}
		confirmStyle := lipgloss.NewStyle().Foreground(muted)
		cancelStyle := lipgloss.NewStyle().Foreground(muted)
		if m.confirmChoice == 0 {
			confirmStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
		} else {
			cancelStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
		}
		lines := []string{
			titleStyle.Render("Confirm Action"),
			fmt.Sprintf("%s: %s", m.pendingConfirm.Label, taskTitle),
			confirmStyle.Render("[confirm]") + "  " + cancelStyle.Render("[cancel]"),
			hintStyle.Render("enter apply • esc cancel • h/l switch • y confirm • n cancel"),
		}
		return style.Render(strings.Join(lines, "\n"))

	case modeQuickActions:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 32, 78))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		lines := []string{titleStyle.Render("Quick Actions")}
		actions := m.quickActions()
		if len(actions) == 0 {
			lines = append(lines, hintStyle.Render("(no actions available)"))
		} else {
			const quickActionWindowSize = 11
			start, end := windowBounds(len(actions), m.quickActionIndex, quickActionWindowSize)
			enabledActiveStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
			disabledStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
			disabledActiveStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("243"))
			for idx := start; idx < end; idx++ {
				action := actions[idx]
				cursor := "  "
				if idx == m.quickActionIndex {
					cursor = "> "
				}
				label := action.Label
				if !action.Enabled && strings.TrimSpace(action.DisabledReason) != "" {
					label += " (" + action.DisabledReason + ")"
				}
				switch {
				case action.Enabled && idx == m.quickActionIndex:
					label = enabledActiveStyle.Render(label)
				case !action.Enabled && idx == m.quickActionIndex:
					label = disabledActiveStyle.Render(label)
				case !action.Enabled:
					label = disabledStyle.Render(label)
				}
				lines = append(lines, cursor+label)
			}
			if len(actions) > quickActionWindowSize {
				lines = append(lines, hintStyle.Render(fmt.Sprintf("showing %d-%d of %d", start+1, end, len(actions))))
			}
		}
		lines = append(lines, hintStyle.Render("j/k navigate • enter run • esc close"))
		return style.Render(strings.Join(lines, "\n"))

	case modeAddTask, modeSearch, modeRenameTask, modeEditTask, modeAddProject, modeEditProject, modeLabelsConfig:
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
			hint = "enter save • esc cancel • tab next field • ctrl+r attach resource • ctrl+y accept label suggestion"
		case modeSearch:
			title = "Search"
			hint = "tab focus • space/enter toggle • ctrl+u clear query • ctrl+r reset filters"
		case modeRenameTask:
			title = "Rename Task"
		case modeEditTask:
			title = "Edit Task"
			hint = "enter save • esc cancel • tab next field • ctrl+r attach resource • ctrl+y accept label suggestion"
		case modeAddProject:
			title = "New Project"
		case modeEditProject:
			title = "Edit Project"
		case modeLabelsConfig:
			title = "Labels Config"
		}

		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		lines := []string{titleStyle.Render(title)}

		switch m.mode {
		case modeSearch:
			queryInput := m.searchInput
			queryInput.SetWidth(max(18, maxWidth-20))
			scope := "current project"
			if m.searchCrossProject {
				scope = "all projects"
			}
			labelStyle := lipgloss.NewStyle().Foreground(muted)
			if m.searchFocus == 0 {
				labelStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
			}
			lines = append(lines, labelStyle.Render("query:")+" "+queryInput.View())

			stateLabel := lipgloss.NewStyle().Foreground(muted)
			if m.searchFocus == 1 {
				stateLabel = lipgloss.NewStyle().Bold(true).Foreground(accent)
			}
			stateParts := make([]string, 0, len(canonicalSearchStatesOrdered))
			for idx, state := range canonicalSearchStatesOrdered {
				check := " "
				if m.isSearchStateEnabled(state) {
					check = "x"
				}
				name := canonicalSearchStateLabels[state]
				if name == "" {
					name = state
				}
				item := fmt.Sprintf("[%s] %s", check, name)
				if idx == clamp(m.searchStateCursor, 0, len(canonicalSearchStatesOrdered)-1) && m.searchFocus == 1 {
					item = lipgloss.NewStyle().Bold(true).Foreground(accent).Render(item)
				}
				stateParts = append(stateParts, item)
			}
			lines = append(lines, stateLabel.Render("states:")+" "+strings.Join(stateParts, "   "))

			scopeLabel := lipgloss.NewStyle().Foreground(muted)
			if m.searchFocus == 2 {
				scopeLabel = lipgloss.NewStyle().Bold(true).Foreground(accent)
			}
			lines = append(lines, scopeLabel.Render("scope: "+scope))

			archivedLabel := lipgloss.NewStyle().Foreground(muted)
			if m.searchFocus == 3 {
				archivedLabel = lipgloss.NewStyle().Bold(true).Foreground(accent)
			}
			if m.showArchived {
				lines = append(lines, archivedLabel.Render("archived: included"))
			} else {
				lines = append(lines, archivedLabel.Render("archived: hidden"))
			}
			applyLabel := hintStyle
			if m.searchFocus == 4 {
				applyLabel = lipgloss.NewStyle().Bold(true).Foreground(accent)
			}
			lines = append(lines, applyLabel.Render("[ apply search ]"))
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
				if i == taskFieldPriority {
					lines = append(lines, labelStyle.Render(fmt.Sprintf("%-12s", label+":"))+" "+m.renderPriorityPicker(accent, muted))
					continue
				}
				in.SetWidth(fieldWidth)
				lines = append(lines, labelStyle.Render(fmt.Sprintf("%-12s", label+":"))+" "+in.View())
			}
			if m.formFocus == taskFieldDue {
				lines = append(lines, hintStyle.Render("ctrl+d or D open due-date picker"))
			}
			if m.formFocus == taskFieldLabels {
				lines = append(lines, hintStyle.Render("ctrl+l inherited label picker • ctrl+y accept autocomplete"))
			}
			if m.formFocus == taskFieldDependsOn || m.formFocus == taskFieldBlockedBy {
				lines = append(lines, hintStyle.Render("dependency fields accept task IDs as csv (use task info/search for IDs)"))
			}
			if len(m.taskFormResourceRefs) > 0 {
				lines = append(lines, hintStyle.Render(fmt.Sprintf("staged resources: %d", len(m.taskFormResourceRefs))))
			}
			if suggestions := m.labelSuggestions(5); len(suggestions) > 0 {
				lines = append(lines, hintStyle.Render("suggested labels: "+strings.Join(suggestions, ", ")))
			}
			inherited := m.taskFormLabelSources()
			lines = append(lines, hintStyle.Render("inherited labels"))
			lines = append(lines, hintStyle.Render(formatLabelSource("global", inherited.Global)))
			lines = append(lines, hintStyle.Render(formatLabelSource("project", inherited.Project)))
			lines = append(lines, hintStyle.Render(formatLabelSource("phase", inherited.Phase)))
			if warning := dueWarning(m.formInputs[taskFieldDue].Value(), time.Now().UTC()); warning != "" {
				lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render(warning))
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
			if m.projectFormFocus == projectFieldRootPath {
				lines = append(lines, hintStyle.Render("ctrl+r browse and select a directory"))
			}
		case modeLabelsConfig:
			fieldWidth := max(18, maxWidth-28)
			labelFields := []string{"global", "project"}
			for i, in := range m.labelsConfigInputs {
				label := fmt.Sprintf("%d.", i+1)
				if i < len(labelFields) {
					label = labelFields[i]
				}
				labelStyle := lipgloss.NewStyle().Foreground(muted)
				if i == m.labelsConfigFocus {
					labelStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
				}
				in.SetWidth(fieldWidth)
				lines = append(lines, labelStyle.Render(fmt.Sprintf("%-12s", label+":"))+" "+in.View())
			}
			lines = append(lines, hintStyle.Render("global applies across projects; project applies to current project only"))
		default:
			lines = append(lines, m.input)
		}

		lines = append(lines, hintStyle.Render(hint))
		return boxStyle.Render(strings.Join(lines, "\n"))
	default:
		return ""
	}
}

// renderPriorityPicker renders output for the current model state.
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

// formatTaskEditInput formats values for display or serialization.
func formatTaskEditInput(task domain.Task) string {
	due := "-"
	if task.DueAt != nil {
		due = formatDueValue(task.DueAt)
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

// parseTaskEditInput parses input into a normalized form.
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

	dueAt, err := parseDueInput(parts[3], current.DueAt)
	if err != nil {
		return app.UpdateTaskInput{}, err
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

// modeLabel handles mode label.
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
	case modeActivityLog:
		return "activity"
	case modeConfirmAction:
		return "confirm"
	case modeResourcePicker:
		return "resources"
	case modeLabelPicker:
		return "labels"
	case modePathsRoots:
		return "paths/roots"
	case modeLabelsConfig:
		return "labels-config"
	default:
		return "normal"
	}
}

// modePrompt handles mode prompt.
func (m Model) modePrompt() string {
	switch m.mode {
	case modeAddTask:
		return "new task title: " + m.input + " (enter save, esc cancel)"
	case modeSearch:
		return "search query: " + m.input + " (enter apply, esc cancel)"
	case modeRenameTask:
		return "rename task: " + m.input + " (enter save, esc cancel)"
	case modeEditTask:
		return "edit task: " + m.input + " (title | description | priority(low|medium|high) | due(YYYY-MM-DD[THH:MM] or -) | labels(csv))"
	case modeDuePicker:
		return "due picker: j/k select, enter apply, esc cancel"
	case modeProjectPicker:
		return "project picker: j/k select, enter choose, esc cancel"
	case modeTaskInfo:
		return "task info: e edit, s subtask, [/] move, b deps, r attach, esc back/close"
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
	case modeActivityLog:
		return "activity log: esc close"
	case modeConfirmAction:
		return "confirm action: enter confirm, esc cancel"
	case modeResourcePicker:
		return "resource picker: type fuzzy filter, j/k select, enter open, a choose/attach, esc cancel"
	case modeLabelPicker:
		return "label picker: j/k select, enter add label, esc cancel"
	case modePathsRoots:
		return "paths/roots: enter save, ctrl+r browse dirs, esc cancel"
	case modeLabelsConfig:
		return "labels config: enter save, esc cancel"
	default:
		return ""
	}
}

// normalizeBoardGroupBy canonicalizes board grouping values.
func normalizeBoardGroupBy(groupBy string) string {
	switch strings.ToLower(strings.TrimSpace(groupBy)) {
	case "priority":
		return "priority"
	case "state":
		return "state"
	default:
		return "none"
	}
}

// formatActivityTimestamp formats activity timestamps for compact modal rendering.
func formatActivityTimestamp(at time.Time) string {
	if at.IsZero() {
		return "--:--:--"
	}
	local := at.Local()
	now := time.Now().In(local.Location())
	if local.Year() != now.Year() || local.YearDay() != now.YearDay() {
		return local.Format("01-02 15:04")
	}
	return local.Format("15:04:05")
}

// columnWidth returns column width.
func (m Model) columnWidth() int {
	return m.columnWidthFor(m.width)
}

// columnWidthFor returns column width for.
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

// columnHeight returns column height.
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

// boardTop handles board top.
func (m Model) boardTop() int {
	// mouse coordinates from tea are 1-based
	// header + optional tabs + spacer
	top := 3
	if len(m.projects) > 1 {
		top++
	}
	return top
}

// clamp clamps the requested operation.
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

// max returns the larger of the provided values.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min returns the smaller of the provided values.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// fitLines fits lines.
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

// overlayOnContent overlays on content.
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

// truncate truncates the requested operation.
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

// summarizeLabels summarizes labels.
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
