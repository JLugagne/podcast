package pgadapter

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/JLugagne/podcast/internal/listening/domain"
	"github.com/JLugagne/podcast/internal/listening/domain/repositories/progressrepo"
)

// progressRepository persists listening progress on PostgreSQL.
type progressRepository struct{ db DBTX }

// NewProgressRepository returns the progress repository backed by db.
func NewProgressRepository(db DBTX) progressrepo.ProgressRepository {
	return &progressRepository{db: db}
}

const progressColumns = `listener_id, episode_id, position_seconds, completed, created_at, updated_at`

func scanProgress(row pgx.Row) (*domain.Progress, error) {
	var (
		listener, episode    string
		position             float64
		completed            bool
		createdAt, updatedAt time.Time
	)
	if err := row.Scan(&listener, &episode, &position, &completed, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	return domain.ReconstituteProgress(
		domain.ListenerID(listener),
		domain.EpisodeRef(episode),
		position,
		completed,
		createdAt,
		updatedAt,
	), nil
}

// Get returns the progress of one listener on one episode.
func (r *progressRepository) Get(ctx context.Context, listener domain.ListenerID, episode domain.EpisodeRef) (*domain.Progress, error) {
	progress, err := scanProgress(r.db.QueryRow(ctx, `SELECT `+progressColumns+`
		FROM listening_progress WHERE listener_id = $1 AND episode_id = $2`, string(listener), string(episode)))
	return progress, mapNotFound(err, domain.ErrProgressNotFound, "get progress")
}

// Save upserts the progress, keeping the original creation time.
func (r *progressRepository) Save(ctx context.Context, progress *domain.Progress) error {
	_, err := r.db.Exec(ctx, `INSERT INTO listening_progress
		(listener_id, episode_id, position_seconds, completed, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (listener_id, episode_id) DO UPDATE
		SET position_seconds = EXCLUDED.position_seconds,
			completed = EXCLUDED.completed,
			updated_at = EXCLUDED.updated_at`,
		string(progress.ListenerID()), string(progress.EpisodeRef()), progress.PositionSeconds(),
		progress.Completed(), progress.CreatedAt(), progress.UpdatedAt())
	if err != nil {
		return wrap("save progress", err)
	}
	return nil
}

// Delete removes the progress. Deleting a row that does not exist is not an
// error.
func (r *progressRepository) Delete(ctx context.Context, listener domain.ListenerID, episode domain.EpisodeRef) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM listening_progress WHERE listener_id = $1 AND episode_id = $2`,
		string(listener), string(episode)); err != nil {
		return wrap("delete progress", err)
	}
	return nil
}

// List returns one listener's progress, most recently updated first. A
// non-positive limit means every matching row.
func (r *progressRepository) List(ctx context.Context, listener domain.ListenerID, filter progressrepo.Filter) ([]*domain.Progress, error) {
	query := `SELECT ` + progressColumns + ` FROM listening_progress WHERE listener_id = $1`
	if filter.InProgressOnly {
		query += ` AND position_seconds > 0 AND NOT completed`
	}
	query += ` ORDER BY updated_at DESC, episode_id DESC LIMIT NULLIF($2, 0)`

	rows, err := r.db.Query(ctx, query, string(listener), filter.Limit)
	if err != nil {
		return nil, wrap("list progress", err)
	}
	defer rows.Close()

	out := make([]*domain.Progress, 0)
	for rows.Next() {
		progress, err := scanProgress(rows)
		if err != nil {
			return nil, wrap("scan progress", err)
		}
		out = append(out, progress)
	}
	if err := rows.Err(); err != nil {
		return nil, wrap("list progress", err)
	}
	return out, nil
}
