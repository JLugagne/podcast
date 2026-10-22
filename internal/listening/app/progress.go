package app

import (
	"context"
	"errors"

	"github.com/JLugagne/podcast/internal/listening/domain"
	"github.com/JLugagne/podcast/internal/listening/domain/repositories/progressrepo"
	"github.com/JLugagne/podcast/internal/listening/domain/usecases/progressuc"
)

// RecordPosition stores a playback position, clamping it with the episode
// duration read from the catalog. An explicit completed=true marks the episode
// played whatever the position is; completed=false or a nil pointer records the
// position, which may complete or re-open the episode.
func (a *App) RecordPosition(ctx context.Context, actor *domain.Actor, episode domain.EpisodeRef, position float64, completed *bool) (*domain.Progress, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	info, err := a.catalog.PublishedEpisode(ctx, episode)
	if err != nil {
		return nil, err
	}
	now := a.now()
	progress, err := a.loadProgress(ctx, listenerOf(actor), episode, now)
	if err != nil {
		return nil, err
	}
	if completed != nil && *completed {
		progress.MarkCompleted(info.DurationSeconds, now)
	} else if err := progress.Record(position, info.DurationSeconds, now); err != nil {
		return nil, err
	}
	if err := a.progress.Save(ctx, progress); err != nil {
		return nil, err
	}
	return progress, nil
}

// MarkPlayed marks the episode played, placing the position at the end of its
// catalog duration. The episode must be published.
func (a *App) MarkPlayed(ctx context.Context, actor *domain.Actor, episode domain.EpisodeRef) (*domain.Progress, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	info, err := a.catalog.PublishedEpisode(ctx, episode)
	if err != nil {
		return nil, err
	}
	now := a.now()
	progress, err := a.loadProgress(ctx, listenerOf(actor), episode, now)
	if err != nil {
		return nil, err
	}
	progress.MarkCompleted(info.DurationSeconds, now)
	if err := a.progress.Save(ctx, progress); err != nil {
		return nil, err
	}
	return progress, nil
}

// MarkUnplayed forgets the episode. It is idempotent: forgetting an episode the
// listener never started succeeds.
func (a *App) MarkUnplayed(ctx context.Context, actor *domain.Actor, episode domain.EpisodeRef) error {
	if err := requireActor(actor); err != nil {
		return err
	}
	return a.progress.Delete(ctx, listenerOf(actor), episode)
}

// ListProgress returns every tracked episode of the listener, most recently
// updated first.
func (a *App) ListProgress(ctx context.Context, actor *domain.Actor) ([]*domain.Progress, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	return a.progress.List(ctx, listenerOf(actor), progressrepo.Filter{})
}

// ContinueListening returns up to limit started-but-unfinished episodes, newest
// first, each enriched with its catalog episode. Episodes that vanished or were
// unpublished in the catalog are skipped without consuming a slot, so callers
// get as many available items as the listener has.
func (a *App) ContinueListening(ctx context.Context, actor *domain.Actor, limit int) ([]progressuc.ContinueItem, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	limit = clampContinueLimit(limit)
	rows, err := a.progress.List(ctx, listenerOf(actor), progressrepo.Filter{
		InProgressOnly: true,
		Limit:          continueScanLimit,
	})
	if err != nil {
		return nil, err
	}

	items := make([]progressuc.ContinueItem, 0, limit)
	for _, progress := range rows {
		if len(items) >= limit {
			break
		}
		info, err := a.catalog.PublishedEpisode(ctx, progress.EpisodeRef())
		if errors.Is(err, domain.ErrEpisodeNotFound) {
			a.log.DebugContext(ctx, "listening: skipping unavailable episode", "episode_id", string(progress.EpisodeRef()))
			continue
		}
		if err != nil {
			return nil, err
		}
		items = append(items, progressuc.ContinueItem{Progress: progress, Episode: info})
	}
	return items, nil
}

// continueScanLimit caps how many in-progress rows are examined to build the
// continue-listening list. Unavailable episodes are skipped without consuming a
// slot, so the scan is wider than the requested limit (capped at
// progressuc.MaxContinueLimit).
const continueScanLimit = 500

// clampContinueLimit applies the continue-listening bounds: a non-positive
// request falls back to the default and anything above the maximum is capped.
func clampContinueLimit(limit int) int {
	switch {
	case limit <= 0:
		return progressuc.DefaultContinueLimit
	case limit > progressuc.MaxContinueLimit:
		return progressuc.MaxContinueLimit
	default:
		return limit
	}
}
