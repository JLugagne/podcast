// Package commands holds the catalog write-side HTTP handlers.
package commands

import (
	"net/http"

	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/episodeuc"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/seriesuc"
	"github.com/JLugagne/podcast/internal/catalog/inbound/http/shared"
	"github.com/JLugagne/podcast/internal/pkg/httpjson"
	"github.com/gorilla/mux"
)

// App is the use-case surface the catalog write handlers need. It is satisfied
// by *app.App.
type App interface {
	seriesuc.SeriesCommands
	episodeuc.EpisodeCommands
}

// maxJSONBytes bounds every JSON request body.
const maxJSONBytes = 1 << 20

// Default upload limits, applied when the caller configures none.
const (
	DefaultMaxAudioBytes int64 = 2 << 30
	DefaultMaxImageBytes int64 = 10 << 20
)

// Options configures the streamed upload limits. Non-positive values fall back
// to the defaults.
type Options struct {
	MaxAudioBytes int64
	MaxImageBytes int64
}

// withDefaults fills every unset limit with its default.
func (o Options) withDefaults() Options {
	if o.MaxAudioBytes <= 0 {
		o.MaxAudioBytes = DefaultMaxAudioBytes
	}
	if o.MaxImageBytes <= 0 {
		o.MaxImageBytes = DefaultMaxImageBytes
	}
	return o
}

// svc holds the handlers' collaborators.
type svc struct {
	app  App
	opts Options
}

// SetupRouter registers every catalog write route on api (the /api/v1
// subrouter).
func SetupRouter(api *mux.Router, app App, opts Options) {
	s := svc{app: app, opts: opts.withDefaults()}

	api.HandleFunc("/series", shared.WithActor(s.createSeries)).Methods(http.MethodPost)
	api.HandleFunc("/series/{id}", shared.WithActor(s.updateSeries)).Methods(http.MethodPut)
	api.HandleFunc("/series/{id}", shared.WithActor(s.deleteSeries)).Methods(http.MethodDelete)
	api.HandleFunc("/series/{id}/cover", shared.WithActor(s.uploadCover)).Methods(http.MethodPut)
	api.HandleFunc("/series/{id}/cover", shared.WithActor(s.removeCover)).Methods(http.MethodDelete)
	api.HandleFunc("/series/{id}/episodes", shared.WithActor(s.createEpisode)).Methods(http.MethodPost)

	api.HandleFunc("/episodes/{id}", shared.WithActor(s.updateEpisode)).Methods(http.MethodPut)
	api.HandleFunc("/episodes/{id}", shared.WithActor(s.deleteEpisode)).Methods(http.MethodDelete)
	api.HandleFunc("/episodes/{id}/audio", shared.WithActor(s.uploadAudio)).Methods(http.MethodPut)
	api.HandleFunc("/episodes/{id}/chapters", shared.WithActor(s.setChapters)).Methods(http.MethodPut)
	api.HandleFunc("/episodes/{id}/publish", shared.WithActor(s.publish)).Methods(http.MethodPost)
	api.HandleFunc("/episodes/{id}/unpublish", shared.WithActor(s.unpublish)).Methods(http.MethodPost)
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

// requiredVersion returns the optimistic-lock version of an update request.
func requiredVersion(version *int) (int, error) {
	if version == nil {
		return 0, shared.ErrVersionRequired
	}
	return *version, nil
}
