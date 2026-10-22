// Package commands holds the listening write-side HTTP handlers.
package commands

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/JLugagne/podcast/internal/listening/domain"
	"github.com/JLugagne/podcast/internal/listening/domain/usecases/progressuc"
	"github.com/JLugagne/podcast/internal/listening/inbound/http/converters"
	"github.com/JLugagne/podcast/internal/listening/inbound/http/shared"
	"github.com/JLugagne/podcast/internal/pkg/httpjson"
	v1 "github.com/JLugagne/podcast/pkg/listening/http/v1"
)

// App is the use-case surface the listening write handlers need. It is
// satisfied by *app.App.
type App interface {
	progressuc.ProgressCommands
}

// maxJSONBytes bounds every JSON request body.
const maxJSONBytes = 1 << 20

// svc holds the handlers' collaborators.
type svc struct {
	app App
}

// SetupRouter registers every listening write route on api (the /api/v1
// subrouter).
func SetupRouter(api *mux.Router, app App) {
	s := svc{app: app}

	api.HandleFunc("/listening/progress/{episodeId}", shared.WithActor(s.recordPosition)).Methods(http.MethodPut)
	api.HandleFunc("/listening/progress/{episodeId}/played", shared.WithActor(s.markPlayed)).Methods(http.MethodPost)
	api.HandleFunc("/listening/progress/{episodeId}", shared.WithActor(s.markUnplayed)).Methods(http.MethodDelete)
}

// decode reads a strict JSON body into req. On failure it writes the error
// response and reports false.
func decode[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var req T
	if err := httpjson.Decode(w, r, &req, maxJSONBytes); err != nil {
		shared.WriteDomainError(w, r, err)
		return req, false
	}
	return req, true
}

// recordPosition handles PUT /listening/progress/{episodeId}.
func (s svc) recordPosition(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	episode, err := shared.PathEpisodeRef(r)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	req, ok := decode[v1.ProgressInput](w, r)
	if !ok {
		return
	}
	progress, err := s.app.RecordPosition(r.Context(), actor, episode, req.PositionSeconds, req.Completed)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	_ = httpjson.Respond(w, http.StatusOK, converters.ToProgress(progress))
}

// markPlayed handles POST /listening/progress/{episodeId}/played.
func (s svc) markPlayed(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	episode, err := shared.PathEpisodeRef(r)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	progress, err := s.app.MarkPlayed(r.Context(), actor, episode)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	_ = httpjson.Respond(w, http.StatusOK, converters.ToProgress(progress))
}

// markUnplayed handles DELETE /listening/progress/{episodeId}.
func (s svc) markUnplayed(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	episode, err := shared.PathEpisodeRef(r)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	if err := s.app.MarkUnplayed(r.Context(), actor, episode); err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	httpjson.NoContent(w)
}
