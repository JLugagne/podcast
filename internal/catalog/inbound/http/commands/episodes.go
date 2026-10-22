package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/episodeuc"
	"github.com/JLugagne/podcast/internal/catalog/inbound/http/converters"
	"github.com/JLugagne/podcast/internal/catalog/inbound/http/shared"
	"github.com/JLugagne/podcast/internal/pkg/httpjson"
	v1 "github.com/JLugagne/podcast/pkg/catalog/http/v1"
)

// createEpisode handles POST /series/{id}/episodes.
func (s svc) createEpisode(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	seriesID, err := shared.PathSeriesID(r)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	req, ok := decode[v1.EpisodeInput](w, r)
	if !ok {
		return
	}
	episode, err := s.app.CreateEpisode(r.Context(), actor, seriesID, episodeuc.CreateInput{
		Number:      req.Number,
		Title:       req.Title,
		Description: req.Description,
	})
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	_ = httpjson.Respond(w, http.StatusCreated, converters.ToEpisode(episode))
}

// updateEpisode handles PUT /episodes/{id}.
func (s svc) updateEpisode(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	id, err := shared.PathEpisodeID(r)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	req, ok := decode[v1.EpisodeInput](w, r)
	if !ok {
		return
	}
	version, err := requiredVersion(req.Version)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	episode, err := s.app.UpdateEpisode(r.Context(), actor, id, converters.ToEpisodeAttributes(req), version)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	_ = httpjson.Respond(w, http.StatusOK, converters.ToEpisode(episode))
}

// deleteEpisode handles DELETE /episodes/{id}.
func (s svc) deleteEpisode(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	id, err := shared.PathEpisodeID(r)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	if err := s.app.DeleteEpisode(r.Context(), actor, id); err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	httpjson.NoContent(w)
}

// uploadAudio handles PUT /episodes/{id}/audio.
//
// The body is a multipart document streamed straight to the use case. The part
// named "file" carries the audio; the optional "duration_seconds" field MUST be
// sent before the file part, because later parts are never read.
func (s svc) uploadAudio(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	id, err := shared.PathEpisodeID(r)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	file, duration, err := readUpload(w, r, s.opts.MaxAudioBytes, true)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	defer file.close()

	episode, err := s.app.UploadAudio(r.Context(), actor, id, episodeuc.AudioUploadInput{
		Reader:          file.Reader,
		ContentType:     file.ContentType,
		OriginalName:    file.FileName,
		DurationSeconds: duration,
	})
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	_ = httpjson.Respond(w, http.StatusOK, converters.ToEpisode(episode))
}

// setChapters handles PUT /episodes/{id}/chapters. The body is a Podcasting 2.0
// JSON chapters document; unknown fields are ignored.
func (s svc) setChapters(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	id, err := shared.PathEpisodeID(r)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	var doc v1.ChaptersDocument
	if err := decodeLenient(w, r, &doc); err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	chapters, err := converters.ToChapters(doc)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	episode, err := s.app.SetChapters(r.Context(), actor, id, chapters)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	_ = httpjson.Respond(w, http.StatusOK, converters.ToEpisode(episode))
}

// publish handles POST /episodes/{id}/publish.
func (s svc) publish(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	id, err := shared.PathEpisodeID(r)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	episode, err := s.app.Publish(r.Context(), actor, id)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	_ = httpjson.Respond(w, http.StatusOK, converters.ToEpisode(episode))
}

// unpublish handles POST /episodes/{id}/unpublish.
func (s svc) unpublish(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	id, err := shared.PathEpisodeID(r)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	episode, err := s.app.Unpublish(r.Context(), actor, id)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	_ = httpjson.Respond(w, http.StatusOK, converters.ToEpisode(episode))
}

// decodeLenient reads a JSON body into dst without rejecting unknown fields:
// the Podcasting 2.0 chapters format is extensible, so unknown members must be
// ignored.
func decodeLenient(w http.ResponseWriter, r *http.Request, dst any) error {
	body := http.MaxBytesReader(w, r.Body, maxJSONBytes)
	dec := json.NewDecoder(body)
	if err := dec.Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return fmt.Errorf("%w: limit %d bytes", httpjson.ErrTooLarge, maxJSONBytes)
		}
		return fmt.Errorf("%w: %w", httpjson.ErrBadRequest, err)
	}
	return nil
}
