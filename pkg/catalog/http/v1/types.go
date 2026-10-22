// Package v1 defines the public JSON contract of the catalog HTTP API. It
// contains request and response types only.
package v1

// Series is the public representation of a catalog series.
type Series struct {
	ID                   string  `json:"id"`
	Slug                 string  `json:"slug"`
	FeedURL              string  `json:"feed_url"`
	Title                string  `json:"title"`
	Author               string  `json:"author"`
	Description          string  `json:"description"`
	AccentColor          string  `json:"accent_color"`
	CoverURL             *string `json:"cover_url"`
	EpisodeCount         int     `json:"episode_count"`
	PublishedCount       int     `json:"published_count"`
	TotalDurationSeconds float64 `json:"total_duration_seconds"`
	CreatedAt            string  `json:"created_at"`
	UpdatedAt            string  `json:"updated_at"`
	Version              int     `json:"version"`
}

// SeriesList is the response body of the series listing.
type SeriesList struct {
	Series []Series `json:"series"`
}

// SeriesInput is the request body of series creation and update. Version is
// required on update only.
type SeriesInput struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Description string `json:"description"`
	AccentColor string `json:"accent_color"`
	Version     *int   `json:"version"`
}

// Audio describes the audio file attached to an episode.
type Audio struct {
	URL             string  `json:"url"`
	ContentType     string  `json:"content_type"`
	SizeBytes       int64   `json:"size_bytes"`
	DurationSeconds float64 `json:"duration_seconds"`
	OriginalName    string  `json:"original_name"`
}

// Chapter is one chapter of an episode in the API representation.
type Chapter struct {
	StartSeconds float64  `json:"start_seconds"`
	EndSeconds   *float64 `json:"end_seconds,omitempty"`
	Title        string   `json:"title"`
	URL          string   `json:"url,omitempty"`
	ImageURL     string   `json:"image_url,omitempty"`
	TOC          bool     `json:"toc"`
}

// Episode is the public representation of an episode.
type Episode struct {
	ID          string    `json:"id"`
	SeriesID    string    `json:"series_id"`
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Audio       *Audio    `json:"audio"`
	Chapters    []Chapter `json:"chapters"`
	PublishedAt *string   `json:"published_at,omitempty"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
	Version     int       `json:"version"`
}

// EpisodeList is the response body of the episode listing.
type EpisodeList struct {
	Episodes []Episode `json:"episodes"`
}

// EpisodeInput is the request body of episode creation and update. Number is
// optional on creation (nil picks the next number) and Version is required on
// update only.
type EpisodeInput struct {
	Number      *int   `json:"number"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     *int   `json:"version"`
}

// ChapterItem is one chapter of a Podcasting 2.0 JSON chapters document.
type ChapterItem struct {
	StartTime float64  `json:"startTime"`
	Title     string   `json:"title"`
	Img       string   `json:"img,omitempty"`
	URL       string   `json:"url,omitempty"`
	TOC       *bool    `json:"toc,omitempty"`
	EndTime   *float64 `json:"endTime,omitempty"`
}

// ChaptersDocument is the Podcasting 2.0 JSON chapters document exchanged by
// the chapters endpoints.
type ChaptersDocument struct {
	Version  string        `json:"version"`
	Chapters []ChapterItem `json:"chapters"`
}

// Error is the body of every API error response.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
