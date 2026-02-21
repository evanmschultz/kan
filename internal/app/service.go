package app

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/evanschultz/kan/internal/domain"
)

type DeleteMode string

const (
	DeleteModeArchive DeleteMode = "archive"
	DeleteModeHard    DeleteMode = "hard"
)

type ServiceConfig struct {
	DefaultDeleteMode DeleteMode
}

type IDGenerator func() string
type Clock func() time.Time

type Service struct {
	repo              Repository
	idGen             IDGenerator
	clock             Clock
	defaultDeleteMode DeleteMode
}

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

	return &Service{
		repo:              repo,
		idGen:             idGen,
		clock:             clock,
		defaultDeleteMode: cfg.DefaultDeleteMode,
	}
}

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

	columnNames := []string{"To Do", "In Progress", "Done"}
	for i, name := range columnNames {
		column, colErr := domain.NewColumn(s.idGen(), project.ID, name, i, 0, now)
		if colErr != nil {
			return domain.Project{}, colErr
		}
		if colErr = s.repo.CreateColumn(ctx, column); colErr != nil {
			return domain.Project{}, colErr
		}
	}

	return project, nil
}

func (s *Service) CreateProject(ctx context.Context, name, description string) (domain.Project, error) {
	now := s.clock()
	project, err := domain.NewProject(s.idGen(), name, description, now)
	if err != nil {
		return domain.Project{}, err
	}
	if err := s.repo.CreateProject(ctx, project); err != nil {
		return domain.Project{}, err
	}
	return project, nil
}

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

type CreateTaskInput struct {
	ProjectID   string
	ColumnID    string
	Title       string
	Description string
	Priority    domain.Priority
	DueAt       *time.Time
	Labels      []string
}

type UpdateTaskInput struct {
	TaskID      string
	Title       string
	Description string
	Priority    domain.Priority
	DueAt       *time.Time
	Labels      []string
}

func (s *Service) CreateTask(ctx context.Context, in CreateTaskInput) (domain.Task, error) {
	tasks, err := s.repo.ListTasks(ctx, in.ProjectID, false)
	if err != nil {
		return domain.Task{}, err
	}
	position := 0
	for _, t := range tasks {
		if t.ColumnID == in.ColumnID && t.Position >= position {
			position = t.Position + 1
		}
	}

	task, err := domain.NewTask(domain.TaskInput{
		ID:          s.idGen(),
		ProjectID:   in.ProjectID,
		ColumnID:    in.ColumnID,
		Position:    position,
		Title:       in.Title,
		Description: in.Description,
		Priority:    in.Priority,
		DueAt:       in.DueAt,
		Labels:      in.Labels,
	}, s.clock())
	if err != nil {
		return domain.Task{}, err
	}

	if err := s.repo.CreateTask(ctx, task); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

func (s *Service) MoveTask(ctx context.Context, taskID, toColumnID string, position int) (domain.Task, error) {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	if err := task.Move(toColumnID, position, s.clock()); err != nil {
		return domain.Task{}, err
	}
	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

func (s *Service) RestoreTask(ctx context.Context, taskID string) (domain.Task, error) {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	task.Restore(s.clock())
	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

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

func (s *Service) UpdateTask(ctx context.Context, in UpdateTaskInput) (domain.Task, error) {
	task, err := s.repo.GetTask(ctx, in.TaskID)
	if err != nil {
		return domain.Task{}, err
	}
	if err := task.UpdateDetails(in.Title, in.Description, in.Priority, in.DueAt, in.Labels, s.clock()); err != nil {
		return domain.Task{}, err
	}
	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

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

func (s *Service) ListProjects(ctx context.Context, includeArchived bool) ([]domain.Project, error) {
	return s.repo.ListProjects(ctx, includeArchived)
}

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

func (s *Service) SearchTasks(ctx context.Context, projectID, query string, includeArchived bool) ([]domain.Task, error) {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return s.ListTasks(ctx, projectID, includeArchived)
	}

	tasks, err := s.repo.ListTasks(ctx, projectID, includeArchived)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Task, 0, len(tasks))
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
