// Package shared holds the helpers every listening HTTP handler relies on:
// actor injection, path parsing, error mapping and JSON writes.
package shared

import (
	"context"
	"errors"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/JLugagne/podcast/internal/listening/domain"
	"github.com/JLugagne/podcast/internal/pkg/httpjson"
	"github.com/JLugagne/podcast/internal/pkg/logger"
)

// actorContextKey is the private type of the actor context key, so no other
// package can collide with it.
type actorContextKey struct{}

// ActorFromRequest returns the actor performing the request. The listener app
// has no authentication, so every request is performed by the local listener.
func ActorFromRequest(r *http.Request) *domain.Actor {
	if actor, ok := r.Context().Value(actorContextKey{}).(*domain.Actor); ok && actor != nil {
		return actor
	}
	return domain.LocalListener()
}

// InjectActor is middleware that stores the local listener in the request
// context. Authentication may replace it later without touching the handlers.
func InjectActor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), actorContextKey{}, domain.LocalListener())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// WithActor adapts an actor-aware handler into an http.HandlerFunc. The actor
// comes from the request context, defaulting to the local listener.
func WithActor(h func(http.ResponseWriter, *http.Request, *domain.Actor)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h(w, r, ActorFromRequest(r))
	}
}

// PathEpisodeRef parses the {episodeId} path variable as an episode reference.
// It returns domain.ErrInvalidID when the value is not a UUID.
func PathEpisodeRef(r *http.Request) (domain.EpisodeRef, error) {
	return domain.ParseEpisodeRef(mux.Vars(r)["episodeId"])
}

// WriteDomainError maps a domain error family to an HTTP status and writes the
// {code, message} body. Unknown errors become a logged generic 500 so internals
// never leak to clients.
func WriteDomainError(w http.ResponseWriter, r *http.Request, err error) {
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		httpjson.Error(w, r, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body too large")
		return
	case errors.Is(err, httpjson.ErrTooLarge):
		httpjson.Error(w, r, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body too large")
		return
	case errors.Is(err, httpjson.ErrBadRequest):
		httpjson.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "malformed request body")
		return
	}

	status := 0
	switch {
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrValidation):
		status = http.StatusUnprocessableEntity
	case errors.Is(err, domain.ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrUnauthorized):
		status = http.StatusUnauthorized
	case errors.Is(err, domain.ErrForbidden):
		status = http.StatusForbidden
	}

	var derr *domain.Error
	if status != 0 && errors.As(err, &derr) {
		httpjson.Error(w, r, status, derr.Code, derr.Message)
		return
	}
	logger.FromContext(r.Context()).Error("listening: unexpected handler error", "error", err)
	httpjson.Error(w, r, http.StatusInternalServerError, "INTERNAL", "internal server error")
}
