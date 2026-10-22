package domain

import "github.com/google/uuid"

// SeriesID is the typed identifier of a catalog series.
type SeriesID string

// EpisodeID is the typed identifier of a catalog episode.
type EpisodeID string

// NewID mints a time-ordered UUIDv7 as a string, so ids sort by creation time.
func NewID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return id.String()
}

func parseUUID(raw string) (string, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return "", ErrInvalidID
	}
	return id.String(), nil
}

// ParseSeriesID validates and canonicalizes a series identifier.
func ParseSeriesID(raw string) (SeriesID, error) {
	id, err := parseUUID(raw)
	return SeriesID(id), err
}

// ParseEpisodeID validates and canonicalizes an episode identifier.
func ParseEpisodeID(raw string) (EpisodeID, error) {
	id, err := parseUUID(raw)
	return EpisodeID(id), err
}
