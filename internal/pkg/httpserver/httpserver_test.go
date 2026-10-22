package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JLugagne/podcast/internal/pkg/httpjson"
	"github.com/JLugagne/podcast/internal/pkg/logger"
)

func TestNewSetsAddr(t *testing.T) {
	s := New("127.0.0.1:9090", http.NewServeMux(), nil)
	if s.Addr() != "127.0.0.1:9090" {
		t.Errorf("Addr = %q", s.Addr())
	}
}

func TestRecoverReturnsJSON500(t *testing.T) {
	var logs bytes.Buffer
	log, err := logger.New("json", "debug", &logs)
	if err != nil {
		t.Fatalf("logger.New: %v", err)
	}
	s := New("", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}), log)

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/panic", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var body httpjson.ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	if body.Code != "INTERNAL_ERROR" {
		t.Errorf("code = %q, want INTERNAL_ERROR", body.Code)
	}
	if !strings.Contains(logs.String(), "boom") {
		t.Errorf("panic was not logged: %q", logs.String())
	}
}

func TestRequestIsLogged(t *testing.T) {
	var logs bytes.Buffer
	log, err := logger.New("text", "info", &logs)
	if err != nil {
		t.Fatalf("logger.New: %v", err)
	}
	s := New("", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if logger.FromContext(r.Context()) != log {
			t.Error("request context does not carry the server logger")
		}
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("tea"))
	}), log)

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/brew", nil))

	if rec.Code != http.StatusTeapot || rec.Body.String() != "tea" {
		t.Fatalf("handler response: status=%d body=%q", rec.Code, rec.Body.String())
	}
	out := logs.String()
	for _, want := range []string{"http: request", "method=GET", "path=/brew", "status=418", "bytes=3"} {
		if !strings.Contains(out, want) {
			t.Errorf("log output missing %q: %q", want, out)
		}
	}
}

func TestServerErrorLoggedAtErrorLevel(t *testing.T) {
	var logs bytes.Buffer
	log, err := logger.New("text", "info", &logs)
	if err != nil {
		t.Fatalf("logger.New: %v", err)
	}
	s := New("", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}), log)

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/down", nil))

	if !strings.Contains(logs.String(), "level=ERROR") {
		t.Errorf("5xx request not logged at ERROR: %q", logs.String())
	}
}

func TestNilLoggerFallsBackToDefault(t *testing.T) {
	s := New("", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	_ = slog.Default()
}
