// Package queries holds the listening read-side HTTP handlers.
package queries

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/JLugagne/podcast/internal/listening/domain"
	"github.com/JLugagne/podcast/internal/listening/domain/usecases/progressuc"
	"github.com/JLugagne/podcast/internal/listening/inbound/http/converters"
	"github.com/JLugagne/podcast/internal/listening/inbound/http/shared"
	"github.com/JLugagne/podcast/internal/pkg/httpjson"
)

// App is the use-case surface the listening read handlers need. It is
// satisfied by *app.App.
type App interface {
	progressuc.ProgressQueries
}

// svc holds the handlers' collaborators.
type svc struct {
	app App
}

// SetupRouter registers every listening read route on api (the /api/v1
// subrouter).
func SetupRouter(api *mux.Router, app App) {
	s := svc{app: app}

	api.HandleFunc("/listening/progress", shared.WithActor(s.listProgress)).Methods(http.MethodGet)
	api.HandleFunc("/listening/continue", shared.WithActor(s.continueListening)).Methods(http.MethodGet)
}

// listProgress handles GET /listening/progress.
func (s svc) listProgress(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	rows, err := s.app.ListProgress(r.Context(), actor)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	_ = httpjson.Respond(w, http.StatusOK, converters.ToProgressList(rows))
}

// continueListening handles GET /listening/continue?limit=.
func (s svc) continueListening(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	limit, err := parseLimit(r)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	items, err := s.app.ContinueListening(r.Context(), actor, limit)
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	_ = httpjson.Respond(w, http.StatusOK, converters.ToContinueList(items))
}

// parseLimit reads the optional limit query parameter. A missing value means
// "use the app default"; a malformed or negative value is a bad request.
func parseLimit(r *http.Request) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return 0, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 0 {
		return 0, fmt.Errorf("%w: limit must be a non-negative integer", httpjson.ErrBadRequest)
	}
	return limit, nil
}
