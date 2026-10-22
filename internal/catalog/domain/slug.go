package domain

import "strings"

// MaxSlugLength is the maximum accepted slug length in bytes.
const MaxSlugLength = 120

// Slugify derives a URL-safe slug from title. The result is lowercased ASCII
// letters and digits, every other run of characters collapsed into a single
// '-', with leading and trailing '-' trimmed. It is truncar on the last '-'
// when it exceeds MaxSlugLength.
func Slugify(title string) string {
	var b strings.Builder
	pendingDash := false
	for _, r := range strings.ToLower(title) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if pendingDash && b.Len() > 0 {
				b.WriteByte('-')
			}
			pendingDash = false
			b.WriteRune(r)
			continue
		}
		pendingDash = true
	}

	slug := b.String()
	if len(slug) > MaxSlugLength {
		slug = slug[:MaxSlugLength]
		slug = strings.TrimRight(slug, "-")
	}
	return slug
}

// validateSlug enforces ^[a-z0-9]+(-[a-z0-9]+)*$ and the length limit.
func validateSlug(slug string) error {
	if slug == "" || len(slug) > MaxSlugLength {
		return ErrInvalidSeries
	}
	prevDash := false
	for i := 0; i < len(slug); i++ {
		c := slug[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			prevDash = false
		case c == '-' && i > 0 && i < len(slug)-1 && !prevDash:
			prevDash = true
		default:
			return ErrInvalidSeries
		}
	}
	return nil
}
