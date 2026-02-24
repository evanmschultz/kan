package app

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/evanschultz/kan/internal/domain"
)

// SnapshotVersion defines a package constant value.
const SnapshotVersion = "kan.snapshot.v1"

// Snapshot represents snapshot data used by this package.
type Snapshot struct {
	Version    string            `json:"version"`
	ExportedAt time.Time         `json:"exported_at"`
	Projects   []SnapshotProject `json:"projects"`
	Columns    []SnapshotColumn  `json:"columns"`
	Tasks      []SnapshotTask    `json:"tasks"`
}

// SnapshotProject represents snapshot project data used by this package.
type SnapshotProject struct {
	ID          string                 `json:"id"`
	Slug        string                 `json:"slug"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Kind        domain.KindID          `json:"kind,omitempty"`
	Metadata    domain.ProjectMetadata `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ArchivedAt  *time.Time             `json:"archived_at,omitempty"`
}

// SnapshotColumn represents snapshot column data used by this package.
type SnapshotColumn struct {
	ID         string     `json:"id"`
	ProjectID  string     `json:"project_id"`
	Name       string     `json:"name"`
	WIPLimit   int        `json:"wip_limit"`
	Position   int        `json:"position"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	ArchivedAt *time.Time `json:"archived_at,omitempty"`
}

// SnapshotTask represents snapshot task data used by this package.
type SnapshotTask struct {
	ID             string                `json:"id"`
	ProjectID      string                `json:"project_id"`
	ParentID       string                `json:"parent_id,omitempty"`
	Kind           domain.WorkKind       `json:"kind"`
	Scope          domain.KindAppliesTo  `json:"scope,omitempty"`
	LifecycleState domain.LifecycleState `json:"lifecycle_state"`
	ColumnID       string                `json:"column_id"`
	Position       int                   `json:"position"`
	Title          string                `json:"title"`
	Description    string                `json:"description"`
	Priority       domain.Priority       `json:"priority"`
	DueAt          *time.Time            `json:"due_at,omitempty"`
	Labels         []string              `json:"labels"`
	Metadata       domain.TaskMetadata   `json:"metadata"`
	CreatedByActor string                `json:"created_by_actor"`
	UpdatedByActor string                `json:"updated_by_actor"`
	UpdatedByType  domain.ActorType      `json:"updated_by_type"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
	StartedAt      *time.Time            `json:"started_at,omitempty"`
	CompletedAt    *time.Time            `json:"completed_at,omitempty"`
	ArchivedAt     *time.Time            `json:"archived_at,omitempty"`
	CanceledAt     *time.Time            `json:"canceled_at,omitempty"`
}

// ExportSnapshot handles export snapshot.
func (s *Service) ExportSnapshot(ctx context.Context, includeArchived bool) (Snapshot, error) {
	projects, err := s.repo.ListProjects(ctx, includeArchived)
	if err != nil {
		return Snapshot{}, err
	}

	snap := Snapshot{
		Version:    SnapshotVersion,
		ExportedAt: s.clock().UTC(),
		Projects:   make([]SnapshotProject, 0, len(projects)),
		Columns:    make([]SnapshotColumn, 0),
		Tasks:      make([]SnapshotTask, 0),
	}
	for _, project := range projects {
		snap.Projects = append(snap.Projects, snapshotProjectFromDomain(project))

		columns, listErr := s.repo.ListColumns(ctx, project.ID, includeArchived)
		if listErr != nil {
			return Snapshot{}, listErr
		}
		for _, column := range columns {
			snap.Columns = append(snap.Columns, snapshotColumnFromDomain(column))
		}

		tasks, listErr := s.repo.ListTasks(ctx, project.ID, includeArchived)
		if listErr != nil {
			return Snapshot{}, listErr
		}
		for _, task := range tasks {
			snap.Tasks = append(snap.Tasks, snapshotTaskFromDomain(task))
		}
	}

	snap.sort()
	return snap, nil
}

// ImportSnapshot handles import snapshot.
func (s *Service) ImportSnapshot(ctx context.Context, snap Snapshot) error {
	if err := snap.Validate(); err != nil {
		return err
	}
	snap.sort()

	for _, project := range snap.Projects {
		if err := s.upsertProject(ctx, project.toDomain()); err != nil {
			return err
		}
	}

	existingColumnsByProject := map[string]map[string]struct{}{}
	for _, project := range snap.Projects {
		columns, err := s.repo.ListColumns(ctx, project.ID, true)
		if err != nil {
			return err
		}
		byID := map[string]struct{}{}
		for _, column := range columns {
			byID[column.ID] = struct{}{}
		}
		existingColumnsByProject[project.ID] = byID
	}

	for _, column := range snap.Columns {
		dc := column.toDomain()
		if _, ok := existingColumnsByProject[dc.ProjectID][dc.ID]; ok {
			if err := s.repo.UpdateColumn(ctx, dc); err != nil {
				return err
			}
			continue
		}
		if err := s.repo.CreateColumn(ctx, dc); err != nil {
			return err
		}
		existingColumnsByProject[dc.ProjectID][dc.ID] = struct{}{}
	}

	for _, task := range snap.Tasks {
		dt := task.toDomain()
		if _, err := s.repo.GetTask(ctx, dt.ID); err == nil {
			if err := s.repo.UpdateTask(ctx, dt); err != nil {
				return err
			}
			continue
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
		if err := s.repo.CreateTask(ctx, dt); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates the requested operation.
func (s *Snapshot) Validate() error {
	if s.Version != "" && s.Version != SnapshotVersion {
		return fmt.Errorf("unsupported snapshot version: %q", s.Version)
	}

	projectIDs := map[string]struct{}{}
	for i, p := range s.Projects {
		if strings.TrimSpace(p.ID) == "" {
			return fmt.Errorf("projects[%d].id is required", i)
		}
		if strings.TrimSpace(p.Name) == "" {
			return fmt.Errorf("projects[%d].name is required", i)
		}
		if p.CreatedAt.IsZero() || p.UpdatedAt.IsZero() {
			return fmt.Errorf("projects[%d] timestamps are required", i)
		}
		if _, exists := projectIDs[p.ID]; exists {
			return fmt.Errorf("duplicate project id: %q", p.ID)
		}
		if domain.NormalizeKindID(p.Kind) == "" {
			p.Kind = domain.DefaultProjectKind
			s.Projects[i].Kind = p.Kind
		}
		projectIDs[p.ID] = struct{}{}
	}

	columnIDs := map[string]struct{}{}
	for i, c := range s.Columns {
		if strings.TrimSpace(c.ID) == "" {
			return fmt.Errorf("columns[%d].id is required", i)
		}
		if strings.TrimSpace(c.ProjectID) == "" {
			return fmt.Errorf("columns[%d].project_id is required", i)
		}
		if strings.TrimSpace(c.Name) == "" {
			return fmt.Errorf("columns[%d].name is required", i)
		}
		if c.Position < 0 {
			return fmt.Errorf("columns[%d].position must be >= 0", i)
		}
		if c.WIPLimit < 0 {
			return fmt.Errorf("columns[%d].wip_limit must be >= 0", i)
		}
		if c.CreatedAt.IsZero() || c.UpdatedAt.IsZero() {
			return fmt.Errorf("columns[%d] timestamps are required", i)
		}
		if _, ok := projectIDs[c.ProjectID]; !ok {
			return fmt.Errorf("columns[%d] references unknown project_id %q", i, c.ProjectID)
		}
		if _, exists := columnIDs[c.ID]; exists {
			return fmt.Errorf("duplicate column id: %q", c.ID)
		}
		columnIDs[c.ID] = struct{}{}
	}

	taskIDs := map[string]struct{}{}
	for i, t := range s.Tasks {
		if strings.TrimSpace(t.ID) == "" {
			return fmt.Errorf("tasks[%d].id is required", i)
		}
		if strings.TrimSpace(t.ProjectID) == "" {
			return fmt.Errorf("tasks[%d].project_id is required", i)
		}
		if strings.TrimSpace(t.ColumnID) == "" {
			return fmt.Errorf("tasks[%d].column_id is required", i)
		}
		if strings.TrimSpace(t.Title) == "" {
			return fmt.Errorf("tasks[%d].title is required", i)
		}
		if t.Position < 0 {
			return fmt.Errorf("tasks[%d].position must be >= 0", i)
		}
		switch t.Priority {
		case domain.PriorityLow, domain.PriorityMedium, domain.PriorityHigh:
		default:
			return fmt.Errorf("tasks[%d].priority must be low|medium|high", i)
		}
		if strings.TrimSpace(string(t.Kind)) == "" {
			t.Kind = domain.WorkKindTask
			s.Tasks[i].Kind = t.Kind
		}
		if t.Scope == "" {
			if strings.TrimSpace(t.ParentID) == "" {
				t.Scope = domain.KindAppliesToTask
			} else {
				t.Scope = domain.KindAppliesToSubtask
			}
			s.Tasks[i].Scope = t.Scope
		}
		if !domain.IsValidWorkItemAppliesTo(t.Scope) {
			return fmt.Errorf("tasks[%d].scope must be branch|phase|task|subtask", i)
		}
		if t.LifecycleState == "" {
			t.LifecycleState = domain.StateTodo
			s.Tasks[i].LifecycleState = t.LifecycleState
		}
		switch t.LifecycleState {
		case domain.StateTodo, domain.StateProgress, domain.StateDone, domain.StateArchived:
		default:
			return fmt.Errorf("tasks[%d].lifecycle_state must be todo|progress|done|archived", i)
		}
		if t.CreatedAt.IsZero() || t.UpdatedAt.IsZero() {
			return fmt.Errorf("tasks[%d] timestamps are required", i)
		}
		if _, ok := projectIDs[t.ProjectID]; !ok {
			return fmt.Errorf("tasks[%d] references unknown project_id %q", i, t.ProjectID)
		}
		if _, ok := columnIDs[t.ColumnID]; !ok {
			return fmt.Errorf("tasks[%d] references unknown column_id %q", i, t.ColumnID)
		}
		if _, exists := taskIDs[t.ID]; exists {
			return fmt.Errorf("duplicate task id: %q", t.ID)
		}
		taskIDs[t.ID] = struct{}{}
	}
	for i, t := range s.Tasks {
		if strings.TrimSpace(t.ParentID) == "" {
			continue
		}
		if t.ParentID == t.ID {
			return fmt.Errorf("tasks[%d].parent_id cannot reference itself", i)
		}
		if _, exists := taskIDs[t.ParentID]; !exists {
			return fmt.Errorf("tasks[%d] references unknown parent_id %q", i, t.ParentID)
		}
	}

	return nil
}

// upsertProject handles upsert project.
func (s *Service) upsertProject(ctx context.Context, p domain.Project) error {
	if _, err := s.repo.GetProject(ctx, p.ID); err == nil {
		return s.repo.UpdateProject(ctx, p)
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	return s.repo.CreateProject(ctx, p)
}

// sort handles sort.
func (s *Snapshot) sort() {
	sort.Slice(s.Projects, func(i, j int) bool {
		return s.Projects[i].ID < s.Projects[j].ID
	})
	sort.Slice(s.Columns, func(i, j int) bool {
		a := s.Columns[i]
		b := s.Columns[j]
		if a.ProjectID == b.ProjectID {
			if a.Position == b.Position {
				return a.ID < b.ID
			}
			return a.Position < b.Position
		}
		return a.ProjectID < b.ProjectID
	})
	sort.Slice(s.Tasks, func(i, j int) bool {
		a := s.Tasks[i]
		b := s.Tasks[j]
		if a.ProjectID == b.ProjectID {
			if a.ColumnID == b.ColumnID {
				if a.Position == b.Position {
					return a.ID < b.ID
				}
				return a.Position < b.Position
			}
			return a.ColumnID < b.ColumnID
		}
		return a.ProjectID < b.ProjectID
	})
}

// snapshotProjectFromDomain handles snapshot project from domain.
func snapshotProjectFromDomain(p domain.Project) SnapshotProject {
	return SnapshotProject{
		ID:          p.ID,
		Slug:        p.Slug,
		Name:        p.Name,
		Description: p.Description,
		Kind:        p.Kind,
		Metadata:    p.Metadata,
		CreatedAt:   p.CreatedAt.UTC(),
		UpdatedAt:   p.UpdatedAt.UTC(),
		ArchivedAt:  copyTimePtr(p.ArchivedAt),
	}
}

// snapshotColumnFromDomain handles snapshot column from domain.
func snapshotColumnFromDomain(c domain.Column) SnapshotColumn {
	return SnapshotColumn{
		ID:         c.ID,
		ProjectID:  c.ProjectID,
		Name:       c.Name,
		WIPLimit:   c.WIPLimit,
		Position:   c.Position,
		CreatedAt:  c.CreatedAt.UTC(),
		UpdatedAt:  c.UpdatedAt.UTC(),
		ArchivedAt: copyTimePtr(c.ArchivedAt),
	}
}

// snapshotTaskFromDomain handles snapshot task from domain.
func snapshotTaskFromDomain(t domain.Task) SnapshotTask {
	return SnapshotTask{
		ID:             t.ID,
		ProjectID:      t.ProjectID,
		ParentID:       t.ParentID,
		Kind:           t.Kind,
		Scope:          t.Scope,
		LifecycleState: t.LifecycleState,
		ColumnID:       t.ColumnID,
		Position:       t.Position,
		Title:          t.Title,
		Description:    t.Description,
		Priority:       t.Priority,
		DueAt:          copyTimePtr(t.DueAt),
		Labels:         append([]string(nil), t.Labels...),
		Metadata:       t.Metadata,
		CreatedByActor: t.CreatedByActor,
		UpdatedByActor: t.UpdatedByActor,
		UpdatedByType:  t.UpdatedByType,
		CreatedAt:      t.CreatedAt.UTC(),
		UpdatedAt:      t.UpdatedAt.UTC(),
		StartedAt:      copyTimePtr(t.StartedAt),
		CompletedAt:    copyTimePtr(t.CompletedAt),
		ArchivedAt:     copyTimePtr(t.ArchivedAt),
		CanceledAt:     copyTimePtr(t.CanceledAt),
	}
}

// toDomain converts domain.
func (p SnapshotProject) toDomain() domain.Project {
	slug := strings.TrimSpace(p.Slug)
	if slug == "" {
		slug = fallbackSlug(p.Name)
	}
	kind := domain.NormalizeKindID(p.Kind)
	if kind == "" {
		kind = domain.DefaultProjectKind
	}
	return domain.Project{
		ID:          strings.TrimSpace(p.ID),
		Slug:        slug,
		Name:        strings.TrimSpace(p.Name),
		Description: strings.TrimSpace(p.Description),
		Kind:        kind,
		Metadata:    p.Metadata,
		CreatedAt:   p.CreatedAt.UTC(),
		UpdatedAt:   p.UpdatedAt.UTC(),
		ArchivedAt:  copyTimePtr(p.ArchivedAt),
	}
}

// toDomain converts domain.
func (c SnapshotColumn) toDomain() domain.Column {
	return domain.Column{
		ID:         strings.TrimSpace(c.ID),
		ProjectID:  strings.TrimSpace(c.ProjectID),
		Name:       strings.TrimSpace(c.Name),
		WIPLimit:   c.WIPLimit,
		Position:   c.Position,
		CreatedAt:  c.CreatedAt.UTC(),
		UpdatedAt:  c.UpdatedAt.UTC(),
		ArchivedAt: copyTimePtr(c.ArchivedAt),
	}
}

// toDomain converts domain.
func (t SnapshotTask) toDomain() domain.Task {
	labels := append([]string(nil), t.Labels...)
	state := t.LifecycleState
	if state == "" {
		state = domain.StateTodo
	}
	kind := t.Kind
	if kind == "" {
		kind = domain.WorkKindTask
	}
	scope := domain.NormalizeKindAppliesTo(t.Scope)
	if scope == "" {
		if strings.TrimSpace(t.ParentID) == "" {
			scope = domain.KindAppliesToTask
		} else {
			scope = domain.KindAppliesToSubtask
		}
	}
	updatedType := t.UpdatedByType
	if updatedType == "" {
		updatedType = domain.ActorTypeUser
	}
	createdBy := strings.TrimSpace(t.CreatedByActor)
	if createdBy == "" {
		createdBy = "kan-user"
	}
	updatedBy := strings.TrimSpace(t.UpdatedByActor)
	if updatedBy == "" {
		updatedBy = createdBy
	}
	return domain.Task{
		ID:             strings.TrimSpace(t.ID),
		ProjectID:      strings.TrimSpace(t.ProjectID),
		ParentID:       strings.TrimSpace(t.ParentID),
		Kind:           kind,
		Scope:          scope,
		LifecycleState: state,
		ColumnID:       strings.TrimSpace(t.ColumnID),
		Position:       t.Position,
		Title:          strings.TrimSpace(t.Title),
		Description:    strings.TrimSpace(t.Description),
		Priority:       t.Priority,
		DueAt:          copyTimePtr(t.DueAt),
		Labels:         labels,
		Metadata:       t.Metadata,
		CreatedByActor: createdBy,
		UpdatedByActor: updatedBy,
		UpdatedByType:  updatedType,
		CreatedAt:      t.CreatedAt.UTC(),
		UpdatedAt:      t.UpdatedAt.UTC(),
		StartedAt:      copyTimePtr(t.StartedAt),
		CompletedAt:    copyTimePtr(t.CompletedAt),
		ArchivedAt:     copyTimePtr(t.ArchivedAt),
		CanceledAt:     copyTimePtr(t.CanceledAt),
	}
}

// fallbackSlug provides fallback slug.
func fallbackSlug(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "-")
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	return strings.Trim(name, "-")
}

// copyTimePtr copies time ptr.
func copyTimePtr(in *time.Time) *time.Time {
	if in == nil {
		return nil
	}
	t := in.UTC().Truncate(time.Second)
	return &t
}
