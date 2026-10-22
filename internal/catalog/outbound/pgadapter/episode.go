package pgadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/episoderepo"
)

// episodeRepository persists episode aggregates on PostgreSQL.
type episodeRepository struct{ db DBTX }

// NewEpisodeRepository returns the episode repository backed by db.
func NewEpisodeRepository(db DBTX) episoderepo.EpisodeRepository { return &episodeRepository{db: db} }

const episodeColumns = `e.id, e.series_id, e.number, e.title, e.description, e.status,
	e.audio_key, e.audio_content_type, e.audio_size_bytes, e.audio_duration_seconds, e.audio_original_name,
	e.chapters, e.published_at, e.created_at, e.updated_at, e.version`

// chapterDTO is the private JSON shape stored in the chapters jsonb column. It
// is deliberately not the public Podcasting 2.0 chapter document.
type chapterDTO struct {
	StartSeconds float64  `json:"start_seconds"`
	Title        string   `json:"title"`
	URL          string   `json:"url,omitempty"`
	ImageURL     string   `json:"image_url,omitempty"`
	EndSeconds   *float64 `json:"end_seconds,omitempty"`
	TOC          bool     `json:"toc"`
}

func encodeChapters(chapters []domain.Chapter) ([]byte, error) {
	dtos := make([]chapterDTO, 0, len(chapters))
	for _, c := range chapters {
		dtos = append(dtos, chapterDTO{
			StartSeconds: c.StartSeconds,
			Title:        c.Title,
			URL:          c.URL,
			ImageURL:     c.ImageURL,
			EndSeconds:   c.EndSeconds,
			TOC:          c.TOC,
		})
	}
	raw, err := json.Marshal(dtos)
	if err != nil {
		return nil, fmt.Errorf("pgadapter: encode chapters: %w", err)
	}
	return raw, nil
}

func decodeChapters(raw []byte) ([]domain.Chapter, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var dtos []chapterDTO
	if err := json.Unmarshal(raw, &dtos); err != nil {
		return nil, fmt.Errorf("pgadapter: decode chapters: %w", err)
	}
	if len(dtos) == 0 {
		return nil, nil
	}
	out := make([]domain.Chapter, len(dtos))
	for i, d := range dtos {
		out[i] = domain.Chapter{
			StartSeconds: d.StartSeconds,
			Title:        d.Title,
			URL:          d.URL,
			ImageURL:     d.ImageURL,
			EndSeconds:   d.EndSeconds,
			TOC:          d.TOC,
		}
	}
	return out, nil
}

func scanEpisode(row pgx.Row) (*domain.Episode, error) {
	var (
		id, seriesID, title, description, status      string
		number                                        int
		audioKey, audioContentType, audioOriginalName *string
		audioSizeBytes                                *int64
		audioDuration                                 *float64
		chaptersRaw                                   []byte
		publishedAt                                   *time.Time
		createdAt, updatedAt                          time.Time
		version                                       int
	)
	if err := row.Scan(&id, &seriesID, &number, &title, &description, &status,
		&audioKey, &audioContentType, &audioSizeBytes, &audioDuration, &audioOriginalName,
		&chaptersRaw, &publishedAt, &createdAt, &updatedAt, &version); err != nil {
		return nil, err
	}
	episodeStatus, err := domain.ParseEpisodeStatus(status)
	if err != nil {
		return nil, err
	}
	chapters, err := decodeChapters(chaptersRaw)
	if err != nil {
		return nil, err
	}
	var audio *domain.AudioFile
	if audioKey != nil {
		audio = &domain.AudioFile{
			Asset: domain.Asset{
				Key:         *audioKey,
				ContentType: derefString(audioContentType),
				SizeBytes:   derefInt64(audioSizeBytes),
			},
			DurationSeconds: derefFloat64(audioDuration),
			OriginalName:    derefString(audioOriginalName),
		}
	}
	return domain.ReconstituteEpisode(
		domain.EpisodeID(id),
		domain.SeriesID(seriesID),
		domain.EpisodeAttributes{Number: number, Title: title, Description: description},
		episodeStatus, audio, chapters, publishedAt, createdAt, updatedAt, version,
	), nil
}

// Create inserts a new episode. The stored version is the aggregate's current
// version, which is 0 for entities returned by domain.NewEpisode.
func (r *episodeRepository) Create(ctx context.Context, e *domain.Episode) error {
	chapters, err := encodeChapters(e.Chapters())
	if err != nil {
		return err
	}
	a := e.Attributes()
	audioKey, audioContentType, audioSizeBytes, audioDuration, audioOriginalName := episodeAudioParams(e.Audio())
	_, err = r.db.Exec(ctx, `INSERT INTO catalog_episodes
		(id, series_id, number, title, description, status, audio_key, audio_content_type, audio_size_bytes,
		 audio_duration_seconds, audio_original_name, chapters, published_at, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		string(e.ID()), string(e.SeriesID()), a.Number, a.Title, a.Description, string(e.Status()),
		audioKey, audioContentType, audioSizeBytes, audioDuration, audioOriginalName,
		chapters, e.PublishedAt(), e.CreatedAt(), e.UpdatedAt(), e.Version())
	if err != nil {
		return mapEpisodeWriteError(err)
	}
	return nil
}

// Update applies optimistic locking on version and increments the stored
// version, writing the new value back on the aggregate.
func (r *episodeRepository) Update(ctx context.Context, e *domain.Episode) error {
	chapters, err := encodeChapters(e.Chapters())
	if err != nil {
		return err
	}
	a := e.Attributes()
	audioKey, audioContentType, audioSizeBytes, audioDuration, audioOriginalName := episodeAudioParams(e.Audio())
	current := e.Version()
	tag, err := r.db.Exec(ctx, `UPDATE catalog_episodes
		SET number = $1, title = $2, description = $3, status = $4,
			audio_key = $5, audio_content_type = $6, audio_size_bytes = $7,
			audio_duration_seconds = $8, audio_original_name = $9,
			chapters = $10, published_at = $11, updated_at = $12, version = version + 1
		WHERE id = $13 AND version = $14`,
		a.Number, a.Title, a.Description, string(e.Status()),
		audioKey, audioContentType, audioSizeBytes, audioDuration, audioOriginalName,
		chapters, e.PublishedAt(), e.UpdatedAt(), string(e.ID()), current)
	if err != nil {
		return mapEpisodeWriteError(err)
	}
	if tag.RowsAffected() == 0 {
		exists, err := r.exists(ctx, e.ID())
		if err != nil {
			return err
		}
		if !exists {
			return domain.ErrEpisodeNotFound
		}
		return domain.ErrVersionConflict
	}
	e.SetVersion(current + 1)
	return nil
}

// Get returns the episode with the given id.
func (r *episodeRepository) Get(ctx context.Context, id domain.EpisodeID) (*domain.Episode, error) {
	e, err := scanEpisode(r.db.QueryRow(ctx, `SELECT `+episodeColumns+` FROM catalog_episodes e WHERE e.id = $1`, string(id)))
	return e, mapNotFound(err, domain.ErrEpisodeNotFound, "get episode")
}

// Delete removes the episode.
func (r *episodeRepository) Delete(ctx context.Context, id domain.EpisodeID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM catalog_episodes WHERE id = $1`, string(id))
	if err != nil {
		return wrap("delete episode", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrEpisodeNotFound
	}
	return nil
}

// ListBySeries returns the episodes of a series ordered by number, optionally
// filtered by status.
func (r *episodeRepository) ListBySeries(ctx context.Context, seriesID domain.SeriesID, f episoderepo.Filter) ([]*domain.Episode, error) {
	status := ""
	if f.Status != nil {
		status = string(*f.Status)
	}
	rows, err := r.db.Query(ctx, `SELECT `+episodeColumns+` FROM catalog_episodes e
		WHERE e.series_id = $1 AND ($2 = '' OR e.status = $2)
		ORDER BY e.number ASC`, string(seriesID), status)
	if err != nil {
		return nil, wrap("list episodes", err)
	}
	defer rows.Close()

	var out []*domain.Episode
	for rows.Next() {
		e, err := scanEpisode(rows)
		if err != nil {
			return nil, wrap("scan episode", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, wrap("list episodes", err)
	}
	return out, nil
}

// CountBySeries returns the number of episodes of a series.
func (r *episodeRepository) CountBySeries(ctx context.Context, seriesID domain.SeriesID) (int, error) {
	var count int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM catalog_episodes WHERE series_id = $1`, string(seriesID)).Scan(&count); err != nil {
		return 0, wrap("count episodes", err)
	}
	return count, nil
}

// NextNumber returns the next free episode number of a series.
func (r *episodeRepository) NextNumber(ctx context.Context, seriesID domain.SeriesID) (int, error) {
	var number int
	if err := r.db.QueryRow(ctx, `SELECT COALESCE(MAX(number), 0) + 1 FROM catalog_episodes WHERE series_id = $1`, string(seriesID)).Scan(&number); err != nil {
		return 0, wrap("next episode number", err)
	}
	return number, nil
}

func (r *episodeRepository) exists(ctx context.Context, id domain.EpisodeID) (bool, error) {
	var one int
	err := r.db.QueryRow(ctx, `SELECT 1 FROM catalog_episodes WHERE id = $1`, string(id)).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, wrap("check episode existence", err)
	}
	return true, nil
}

func episodeAudioParams(audio *domain.AudioFile) (*string, *string, *int64, *float64, *string) {
	if audio == nil {
		return nil, nil, nil, nil, nil
	}
	return &audio.Key, &audio.ContentType, &audio.SizeBytes, &audio.DurationSeconds, &audio.OriginalName
}

func mapEpisodeWriteError(err error) error {
	code, constraint := pgError(err)
	switch {
	case code == uniqueViolation && constraint == episodeNumberConstraint:
		return domain.ErrEpisodeNumberTaken
	case code == foreignKeyViolation && constraint == episodeSeriesFKConstraint:
		return domain.ErrSeriesNotFound
	}
	return wrap("write episode", err)
}
