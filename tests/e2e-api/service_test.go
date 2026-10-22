package e2eapi_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/JLugagne/podcast/internal/catalog"
)

// TestMigrationsIdempotent proves that running the catalog migrations twice is
// harmless, which catalog.New relies on at every start.
func TestMigrationsIdempotent(t *testing.T) {
	pool := requireDB(t)
	for run := 1; run <= 2; run++ {
		if err := catalog.Migrate(context.Background(), pool, nil); err != nil {
			t.Fatalf("catalog.Migrate run %d: %v", run, err)
		}
	}
}

// TestHealthz covers the database-backed health probe.
func TestHealthz(t *testing.T) {
	e := newEnv(t)
	resp := e.get("/healthz").wantStatus(http.StatusOK)
	var body map[string]string
	resp.decode(&body)
	if body["status"] != "ok" {
		t.Fatalf("health status = %q, want ok", body["status"])
	}
}

// TestAdminSPA covers the admin single-page application, its history fallback
// and the root redirect.
// TestAdminSPA covers the admin single-page application and its history
// fallback.
func TestAdminSPA(t *testing.T) {
	e := newEnv(t)

	index := e.get("/admin/").wantStatus(http.StatusOK)
	if got := index.header.Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("/admin/ content type = %q, want text/html", got)
	}
	if !strings.Contains(string(index.body), `id="root"`) {
		t.Fatalf("/admin/ does not serve the SPA shell: %q", string(index.body))
	}

	// Client-side routes fall back to index.html.
	deep := e.get("/admin/series/00000000-0000-0000-0000-000000000000").wantStatus(http.StatusOK)
	if !strings.Contains(string(deep.body), `id="root"`) {
		t.Fatalf("deep SPA route did not fall back to index.html: %q", string(deep.body))
	}
}

// TestPlayerSPA covers the listener single-page application served at /, its
// history fallback, and the JSON 404 of unknown API and feed paths.
func TestPlayerSPA(t *testing.T) {
	e := newEnv(t)

	index := e.get("/").wantStatus(http.StatusOK)
	if got := index.header.Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("/ content type = %q, want text/html", got)
	}
	if !strings.Contains(string(index.body), `id="root"`) {
		t.Fatalf("/ does not serve the player SPA shell: %q", string(index.body))
	}

	for _, target := range []string{"/series/abc", "/play", "/some/deep/route"} {
		rec := e.get(target).wantStatus(http.StatusOK)
		if !strings.Contains(string(rec.body), `id="root"`) {
			t.Fatalf("%s did not fall back to the player index: %q", target, string(rec.body))
		}
	}

	// Unknown API and feed paths must not be swallowed by the SPA fallback.
	e.get("/api/v1/nope").wantError(http.StatusNotFound, "NOT_FOUND")
	e.get("/feeds/unknown.xml").wantError(http.StatusNotFound, "SERIES_NOT_FOUND")
}
