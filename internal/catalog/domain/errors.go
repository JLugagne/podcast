// Package domain holds the catalog context's entities, value objects, pure
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
// sentinel.
var (
	ErrInvalidID = Wrap(ErrValidation, "INVALID_ID", "invalid identifier")

	ErrSeriesNotFound  = Wrap(ErrNotFound, "SERIES_NOT_FOUND", "series not found")
	ErrEpisodeNotFound = Wrap(ErrNotFound, "EPISODE_NOT_FOUND", "episode not found")
	ErrAssetNotFound   = Wrap(ErrNotFound, "ASSET_NOT_FOUND", "asset not found")

	ErrSlugTaken          = Wrap(ErrConflict, "SLUG_TAKEN", "slug already in use")
	ErrEpisodeNumberTaken = Wrap(ErrConflict, "EPISODE_NUMBER_TAKEN", "episode number already in use")
	ErrVersionConflict    = Wrap(ErrConflict, "VERSION_CONFLICT", "the resource was modified concurrently")
	ErrSeriesNotEmpty     = Wrap(ErrConflict, "SERIES_NOT_EMPTY", "series still has episodes")

	ErrInvalidSeries          = Wrap(ErrValidation, "INVALID_SERIES", "invalid series")
	ErrInvalidEpisode         = Wrap(ErrValidation, "INVALID_EPISODE", "invalid episode")
	ErrInvalidChapters        = Wrap(ErrValidation, "INVALID_CHAPTERS", "invalid chapters")
	ErrChapterOutOfRange      = Wrap(ErrValidation, "CHAPTER_OUT_OF_RANGE", "chapter start is beyond the audio duration")
	ErrUnsupportedAudioType   = Wrap(ErrValidation, "UNSUPPORTED_AUDIO_TYPE", "unsupported audio content type")
	ErrUnsupportedImageType   = Wrap(ErrValidation, "UNSUPPORTED_IMAGE_TYPE", "unsupported image content type")
	ErrEpisodeAudioMissing    = Wrap(ErrValidation, "EPISODE_AUDIO_MISSING", "episode has no audio")
	ErrEpisodeChaptersMissing = Wrap(ErrValidation, "EPISODE_CHAPTERS_MISSING", "episode has no chapters")
)
