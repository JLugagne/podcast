import { HttpResponse, http } from "msw/http";
import { parseChaptersDocument } from "../api/chapters";
import type { ChapterDTO, Episode, EpisodeInput, Series, SeriesInput } from "../api/types";

const NOW = "2026-01-01T00:00:00.000Z";

export interface RecordedRequest {
  method: string;
  path: string;
  contentType: string | null;
  body: unknown;
}

export interface ErrorOverride {
  status: number;
  code: string;
  message: string;
}

interface TestStore {
  series: Series[];
  episodes: Episode[];
  requests: RecordedRequest[];
  chaptersDocuments: Map<string, string>;
  failNextChapters: ErrorOverride | null;
  failNextAudio: ErrorOverride | null;
  failNextCover: ErrorOverride | null;
  failNextEpisodeUpdate: ErrorOverride | null;
  nextSeriesId: number;
  nextEpisodeId: number;
}

export const store: TestStore = {
  series: [],
  episodes: [],
  requests: [],
  chaptersDocuments: new Map(),
  failNextChapters: null,
  failNextAudio: null,
  failNextCover: null,
  failNextEpisodeUpdate: null,
  nextSeriesId: 1,
  nextEpisodeId: 1,
};

export function resetStore(): void {
  store.series = [];
  store.episodes = [];
  store.requests = [];
  store.chaptersDocuments = new Map();
  store.failNextChapters = null;
  store.failNextAudio = null;
  store.failNextCover = null;
  store.failNextEpisodeUpdate = null;
  store.nextSeriesId = 1;
  store.nextEpisodeId = 1;
}

export function seedSeries(overrides: Partial<Series> = {}): Series {
  const number = store.nextSeriesId++;
  const slug = overrides.slug ?? `test-series-${number}`;
  const series: Series = {
    id: `series-${number}`,
    slug,
    feed_url: `/feeds/${slug}.xml`,
    title: "Test Series",
    author: "Test Author",
    description: "A series used by tests.",
    accent_color: "#7A9CFF",
    cover_url: null,
    episode_count: 0,
    published_count: 0,
    total_duration_seconds: 0,
    created_at: NOW,
    updated_at: NOW,
    version: 1,
    ...overrides,
  };
  store.series.push(series);
  return series;
}

export function seedEpisode(seriesId: string, overrides: Partial<Episode> = {}): Episode {
  const sameSeries = store.episodes.filter((episode) => episode.series_id === seriesId);
  const episode: Episode = {
    id: `episode-${store.nextEpisodeId++}`,
    series_id: seriesId,
    number: sameSeries.length + 1,
    title: "Test Episode",
    description: "",
    status: "draft",
    audio: null,
    chapters: [],
    published_at: null,
    created_at: NOW,
    updated_at: NOW,
    version: 1,
    ...overrides,
  };
  store.episodes.push(episode);
  return episode;
}

function seriesView(series: Series): Series {
  const episodes = store.episodes.filter((episode) => episode.series_id === series.id);
  return {
    ...series,
    feed_url: `/feeds/${series.slug}.xml`,
    episode_count: episodes.length,
    published_count: episodes.filter((episode) => episode.status === "published").length,
    total_duration_seconds: episodes.reduce(
      (total, episode) => total + (episode.audio?.duration_seconds ?? 0),
      0,
    ),
  };
}

function deriveSlug(title: string): string {
  return title
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 120);
}

function apiError(status: number, code: string, message: string) {
  return HttpResponse.json({ code, message }, { status });
}

function record(request: Request, body: unknown): void {
  store.requests.push({
    method: request.method,
    path: new URL(request.url).pathname,
    contentType: request.headers.get("content-type"),
    body,
  });
}

function findSeries(id: string): Series | undefined {
  return store.series.find((series) => series.id === id);
}

function findEpisode(id: string): Episode | undefined {
  return store.episodes.find((episode) => episode.id === id);
}

function chaptersDocument(episode: Episode): string {
  return (
    store.chaptersDocuments.get(episode.id) ??
    JSON.stringify({ version: "1.2.0", chapters: chaptersToDocumentChapters(episode.chapters) })
  );
}

function chaptersToDocumentChapters(chapters: ChapterDTO[]) {
  return [...chapters]
    .sort((a, b) => a.start_seconds - b.start_seconds)
    .map((chapter) => ({
      startTime: chapter.start_seconds,
      title: chapter.title,
      toc: chapter.toc,
      ...(chapter.end_seconds !== undefined && chapter.end_seconds !== null
        ? { endTime: chapter.end_seconds }
        : {}),
      ...(chapter.url ? { url: chapter.url } : {}),
      ...(chapter.image_url ? { img: chapter.image_url } : {}),
    }));
}

async function readUpload(request: Request) {
  const form = await request.formData();
  const order = [...form.keys()];
  const file = form.get("file");
  const isFile = typeof file === "object" && file !== null && "name" in file;
  const fileName = isFile ? String((file as File).name) : null;
  const size = isFile ? Number((file as File).size) : 0;
  const fields = Object.fromEntries(
    [...form.entries()]
      .filter(([key]) => key !== "file")
      .map(([key, value]) => [key, String(value)]),
  );
  record(request, { field: "file", fileName, size, fields, order });
  return {
    isFile,
    fileName,
    size,
    order,
    durationSeconds: Number(form.get("duration_seconds") ?? 0),
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export const handlers = [
  http.get("/api/v1/series", ({ request }) => {
    const query = new URL(request.url).searchParams.get("q")?.trim().toLowerCase() ?? "";
    const list = store.series
      .filter(
        (series) =>
          query.length === 0 ||
          series.title.toLowerCase().includes(query) ||
          series.author.toLowerCase().includes(query) ||
          series.slug.includes(query),
      )
      .map(seriesView);
    return HttpResponse.json({ series: list });
  }),

  http.post("/api/v1/series", async ({ request }) => {
    const body = (await request.json()) as SeriesInput;
    record(request, body);
    if (!body.title?.trim()) {
      return apiError(422, "INVALID_SERIES", "The series title is required.");
    }
    const slug = (body.slug?.trim() || deriveSlug(body.title)).trim();
    if (store.series.some((series) => series.slug === slug)) {
      return apiError(409, "SLUG_TAKEN", "This slug is already used by another series.");
    }
    const series = seedSeries({
      slug,
      title: body.title.trim(),
      author: body.author ?? "",
      description: body.description ?? "",
      accent_color: body.accent_color ?? "#7A9CFF",
    });
    return HttpResponse.json(seriesView(series), { status: 201 });
  }),

  http.get("/api/v1/series/:id", ({ params }) => {
    const series = findSeries(String(params.id));
    if (!series) return apiError(404, "SERIES_NOT_FOUND", "The series was not found.");
    return HttpResponse.json(seriesView(series));
  }),

  http.put("/api/v1/series/:id", async ({ request, params }) => {
    const series = findSeries(String(params.id));
    const body = (await request.json()) as SeriesInput;
    record(request, body);
    if (!series) return apiError(404, "SERIES_NOT_FOUND", "The series was not found.");
    if (body.version === undefined) {
      return apiError(422, "VERSION_REQUIRED", "The version field is required.");
    }
    if (body.version !== series.version) {
      return apiError(409, "VERSION_CONFLICT", "The series was modified in another session.");
    }
    if (!body.title?.trim()) {
      return apiError(422, "INVALID_SERIES", "The series title is required.");
    }
    const slug = body.slug?.trim() || series.slug;
    if (store.series.some((other) => other.id !== series.id && other.slug === slug)) {
      return apiError(409, "SLUG_TAKEN", "This slug is already used by another series.");
    }
    series.title = body.title.trim();
    series.slug = slug;
    series.feed_url = `/feeds/${slug}.xml`;
    series.author = body.author ?? "";
    series.description = body.description ?? "";
    series.accent_color = body.accent_color ?? series.accent_color;
    series.version += 1;
    return HttpResponse.json(seriesView(series));
  }),

  http.delete("/api/v1/series/:id", ({ params }) => {
    const id = String(params.id);
    const series = findSeries(id);
    if (!series) return apiError(404, "SERIES_NOT_FOUND", "The series was not found.");
    if (store.episodes.some((episode) => episode.series_id === id)) {
      return apiError(409, "SERIES_NOT_EMPTY", "The series still has episodes.");
    }
    store.series = store.series.filter((other) => other.id !== id);
    return new HttpResponse(null, { status: 204 });
  }),

  http.put("/api/v1/series/:id/cover", async ({ request, params }) => {
    const series = findSeries(String(params.id));
    if (!series) return apiError(404, "SERIES_NOT_FOUND", "The series was not found.");
    const { isFile } = await readUpload(request);
    if (store.failNextCover) {
      const failure = store.failNextCover;
      store.failNextCover = null;
      return apiError(failure.status, failure.code, failure.message);
    }
    if (!isFile) return apiError(422, "MISSING_FILE", "A cover image is required.");
    series.cover_url = `/api/v1/series/${series.id}/cover`;
    series.version += 1;
    return HttpResponse.json(seriesView(series));
  }),

  http.delete("/api/v1/series/:id/cover", ({ params }) => {
    const series = findSeries(String(params.id));
    if (!series) return apiError(404, "SERIES_NOT_FOUND", "The series was not found.");
    series.cover_url = null;
    series.version += 1;
    return HttpResponse.json(seriesView(series));
  }),

  http.get("/api/v1/series/:id/cover", () => {
    return new HttpResponse(new Uint8Array([137, 80, 78, 71]), {
      headers: { "Content-Type": "image/png" },
    });
  }),

  http.get("/api/v1/series/:id/episodes", ({ request, params }) => {
    const status = new URL(request.url).searchParams.get("status");
    const list = store.episodes
      .filter(
        (episode) =>
          episode.series_id === String(params.id) && (status === null || episode.status === status),
      )
      .sort((a, b) => a.number - b.number);
    return HttpResponse.json({ episodes: list });
  }),

  http.post("/api/v1/series/:id/episodes", async ({ request, params }) => {
    const seriesId = String(params.id);
    const body = (await request.json()) as EpisodeInput;
    record(request, body);
    if (!findSeries(seriesId)) {
      return apiError(404, "SERIES_NOT_FOUND", "The series was not found.");
    }
    if (!body.title?.trim()) {
      return apiError(422, "INVALID_EPISODE", "The episode title is required.");
    }
    const numbers = store.episodes
      .filter((episode) => episode.series_id === seriesId)
      .map((episode) => episode.number);
    const number = body.number ?? (numbers.length > 0 ? Math.max(...numbers) + 1 : 1);
    if (numbers.includes(number)) {
      return apiError(409, "EPISODE_NUMBER_TAKEN", "This episode number is already used.");
    }
    const episode = seedEpisode(seriesId, {
      number,
      title: body.title.trim(),
      description: body.description ?? "",
    });
    return HttpResponse.json(episode, { status: 201 });
  }),

  http.get("/api/v1/episodes/:id", ({ params }) => {
    const episode = findEpisode(String(params.id));
    if (!episode) return apiError(404, "EPISODE_NOT_FOUND", "The episode was not found.");
    return HttpResponse.json(episode);
  }),

  http.put("/api/v1/episodes/:id", async ({ request, params }) => {
    const episode = findEpisode(String(params.id));
    const body = (await request.json()) as EpisodeInput;
    record(request, body);
    if (!episode) return apiError(404, "EPISODE_NOT_FOUND", "The episode was not found.");
    if (body.version === undefined) {
      return apiError(422, "VERSION_REQUIRED", "The version field is required.");
    }
    if (body.version !== episode.version) {
      return apiError(409, "VERSION_CONFLICT", "The episode was modified in another session.");
    }
    if (store.failNextEpisodeUpdate) {
      const failure = store.failNextEpisodeUpdate;
      store.failNextEpisodeUpdate = null;
      return apiError(failure.status, failure.code, failure.message);
    }
    if (!body.title?.trim()) {
      return apiError(422, "INVALID_EPISODE", "The episode title is required.");
    }
    if (body.number !== undefined) {
      const taken = store.episodes.some(
        (other) =>
          other.id !== episode.id &&
          other.series_id === episode.series_id &&
          other.number === body.number,
      );
      if (taken)
        return apiError(409, "EPISODE_NUMBER_TAKEN", "This episode number is already used.");
      episode.number = body.number;
    }
    episode.title = body.title.trim();
    episode.description = body.description ?? "";
    episode.version += 1;
    return HttpResponse.json(episode);
  }),

  http.delete("/api/v1/episodes/:id", ({ params }) => {
    const id = String(params.id);
    if (!findEpisode(id)) {
      return apiError(404, "EPISODE_NOT_FOUND", "The episode was not found.");
    }
    store.episodes = store.episodes.filter((episode) => episode.id !== id);
    store.chaptersDocuments.delete(id);
    return new HttpResponse(null, { status: 204 });
  }),

  http.put("/api/v1/episodes/:id/audio", async ({ request, params }) => {
    const episode = findEpisode(String(params.id));
    if (!episode) return apiError(404, "EPISODE_NOT_FOUND", "The episode was not found.");
    const upload = await readUpload(request);
    if (store.failNextAudio) {
      const failure = store.failNextAudio;
      store.failNextAudio = null;
      return apiError(failure.status, failure.code, failure.message);
    }
    if (!upload.isFile) return apiError(422, "MISSING_FILE", "An audio file is required.");
    episode.audio = {
      url: `/api/v1/episodes/${episode.id}/audio`,
      content_type: "audio/mpeg",
      size_bytes: upload.size,
      duration_seconds: upload.durationSeconds,
      original_name: upload.fileName ?? "audio.mp3",
    };
    episode.version += 1;
    return HttpResponse.json(episode);
  }),

  http.get("/api/v1/episodes/:id/audio", () => {
    return new HttpResponse(new Uint8Array([1, 2, 3]), {
      headers: { "Content-Type": "audio/mpeg" },
    });
  }),

  http.put("/api/v1/episodes/:id/chapters", async ({ request, params }) => {
    const episode = findEpisode(String(params.id));
    const raw = await request.text();
    record(request, raw);
    if (!episode) return apiError(404, "EPISODE_NOT_FOUND", "The episode was not found.");
    if (store.failNextChapters) {
      const failure = store.failNextChapters;
      store.failNextChapters = null;
      return apiError(failure.status, failure.code, failure.message);
    }
    let parsedRaw: unknown = null;
    try {
      parsedRaw = JSON.parse(raw);
    } catch {
      // parseChaptersDocument below reports the JSON error.
    }
    if (
      isRecord(parsedRaw) &&
      typeof parsedRaw.version === "string" &&
      !/^1\.\d+(?:\.\d+)*$/.test(parsedRaw.version.trim())
    ) {
      return apiError(
        422,
        "INVALID_CHAPTERS_VERSION",
        "The chapters document version is not supported.",
      );
    }
    const parsed = parseChaptersDocument(raw);
    if (!parsed.ok) {
      return apiError(422, "INVALID_CHAPTERS", parsed.errors.join(" "));
    }
    episode.chapters = parsed.chapters;
    episode.version += 1;
    store.chaptersDocuments.set(episode.id, raw);
    return HttpResponse.json(episode);
  }),

  http.get("/api/v1/episodes/:id/chapters", ({ params }) => {
    const episode = findEpisode(String(params.id));
    if (!episode) return apiError(404, "EPISODE_NOT_FOUND", "The episode was not found.");
    return new HttpResponse(chaptersDocument(episode), {
      headers: { "Content-Type": "application/json+chapters" },
    });
  }),

  http.post("/api/v1/episodes/:id/publish", ({ request, params }) => {
    record(request, null);
    const episode = findEpisode(String(params.id));
    if (!episode) return apiError(404, "EPISODE_NOT_FOUND", "The episode was not found.");
    if (!episode.audio) {
      return apiError(422, "EPISODE_AUDIO_MISSING", "The episode has no audio file.");
    }
    if (episode.chapters.length === 0) {
      return apiError(422, "EPISODE_CHAPTERS_MISSING", "The episode has no chapters.");
    }
    episode.status = "published";
    episode.published_at = NOW;
    episode.version += 1;
    return HttpResponse.json(episode);
  }),

  http.post("/api/v1/episodes/:id/unpublish", ({ request, params }) => {
    record(request, null);
    const episode = findEpisode(String(params.id));
    if (!episode) return apiError(404, "EPISODE_NOT_FOUND", "The episode was not found.");
    episode.status = "draft";
    episode.published_at = null;
    episode.version += 1;
    return HttpResponse.json(episode);
  }),
];
