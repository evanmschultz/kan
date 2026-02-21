package domain

import "errors"

var (
	ErrInvalidID       = errors.New("invalid id")
	ErrInvalidName     = errors.New("invalid name")
	ErrInvalidTitle    = errors.New("invalid title")
	ErrInvalidPriority = errors.New("invalid priority")
	ErrInvalidPosition = errors.New("invalid position")
	ErrInvalidColumnID = errors.New("invalid column id")
)
