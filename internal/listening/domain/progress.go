package domain

import "time"

// completionWindowSeconds is how close to the end of an episode counts as
// finished: reaching duration-2s marks the episode completed.
const completionWindowSeconds = 2.0

// Progress is the listening aggregate: how far one listener got into one
// catalog episode. The listener id is part of the identity, so several
// listeners can share the same table later.
type Progress struct {
	listenerID      ListenerID
	episodeRef      EpisodeRef
	positionSeconds float64
	completed       bool
	createdAt       time.Time
	updatedAt       time.Time
}

// NewProgress starts tracking an episode at position zero.
func NewProgress(listener ListenerID, episode EpisodeRef, now time.Time) *Progress {
	return &Progress{listenerID: listener, episodeRef: episode, createdAt: now, updatedAt: now}
}

// ReconstituteProgress rebuilds a progress from trusted storage without
// validation.
func ReconstituteProgress(listener ListenerID, episode EpisodeRef, positionSeconds float64, completed bool, createdAt, updatedAt time.Time) *Progress {
	return &Progress{
		listenerID:      listener,
		episodeRef:      episode,
		positionSeconds: positionSeconds,
		completed:       completed,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
	}
}

// ListenerID returns the listener the progress belongs to.
func (p *Progress) ListenerID() ListenerID { return p.listenerID }

// EpisodeRef returns the catalog episode tracked by the progress.
func (p *Progress) EpisodeRef() EpisodeRef { return p.episodeRef }

// PositionSeconds returns the last recorded playback position.
func (p *Progress) PositionSeconds() float64 { return p.positionSeconds }

// Completed reports whether the episode counts as played.
func (p *Progress) Completed() bool { return p.completed }

// CreatedAt returns the creation time.
func (p *Progress) CreatedAt() time.Time { return p.createdAt }

// UpdatedAt returns the last modification time.
func (p *Progress) UpdatedAt() time.Time { return p.updatedAt }

// Record stores a playback position. A negative position is rejected; with a
// known duration the position is clamped to it, and reaching the last
// completionWindowSeconds marks the episode completed. Recording an earlier
// position on a completed episode re-opens it.
func (p *Progress) Record(position, durationSeconds float64, now time.Time) error {
	if position < 0 {
		return ErrInvalidPosition
	}
	if durationSeconds > 0 && position > durationSeconds {
		position = durationSeconds
	}
	p.positionSeconds = position
	p.updatedAt = now
	if durationSeconds <= 0 {
		return nil
	}
	p.completed = position >= durationSeconds-completionWindowSeconds
	return nil
}

// MarkCompleted marks the episode played and places the position at its end.
func (p *Progress) MarkCompleted(durationSeconds float64, now time.Time) {
	if durationSeconds < 0 {
		durationSeconds = 0
	}
	p.positionSeconds = durationSeconds
	p.completed = true
	p.updatedAt = now
}

// Reset returns the episode to "not started".
func (p *Progress) Reset(now time.Time) {
	p.positionSeconds = 0
	p.completed = false
	p.updatedAt = now
}
