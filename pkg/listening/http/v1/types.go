// Package v1 defines the public JSON contract of the listening HTTP API. It
// contains request and response types only.
package v1

// Progress is the public representation of one listener's progress on one
// episode.
type Progress struct {
	EpisodeID       string  `json:"episode_id"`
	PositionSeconds float64 `json:"position_seconds"`
	Completed       bool    `json:"completed"`
	UpdatedAt       string  `json:"updated_at"`
}

// ProgressList is the response body of the progress listing.
type ProgressList struct {
	Progress []Progress `json:"progress"`
}

// ProgressInput is the request body of a position update. Completed is
// optional: nil records the position, true marks the episode played and false
// records the position on a re-opened episode.
type ProgressInput struct {
	PositionSeconds float64 `json:"position_seconds"`
	Completed       *bool   `json:"completed,omitempty"`
}

// EpisodeSummary is the catalog episode information embedded in a
// continue-listening item.
type EpisodeSummary struct {
	ID              string  `json:"id"`
	Number          int     `json:"number"`
	Title           string  `json:"title"`
	DurationSeconds float64 `json:"duration_seconds"`
	ChapterCount    int     `json:"chapter_count"`
	SeriesID        string  `json:"series_id"`
	SeriesTitle     string  `json:"series_title"`
	SeriesSlug      string  `json:"series_slug"`
	AccentColor     string  `json:"accent_color"`
	CoverURL        *string `json:"cover_url"`
}

// ContinueItem pairs an in-progress row with its catalog episode.
type ContinueItem struct {
	Progress Progress       `json:"progress"`
	Episode  EpisodeSummary `json:"episode"`
}

// ContinueList is the response body of the continue-listening listing.
type ContinueList struct {
	Items []ContinueItem `json:"items"`
}

// Error is the body of every API error response.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
