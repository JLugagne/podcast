package queries

import (
	"encoding/json"
	"net/http"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/episodeuc"
	"github.com/JLugagne/podcast/internal/catalog/inbound/http/converters"
	"github.com/JLugagne/podcast/internal/catalog/inbound/http/shared"
	"github.com/JLugagne/podcast/internal/pkg/httpjson"
)

// listEpisodes handles GET /series/{id}/episodes?status=.
func (s svc) listEpisodes(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	seriesID, err := shared.PathSeriesID(r)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	status, err := shared.ParseStatus(r)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	episodes, err := s.app.ListEpisodes(r.Context(), actor, seriesID, episodeuc.ListQuery{Status: status})
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	_ = httpjson.Respond(w, http.StatusOK, converters.ToEpisodeList(episodes))
}

// getEpisode handles GET /episodes/{id}.
func (s svc) getEpisode(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	id, err := shared.PathEpisodeID(r)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	episode, err := s.app.GetEpisode(r.Context(), actor, id)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	_ = httpjson.Respond(w, http.StatusOK, converters.ToEpisode(episode))
}

// openAudio handles GET /episodes/{id}/audio.
func (s svc) openAudio(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	id, err := shared.PathEpisodeID(r)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	blob, err := s.app.OpenAudio(r.Context(), actor, id)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	serveBlob(w, r, blob)
}

// getChapters handles GET /episodes/{id}/chapters and serves the Podcasting 2.0
// JSON chapters document.
func (s svc) getChapters(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	id, err := shared.PathEpisodeID(r)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	episode, err := s.app.GetEpisode(r.Context(), actor, id)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json+chapters")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(converters.ToChaptersDocument(episode.Chapters()))
}
