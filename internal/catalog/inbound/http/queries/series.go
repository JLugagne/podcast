package queries

import (
	"net/http"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/seriesuc"
	"github.com/JLugagne/podcast/internal/catalog/inbound/http/converters"
	"github.com/JLugagne/podcast/internal/catalog/inbound/http/shared"
	"github.com/JLugagne/podcast/internal/pkg/httpjson"
)

// listSeries handles GET /series?q=.
func (s svc) listSeries(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	views, err := s.app.ListSeries(r.Context(), actor, seriesuc.ListQuery{
		Query: r.URL.Query().Get("q"),
	})
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	_ = httpjson.Respond(w, http.StatusOK, converters.ToSeriesList(views))
}

// getSeries handles GET /series/{id}.
func (s svc) getSeries(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	id, err := shared.PathSeriesID(r)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	view, err := s.app.GetSeries(r.Context(), actor, id)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	_ = httpjson.Respond(w, http.StatusOK, converters.ToSeries(view))
}

// openCover handles GET /series/{id}/cover.
func (s svc) openCover(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	id, err := shared.PathSeriesID(r)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	blob, err := s.app.OpenCover(r.Context(), actor, id)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	serveBlob(w, r, blob)
}
