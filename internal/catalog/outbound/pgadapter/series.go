package pgadapter

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/seriesrepo"
)

// seriesRepository persists series aggregates on PostgreSQL.
type seriesRepository struct{ db DBTX }

// NewSeriesRepository returns the series repository backed by db.
func NewSeriesRepository(db DBTX) seriesrepo.SeriesRepository { return &seriesRepository{db: db} }

const seriesColumns = `s.id, s.slug, s.title, s.author, s.description, s.accent_color,
	s.cover_key, s.cover_content_type, s.cover_size_bytes, s.created_at, s.updated_at, s.version`

func scanSeries(row pgx.Row) (*domain.Series, error) {
	var (
		id, slug, title, author, description, accentColor string
		coverKey, coverContentType                        *string
		coverSizeBytes                                    *int64
		createdAt, updatedAt                              time.Time
		version                                           int
	)
	if err := row.Scan(&id, &slug, &title, &author, &description, &accentColor,
		&coverKey, &coverContentType, &coverSizeBytes, &createdAt, &updatedAt, &version); err != nil {
		return nil, err
	}
	var cover *domain.Asset
	if coverKey != nil {
		cover = &domain.Asset{
			Key:         *coverKey,
			ContentType: derefString(coverContentType),
			SizeBytes:   derefInt64(coverSizeBytes),
		}
	}
	return domain.ReconstituteSeries(domain.SeriesID(id), domain.SeriesAttributes{
		Title:       title,
		Author:      author,
		Description: description,
		AccentColor: accentColor,
		Slug:        slug,
	}, cover, createdAt, updatedAt, version), nil
}

// Create inserts a new series. The stored version is the aggregate's current
// version, which is 0 for entities returned by domain.NewSeries.
func (r *seriesRepository) Create(ctx context.Context, s *domain.Series) error {
	a := s.Attributes()
	coverKey, coverContentType, coverSizeBytes := coverParams(s.Cover())
	_, err := r.db.Exec(ctx, `INSERT INTO catalog_series
		(id, slug, title, author, description, accent_color, cover_key, cover_content_type, cover_size_bytes, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		string(s.ID()), a.Slug, a.Title, a.Author, a.Description, a.AccentColor,
		coverKey, coverContentType, coverSizeBytes, s.CreatedAt(), s.UpdatedAt(), s.Version())
	if err != nil {
		return mapSeriesWriteError(err)
	}
	return nil
}

// Update applies optimistic locking on version and increments the stored
// version, writing the new value back on the aggregate.
func (r *seriesRepository) Update(ctx context.Context, s *domain.Series) error {
	a := s.Attributes()
	coverKey, coverContentType, coverSizeBytes := coverParams(s.Cover())
	current := s.Version()
	tag, err := r.db.Exec(ctx, `UPDATE catalog_series
		SET slug = $1, title = $2, author = $3, description = $4, accent_color = $5,
			cover_key = $6, cover_content_type = $7, cover_size_bytes = $8,
			updated_at = $9, version = version + 1
		WHERE id = $10 AND version = $11`,
		a.Slug, a.Title, a.Author, a.Description, a.AccentColor,
		coverKey, coverContentType, coverSizeBytes, s.UpdatedAt(), string(s.ID()), current)
	if err != nil {
		return mapSeriesWriteError(err)
	}
	if tag.RowsAffected() == 0 {
		exists, err := r.exists(ctx, s.ID())
		if err != nil {
			return err
		}
		if !exists {
			return domain.ErrSeriesNotFound
		}
		return domain.ErrVersionConflict
	}
	s.SetVersion(current + 1)
	return nil
}

// Get returns the series with the given id.
func (r *seriesRepository) Get(ctx context.Context, id domain.SeriesID) (*domain.Series, error) {
	s, err := scanSeries(r.db.QueryRow(ctx, `SELECT `+seriesColumns+` FROM catalog_series s WHERE s.id = $1`, string(id)))
	return s, mapNotFound(err, domain.ErrSeriesNotFound, "get series")
}

// GetBySlug returns the series owning slug.
func (r *seriesRepository) GetBySlug(ctx context.Context, slug string) (*domain.Series, error) {
	s, err := scanSeries(r.db.QueryRow(ctx, `SELECT `+seriesColumns+` FROM catalog_series s WHERE s.slug = $1`, slug))
	return s, mapNotFound(err, domain.ErrSeriesNotFound, "get series by slug")
}

// List returns the series matching the filter, most recently created first.
func (r *seriesRepository) List(ctx context.Context, f seriesrepo.Filter) ([]*domain.Series, error) {
	pattern := "%" + escapeLike(strings.TrimSpace(f.Query)) + "%"
	rows, err := r.db.Query(ctx, `SELECT `+seriesColumns+` FROM catalog_series s
		WHERE s.slug ILIKE $1 ESCAPE '\' OR s.title ILIKE $1 ESCAPE '\' OR s.author ILIKE $1 ESCAPE '\'
		ORDER BY s.created_at DESC, s.id DESC
		LIMIT NULLIF($2, 0) OFFSET $3`, pattern, f.Limit, f.Offset)
	if err != nil {
		return nil, wrap("list series", err)
	}
	defer rows.Close()

	var out []*domain.Series
	for rows.Next() {
		s, err := scanSeries(rows)
		if err != nil {
			return nil, wrap("scan series", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, wrap("list series", err)
	}
	return out, nil
}

// Delete removes the series, or fails with domain.ErrSeriesNotEmpty while
// episodes still reference it.
func (r *seriesRepository) Delete(ctx context.Context, id domain.SeriesID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM catalog_series WHERE id = $1`, string(id))
	if err != nil {
		if code, _ := pgError(err); code == foreignKeyViolation || code == restrictViolation {
			return domain.ErrSeriesNotEmpty
		}
		return wrap("delete series", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrSeriesNotFound
	}
	return nil
}

func (r *seriesRepository) exists(ctx context.Context, id domain.SeriesID) (bool, error) {
	var one int
	err := r.db.QueryRow(ctx, `SELECT 1 FROM catalog_series WHERE id = $1`, string(id)).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, wrap("check series existence", err)
	}
	return true, nil
}

func coverParams(cover *domain.Asset) (*string, *string, *int64) {
	if cover == nil {
		return nil, nil, nil
	}
	return &cover.Key, &cover.ContentType, &cover.SizeBytes
}

func mapSeriesWriteError(err error) error {
	code, constraint := pgError(err)
	if code == uniqueViolation && constraint == seriesSlugConstraint {
		return domain.ErrSlugTaken
	}
	return wrap("write series", err)
}

// mapNotFound translates a pgx.ErrNoRows scan into notFound, wraps any other
// error with op, and returns nil when err is nil.
func mapNotFound(err error, notFound error, op string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound
	}
	if err != nil {
		return wrap(op, err)
	}
	return nil
}

// escapeLike neutralizes the wildcards of a user-supplied ILIKE pattern.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
