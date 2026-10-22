// Package feed is the catalog context's RSS 2.0 inbound adapter. It serves one
// feed per series under /feeds/{slug}.xml, with absolute links built from the
// configured public URL.
package feed

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/episodeuc"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/seriesuc"
	"github.com/JLugagne/podcast/internal/catalog/inbound/http/shared"
)

const (
	// contentType is the media type of every feed response.
	contentType = "application/rss+xml; charset=utf-8"
	// feedPath is the feed route pattern, registered on the root router.
	feedPath = "/feeds/{slug}.xml"
	// feedLanguage is the language declared by the channel.
	feedLanguage = "en"
	// itunesNamespace is the Apple Podcasts namespace.
	itunesNamespace = "http://www.itunes.com/dtds/podcast-1.0.dtd"
	// podcastNamespace is the Podcasting 2.0 namespace.
	podcastNamespace = "https://podcastindex.org/namespace/1.0"
	// chaptersContentType is the media type of the Podcasting 2.0 JSON chapters
	// documents linked from the items.
	chaptersContentType = "application/json+chapters"
	// audioURLPrefix is the path prefix of the episode audio endpoint.
	audioURLPrefix = "/api/v1/episodes/"
	// coverURLPrefix is the path prefix of the series cover endpoint.
	coverURLPrefix = "/api/v1/series/"
	// seriesPagePrefix is the path prefix of the listener series page.
	seriesPagePrefix = "/series/"
)

// App is the catalog use-case surface the feed needs. It is satisfied by
// *app.App.
type App interface {
	seriesuc.SeriesQueries
	episodeuc.EpisodeQueries
}

// Options configures the feed adapter. A zero PublicURL yields path-absolute
// links and a nil Now falls back to time.Now.
type Options struct {
	PublicURL string
	Now       func() time.Time
}

// svc holds the handler's collaborators.
type svc struct {
	app       App
	publicURL string
	now       func() time.Time
}

// SetupRouter registers GET /feeds/{slug}.xml on r.
func SetupRouter(r *mux.Router, app App, opts Options) {
	s := svc{
		app:       app,
		publicURL: strings.TrimRight(strings.TrimSpace(opts.PublicURL), "/"),
		now:       opts.Now,
	}
	if s.now == nil {
		s.now = time.Now
	}
	r.HandleFunc(feedPath, shared.WithActor(s.serveFeed)).Methods(http.MethodGet, http.MethodHead)
}

// serveFeed writes the RSS document of the series owning the requested slug.
func (s svc) serveFeed(w http.ResponseWriter, r *http.Request, actor *domain.Actor) {
	view, err := s.app.GetSeriesBySlug(r.Context(), actor, mux.Vars(r)["slug"])
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	published := domain.EpisodeStatusPublished
	episodes, err := s.app.ListEpisodes(r.Context(), actor, view.Series.ID(), episodeuc.ListQuery{Status: &published})
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	body, err := Build(view.Series, episodes, s.publicURL, s.now())
	if err != nil {
		shared.WriteDomainError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	// The document is rendered by encoding/xml, which escapes every value.
	_, _ = w.Write(body) //nolint:gosec // G705: values are XML-escaped by encoding/xml
}

// rssFeed is the root element of the document.
type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	ITunes  string     `xml:"xmlns:itunes,attr"`
	Podcast string     `xml:"xmlns:podcast,attr"`
	Channel rssChannel `xml:"channel"`
}

// rssChannel is the channel element of the feed.
type rssChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	Language      string    `xml:"language"`
	LastBuildDate string    `xml:"lastBuildDate"`
	ITunesAuthor  string    `xml:"itunes:author"`
	ITunesImage   *rssImage `xml:"itunes:image,omitempty"`
	Items         []rssItem `xml:"item"`
}

// rssImage is the itunes:image element.
type rssImage struct {
	Href string `xml:"href,attr"`
}

// rssItem is one episode of the channel.
type rssItem struct {
	Title           string        `xml:"title"`
	Description     string        `xml:"description"`
	GUID            rssGUID       `xml:"guid"`
	PubDate         string        `xml:"pubDate"`
	Enclosure       *rssEnclosure `xml:"enclosure,omitempty"`
	ITunesDuration  string        `xml:"itunes:duration"`
	ITunesEpisode   int           `xml:"itunes:episode"`
	PodcastChapters *rssChapters  `xml:"podcast:chapters,omitempty"`
}

// rssGUID is the guid element, never a permalink.
type rssGUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

// rssEnclosure is the enclosure element of an item.
type rssEnclosure struct {
	URL    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

// rssChapters is the podcast:chapters element of an item.
type rssChapters struct {
	URL  string `xml:"url,attr"`
	Type string `xml:"type,attr"`
}

// Build renders the RSS 2.0 document of one series. Only published episodes are
// listed, newest first, and every URL is absolute against publicURL.
func Build(series *domain.Series, episodes []*domain.Episode, publicURL string, now time.Time) ([]byte, error) {
	base := strings.TrimRight(strings.TrimSpace(publicURL), "/")
	channel := rssChannel{
		Title:         series.Title(),
		Link:          base + seriesPagePrefix + string(series.ID()),
		Description:   series.Description(),
		Language:      feedLanguage,
		LastBuildDate: now.UTC().Format(time.RFC1123Z),
		ITunesAuthor:  series.Author(),
		Items:         make([]rssItem, 0, len(episodes)),
	}
	if series.HasCover() {
		channel.ITunesImage = &rssImage{Href: base + coverURLPrefix + string(series.ID()) + "/cover"}
	}
	for _, episode := range publishedNewestFirst(episodes) {
		channel.Items = append(channel.Items, toItem(episode, base))
	}

	doc := rssFeed{
		Version: "2.0",
		ITunes:  itunesNamespace,
		Podcast: podcastNamespace,
		Channel: channel,
	}
	body, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), body...), nil
}

// publishedNewestFirst keeps the published episodes and orders them by
// publication date descending, then by episode number descending.
func publishedNewestFirst(episodes []*domain.Episode) []*domain.Episode {
	published := make([]*domain.Episode, 0, len(episodes))
	for _, episode := range episodes {
		if episode.Status() == domain.EpisodeStatusPublished {
			published = append(published, episode)
		}
	}
	sort.SliceStable(published, func(i, j int) bool {
		left, right := published[i], published[j]
		leftAt, rightAt := left.PublishedAt(), right.PublishedAt()
		switch {
		case leftAt == nil && rightAt == nil:
			return left.Number() > right.Number()
		case leftAt == nil:
			return false
		case rightAt == nil:
			return true
		case leftAt.Equal(*rightAt):
			return left.Number() > right.Number()
		default:
			return leftAt.After(*rightAt)
		}
	})
	return published
}

// toItem maps one published episode to its RSS item.
func toItem(episode *domain.Episode, base string) rssItem {
	item := rssItem{
		Title:         episode.Title(),
		Description:   episode.Description(),
		GUID:          rssGUID{IsPermaLink: "false", Value: string(episode.ID())},
		ITunesEpisode: episode.Number(),
	}
	if published := episode.PublishedAt(); published != nil {
		item.PubDate = published.UTC().Format(time.RFC1123Z)
	}
	if audio := episode.Audio(); audio != nil {
		item.Enclosure = &rssEnclosure{
			URL:    base + audioURLPrefix + string(episode.ID()) + "/audio",
			Length: audio.SizeBytes,
			Type:   audio.ContentType,
		}
		item.ITunesDuration = strconv.Itoa(int(audio.DurationSeconds))
	}
	if episode.HasChapters() {
		item.PodcastChapters = &rssChapters{
			URL:  base + audioURLPrefix + string(episode.ID()) + "/chapters",
			Type: chaptersContentType,
		}
	}
	return item
}
