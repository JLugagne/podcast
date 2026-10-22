package shared

import (
	"errors"
	"net/http"

	"github.com/JLugagne/podcast/internal/catalog/domain"
)

// ErrMissingFile reports a multipart body without a part named "file".
var ErrMissingFile = domain.Wrap(domain.ErrValidation, "MISSING_FILE", "multipart body must contain a file part")

// ErrInvalidMultipart reports a multipart body that could not be parsed.
var ErrInvalidMultipart = domain.Wrap(domain.ErrValidation, "INVALID_MULTIPART", "malformed multipart body")

// ErrVersionRequired reports an update request without the optimistic-lock
// version.
var ErrVersionRequired = domain.Wrap(domain.ErrValidation, "VERSION_REQUIRED", "version is required")

// ErrInvalidDuration reports an invalid duration_seconds multipart field.
var ErrInvalidDuration = domain.Wrap(domain.ErrValidation, "INVALID_DURATION", "duration_seconds must be a non-negative number")

// MultipartError keeps request-size errors untouched so they map to 413, and
// turns every other multipart read failure into ErrInvalidMultipart.
func MultipartError(err error) error {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return err
	}
	return ErrInvalidMultipart
}
