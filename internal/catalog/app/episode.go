package app

import (
	"context"
	"fmt"
	"math"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/blobstore"
	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/episoderepo"
	"github.com/JLugagne/podcast/internal/catalog/domain/uow"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/episodeuc"
)

// CreateEpisode creates an episode in the series. A nil Number picks the next
// available number.
func (a *App) CreateEpisode(ctx context.Context, actor *domain.Actor, seriesID domain.SeriesID, in episodeuc.CreateInput) (*domain.Episode, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	now := a.now()
	var out *domain.Episode
	err := a.uow.Do(ctx, func(ctx context.Context, r uow.Repositories) error {
		if _, err := r.Series().Get(ctx, seriesID); err != nil {
			return err
		}
		number := 0
		if in.Number != nil {
			number = *in.Number
		} else {
			n, err := r.Episodes().NextNumber(ctx, seriesID)
			if err != nil {
				return err
			}
			number = n
		}
		e, err := domain.NewEpisode(domain.EpisodeID(domain.NewID()), seriesID, domain.EpisodeAttributes{
			Number:      number,
			Title:       in.Title,
			Description: in.Description,
		}, now)
		if err != nil {
			return err
		}
		if err := r.Episodes().Create(ctx, e); err != nil {
			return err
		}
		out = e
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateEpisode replaces the episode attributes when expectedVersion matches
// the stored version.
func (a *App) UpdateEpisode(ctx context.Context, actor *domain.Actor, id domain.EpisodeID, attrs domain.EpisodeAttributes, expectedVersion int) (*domain.Episode, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	now := a.now()
	var out *domain.Episode
	err := a.uow.Do(ctx, func(ctx context.Context, r uow.Repositories) error {
		e, err := r.Episodes().Get(ctx, id)
		if err != nil {
			return err
		}
		if e.Version() != expectedVersion {
			return domain.ErrVersionConflict
		}
		if err := e.Update(attrs, now); err != nil {
			return err
		}
		if err := r.Episodes().Update(ctx, e); err != nil {
			return err
		}
		out = e
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteEpisode deletes an episode and its audio blob after the commit.
func (a *App) DeleteEpisode(ctx context.Context, actor *domain.Actor, id domain.EpisodeID) error {
	if err := requireActor(actor); err != nil {
		return err
	}
	var audioKey string
	err := a.uow.Do(ctx, func(ctx context.Context, r uow.Repositories) error {
		e, err := r.Episodes().Get(ctx, id)
		if err != nil {
			return err
		}
		if audio := e.Audio(); audio != nil {
			audioKey = audio.Key
		}
		return r.Episodes().Delete(ctx, id)
	})
	if err != nil {
		return err
	}
	a.deleteBlob(ctx, "catalog: delete episode audio", audioKey)
	return nil
}

// UploadAudio stores a new audio blob and swaps it into the episode. The blob
// is written before the transaction; the previous one is deleted after the
// commit.
func (a *App) UploadAudio(ctx context.Context, actor *domain.Actor, id domain.EpisodeID, in episodeuc.AudioUploadInput) (*domain.Episode, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	contentType := domain.NormalizeContentType(in.ContentType)
	if err := domain.ValidateAudioContentType(contentType); err != nil {
		return nil, err
	}
	if math.IsNaN(in.DurationSeconds) || math.IsInf(in.DurationSeconds, 0) || in.DurationSeconds < 0 {
		return nil, domain.ErrInvalidEpisode
	}

	key := fmt.Sprintf("episodes/%s/audio-%s%s", id, domain.NewID(), domain.AudioExtension(contentType))
	size, err := a.blobs.Put(ctx, key, in.Reader, contentType)
	if err != nil {
		return nil, err
	}

	now := a.now()
	var (
		out    *domain.Episode
		oldKey string
	)
	err = a.uow.Do(ctx, func(ctx context.Context, r uow.Repositories) error {
		e, err := r.Episodes().Get(ctx, id)
		if err != nil {
			return err
		}
		if audio := e.Audio(); audio != nil {
			oldKey = audio.Key
		}
		if err := e.AttachAudio(domain.AudioFile{
			Asset:           domain.Asset{Key: key, ContentType: contentType, SizeBytes: size},
			DurationSeconds: in.DurationSeconds,
			OriginalName:    in.OriginalName,
		}, now); err != nil {
			return err
		}
		if err := r.Episodes().Update(ctx, e); err != nil {
			return err
		}
		out = e
		return nil
	})
	if err != nil {
		a.deleteBlob(ctx, "catalog: discard unreferenced audio", key)
		return nil, err
	}
	if oldKey != key {
		a.deleteBlob(ctx, "catalog: delete replaced audio", oldKey)
	}
	return out, nil
}

// SetChapters replaces the episode chapters.
func (a *App) SetChapters(ctx context.Context, actor *domain.Actor, id domain.EpisodeID, chapters []domain.Chapter) (*domain.Episode, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	now := a.now()
	var out *domain.Episode
	err := a.uow.Do(ctx, func(ctx context.Context, r uow.Repositories) error {
		e, err := r.Episodes().Get(ctx, id)
		if err != nil {
			return err
		}
		if err := e.SetChapters(chapters, now); err != nil {
			return err
		}
		if err := r.Episodes().Update(ctx, e); err != nil {
			return err
		}
		out = e
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Publish publishes an episode that has audio and chapters.
func (a *App) Publish(ctx context.Context, actor *domain.Actor, id domain.EpisodeID) (*domain.Episode, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	now := a.now()
	var out *domain.Episode
	err := a.uow.Do(ctx, func(ctx context.Context, r uow.Repositories) error {
		e, err := r.Episodes().Get(ctx, id)
		if err != nil {
			return err
		}
		if err := e.Publish(now); err != nil {
			return err
		}
		if err := r.Episodes().Update(ctx, e); err != nil {
			return err
		}
		out = e
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Unpublish moves an episode back to draft.
func (a *App) Unpublish(ctx context.Context, actor *domain.Actor, id domain.EpisodeID) (*domain.Episode, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	now := a.now()
	var out *domain.Episode
	err := a.uow.Do(ctx, func(ctx context.Context, r uow.Repositories) error {
		e, err := r.Episodes().Get(ctx, id)
		if err != nil {
			return err
		}
		e.Unpublish(now)
		if err := r.Episodes().Update(ctx, e); err != nil {
			return err
		}
		out = e
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetEpisode returns one episode.
func (a *App) GetEpisode(ctx context.Context, actor *domain.Actor, id domain.EpisodeID) (*domain.Episode, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	return a.episodes.Get(ctx, id)
}

// ListEpisodes returns the episodes of a series, ordered by number.
func (a *App) ListEpisodes(ctx context.Context, actor *domain.Actor, seriesID domain.SeriesID, q episodeuc.ListQuery) ([]*domain.Episode, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	return a.episodes.ListBySeries(ctx, seriesID, episoderepo.Filter{Status: q.Status})
}

// OpenAudio opens the episode audio blob.
func (a *App) OpenAudio(ctx context.Context, actor *domain.Actor, id domain.EpisodeID) (blobstore.Blob, error) {
	if err := requireActor(actor); err != nil {
		return blobstore.Blob{}, err
	}
	e, err := a.episodes.Get(ctx, id)
	if err != nil {
		return blobstore.Blob{}, err
	}
	audio := e.Audio()
	if audio == nil {
		return blobstore.Blob{}, domain.ErrAssetNotFound
	}
	return a.blobs.Open(ctx, audio.Key)
}
