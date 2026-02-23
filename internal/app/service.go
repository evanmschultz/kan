package app

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/evanschultz/kan/internal/domain"
)

// DeleteMode represents a selectable mode.
type DeleteMode string

// DeleteModeArchive and related constants define package defaults.
const (
	DeleteModeArchive DeleteMode = "archive"
	DeleteModeHard    DeleteMode = "hard"
)

// ServiceConfig holds configuration for service.
type ServiceConfig struct {
	DefaultDeleteMode        DeleteMode
	StateTemplates           []StateTemplate
	AutoCreateProjectColumns bool
}

// StateTemplate represents state template data used by this package.
type StateTemplate struct {
	ID       string
	Name     string
	WIPLimit int
	Position int
}

// IDGenerator returns unique identifiers for new entities.
type IDGenerator func() string

// Clock returns the current time.
type Clock func() time.Time

// Service represents service data used by this package.
type Service struct {
	repo              Repository
	idGen             IDGenerator
	clock             Clock
	defaultDeleteMode DeleteMode
	stateTemplates    []StateTemplate
	autoProjectCols   bool
}

// NewService constructs a new value for this package.
func NewService(repo Repository, idGen IDGenerator, clock Clock, cfg ServiceConfig) *Service {
	if idGen == nil {
		idGen = func() string { return "" }
	}
	if clock == nil {
		clock = time.Now
	}
	if cfg.DefaultDeleteMode == "" {
		cfg.DefaultDeleteMode = DeleteModeArchive
	}
	templates := sanitizeStateTemplates(cfg.StateTemplates)
	if len(templates) == 0 {
		templates = defaultStateTemplates()
	}

	return &Service{
		repo:              repo,
		idGen:             idGen,
		clock:             clock,
		defaultDeleteMode: cfg.DefaultDeleteMode,
		stateTemplates:    templates,
		autoProjectCols:   cfg.AutoCreateProjectColumns,
	}
}

// EnsureDefaultProject ensures default project.
func (s *Service) EnsureDefaultProject(ctx context.Context) (domain.Project, error) {
	projects, err := s.repo.ListProjects(ctx, false)
	if err != nil {
		return domain.Project{}, err
	}
	if len(projects) > 0 {
		return projects[0], nil
	}

	now := s.clock()
	project, err := domain.NewProject(s.idGen(), "Inbox", "Default project", now)
	if err != nil {
		return domain.Project{}, err
	}
	if err := s.repo.CreateProject(ctx, project); err != nil {
		return domain.Project{}, err
	}

	if err := s.createDefaultColumns(ctx, project.ID, now); err != nil {
		return domain.Project{}, err
	}

	return project, nil
}

// CreateProjectInput holds input values for create project operations.
type CreateProjectInput struct {
	Name        string
	Description string
	Metadata    domain.ProjectMetadata
}

// CreateProject creates project.
func (s *Service) CreateProject(ctx context.Context, name, description string) (domain.Project, error) {
	return s.CreateProjectWithMetadata(ctx, CreateProjectInput{
		Name:        name,
		Description: description,
	})
}

// CreateProjectWithMetadata creates project with metadata.
func (s *Service) CreateProjectWithMetadata(ctx context.Context, in CreateProjectInput) (domain.Project, error) {
	now := s.clock()
	project, err := domain.NewProject(s.idGen(), in.Name, in.Description, now)
	if err != nil {
		return domain.Project{}, err
	}
	if err := project.UpdateDetails(project.Name, project.Description, in.Metadata, now); err != nil {
		return domain.Project{}, err
	}
	if err := s.repo.CreateProject(ctx, project); err != nil {
		return domain.Project{}, err
	}
	if s.autoProjectCols {
		if err := s.createDefaultColumns(ctx, project.ID, now); err != nil {
			return domain.Project{}, err
		}
	}
	return project, nil
}

// UpdateProjectInput holds input values for update project operations.
type UpdateProjectInput struct {
	ProjectID   string
	Name        string
	Description string
	Metadata    domain.ProjectMetadata
}

// UpdateProject updates state for the requested operation.
func (s *Service) UpdateProject(ctx context.Context, in UpdateProjectInput) (domain.Project, error) {
	project, err := s.repo.GetProject(ctx, in.ProjectID)
	if err != nil {
		return domain.Project{}, err
	}
	if err := project.UpdateDetails(in.Name, in.Description, in.Metadata, s.clock()); err != nil {
		return domain.Project{}, err
	}
	if err := s.repo.UpdateProject(ctx, project); err != nil {
		return domain.Project{}, err
	}
	return project, nil
}

// CreateColumn creates column.
func (s *Service) CreateColumn(ctx context.Context, projectID, name string, position, wipLimit int) (domain.Column, error) {
	column, err := domain.NewColumn(s.idGen(), projectID, name, position, wipLimit, s.clock())
	if err != nil {
		return domain.Column{}, err
	}
	if err := s.repo.CreateColumn(ctx, column); err != nil {
		return domain.Column{}, err
	}
	return column, nil
}

// CreateTaskInput holds input values for create task operations.
type CreateTaskInput struct {
	ProjectID      string
	ParentID       string
	Kind           domain.WorkKind
	ColumnID       string
	Title          string
	Description    string
	Priority       domain.Priority
	DueAt          *time.Time
	Labels         []string
	Metadata       domain.TaskMetadata
	CreatedByActor string
	UpdatedByActor string
	UpdatedByType  domain.ActorType
}

// UpdateTaskInput holds input values for update task operations.
type UpdateTaskInput struct {
	TaskID      string
	Title       string
	Description string
	Priority    domain.Priority
	DueAt       *time.Time
	Labels      []string
	Metadata    *domain.TaskMetadata
	UpdatedBy   string
	UpdatedType domain.ActorType
}

// SearchTasksFilter defines filtering criteria for queries.
type SearchTasksFilter struct {
	ProjectID       string
	Query           string
	CrossProject    bool
	IncludeArchived bool
	States          []string
}

// TaskMatch describes a matched result.
type TaskMatch struct {
	Project domain.Project
	Task    domain.Task
	StateID string
}

// CreateTask creates task.
func (s *Service) CreateTask(ctx context.Context, in CreateTaskInput) (domain.Task, error) {
	tasks, err := s.repo.ListTasks(ctx, in.ProjectID, false)
	if err != nil {
		return domain.Task{}, err
	}
	columns, err := s.repo.ListColumns(ctx, in.ProjectID, true)
	if err != nil {
		return domain.Task{}, err
	}
	lifecycleState := lifecycleStateForColumnID(columns, in.ColumnID)
	if lifecycleState == "" {
		lifecycleState = domain.StateTodo
	}
	position := 0
	for _, t := range tasks {
		if t.ColumnID == in.ColumnID && t.Position >= position {
			position = t.Position + 1
		}
	}

	task, err := domain.NewTask(domain.TaskInput{
		ID:             s.idGen(),
		ProjectID:      in.ProjectID,
		ParentID:       in.ParentID,
		Kind:           in.Kind,
		LifecycleState: lifecycleState,
		ColumnID:       in.ColumnID,
		Position:       position,
		Title:          in.Title,
		Description:    in.Description,
		Priority:       in.Priority,
		DueAt:          in.DueAt,
		Labels:         in.Labels,
		Metadata:       in.Metadata,
		CreatedByActor: in.CreatedByActor,
		UpdatedByActor: in.UpdatedByActor,
		UpdatedByType:  in.UpdatedByType,
	}, s.clock())
	if err != nil {
		return domain.Task{}, err
	}

	if err := s.repo.CreateTask(ctx, task); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

// MoveTask moves task.
func (s *Service) MoveTask(ctx context.Context, taskID, toColumnID string, position int) (domain.Task, error) {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	columns, err := s.repo.ListColumns(ctx, task.ProjectID, true)
	if err != nil {
		return domain.Task{}, err
	}
	fromState := lifecycleStateForColumnID(columns, task.ColumnID)
	if fromState == "" {
		fromState = task.LifecycleState
	}
	toState := lifecycleStateForColumnID(columns, toColumnID)
	if toState == "" {
		toState = fromState
	}
	if fromState == domain.StateTodo && toState == domain.StateProgress {
		if unmet := task.StartCriteriaUnmet(); len(unmet) > 0 {
			return domain.Task{}, fmt.Errorf("%w: start criteria unmet (%s)", domain.ErrTransitionBlocked, strings.Join(unmet, ", "))
		}
	}
	if toState == domain.StateDone {
		projectTasks, listErr := s.repo.ListTasks(ctx, task.ProjectID, true)
		if listErr != nil {
			return domain.Task{}, listErr
		}
		children := make([]domain.Task, 0)
		for _, candidate := range projectTasks {
			if candidate.ParentID == task.ID {
				children = append(children, candidate)
			}
		}
		for _, child := range children {
			if child.ArchivedAt != nil {
				continue
			}
			if child.LifecycleState != domain.StateDone {
				return domain.Task{}, fmt.Errorf("%w: completion criteria unmet (subtasks must be done before moving to done)", domain.ErrTransitionBlocked)
			}
		}
		if unmet := task.CompletionCriteriaUnmet(children); len(unmet) > 0 {
			return domain.Task{}, fmt.Errorf("%w: completion criteria unmet (%s)", domain.ErrTransitionBlocked, strings.Join(unmet, ", "))
		}
	}
	if err := task.Move(toColumnID, position, s.clock()); err != nil {
		return domain.Task{}, err
	}
	if err := task.SetLifecycleState(toState, s.clock()); err != nil {
		return domain.Task{}, err
	}
	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

// RestoreTask restores task.
func (s *Service) RestoreTask(ctx context.Context, taskID string) (domain.Task, error) {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	task.Restore(s.clock())
	columns, err := s.repo.ListColumns(ctx, task.ProjectID, true)
	if err != nil {
		return domain.Task{}, err
	}
	restoredState := lifecycleStateForColumnID(columns, task.ColumnID)
	if restoredState == "" {
		restoredState = domain.StateTodo
	}
	if err := task.SetLifecycleState(restoredState, s.clock()); err != nil {
		return domain.Task{}, err
	}
	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

// RenameTask renames task.
func (s *Service) RenameTask(ctx context.Context, taskID, title string) (domain.Task, error) {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	if err := task.UpdateDetails(title, task.Description, task.Priority, task.DueAt, task.Labels, s.clock()); err != nil {
		return domain.Task{}, err
	}
	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

// UpdateTask updates state for the requested operation.
func (s *Service) UpdateTask(ctx context.Context, in UpdateTaskInput) (domain.Task, error) {
	task, err := s.repo.GetTask(ctx, in.TaskID)
	if err != nil {
		return domain.Task{}, err
	}
	if err := task.UpdateDetails(in.Title, in.Description, in.Priority, in.DueAt, in.Labels, s.clock()); err != nil {
		return domain.Task{}, err
	}
	if in.Metadata != nil {
		actorType := in.UpdatedType
		if actorType == "" {
			actorType = domain.ActorTypeUser
		}
		if err := task.UpdatePlanningMetadata(*in.Metadata, in.UpdatedBy, actorType, s.clock()); err != nil {
			return domain.Task{}, err
		}
	}
	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

// DeleteTask deletes task.
func (s *Service) DeleteTask(ctx context.Context, taskID string, mode DeleteMode) error {
	if mode == "" {
		mode = s.defaultDeleteMode
	}

	switch mode {
	case DeleteModeArchive:
		task, err := s.repo.GetTask(ctx, taskID)
		if err != nil {
			return err
		}
		task.Archive(s.clock())
		return s.repo.UpdateTask(ctx, task)
	case DeleteModeHard:
		return s.repo.DeleteTask(ctx, taskID)
	default:
		return ErrInvalidDeleteMode
	}
}

// ListProjects lists projects.
func (s *Service) ListProjects(ctx context.Context, includeArchived bool) ([]domain.Project, error) {
	return s.repo.ListProjects(ctx, includeArchived)
}

// ListColumns lists columns.
func (s *Service) ListColumns(ctx context.Context, projectID string, includeArchived bool) ([]domain.Column, error) {
	columns, err := s.repo.ListColumns(ctx, projectID, includeArchived)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(columns, func(a, b domain.Column) int {
		return a.Position - b.Position
	})
	return columns, nil
}

// ListTasks lists tasks.
func (s *Service) ListTasks(ctx context.Context, projectID string, includeArchived bool) ([]domain.Task, error) {
	tasks, err := s.repo.ListTasks(ctx, projectID, includeArchived)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(tasks, func(a, b domain.Task) int {
		if a.ColumnID == b.ColumnID {
			return a.Position - b.Position
		}
		return strings.Compare(a.ColumnID, b.ColumnID)
	})
	return tasks, nil
}

// ListProjectChangeEvents lists recent change events for a project.
func (s *Service) ListProjectChangeEvents(ctx context.Context, projectID string, limit int) ([]domain.ChangeEvent, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, domain.ErrInvalidID
	}
	return s.repo.ListProjectChangeEvents(ctx, projectID, limit)
}

// GetProjectDependencyRollup summarizes dependency and blocked-state counts.
func (s *Service) GetProjectDependencyRollup(ctx context.Context, projectID string) (domain.DependencyRollup, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return domain.DependencyRollup{}, domain.ErrInvalidID
	}
	if _, err := s.repo.GetProject(ctx, projectID); err != nil {
		return domain.DependencyRollup{}, err
	}
	tasks, err := s.repo.ListTasks(ctx, projectID, false)
	if err != nil {
		return domain.DependencyRollup{}, err
	}
	return buildDependencyRollup(projectID, tasks), nil
}

// ListChildTasks lists child tasks for a parent within the same project.
func (s *Service) ListChildTasks(ctx context.Context, projectID, parentID string, includeArchived bool) ([]domain.Task, error) {
	parentID = strings.TrimSpace(parentID)
	if parentID == "" {
		return nil, domain.ErrInvalidParentID
	}
	tasks, err := s.ListTasks(ctx, projectID, includeArchived)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Task, 0)
	for _, task := range tasks {
		if task.ParentID == parentID {
			out = append(out, task)
		}
	}
	return out, nil
}

// ReparentTask changes parent task relationship.
func (s *Service) ReparentTask(ctx context.Context, taskID, parentID string) (domain.Task, error) {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	if parentID != "" {
		parent, parentErr := s.repo.GetTask(ctx, parentID)
		if parentErr != nil {
			return domain.Task{}, parentErr
		}
		if parent.ProjectID != task.ProjectID {
			return domain.Task{}, domain.ErrInvalidParentID
		}
	}
	if err := task.Reparent(parentID, s.clock()); err != nil {
		return domain.Task{}, err
	}
	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

// SearchTaskMatches finds task matches using project, state, and archive filters.
func (s *Service) SearchTaskMatches(ctx context.Context, in SearchTasksFilter) ([]TaskMatch, error) {
	stateFilter := map[string]struct{}{}
	for _, raw := range in.States {
		state := strings.TrimSpace(strings.ToLower(raw))
		if state == "" {
			continue
		}
		stateFilter[state] = struct{}{}
	}
	allowAllStates := len(stateFilter) == 0
	wantsArchivedState := allowAllStates
	if !allowAllStates {
		_, wantsArchivedState = stateFilter["archived"]
	}

	targetProjects := []domain.Project{}
	if in.CrossProject {
		projects, err := s.repo.ListProjects(ctx, in.IncludeArchived)
		if err != nil {
			return nil, err
		}
		targetProjects = append(targetProjects, projects...)
	} else {
		projectID := strings.TrimSpace(in.ProjectID)
		if projectID == "" {
			return nil, domain.ErrInvalidID
		}
		project, err := s.repo.GetProject(ctx, projectID)
		if err != nil {
			return nil, err
		}
		if !in.IncludeArchived && project.ArchivedAt != nil {
			return nil, nil
		}
		targetProjects = append(targetProjects, project)
	}

	query := strings.TrimSpace(strings.ToLower(in.Query))
	out := make([]TaskMatch, 0)
	for _, project := range targetProjects {
		columns, err := s.repo.ListColumns(ctx, project.ID, true)
		if err != nil {
			return nil, err
		}
		stateByColumn := make(map[string]string, len(columns))
		for _, column := range columns {
			stateByColumn[column.ID] = normalizeStateID(column.Name)
		}

		tasks, err := s.repo.ListTasks(ctx, project.ID, true)
		if err != nil {
			return nil, err
		}
		for _, task := range tasks {
			stateID := stateByColumn[task.ColumnID]
			if stateID == "" {
				stateID = string(task.LifecycleState)
			}
			if stateID == "" {
				stateID = "todo"
			}
			if task.ArchivedAt != nil {
				if !in.IncludeArchived || !wantsArchivedState {
					continue
				}
				stateID = "archived"
			} else if !allowAllStates {
				if _, ok := stateFilter[stateID]; !ok {
					continue
				}
			}

			if query != "" {
				if !strings.Contains(strings.ToLower(task.Title), query) &&
					!strings.Contains(strings.ToLower(task.Description), query) &&
					!labelsContainQuery(task.Labels, query) {
					continue
				}
			}

			out = append(out, TaskMatch{
				Project: project,
				Task:    task,
				StateID: stateID,
			})
		}
	}

	slices.SortFunc(out, func(a, b TaskMatch) int {
		if a.Project.ID == b.Project.ID {
			if a.StateID == b.StateID {
				if a.Task.ColumnID == b.Task.ColumnID {
					if a.Task.Position == b.Task.Position {
						return strings.Compare(a.Task.ID, b.Task.ID)
					}
					return a.Task.Position - b.Task.Position
				}
				return strings.Compare(a.Task.ColumnID, b.Task.ColumnID)
			}
			return strings.Compare(a.StateID, b.StateID)
		}
		return strings.Compare(a.Project.ID, b.Project.ID)
	})

	return out, nil
}

// labelsContainQuery handles labels contain query.
func labelsContainQuery(labels []string, query string) bool {
	for _, label := range labels {
		if strings.Contains(strings.ToLower(label), query) {
			return true
		}
	}
	return false
}

// buildDependencyRollup computes aggregate dependency and blocked-state counts.
func buildDependencyRollup(projectID string, tasks []domain.Task) domain.DependencyRollup {
	rollup := domain.DependencyRollup{
		ProjectID:  projectID,
		TotalItems: len(tasks),
	}
	stateByID := make(map[string]domain.LifecycleState, len(tasks))
	for _, task := range tasks {
		stateByID[task.ID] = task.LifecycleState
	}
	for _, task := range tasks {
		dependsOn := uniqueNonEmptyIDs(task.Metadata.DependsOn)
		blockedBy := uniqueNonEmptyIDs(task.Metadata.BlockedBy)

		if len(dependsOn) > 0 {
			rollup.ItemsWithDependencies++
			rollup.DependencyEdges += len(dependsOn)
		}
		if len(blockedBy) > 0 || strings.TrimSpace(task.Metadata.BlockedReason) != "" {
			rollup.BlockedItems++
		}
		rollup.BlockedByEdges += len(blockedBy)

		// Dependencies are unresolved when the target is missing or not done.
		for _, depID := range dependsOn {
			state, ok := stateByID[depID]
			if !ok || state != domain.StateDone {
				rollup.UnresolvedDependencyEdges++
			}
		}
	}
	return rollup
}

// uniqueNonEmptyIDs trims and de-duplicates IDs while preserving order.
func uniqueNonEmptyIDs(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, raw := range in {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// defaultStateTemplates returns default state templates.
func defaultStateTemplates() []StateTemplate {
	return []StateTemplate{
		{ID: "todo", Name: "To Do", WIPLimit: 0, Position: 0},
		{ID: "progress", Name: "In Progress", WIPLimit: 0, Position: 1},
		{ID: "done", Name: "Done", WIPLimit: 0, Position: 2},
	}
}

// sanitizeStateTemplates handles sanitize state templates.
func sanitizeStateTemplates(in []StateTemplate) []StateTemplate {
	if len(in) == 0 {
		return nil
	}
	out := make([]StateTemplate, 0, len(in))
	seen := map[string]struct{}{}
	for idx, state := range in {
		state.Name = strings.TrimSpace(state.Name)
		state.ID = strings.TrimSpace(strings.ToLower(state.ID))
		if state.Name == "" {
			continue
		}
		if state.ID == "" {
			state.ID = normalizeStateID(state.Name)
		}
		dedupeID := strings.ReplaceAll(state.ID, "-", "")
		if _, ok := seen[dedupeID]; ok {
			continue
		}
		seen[dedupeID] = struct{}{}
		if state.Position < 0 {
			state.Position = idx
		}
		if state.WIPLimit < 0 {
			state.WIPLimit = 0
		}
		out = append(out, state)
	}
	slices.SortFunc(out, func(a, b StateTemplate) int {
		if a.Position == b.Position {
			return strings.Compare(a.ID, b.ID)
		}
		return a.Position - b.Position
	})
	return out
}

// normalizeStateID normalizes state id.
func normalizeStateID(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	normalized := strings.Trim(b.String(), "-")
	switch normalized {
	case "to-do", "todo":
		return "todo"
	case "in-progress", "progress", "doing":
		return "progress"
	case "done", "complete", "completed":
		return "done"
	default:
		return normalized
	}
}

// lifecycleStateForColumnID resolves canonical lifecycle state for a column.
func lifecycleStateForColumnID(columns []domain.Column, columnID string) domain.LifecycleState {
	for _, column := range columns {
		if column.ID != columnID {
			continue
		}
		switch normalizeStateID(column.Name) {
		case "todo":
			return domain.StateTodo
		case "progress":
			return domain.StateProgress
		case "done":
			return domain.StateDone
		case "archived":
			return domain.StateArchived
		default:
			return domain.StateTodo
		}
	}
	return ""
}

// createDefaultColumns creates default columns.
func (s *Service) createDefaultColumns(ctx context.Context, projectID string, now time.Time) error {
	for idx, state := range s.stateTemplates {
		position := state.Position
		if position < 0 {
			position = idx
		}
		column, err := domain.NewColumn(s.idGen(), projectID, state.Name, position, state.WIPLimit, now)
		if err != nil {
			return fmt.Errorf("create default column %q: %w", state.Name, err)
		}
		if err := s.repo.CreateColumn(ctx, column); err != nil {
			return fmt.Errorf("persist default column %q: %w", state.Name, err)
		}
	}
	return nil
}
