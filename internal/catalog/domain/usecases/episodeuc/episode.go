// Package episodeuc declares the inbound use-case ports for episodes.
package episodeuc

import (
	"context"
	"io"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/blobstore"
)

// CreateInput creates an episode. A nil Number means "next available number".
type CreateInput struct {
	Number      *int
	Title       string
	Description string
}

// AudioUploadInput is a streamed audio upload.
type AudioUploadInput struct {
	Reader          io.Reader
	ContentType     string
	OriginalName    string
	DurationSeconds float64
}

// ListQuery narrows ListEpisodes. A nil Status returns every episode.
type ListQuery struct {
	Status *domain.EpisodeStatus
}

// EpisodeCommands mutate episodes.
type EpisodeCommands interface {
	CreateEpisode(ctx context.Context, actor *domain.Actor, seriesID domain.SeriesID, in CreateInput) (*domain.Episode, error)
	UpdateEpisode(ctx context.Context, actor *domain.Actor, id domain.EpisodeID, attrs domain.EpisodeAttributes, expectedVersion int) (*domain.Episode, error)
	DeleteEpisode(ctx context.Context, actor *domain.Actor, id domain.EpisodeID) error
	UploadAudio(ctx context.Context, actor *domain.Actor, id domain.EpisodeID, in AudioUploadInput) (*domain.Episode, error)
	SetChapters(ctx context.Context, actor *domain.Actor, id domain.EpisodeID, chapters []domain.Chapter) (*domain.Episode, error)
	Publish(ctx context.Context, actor *domain.Actor, id domain.EpisodeID) (*domain.Episode, error)
	Unpublish(ctx context.Context, actor *domain.Actor, id domain.EpisodeID) (*domain.Episode, error)
}

// EpisodeQueries never mutate state.
type EpisodeQueries interface {
	GetEpisode(ctx context.Context, actor *domain.Actor, id domain.EpisodeID) (*domain.Episode, error)
	ListEpisodes(ctx context.Context, actor *domain.Actor, seriesID domain.SeriesID, q ListQuery) ([]*domain.Episode, error)
	OpenAudio(ctx context.Context, actor *domain.Actor, id domain.EpisodeID) (blobstore.Blob, error)
}
