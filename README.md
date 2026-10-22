# Podcast

A self-hosted podcast service for chaptered shows:

- a **listener app** (`/`): library, continue listening, series pages and a
  player with a chapter-segmented progress bar, mini player, playback speed,
  sleep timer, lock-screen controls (Media Session) and keyboard shortcuts;
- an **admin app** (`/admin/`): create series and episodes, upload covers and
  audio, import Podcasting 2.0 JSON chapters, publish;
- **RSS feeds** (`/feeds/<slug>.xml`) with `podcast:chapters`, so any podcast
  app can subscribe;
- a Go API (`/api/v1`), PostgreSQL for metadata and blob storage on disk.

## Quick start

```sh
cp .env.example .env
docker compose up --build
```

Open <http://localhost:8080/admin/> to create a series and publish episodes,
then listen at <http://localhost:8080/>. The `migrate` service applies the
database migrations before the app starts. Set `PODCAST_HTTP__PUBLIC_URL`
when the service is reached through another address: feeds use it to build
absolute links.

To browse the database, start the optional Adminer container:

```sh
docker compose --profile tools up -d
# http://localhost:8081 — system: PostgreSQL, server: postgres,
# user/password/database: podcast
```

## Development

Start only PostgreSQL, then run the API and the Vite dev server on the host:

```sh
make dev-db
make run

cd web/admin
npm install
npm run dev
```

The Vite dev server proxies `/api` to <http://localhost:8080>.

Useful targets:

| Target | Description |
|---|---|
| `make build` | Build `bin/podcast` (the admin UI is not rebuilt) |
| `make web` | Install and build the admin UI into `web/admin/dist` |
| `make run` | Run the API from source |
| `make migrate` | Apply the database migrations and exit |
| `make test` | Run every Go test with the race detector (needs Docker) |
| `make test-short` | Run the Go unit tests only (no Docker, a few seconds) |
| `make test-int` | Run the Postgres-backed tests (adapters + e2e API) |
| `make web-test` | Run the admin and listener UI tests (Vitest + MSW) |
| `make e2e-ui` | Start the compose stack and run the Playwright journeys (admin desktop, listener mobile) |
| `make lint` | Run `go vet`, `golangci-lint` (pinned, via `go run`) and go-arch-lint |
| `make arch-lint` | Check the go-arch-lint boundaries |
| `make up` / `make down` | Start / stop the full compose stack |
| `make dev-db` | Start only the development PostgreSQL |
| `make tidy` | Synchronise `go.mod` and `go.sum` |

## Configuration

Configuration is read from built-in defaults, an optional YAML file
(`configs/podcast.example.yaml`) and `PODCAST_` environment variables. A double
underscore nests keys, so `PODCAST_DATABASE__URL` sets `database.url`.

| Key | Default | Notes |
|---|---|---|
| `http.addr` | `:8080` | Listen address |
| `http.public_url` | `http://localhost:8080` | Absolute base URL used in feed links (no trailing slash) |
| `database.url` | — | Required PostgreSQL DSN |
| `storage.dir` | `./data/blobs` | Blob storage directory |
| `upload.max_audio_bytes` | `2147483648` | 2 GiB |
| `upload.max_image_bytes` | `10485760` | 10 MiB |
| `log.format` | `text` | `text` or `json` |
| `log.level` | `info` | `debug`, `info`, `warn`, `error` |

## Layout

```
cmd/podcast/          cobra commands: serve, migrate
internal/pkg/         shared kernel: config, logger, httpjson, httpserver, postgres, spa
internal/catalog/     catalog bounded context: series, episodes, chapters, RSS feeds
internal/listening/   listening bounded context: progress, continue listening (bridged to catalog)
pkg/catalog/http/v1/  public JSON contract
web/admin/            Vite + React admin UI, embedded into the binary
web/player/           Vite + React listener app, embedded into the binary
tests/e2e-api/        black-box API tests over a real PostgreSQL (testcontainers)
tests/e2e-ui/         Playwright journey against the running stack (screenshots in screenshots/)
```

## API

All routes live under `/api/v1`. There is no authentication: the service is
meant to run locally.

| Method | Path | Purpose |
|---|---|---|
| GET, POST | `/series` | List (`?q=`) or create series |
| GET, PUT, DELETE | `/series/{id}` | Read, update (send `version`), delete (only when empty) |
| GET, PUT, DELETE | `/series/{id}/cover` | Cover image (multipart field `file`) |
| GET, POST | `/series/{id}/episodes` | List (`?status=draft\|published`) or create episodes |
| GET, PUT, DELETE | `/episodes/{id}` | Read, update (send `version`), delete |
| GET, PUT | `/episodes/{id}/audio` | Stream with Range support / upload (multipart: `duration_seconds` first, then `file`) |
| GET, PUT | `/episodes/{id}/chapters` | Podcasting 2.0 JSON chapters document |
| POST | `/episodes/{id}/publish`, `/episodes/{id}/unpublish` | Publishing requires audio and at least one chapter |

Listening progress (single local listener for now):

| Method | Path | Purpose |
|---|---|---|
| GET | `/listening/progress` | Saved positions for every episode |
| PUT | `/listening/progress/{episodeId}` | Save a position (`position_seconds`, optional `completed`) |
| POST | `/listening/progress/{episodeId}/played` | Mark as played |
| DELETE | `/listening/progress/{episodeId}` | Mark as unplayed |
| GET | `/listening/continue?limit=5` | In-progress episodes, newest first, with episode and series details |

Errors are returned as `{"code": "...", "message": "..."}` with 400, 404, 409,
413 or 422.

## Chapters format

Chapters use the [Podcasting 2.0 JSON chapters](https://github.com/Podcastindex-org/podcast-namespace/blob/main/docs/examples/chapters/jsonChapters.md)
format:

```json
{
  "version": "1.2.0",
  "chapters": [
    { "startTime": 0, "title": "Intro" },
    { "startTime": 95.5, "title": "Main topic", "url": "https://example.com", "img": "https://example.com/art.png" }
  ]
}
```

Rules: the first chapter starts at 0, start times are unique, every title is
set, and when the audio duration is known every chapter starts before it.

## Health check

`GET /healthz` returns `200 OK` when the process is running.
