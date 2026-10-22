// Package domain holds the listening context's entities, value objects, pure
// services and errors.
package domain

// Error is a domain error with a stable machine Code and a human Message,
// optionally filed under a category sentinel so adapters can map a whole family
// to one transport status with errors.Is.
type Error struct {
	Code    string
	Message string
	parent  error
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func (e *Error) Unwrap() error { return e.parent }

// NewError creates a category sentinel.
func NewError(code, message string) *Error { return &Error{Code: code, Message: message} }

// Wrap creates a concrete error filed under category.
func Wrap(category *Error, code, message string) *Error {
	return &Error{Code: code, Message: message, parent: category}
}

// Category sentinels classify a family of errors so adapters can map them to
// one transport status with errors.Is.
var (
	ErrNotFound     = NewError("NOT_FOUND", "resource not found")
	ErrValidation   = NewError("VALIDATION_FAILED", "validation failed")
	ErrConflict     = NewError("CONFLICT", "conflicting state")
	ErrForbidden    = NewError("FORBIDDEN", "operation not permitted")
	ErrUnauthorized = NewError("UNAUTHORIZED", "authentication required")
)

// Concrete errors returned by the domain, each filed under a category
// sentinel. ErrEpisodeNotFound covers both a missing episode and one that is
// still a draft.
var (
	ErrInvalidID        = Wrap(ErrValidation, "INVALID_ID", "invalid identifier")
	ErrInvalidPosition  = Wrap(ErrValidation, "INVALID_POSITION", "position must not be negative")
	ErrEpisodeNotFound  = Wrap(ErrNotFound, "EPISODE_NOT_FOUND", "episode not found")
	ErrProgressNotFound = Wrap(ErrNotFound, "PROGRESS_NOT_FOUND", "progress not found")
)
