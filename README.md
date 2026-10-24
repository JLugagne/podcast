# Podcast

A static, chaptered podcast site published on GitHub Pages.

- **Content lives in git** under `content/`: one JSON per series and one JSON per
  episode. An episode JSON carries its MP3 URL and its chapters.
- **A listener app** (`web/player/`, Vite + React) reads the generated JSON and
  plays the audio. Listening progress (position, played/unplayed, continue
  listening) is stored in the browser's `localStorage`, so it survives reloads
  and sessions on that device.
- **A build script** (`web/player/scripts/content.mjs`) validates the content,
  writes `data/index.json` plus one `data/episodes/<series>/<episode>.json` per
  episode, copies relative audio files, and writes the `404.html` SPA fallback.
- **Audio is hosted outside the site** (Cloudflare R2 behind a Worker, or any
  HTTPS origin), referenced by absolute URL. There is no server, no database
  and no RSS feed.

## Quick start

```sh
make site
cd web/player && npm run preview   # http://localhost:4173
```

`make site` installs the player dependencies, runs the Vite build and generates
the site data into `web/player/dist`.

## Content format

```
content/
  site.json
  series/<slug>/series.json
  series/<slug>/episodes/<episode-slug>.json
```

`content/site.json`:

```json
{ "title": "Podcasts", "description": "", "author": "Ada" }
```

`content/series/<slug>/series.json`:

```json
{
  "title": "NLP et LLM",
  "author": "Ada",
  "description": "",
  "accent_color": "#7A9CFF",
  "cover_url": "https://media.example.com/cover.jpg"
}
```

`content/series/<slug>/episodes/<episode-slug>.json`:

```json
{
  "number": 1,
  "title": "Les origines",
  "description": "",
  "published_at": "2026-10-24",
  "duration_seconds": 2876.8,
  "audio_url": "https://media.example.com/nlp-et-llm/01-les-origines.mp3",
  "chapters": [
    { "startTime": 0, "title": "Introduction" },
    { "startTime": 179, "title": "Alan Turing" }
  ]
}
```

Rules enforced by `make check`:

- series and episode slugs match `^[a-z0-9]+(-[a-z0-9]+)*$`;
- episode numbers are unique per series;
- `published_at` is `YYYY-MM-DD` or RFC 3339;
- `audio_url` is an absolute HTTP(S) URL, or a path relative to the series
  directory (copied into the build under `media/`);
- chapters are sorted, the first starts at 0, start times are unique and every
  chapter starts before `duration_seconds`.

`make check` prints every problem at once; `npm run check` in `web/player` does
the same.

## Development

```sh
cd web/player
npm install
npm run dev          # generates the data into public/, then starts Vite on :5174
npm run build        # tsc + vite build + content generation
npm test             # Vitest
npm run check        # content validation
```

Set `CONTENT_DIR` to build from another content tree (the e2e suite does this).

## Deployment

`.github/workflows/pages.yml` builds the site and publishes
`web/player/dist` with the official Pages actions. For a project site served
under `https://<user>.github.io/<repo>/`, the workflow builds with
`BASE_PATH=/<repo>/`; a user site uses the default `/`.

The build writes `404.html` (a copy of `index.html`) so deep links such as
`/series/<slug>` resolve on Pages, and `.nojekyll` so Pages serves the output
as-is.

## Audio hosting

Keep MP3s out of git: GitHub rejects files above 100 MiB, Git LFS is not served
by Pages and its bandwidth quota disables downloads once exceeded. Upload the
audio to R2 (or any HTTPS host) and put the absolute URL in `audio_url`.

## Layout

```
content/                 series and episode JSON documents
web/player/              the listener app and the content build script
web/player/scripts/      content validation and site generation (Node)
tests/e2e-ui/            Playwright journey against a fixture content tree
.adrs/                   architecture decision records
```
