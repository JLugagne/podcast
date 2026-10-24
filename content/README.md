# Content

The site is generated from this directory. `make check` validates it and
`make site` builds it.

```
site.json
series/<slug>/series.json
series/<slug>/episodes/<episode-slug>.json
```

- The directory name is the series slug and the file name is the episode slug;
  both must match `^[a-z0-9]+(-[a-z0-9]+)*$`.
- Episode `audio_url` is either an absolute HTTP(S) URL (the normal case, for
  audio stored on R2 or another host) or a path relative to the series
  directory, which the build copies under `media/`.
- Chapters use the Podcasting 2.0 field names (`startTime`, `endTime`, `img`,
  `url`, `toc`). The first chapter must start at 0 and every chapter must start
  before `duration_seconds`.

`series/nlp-et-llm/episodes/01-les-origines.json.example` is a ready-to-fill
example: the chapters were imported from the original episode. To publish it,
copy it to `01-les-origines.json` and replace the placeholder `audio_url` with
the real URL of the MP3.
