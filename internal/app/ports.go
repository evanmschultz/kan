package app

import (
	"context"
	"time"

	"github.com/evanschultz/kan/internal/domain"
)

// Repository represents repository data used by this package.
type Repository interface {
	CreateProject(context.Context, domain.Project) error
	UpdateProject(context.Context, domain.Project) error
	GetProject(context.Context, string) (domain.Project, error)
	ListProjects(context.Context, bool) ([]domain.Project, error)
	SetProjectAllowedKinds(context.Context, string, []domain.KindID) error
	ListProjectAllowedKinds(context.Context, string) ([]domain.KindID, error)

	CreateKindDefinition(context.Context, domain.KindDefinition) error
	UpdateKindDefinition(context.Context, domain.KindDefinition) error
	GetKindDefinition(context.Context, domain.KindID) (domain.KindDefinition, error)
	ListKindDefinitions(context.Context, bool) ([]domain.KindDefinition, error)

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

	CreateCapabilityLease(context.Context, domain.CapabilityLease) error
	UpdateCapabilityLease(context.Context, domain.CapabilityLease) error
	GetCapabilityLease(context.Context, string) (domain.CapabilityLease, error)
	ListCapabilityLeasesByScope(context.Context, string, domain.CapabilityScopeType, string) ([]domain.CapabilityLease, error)
	RevokeCapabilityLeasesByScope(context.Context, string, domain.CapabilityScopeType, string, time.Time, string) error
}
