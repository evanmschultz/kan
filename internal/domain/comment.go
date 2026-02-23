package domain

import (
	"slices"
	"strings"
	"time"
)

// CommentTargetType identifies the entity type a comment belongs to.
type CommentTargetType string

// Comment target type values.
const (
	CommentTargetTypeProject  CommentTargetType = "project"
	CommentTargetTypeTask     CommentTargetType = CommentTargetType(WorkKindTask)
	CommentTargetTypeSubtask  CommentTargetType = CommentTargetType(WorkKindSubtask)
	CommentTargetTypePhase    CommentTargetType = CommentTargetType(WorkKindPhase)
	CommentTargetTypeDecision CommentTargetType = CommentTargetType(WorkKindDecision)
	CommentTargetTypeNote     CommentTargetType = CommentTargetType(WorkKindNote)
)

// validCommentTargetTypes stores supported target-type values.
var validCommentTargetTypes = []CommentTargetType{
	CommentTargetTypeProject,
	CommentTargetTypeTask,
	CommentTargetTypeSubtask,
	CommentTargetTypePhase,
	CommentTargetTypeDecision,
	CommentTargetTypeNote,
}

// CommentTarget identifies a concrete target within a project.
type CommentTarget struct {
	ProjectID  string
	TargetType CommentTargetType
	TargetID   string
}

// Comment stores an ownership-attributed note attached to a target.
type Comment struct {
	ID           string
	ProjectID    string
	TargetType   CommentTargetType
	TargetID     string
	BodyMarkdown string
	ActorType    ActorType
	AuthorName   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// CommentInput holds input values for comment creation operations.
type CommentInput struct {
	ID           string
	ProjectID    string
	TargetType   CommentTargetType
	TargetID     string
	BodyMarkdown string
	ActorType    ActorType
	AuthorName   string
}

// NewComment constructs a normalized comment.
func NewComment(in CommentInput, now time.Time) (Comment, error) {
	in.ID = strings.TrimSpace(in.ID)
	if in.ID == "" {
		return Comment{}, ErrInvalidID
	}

	target, err := NormalizeCommentTarget(CommentTarget{
		ProjectID:  in.ProjectID,
		TargetType: in.TargetType,
		TargetID:   in.TargetID,
	})
	if err != nil {
		return Comment{}, err
	}

	body := strings.TrimSpace(in.BodyMarkdown)
	if body == "" {
		return Comment{}, ErrInvalidBodyMarkdown
	}

	actorType := normalizeActorTypeValue(in.ActorType)
	if actorType == "" {
		actorType = ActorTypeUser
	}
	if !isValidActorType(actorType) {
		return Comment{}, ErrInvalidActorType
	}

	authorName := strings.TrimSpace(in.AuthorName)
	if authorName == "" {
		authorName = "kan-user"
	}

	timestamp := now.UTC()
	return Comment{
		ID:           in.ID,
		ProjectID:    target.ProjectID,
		TargetType:   target.TargetType,
		TargetID:     target.TargetID,
		BodyMarkdown: body,
		ActorType:    actorType,
		AuthorName:   authorName,
		CreatedAt:    timestamp,
		UpdatedAt:    timestamp,
	}, nil
}

// NormalizeCommentTarget validates and canonicalizes comment target identifiers.
func NormalizeCommentTarget(target CommentTarget) (CommentTarget, error) {
	target.ProjectID = strings.TrimSpace(target.ProjectID)
	target.TargetID = strings.TrimSpace(target.TargetID)
	target.TargetType = NormalizeCommentTargetType(target.TargetType)

	if target.ProjectID == "" {
		return CommentTarget{}, ErrInvalidID
	}
	if target.TargetID == "" {
		return CommentTarget{}, ErrInvalidTargetID
	}
	if !IsValidCommentTargetType(target.TargetType) {
		return CommentTarget{}, ErrInvalidTargetType
	}
	return target, nil
}

// NormalizeCommentTargetType canonicalizes target types to their stored form.
func NormalizeCommentTargetType(targetType CommentTargetType) CommentTargetType {
	return CommentTargetType(strings.TrimSpace(strings.ToLower(string(targetType))))
}

// IsValidCommentTargetType reports whether the target type is supported.
func IsValidCommentTargetType(targetType CommentTargetType) bool {
	targetType = NormalizeCommentTargetType(targetType)
	return slices.Contains(validCommentTargetTypes, targetType)
}

// normalizeActorTypeValue canonicalizes actor type values without applying defaults.
func normalizeActorTypeValue(actorType ActorType) ActorType {
	return ActorType(strings.TrimSpace(strings.ToLower(string(actorType))))
}
