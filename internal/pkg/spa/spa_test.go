package spa

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func testFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":    {Data: []byte("<!doctype html><title>admin</title>")},
		"assets/app.js": {Data: []byte("console.log('app')")},
		"robots.txt":    {Data: []byte("User-agent: *")},
	}
}

func get(t *testing.T, h http.Handler, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), method, target, nil))
	return rec
}

func TestServesIndexAndAssets(t *testing.T) {
	h := Handler(testFS(), "/admin")

	rec := get(t, h, http.MethodGet, "/admin/")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "admin") {
		t.Fatalf("index: status=%d body=%q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != noCache {
		t.Errorf("index Cache-Control = %q, want %q", got, noCache)
	}

	rec = get(t, h, http.MethodGet, "/admin/assets/app.js")
	if rec.Code != http.StatusOK || rec.Body.String() != "console.log('app')" {
		t.Fatalf("asset: status=%d body=%q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != immutableAge {
		t.Errorf("asset Cache-Control = %q, want %q", got, immutableAge)
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "javascript") {
		t.Errorf("asset Content-Type = %q", got)
	}
}

func TestFallsBackToIndexForUnknownRoutes(t *testing.T) {
	h := Handler(testFS(), "/admin")
	for _, target := range []string{"/admin/series", "/admin/episodes/42", "/admin/series/1/episodes"} {
		rec := get(t, h, http.MethodGet, target)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "admin") {
			t.Errorf("%s: status=%d body=%q", target, rec.Code, rec.Body.String())
		}
	}
}

func TestMissingAssetIsNotFound(t *testing.T) {
	h := Handler(testFS(), "/admin")
	rec := get(t, h, http.MethodGet, "/admin/assets/missing.js")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestRedirectsBarePrefix(t *testing.T) {
	h := Handler(testFS(), "/admin")
	rec := get(t, h, http.MethodGet, "/admin?x=1")
	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want 301", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/admin/?x=1" {
		t.Errorf("Location = %q, want /admin/?x=1", loc)
	}
}

func TestOutsidePrefixIsNotFound(t *testing.T) {
	h := Handler(testFS(), "/admin")
	for _, target := range []string{"/", "/other", "/adminx/"} {
		if rec := get(t, h, http.MethodGet, target); rec.Code != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 404", target, rec.Code)
		}
	}
}

func TestEmptyPrefixServesRoot(t *testing.T) {
	h := Handler(testFS(), "")
	rec := get(t, h, http.MethodGet, "/")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "admin") {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	rec = get(t, h, http.MethodGet, "/robots.txt")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "User-agent") {
		t.Fatalf("robots: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestRejectsNonGetMethods(t *testing.T) {
	h := Handler(testFS(), "/admin")
	rec := get(t, h, http.MethodPost, "/admin/")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}
