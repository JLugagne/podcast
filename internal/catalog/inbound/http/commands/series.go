package commands

import (
	"net/http"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/seriesuc"
	"github.com/JLugagne/podcast/internal/catalog/inbound/http/converters"
	"github.com/JLugagne/podcast/internal/catalog/inbound/http/shared"
	"github.com/JLugagne/podcast/internal/pkg/httpjson"
	v1 "github.com/JLugagne/podcast/pkg/catalog/http/v1"
)

// createSeries handles POST /series.
func (s svc) createSeries(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	req, ok := decode[v1.SeriesInput](w, r)
	if !ok {
		return
	}
	view, err := s.app.CreateSeries(r.Context(), actor, converters.ToSeriesAttributes(req))
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	_ = httpjson.Respond(w, http.StatusCreated, converters.ToSeries(view))
}

// updateSeries handles PUT /series/{id}.
func (s svc) updateSeries(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	id, err := shared.PathSeriesID(r)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	req, ok := decode[v1.SeriesInput](w, r)
	if !ok {
		return
	}
	version, err := requiredVersion(req.Version)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	view, err := s.app.UpdateSeries(r.Context(), actor, id, converters.ToSeriesAttributes(req), version)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	_ = httpjson.Respond(w, http.StatusOK, converters.ToSeries(view))
}

// deleteSeries handles DELETE /series/{id}.
func (s svc) deleteSeries(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	id, err := shared.PathSeriesID(r)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	if err := s.app.DeleteSeries(r.Context(), actor, id); err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	httpjson.NoContent(w)
}

// uploadCover handles PUT /series/{id}/cover.
//
// The body is a multipart document streamed straight to the use case; the part
// named "file" carries the image.
func (s svc) uploadCover(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	id, err := shared.PathSeriesID(r)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	file, _, err := readUpload(w, r, s.opts.MaxImageBytes, false)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	defer file.close()

	view, err := s.app.UploadCover(r.Context(), actor, id, seriesuc.UploadInput{
		Reader:      file.Reader,
		ContentType: file.ContentType,
	})
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	_ = httpjson.Respond(w, http.StatusOK, converters.ToSeries(view))
}

// removeCover handles DELETE /series/{id}/cover.
func (s svc) removeCover(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	id, err := shared.PathSeriesID(r)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	view, err := s.app.RemoveCover(r.Context(), actor, id)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	_ = httpjson.Respond(w, http.StatusOK, converters.ToSeries(view))
}
