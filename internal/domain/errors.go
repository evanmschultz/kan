package domain

import "errors"

// ErrInvalidID and related errors describe validation and runtime failures.
var (
	ErrInvalidID       = errors.New("invalid id")
	ErrInvalidName     = errors.New("invalid name")
	ErrInvalidTitle    = errors.New("invalid title")
	ErrInvalidPriority = errors.New("invalid priority")
	ErrInvalidPosition = errors.New("invalid position")
	ErrInvalidColumnID = errors.New("invalid column id")
)
