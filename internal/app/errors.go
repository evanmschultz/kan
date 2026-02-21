package app

import "errors"

var (
	ErrNotFound          = errors.New("not found")
	ErrInvalidDeleteMode = errors.New("invalid delete mode")
)
