package domain

import "errors"

// ErrInvalidID and related errors describe validation and runtime failures.
var (
	ErrInvalidID             = errors.New("invalid id")
	ErrInvalidName           = errors.New("invalid name")
	ErrInvalidTitle          = errors.New("invalid title")
	ErrInvalidPriority       = errors.New("invalid priority")
	ErrInvalidPosition       = errors.New("invalid position")
	ErrInvalidColumnID       = errors.New("invalid column id")
	ErrInvalidParentID       = errors.New("invalid parent id")
	ErrInvalidKind           = errors.New("invalid kind")
	ErrInvalidLifecycleState = errors.New("invalid lifecycle state")
	ErrInvalidActorType      = errors.New("invalid actor type")
	ErrTransitionBlocked     = errors.New("transition blocked by completion contract")
)
