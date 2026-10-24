# Migration plan — static site on GitHub Pages

Tracks the execution of [ADR 001](./001-static-site-github-pages.md).

## Target layout

```
content/
  site.json                                  # site title, description, author
  series/<slug>/series.json                  # title, author, description, accent_color, cover_url
  series/<slug>/episodes/<episode-slug>.json # number, title, description, published_at,
                                             # duration_seconds, audio_url, chapters[]
web/player/                                  # the only UI (Vite + React)
web/player/dist/                             # build output, published to GitHub Pages
  data/index.json                            # site + series with episode summaries
  data/episodes/<series>/<episode>.json      # full episode: audio URL + chapters
  404.html                                   # SPA fallback for deep links
internal/site/                               # REMOVED — generation moved to Node
cmd/podcast/                                 # REMOVED — generation moved to Node
```

## Steps

1. **Content model and loader** — `web/player/scripts/content.mjs`: site,
   series, episode and chapter types; JSON decoding; validation (slug shape,
   unique series slugs, unique episode numbers and slugs, title/description
   bounds, absolute `audio_url`, chapter ordering and duration bounds).
2. **Site writer** — the same script: write `data/index.json`, one
   `data/episodes/<series>/<episode>.json` per episode, copy relative audio
   files into `media/`, write `404.html` and `.nojekyll`.
3. **Commands** — `npm run check` (validate only) and `npm run build` (Vite
   build followed by generation). No Go toolchain is kept: the repository is a
   single Node/Vite project.
4. **Remove the service** — delete `internal/catalog`, `internal/listening`,
   `internal/pkg`, `pkg/`, the old `cmd` commands, `main.go`, `go.mod`,
   `go.sum`, the Go lint configs, `web/admin`, `web/player/embed.go`,
   `tests/e2e-api`, Docker/compose, configs and env files.
5. **Player data layer** — replace the API client with static JSON loaders,
   adapt types, hooks, pages and the playback provider; series routes key on
   slugs; episode ids are `<series-slug>/<episode-slug>`.
6. **Local progress** — implement progress in `localStorage` (position,
   completed, updated_at) with the same query keys so the UI keeps working;
   "continue listening" is computed client-side.
7. **Remove RSS surfaces** — no feed panel, no feed URL in the series page, no
   feed generator.
8. **Tests** — Go tests for loading/validation/building; player Vitest suite
   adapted to static handlers; Playwright listener journey against a fixture
   content tree served by `vite preview`.
9. **Build and deploy** — Makefile targets (`site`, `check`, `test`, `e2e-ui`),
   `.github/workflows/pages.yml`, updated `.go-arch-lint.yml`, `.gitignore`,
   README.
10. **Verify** — `go test ./...`, `go vet ./...`, `npm run typecheck`,
    `npm test -- --run`, `make site`, `npx playwright test` for the listener
    journey.

## Out of scope

- uploading audio to the external host (done with the host's tooling);
- an authoring UI (content is edited as files);
- cross-device progress sync.
