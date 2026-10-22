package app

import (
	"context"
	"fmt"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/blobstore"
	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/episoderepo"
	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/seriesrepo"
	"github.com/JLugagne/podcast/internal/catalog/domain/uow"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/seriesuc"
)

// CreateSeries creates a series from attrs.
func (a *App) CreateSeries(ctx context.Context, actor *domain.Actor, attrs domain.SeriesAttributes) (*seriesuc.SeriesView, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	s, err := domain.NewSeries(domain.SeriesID(domain.NewID()), attrs, a.now())
	if err != nil {
		return nil, err
	}
	if err := a.uow.Do(ctx, func(ctx context.Context, r uow.Repositories) error {
		return r.Series().Create(ctx, s)
	}); err != nil {
		return nil, err
	}
	return &seriesuc.SeriesView{Series: s}, nil
}

// UpdateSeries replaces the series attributes when expectedVersion matches the
// stored version.
func (a *App) UpdateSeries(ctx context.Context, actor *domain.Actor, id domain.SeriesID, attrs domain.SeriesAttributes, expectedVersion int) (*seriesuc.SeriesView, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	now := a.now()
	var out *seriesuc.SeriesView
	err := a.uow.Do(ctx, func(ctx context.Context, r uow.Repositories) error {
		s, err := r.Series().Get(ctx, id)
		if err != nil {
			return err
		}
		if s.Version() != expectedVersion {
			return domain.ErrVersionConflict
		}
		if err := s.Update(attrs, now); err != nil {
			return err
		}
		if err := r.Series().Update(ctx, s); err != nil {
			return err
		}
		out, err = a.seriesView(ctx, r.Episodes(), s)
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteSeries deletes a series and its cover blob. It fails with
// domain.ErrSeriesNotEmpty while episodes still exist.
func (a *App) DeleteSeries(ctx context.Context, actor *domain.Actor, id domain.SeriesID) error {
	if err := requireActor(actor); err != nil {
		return err
	}
	var coverKey string
	err := a.uow.Do(ctx, func(ctx context.Context, r uow.Repositories) error {
		s, err := r.Series().Get(ctx, id)
		if err != nil {
			return err
		}
		count, err := r.Episodes().CountBySeries(ctx, id)
		if err != nil {
			return err
		}
		if count > 0 {
			return domain.ErrSeriesNotEmpty
		}
		if err := r.Series().Delete(ctx, id); err != nil {
			return err
		}
		if c := s.Cover(); c != nil {
			coverKey = c.Key
		}
		return nil
	})
	if err != nil {
		return err
	}
	a.deleteBlob(ctx, "catalog: delete series cover", coverKey)
	return nil
}

// UploadCover stores a new cover blob and swaps it into the series. The blob is
// written before the transaction so a storage failure leaves the series
// untouched; the previous blob is deleted after the commit.
func (a *App) UploadCover(ctx context.Context, actor *domain.Actor, id domain.SeriesID, in seriesuc.UploadInput) (*seriesuc.SeriesView, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	contentType := domain.NormalizeContentType(in.ContentType)
	if err := domain.ValidateImageContentType(contentType); err != nil {
		return nil, err
	}

	key := fmt.Sprintf("series/%s/cover-%s%s", id, domain.NewID(), domain.ImageExtension(contentType))
	size, err := a.blobs.Put(ctx, key, in.Reader, contentType)
	if err != nil {
		return nil, err
	}

	now := a.now()
	var (
		out    *seriesuc.SeriesView
		oldKey string
	)
	err = a.uow.Do(ctx, func(ctx context.Context, r uow.Repositories) error {
		s, err := r.Series().Get(ctx, id)
		if err != nil {
			return err
		}
		if c := s.Cover(); c != nil {
			oldKey = c.Key
		}
		s.SetCover(domain.Asset{Key: key, ContentType: contentType, SizeBytes: size}, now)
		if err := r.Series().Update(ctx, s); err != nil {
			return err
		}
		out, err = a.seriesView(ctx, r.Episodes(), s)
		return err
	})
	if err != nil {
		a.deleteBlob(ctx, "catalog: discard unreferenced cover", key)
		return nil, err
	}
	if oldKey != key {
		a.deleteBlob(ctx, "catalog: delete replaced cover", oldKey)
	}
	return out, nil
}

// RemoveCover detaches the cover and deletes its blob after the commit.
func (a *App) RemoveCover(ctx context.Context, actor *domain.Actor, id domain.SeriesID) (*seriesuc.SeriesView, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	now := a.now()
	var (
		out    *seriesuc.SeriesView
		oldKey string
	)
	err := a.uow.Do(ctx, func(ctx context.Context, r uow.Repositories) error {
		s, err := r.Series().Get(ctx, id)
		if err != nil {
			return err
		}
		if c := s.Cover(); c != nil {
			oldKey = c.Key
		}
		s.ClearCover(now)
		if err := r.Series().Update(ctx, s); err != nil {
			return err
		}
		out, err = a.seriesView(ctx, r.Episodes(), s)
		return err
	})
	if err != nil {
		return nil, err
	}
	a.deleteBlob(ctx, "catalog: delete removed cover", oldKey)
	return out, nil
}

// GetSeries returns one series with its derived counters.
func (a *App) GetSeries(ctx context.Context, actor *domain.Actor, id domain.SeriesID) (*seriesuc.SeriesView, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	s, err := a.series.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return a.seriesView(ctx, a.episodes, s)
}

// ListSeries returns the series matching the query with their derived counters.
func (a *App) ListSeries(ctx context.Context, actor *domain.Actor, q seriesuc.ListQuery) ([]*seriesuc.SeriesView, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	list, err := a.series.List(ctx, seriesrepo.Filter{Query: q.Query, Limit: q.Limit, Offset: q.Offset})
	if err != nil {
		return nil, err
	}
	views := make([]*seriesuc.SeriesView, 0, len(list))
	for _, s := range list {
		v, err := a.seriesView(ctx, a.episodes, s)
		if err != nil {
			return nil, err
		}
		views = append(views, v)
	}
	return views, nil
}

// OpenCover opens the series cover blob.
func (a *App) OpenCover(ctx context.Context, actor *domain.Actor, id domain.SeriesID) (blobstore.Blob, error) {
	if err := requireActor(actor); err != nil {
		return blobstore.Blob{}, err
	}
	s, err := a.series.Get(ctx, id)
	if err != nil {
		return blobstore.Blob{}, err
	}
	cover := s.Cover()
	if cover == nil {
		return blobstore.Blob{}, domain.ErrAssetNotFound
	}
	return a.blobs.Open(ctx, cover.Key)
}

func (a *App) seriesView(ctx context.Context, episodes episoderepo.EpisodeRepository, s *domain.Series) (*seriesuc.SeriesView, error) {
	list, err := episodes.ListBySeries(ctx, s.ID(), episoderepo.Filter{})
	if err != nil {
		return nil, err
	}
	v := &seriesuc.SeriesView{Series: s, EpisodeCount: len(list)}
	for _, e := range list {
		if e.Status() == domain.EpisodeStatusPublished {
			v.PublishedCount++
		}
		if audio := e.Audio(); audio != nil {
			v.TotalDurationSeconds += audio.DurationSeconds
		}
	}
	return v, nil
}

// GetSeriesBySlug returns one series with its derived counters, looked up by
// its slug.
func (a *App) GetSeriesBySlug(ctx context.Context, actor *domain.Actor, slug string) (*seriesuc.SeriesView, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	s, err := a.series.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return a.seriesView(ctx, a.episodes, s)
}
