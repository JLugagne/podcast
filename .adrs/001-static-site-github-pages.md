# ADR 001 — Static site on GitHub Pages

Date: 2026-10-24
Scope: strategic
Author: architect
Decided: with human input
Revisit: false

## Context

The service was built as a self-hosted Go API with PostgreSQL, a blob store and
an admin SPA, plus an RSS feed per series. In practice the product is a
personal podcast: one author, one listener, no authentication, no multi-tenancy.
That shape makes most of the runtime infrastructure unnecessary:

- the admin UI exists to upload files and edit rows that could be files in git;
- PostgreSQL stores a catalogue whose source of truth is the audio production
  directory and its chapters JSON;
- the RSS feed was an integration surface nobody consumes — the listener app
  talks to the JSON API;
- the listener app is already a pure SPA and only needs read access to series,
  episodes and audio;
- listening progress is already single-listener, so it can live in the browser.

Hosting audio in git is not viable (GitHub's 100 MiB per-file hard limit, LFS
bandwidth quota and no LFS support on Pages), so audio lives outside the repo
behind a stable HTTPS URL.

## Decision

Replace the service with a static site:

1. **Content lives in git** under `content/`, one JSON file per series and one
   JSON file per episode. An episode JSON carries its metadata, the absolute
   `audio_url` of the MP3 and its chapters (Podcasting 2.0 camelCase fields).
2. **A Node build script** (`web/player/scripts/content.mjs`, run by
   `npm run build` and `npm run check`) validates the content and emits the
   site data: one `data/index.json` for listings and one
   `data/episodes/<series>/<episode>.json` per episode, so every episode has a
   single JSON with its MP3 URL and chapters.
3. **The listener SPA is the only UI.** It reads the generated JSON with
   relative URLs, works under any base path, and stores listening progress in
   `localStorage`.
4. **Deployment is GitHub Pages**, published from `web/player/dist` by a
   GitHub Actions workflow. Deep links are handled by a `404.html` copy of the
   entry document.
5. **Audio is hosted outside GitHub Pages** (Cloudflare R2 behind a Worker, or
   any HTTPS origin) and referenced by absolute URL from each episode JSON.
   Relative `audio_url` values are copied into the build for local previews and
   end-to-end tests.
6. **Removed**: the admin SPA, the HTTP API, PostgreSQL and its migrations, the
   blob store, the listening bounded context and its server-side progress, the
   RSS feed generator and feed routes, Docker/compose, and the API e2e suite.

## Consequences

Positive:

- the runtime surface shrinks to static files: no database, no server, no
  migrations, no upload pipeline, no Docker;
- content is reviewable and versioned; publishing is a commit;
- hosting is free and cacheable, and the whole site can be rebuilt from git;
- the generator reuses the existing domain rules for chapters, slugs and
  episode validation, so behaviour is preserved where it matters.

Negative:

- no server-side progress: progress is per browser profile and does not sync
  across devices;
- no private drafts: everything committed to the published branch is public;
- publishing requires a local toolchain (an editor, Node and the audio
  duration) and a commit instead of a form;
- audio must be uploaded separately to its external host;
- no RSS feed for third-party podcast apps (a deliberate removal).

Risks:

- an absolute `audio_url` is immutable once published; moving audio breaks
  listeners. The build validates that URLs are absolute HTTP(S).
- GitHub Pages cannot set response headers; the JSON is served with GitHub's
  defaults. Acceptable for a same-origin SPA.

## Related

- Features: static-hosting migration (`.adrs/001-static-site-github-pages-plan.md`)
- Prior ADRs: None
- Superseded ADRs: None
