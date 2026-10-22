package feed_test

import (
	"context"
	"encoding/xml"
	"errors"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/blobstore"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/episodeuc"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/seriesuc"
	"github.com/JLugagne/podcast/internal/catalog/inbound/feed"
)

// update rewrites the golden file instead of comparing against it.
var update = flag.Bool("update", false, "update the golden files")

const (
	testSeriesID = "018f0000-0000-7000-8000-000000000001"
	testFirstID  = "018f0000-0000-7000-8000-000000000002"
	testSecondID = "018f0000-0000-7000-8000-000000000003"
	testDraftID  = "018f0000-0000-7000-8000-000000000004"
	publicURL    = "https://podcast.example"
	goldenPath   = "testdata/feed.golden.xml"
)

var testNow = time.Date(2026, 10, 21, 12, 30, 0, 0, time.UTC)

// testSeries builds a stored series, with a cover when cover is true.
func testSeries(cover bool) *domain.Series {
	var asset *domain.Asset
	if cover {
		asset = &domain.Asset{Key: "series/x/cover.png", ContentType: "image/png", SizeBytes: 8}
	}
	return domain.ReconstituteSeries(
		domain.SeriesID(testSeriesID),
		domain.SeriesAttributes{
			Title:       "Deep Dive",
			Author:      "Ada Lovelace",
			Description: "A show about numbers & ideas",
			AccentColor: domain.DefaultAccentColor,
			Slug:        "deep-dive",
		},
		asset, testNow, testNow.Add(time.Hour), 3,
	)
}

// testEpisode builds a stored episode with audio and, when chapters is true,
// two chapters.
func testEpisode(id, title string, number int, status domain.EpisodeStatus, published *time.Time, duration float64, chapters bool) *domain.Episode {
	audio := &domain.AudioFile{
		Asset:           domain.Asset{Key: "episodes/" + id + "/audio.mp3", ContentType: "audio/mpeg", SizeBytes: 2048},
		DurationSeconds: duration,
		OriginalName:    "episode.mp3",
	}
	list := []domain.Chapter(nil)
	if chapters {
		list = []domain.Chapter{
			{StartSeconds: 0, Title: "Intro", TOC: true},
			{StartSeconds: 30, Title: "Body", TOC: true},
		}
	}
	return domain.ReconstituteEpisode(
		domain.EpisodeID(id),
		domain.SeriesID(testSeriesID),
		domain.EpisodeAttributes{Number: number, Title: title, Description: title + " description"},
		status, audio, list, published,
		testNow, testNow, 1,
	)
}

// feedEpisodes returns the fixture episodes in a deliberately unordered list.
func feedEpisodes() []*domain.Episode {
	firstPublished := time.Date(2026, 10, 19, 8, 0, 0, 0, time.UTC)
	secondPublished := time.Date(2026, 10, 20, 8, 0, 0, 0, time.UTC)
	return []*domain.Episode{
		testEpisode(testDraftID, "Not yet", 3, domain.EpisodeStatusDraft, nil, 10, false),
		testEpisode(testFirstID, "First Steps", 1, domain.EpisodeStatusPublished, &firstPublished, 95.5, true),
		testEpisode(testSecondID, "Second Thoughts", 2, domain.EpisodeStatusPublished, &secondPublished, 3600, false),
	}
}

// fakeApp implements the feed App interface; unset methods panic.
type fakeApp struct {
	getSeriesBySlugFn func(ctx context.Context, actor *domain.Actor, slug string) (*seriesuc.SeriesView, error)
	listEpisodesFn    func(ctx context.Context, actor *domain.Actor, seriesID domain.SeriesID, q episodeuc.ListQuery) ([]*domain.Episode, error)
}

func (f *fakeApp) GetSeries(_ context.Context, _ *domain.Actor, _ domain.SeriesID) (*seriesuc.SeriesView, error) {
	panic("fakeApp.GetSeries not expected")
}

func (f *fakeApp) GetSeriesBySlug(ctx context.Context, actor *domain.Actor, slug string) (*seriesuc.SeriesView, error) {
	if f.getSeriesBySlugFn == nil {
		panic("fakeApp.GetSeriesBySlugFn not set")
	}
	return f.getSeriesBySlugFn(ctx, actor, slug)
}

func (f *fakeApp) ListSeries(_ context.Context, _ *domain.Actor, _ seriesuc.ListQuery) ([]*seriesuc.SeriesView, error) {
	panic("fakeApp.ListSeries not expected")
}

func (f *fakeApp) OpenCover(_ context.Context, _ *domain.Actor, _ domain.SeriesID) (blobstore.Blob, error) {
	panic("fakeApp.OpenCover not expected")
}

func (f *fakeApp) GetEpisode(_ context.Context, _ *domain.Actor, _ domain.EpisodeID) (*domain.Episode, error) {
	panic("fakeApp.GetEpisode not expected")
}

func (f *fakeApp) ListEpisodes(ctx context.Context, actor *domain.Actor, seriesID domain.SeriesID, q episodeuc.ListQuery) ([]*domain.Episode, error) {
	if f.listEpisodesFn == nil {
		panic("fakeApp.ListEpisodesFn not set")
	}
	return f.listEpisodesFn(ctx, actor, seriesID, q)
}

func (f *fakeApp) OpenAudio(_ context.Context, _ *domain.Actor, _ domain.EpisodeID) (blobstore.Blob, error) {
	panic("fakeApp.OpenAudio not expected")
}

// newRouter mounts the feed route with a fixed clock.
func newRouter(t *testing.T, app feed.App, url string) http.Handler {
	t.Helper()
	router := mux.NewRouter()
	feed.SetupRouter(router, app, feed.Options{PublicURL: url, Now: func() time.Time { return testNow }})
	return router
}

// serve runs req through h and returns the recorder.
func serve(h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// parsedFeed mirrors the RSS document so the tests can parse it back with
// encoding/xml and assert on the result.
type parsedFeed struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel struct {
		Title         string `xml:"title"`
		Link          string `xml:"link"`
		Description   string `xml:"description"`
		Language      string `xml:"language"`
		LastBuildDate string `xml:"lastBuildDate"`
		Author        string `xml:"author"`
		Image         struct {
			Href string `xml:"href,attr"`
		} `xml:"image"`
		Items []struct {
			Title       string `xml:"title"`
			Description string `xml:"description"`
			GUID        struct {
				IsPermaLink string `xml:"isPermaLink,attr"`
				Value       string `xml:",chardata"`
			} `xml:"guid"`
			PubDate   string `xml:"pubDate"`
			Enclosure struct {
				URL    string `xml:"url,attr"`
				Length int64  `xml:"length,attr"`
				Type   string `xml:"type,attr"`
			} `xml:"enclosure"`
			Duration string `xml:"duration"`
			Episode  int    `xml:"episode"`
			Chapters struct {
				URL  string `xml:"url,attr"`
				Type string `xml:"type,attr"`
			} `xml:"chapters"`
		} `xml:"item"`
	} `xml:"channel"`
}

// TestBuildGolden compares the rendered document with its golden file. Run
// `go test ./internal/catalog/inbound/feed -update` to refresh it.
func TestBuildGolden(t *testing.T) {
	body, err := feed.Build(testSeries(true), feedEpisodes(), publicURL+"/", testNow)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if *update {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o750); err != nil {
			t.Fatalf("create golden directory: %v", err)
		}
		if err := os.WriteFile(goldenPath, body, 0o600); err != nil {
			t.Fatalf("write golden file: %v", err)
		}
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	if string(body) != string(want) {
		t.Fatalf("feed does not match %s:\n--- got ---\n%s\n--- want ---\n%s", goldenPath, body, want)
	}
}

// TestBuildParsesBack parses the document and asserts the documented fields.
func TestBuildParsesBack(t *testing.T) {
	body, err := feed.Build(testSeries(true), feedEpisodes(), publicURL, testNow)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	var doc parsedFeed
	if err := xml.Unmarshal(body, &doc); err != nil {
		t.Fatalf("xml.Unmarshal() error = %v\n%s", err, body)
	}
	if doc.Version != "2.0" {
		t.Fatalf("version = %q, want 2.0", doc.Version)
	}
	channel := doc.Channel
	if channel.Title != "Deep Dive" || channel.Author != "Ada Lovelace" || channel.Language != "en" {
		t.Fatalf("unexpected channel: %+v", channel)
	}
	if channel.Link != publicURL+"/series/"+testSeriesID {
		t.Fatalf("channel link = %q", channel.Link)
	}
	if channel.Image.Href != publicURL+"/api/v1/series/"+testSeriesID+"/cover" {
		t.Fatalf("image href = %q", channel.Image.Href)
	}
	if channel.LastBuildDate != testNow.Format(time.RFC1123Z) {
		t.Fatalf("lastBuildDate = %q", channel.LastBuildDate)
	}
	if len(channel.Items) != 2 {
		t.Fatalf("items = %d, want 2 published episodes", len(channel.Items))
	}

	newest := channel.Items[0]
	if newest.Title != "Second Thoughts" || newest.Episode != 2 {
		t.Fatalf("first item = %+v, want the newest episode", newest)
	}
	if newest.GUID.IsPermaLink != "false" || newest.GUID.Value != testSecondID {
		t.Fatalf("guid = %+v", newest.GUID)
	}
	if newest.PubDate != "Tue, 20 Oct 2026 08:00:00 +0000" {
		t.Fatalf("pubDate = %q", newest.PubDate)
	}
	if newest.Enclosure.URL != publicURL+"/api/v1/episodes/"+testSecondID+"/audio" ||
		newest.Enclosure.Length != 2048 || newest.Enclosure.Type != "audio/mpeg" {
		t.Fatalf("enclosure = %+v", newest.Enclosure)
	}
	if newest.Duration != "3600" {
		t.Fatalf("itunes:duration = %q, want 3600", newest.Duration)
	}
	if newest.Chapters.URL != "" {
		t.Fatalf("episode without chapters has a chapters link: %+v", newest.Chapters)
	}

	older := channel.Items[1]
	if older.Title != "First Steps" || older.Episode != 1 {
		t.Fatalf("second item = %+v", older)
	}
	if older.Chapters.URL != publicURL+"/api/v1/episodes/"+testFirstID+"/chapters" ||
		older.Chapters.Type != "application/json+chapters" {
		t.Fatalf("chapters link = %+v", older.Chapters)
	}
	if older.Duration != "95" {
		t.Fatalf("itunes:duration = %q, want 95", older.Duration)
	}
}

// TestBuildOmitsMissingCoverAndChapters checks the optional elements.
// TestBuildOmitsMissingCoverAndChapters checks the optional elements.
func TestBuildOmitsMissingCoverAndChapters(t *testing.T) {
	published := time.Date(2026, 10, 19, 8, 0, 0, 0, time.UTC)
	episodes := []*domain.Episode{
		testEpisode(testFirstID, "First Steps", 1, domain.EpisodeStatusPublished, &published, 95.5, false),
	}
	body, err := feed.Build(testSeries(false), episodes, publicURL, testNow)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	for _, unwanted := range []string{"itunes:image", "podcast:chapters"} {
		if strings.Contains(string(body), unwanted) {
			t.Fatalf("feed contains %q although the fixture has none:\n%s", unwanted, body)
		}
	}
}

// TestServeFeed covers the happy path of GET /feeds/{slug}.xml.
func TestServeFeed(t *testing.T) {
	app := &fakeApp{
		getSeriesBySlugFn: func(_ context.Context, actor *domain.Actor, slug string) (*seriesuc.SeriesView, error) {
			if slug != "deep-dive" {
				t.Errorf("slug = %q, want deep-dive", slug)
			}
			if actor.ID() != domain.LocalAdminID {
				t.Errorf("actor = %q, want the local admin", actor.ID())
			}
			return &seriesuc.SeriesView{Series: testSeries(true)}, nil
		},
		listEpisodesFn: func(_ context.Context, _ *domain.Actor, seriesID domain.SeriesID, q episodeuc.ListQuery) ([]*domain.Episode, error) {
			if seriesID != domain.SeriesID(testSeriesID) {
				t.Errorf("series id = %q", seriesID)
			}
			if q.Status == nil || *q.Status != domain.EpisodeStatusPublished {
				t.Errorf("status filter = %v, want published", q.Status)
			}
			return feedEpisodes(), nil
		},
	}

	rec := serve(newRouter(t, app, publicURL), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/feeds/deep-dive.xml", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/rss+xml; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	var doc parsedFeed
	if err := xml.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("response is not valid XML: %v\n%s", err, rec.Body.String())
	}
	if len(doc.Channel.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(doc.Channel.Items))
	}
}

// TestServeFeedUnknownSlug returns a JSON 404 for an unknown slug.
func TestServeFeedUnknownSlug(t *testing.T) {
	app := &fakeApp{
		getSeriesBySlugFn: func(context.Context, *domain.Actor, string) (*seriesuc.SeriesView, error) {
			return nil, domain.ErrSeriesNotFound
		},
	}

	rec := serve(newRouter(t, app, publicURL), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/feeds/missing.xml", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body %s)", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("Content-Type = %q, want JSON", got)
	}
	if !strings.Contains(rec.Body.String(), `"SERIES_NOT_FOUND"`) {
		t.Fatalf("body = %s, want the SERIES_NOT_FOUND code", rec.Body.String())
	}
}

// TestServeFeedListError maps a repository failure to an error response.
func TestServeFeedListError(t *testing.T) {
	app := &fakeApp{
		getSeriesBySlugFn: func(context.Context, *domain.Actor, string) (*seriesuc.SeriesView, error) {
			return &seriesuc.SeriesView{Series: testSeries(true)}, nil
		},
		listEpisodesFn: func(context.Context, *domain.Actor, domain.SeriesID, episodeuc.ListQuery) ([]*domain.Episode, error) {
			return nil, errors.New("boom")
		},
	}

	rec := serve(newRouter(t, app, publicURL), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/feeds/deep-dive.xml", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body %s)", rec.Code, rec.Body.String())
	}
}
