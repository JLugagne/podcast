// Package progressuc declares the inbound use-case ports for listening
// progress.
package progressuc

import (
	"context"

	"github.com/JLugagne/podcast/internal/listening/domain"
	"github.com/JLugagne/podcast/internal/listening/domain/catalogport"
)

// Continue-listening bounds, applied by the app: a non-positive requested limit
// falls back to DefaultContinueLimit and anything above MaxContinueLimit is
// capped.
const (
	DefaultContinueLimit = 5
	MaxContinueLimit     = 50
)

// ContinueItem is one in-progress row enriched with its catalog episode.
type ContinueItem struct {
	Progress *domain.Progress
	Episode  catalogport.EpisodeInfo
}

// ProgressCommands mutate listening progress.
type ProgressCommands interface {
	RecordPosition(ctx context.Context, actor *domain.Actor, episode domain.EpisodeRef, position float64, completed *bool) (*domain.Progress, error)
	MarkPlayed(ctx context.Context, actor *domain.Actor, episode domain.EpisodeRef) (*domain.Progress, error)
	MarkUnplayed(ctx context.Context, actor *domain.Actor, episode domain.EpisodeRef) error
}

// ProgressQueries never mutate state.
type ProgressQueries interface {
	ListProgress(ctx context.Context, actor *domain.Actor) ([]*domain.Progress, error)
	ContinueListening(ctx context.Context, actor *domain.Actor, limit int) ([]ContinueItem, error)
}
