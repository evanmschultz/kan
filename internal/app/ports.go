package app

import (
	"context"

	"github.com/evanschultz/kan/internal/domain"
)

// Repository represents repository data used by this package.
type Repository interface {
	CreateProject(context.Context, domain.Project) error
	UpdateProject(context.Context, domain.Project) error
	GetProject(context.Context, string) (domain.Project, error)
	ListProjects(context.Context, bool) ([]domain.Project, error)

	CreateColumn(context.Context, domain.Column) error
	UpdateColumn(context.Context, domain.Column) error
	ListColumns(context.Context, string, bool) ([]domain.Column, error)

	CreateTask(context.Context, domain.Task) error
	UpdateTask(context.Context, domain.Task) error
	GetTask(context.Context, string) (domain.Task, error)
	ListTasks(context.Context, string, bool) ([]domain.Task, error)
	DeleteTask(context.Context, string) error
	CreateComment(context.Context, domain.Comment) error
	ListCommentsByTarget(context.Context, domain.CommentTarget) ([]domain.Comment, error)
	ListProjectChangeEvents(context.Context, string, int) ([]domain.ChangeEvent, error)
}
