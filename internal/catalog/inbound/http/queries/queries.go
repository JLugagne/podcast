// Package queries holds the catalog read-side HTTP handlers.
package queries

import (
	"net/http"

	"github.com/JLugagne/podcast/internal/catalog/domain/blobstore"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/episodeuc"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/seriesuc"
	"github.com/JLugagne/podcast/internal/catalog/inbound/http/shared"
	"github.com/gorilla/mux"
)

// App is the use-case surface the read handlers need. It is satisfied by
// *app.App.
type App interface {
	seriesuc.SeriesQueries
	episodeuc.EpisodeQueries
}

// svc holds the handlers' collaborators.
type svc struct {
	app App
}

// SetupRouter registers every catalog read route on api (the /api/v1
// subrouter).
func SetupRouter(api *mux.Router, app App) {
	s := svc{app: app}

	api.HandleFunc("/series", shared.WithActor(s.listSeries)).Methods(http.MethodGet)
	api.HandleFunc("/series/{id}", shared.WithActor(s.getSeries)).Methods(http.MethodGet)
	api.HandleFunc("/series/{id}/cover", shared.WithActor(s.openCover)).Methods(http.MethodGet)
	api.HandleFunc("/series/{id}/episodes", shared.WithActor(s.listEpisodes)).Methods(http.MethodGet)

	api.HandleFunc("/episodes/{id}", shared.WithActor(s.getEpisode)).Methods(http.MethodGet)
	api.HandleFunc("/episodes/{id}/audio", shared.WithActor(s.openAudio)).Methods(http.MethodGet)
	api.HandleFunc("/episodes/{id}/chapters", shared.WithActor(s.getChapters)).Methods(http.MethodGet)
}

// serveBlob streams a stored blob with Range support. Media players rely on
// range requests to seek, and the content never changes in place, so caches are
// told to revalidate.
func serveBlob(w http.ResponseWriter, r *http.Request, blob blobstore.Blob) {
	defer func() { _ = blob.Close() }()
	w.Header().Set("Content-Type", blob.ContentType)
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeContent(w, r, "", blob.ModTime, blob)
}
