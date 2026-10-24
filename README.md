# Podcast

An installable web app (PWA / WebAPK) that plays podcasts from their RSS feed.
It is a static site published on GitHub Pages: there is no server and no database.

- **Add a podcast** by pasting the URL of its RSS feed. The feed is parsed in the
  browser and the subscription (show, episodes) is saved in `localStorage`.
- **Listening progress** (position, played/unplayed, continue listening) is saved
  in `localStorage` too.
- **Offline**: an episode can be downloaded from its `…` menu ("Download for
  offline"). The audio is stored with the Cache API and played from there when
  available. The app shell is cached by a service worker, so the app opens
  without a connection.
- **Install**: on Android, Chrome offers "Install app" (a WebAPK) thanks to the
  web manifest and service worker.
- **Chapters** are read from the `podcast:chapters` JSON document of an episode
  when the feed provides one.

## CORS

The browser can only read a feed or download audio when the server sends CORS
headers. When a feed is blocked, open "Feed blocked by the browser?" under the
add form and set a CORS proxy prefix (the feed URL is appended, URL-encoded).
Audio downloads need the audio host to allow CORS; streaming works either way.
Everything is stored on the device, and nothing is sent to any other service.

## Development

```sh
cd web/player
npm install
npm run dev          # Vite on :5174
npm test             # Vitest
npm run lint         # Biome
npm run build        # tsc + vite build + GitHub Pages fallback files
npm run icons        # regenerate the PWA icons in public/
```

`make site` installs dependencies and builds into `web/player/dist`.

## Deployment

`.github/workflows/pages.yml` runs the tests, builds the site and publishes
`web/player/dist` with the official Pages actions. For a project site served
under `https://<user>.github.io/<repo>/`, the workflow builds with
`BASE_PATH=/<repo>/`. The build writes `404.html` (a copy of `index.html`) so
deep links resolve, and `.nojekyll`.

## Layout

```
web/player/src/lib/feed.ts        RSS parsing
web/player/src/lib/library.ts     subscriptions in localStorage, feed fetching
web/player/src/lib/downloads.ts   offline audio (Cache API)
web/player/public/                manifest, service worker, icons
```

The Playwright journey lives in `tests/e2e-ui` (`make e2e-ui`, needs `ffmpeg`). It serves a fake podcast host (feed, chapters, audio) through request interception.
