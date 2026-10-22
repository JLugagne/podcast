// Package httpserver builds the HTTP server with panic recovery, request
// logging and graceful shutdown.
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/JLugagne/podcast/internal/pkg/httpjson"
	"github.com/JLugagne/podcast/internal/pkg/logger"
)

const (
	readHeaderTimeout = 10 * time.Second
	idleTimeout       = 2 * time.Minute
	shutdownTimeout   = 15 * time.Second
)

// Server owns an http.Server and its shutdown lifecycle.
type Server struct {
	server *http.Server
	log    *slog.Logger
}

// New builds a server listening on addr. It deliberately sets no ReadTimeout
// and no WriteTimeout: audio uploads and downloads may legitimately take
// minutes, and a fixed deadline would cut them off. ReadHeaderTimeout still
// bounds slow-header attacks and IdleTimeout reclaims idle keep-alive
// connections.
func New(addr string, handler http.Handler, log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}
	if handler == nil {
		handler = http.NewServeMux()
	}
	wrapped := withLogger(handler, log)
	wrapped = withLogging(wrapped, log)
	wrapped = withRecover(wrapped, log)
	return &Server{
		server: &http.Server{
			Addr:              addr,
			Handler:           wrapped,
			ReadHeaderTimeout: readHeaderTimeout,
			IdleTimeout:       idleTimeout,
		},
		log: log,
	}
}

// Addr returns the configured listen address.
func (s *Server) Addr() string { return s.server.Addr }

// Handler returns the wrapped handler, useful for tests.
func (s *Server) Handler() http.Handler { return s.server.Handler }

// Run serves until ctx is cancelled, then shuts down gracefully. It returns
// nil on a clean shutdown.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() { errCh <- s.server.ListenAndServe() }()
	s.log.Info("http: server listening", slog.String("addr", s.server.Addr))

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("httpserver: listen: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("httpserver: shutdown: %w", err)
		}
		s.log.Info("http: server stopped")
		return nil
	}
}

// withLogger attaches the server logger to every request context.
func withLogger(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(logger.WithLogger(r.Context(), log)))
	})
}

// withRecover turns a panic into a logged 500 JSON response.
func withRecover(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			if err, ok := v.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(v)
			}
			log.Error("http: panic serving request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Any("panic", v),
				slog.String("stack", string(debug.Stack())),
			)
			httpjson.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		}()
		next.ServeHTTP(w, r)
	})
}

// withLogging logs one line per request.
func withLogging(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}
		attrs := []slog.Attr{
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", status),
			slog.Int("bytes", rec.bytes),
			slog.Duration("duration", time.Since(start)),
		}
		log.LogAttrs(r.Context(), requestLevel(status), "http: request", attrs...)
	})
}

// requestLevel reports server errors at Error, everything else at Info.
func requestLevel(status int) slog.Level {
	if status >= http.StatusInternalServerError {
		return slog.LevelError
	}
	return slog.LevelInfo
}

// responseRecorder captures the status and body size written by a handler.
type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *responseRecorder) WriteHeader(code int) {
	if r.status == 0 {
		r.status = code
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// Unwrap lets http.ResponseController reach the underlying writer (Flush,
// deadlines) through the recorder.
func (r *responseRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }
