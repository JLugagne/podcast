package e2eapi_test

import (
	"encoding/xml"
	"net/http"
	"strings"
	"testing"
	"time"

	v1 "github.com/JLugagne/podcast/pkg/catalog/http/v1"
)

// feedDocument mirrors the RSS document served by /feeds/{slug}.xml so the
// tests can parse it back with encoding/xml.
type feedDocument struct {
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
			Title string `xml:"title"`
			GUID  struct {
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

// TestFeedRSS covers the RSS feed: published items only, newest first, absolute
// enclosure and chapters URLs, and the channel metadata.
func TestFeedRSS(t *testing.T) {
	e := newEnv(t)
	series := e.createSeries(v1.SeriesInput{
		Title:       "Feed Show",
		Author:      "Ada Lovelace",
		Description: "Numbers & ideas",
	})
	e.uploadCover(series.ID, []byte("\x89PNG\r\n\x1a\nfake-cover"), "image/png", "cover.png").wantStatus(http.StatusOK)

	draft := e.createEpisode(series.ID, v1.EpisodeInput{Number: intPtr(3), Title: "Still Draft"})
	e.uploadAudio(draft.ID, []byte("draft-audio"), "draft.mp3", floatPtr(120))
	e.setChapters(draft.ID, validChapters())

	first := e.createEpisode(series.ID, v1.EpisodeInput{Number: intPtr(1), Title: "First Broadcast", Description: "First description"})
	e.uploadAudio(first.ID, []byte("first-audio"), "first.mp3", floatPtr(95))
	e.setChapters(first.ID, validChapters())
	publishedFirst := e.publish(first.ID)

	second := e.createEpisode(series.ID, v1.EpisodeInput{Number: intPtr(2), Title: "Second Broadcast", Description: "Second description"})
	e.uploadAudio(second.ID, []byte("second-audio"), "second.mp3", floatPtr(3600))
	e.setChapters(second.ID, validChapters())
	publishedSecond := e.publish(second.ID)

	resp := e.get("/feeds/" + series.Slug + ".xml").wantStatus(http.StatusOK)
	if got := resp.header.Get("Content-Type"); got != "application/rss+xml; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}

	var doc feedDocument
	if err := xml.Unmarshal(resp.body, &doc); err != nil {
		t.Fatalf("feed is not valid XML: %v\n%s", err, string(resp.body))
	}
	if doc.Version != "2.0" {
		t.Fatalf("rss version = %q, want 2.0", doc.Version)
	}
	channel := doc.Channel
	if channel.Title != series.Title || channel.Author != "Ada Lovelace" || channel.Language != "en" {
		t.Fatalf("unexpected channel: %+v", channel)
	}
	if channel.Description != "Numbers & ideas" {
		t.Fatalf("channel description = %q", channel.Description)
	}
	if channel.Link != publicURL+"/series/"+series.ID {
		t.Fatalf("channel link = %q", channel.Link)
	}
	if channel.Image.Href != publicURL+"/api/v1/series/"+series.ID+"/cover" {
		t.Fatalf("image href = %q", channel.Image.Href)
	}
	if _, err := time.Parse(time.RFC1123Z, channel.LastBuildDate); err != nil {
		t.Fatalf("lastBuildDate = %q is not RFC1123Z: %v", channel.LastBuildDate, err)
	}

	if len(channel.Items) != 2 {
		t.Fatalf("items = %d, want the 2 published episodes\n%s", len(channel.Items), string(resp.body))
	}
	newest, older := channel.Items[0], channel.Items[1]
	if newest.Title != "Second Broadcast" || newest.GUID.Value != second.ID {
		t.Fatalf("first item = %+v, want the newest episode %q", newest, second.ID)
	}
	if older.Title != "First Broadcast" || older.GUID.Value != first.ID {
		t.Fatalf("second item = %+v, want %q", older, first.ID)
	}
	for _, item := range channel.Items {
		if item.GUID.Value == draft.ID {
			t.Fatalf("draft episode %q is listed in the feed", draft.ID)
		}
		if item.GUID.IsPermaLink != "false" {
			t.Fatalf("guid %q isPermaLink = %q, want false", item.GUID.Value, item.GUID.IsPermaLink)
		}
		if _, err := time.Parse(time.RFC1123Z, item.PubDate); err != nil {
			t.Fatalf("pubDate = %q is not RFC1123Z: %v", item.PubDate, err)
		}
	}

	if newest.Enclosure.URL != publicURL+"/api/v1/episodes/"+second.ID+"/audio" {
		t.Fatalf("enclosure url = %q", newest.Enclosure.URL)
	}
	if newest.Enclosure.Length != int64(len("second-audio")) || newest.Enclosure.Type != "audio/mpeg" {
		t.Fatalf("enclosure = %+v", newest.Enclosure)
	}
	if newest.Duration != "3600" || newest.Episode != 2 {
		t.Fatalf("duration/episode = %q/%d", newest.Duration, newest.Episode)
	}
	if newest.Chapters.URL != publicURL+"/api/v1/episodes/"+second.ID+"/chapters" ||
		newest.Chapters.Type != "application/json+chapters" {
		t.Fatalf("chapters link = %+v", newest.Chapters)
	}
	if older.Duration != "95" {
		t.Fatalf("first episode duration = %q, want 95", older.Duration)
	}
	if publishedSecond.PublishedAt == nil || publishedFirst.PublishedAt == nil {
		t.Fatal("published episodes carry no published_at")
	}

	// Unpublishing removes the episode from the feed.
	e.sendJSON(http.MethodPost, "/api/v1/episodes/"+second.ID+"/unpublish", nil, nil).wantStatus(http.StatusOK)
	after := e.get("/feeds/" + series.Slug + ".xml").wantStatus(http.StatusOK)
	var refreshed feedDocument
	if err := xml.Unmarshal(after.body, &refreshed); err != nil {
		t.Fatalf("refreshed feed is not valid XML: %v", err)
	}
	if len(refreshed.Channel.Items) != 1 || refreshed.Channel.Items[0].GUID.Value != first.ID {
		t.Fatalf("refreshed feed items = %+v, want only %q", refreshed.Channel.Items, first.ID)
	}
}

// TestFeedUnknownSlug covers the 404 of an unknown feed slug.
func TestFeedUnknownSlug(t *testing.T) {
	e := newEnv(t)

	resp := e.get("/feeds/no-such-series.xml")
	resp.wantError(http.StatusNotFound, "SERIES_NOT_FOUND")
	if got := resp.header.Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("Content-Type = %q, want JSON", got)
	}
}
