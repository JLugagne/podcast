import { HttpResponse, http } from "msw/http";
import type { ChapterDTO, Episode, EpisodeSummary, Progress, Series } from "../api/types";

const NOW = "2026-01-01T00:00:00.000Z";

export interface RecordedRequest {
  method: string;
  path: string;
  body: unknown;
}

interface TestStore {
  series: Series[];
  episodes: Episode[];
  progress: Map<string, Progress>;
  requests: RecordedRequest[];
  nextSeriesId: number;
  nextEpisodeId: number;
  clock: number;
}

export const store: TestStore = {
  series: [],
  episodes: [],
  progress: new Map(),
  requests: [],
  nextSeriesId: 1,
  nextEpisodeId: 1,
  clock: 0,
};

export function resetStore(): void {
  store.series = [];
  store.episodes = [];
  store.progress = new Map();
  store.requests = [];
  store.nextSeriesId = 1;
  store.nextEpisodeId = 1;
  store.clock = 0;
}

function nextTimestamp(): string {
  store.clock += 1;
  return new Date(Date.parse(NOW) + store.clock * 1000).toISOString();
}

export function seedSeries(overrides: Partial<Series> = {}): Series {
  const number = store.nextSeriesId++;
  const slug = overrides.slug ?? `test-series-${number}`;
  const series: Series = {
    id: `series-${number}`,
    slug,
    title: "Test Series",
    author: "Test Author",
    description: "A series used by tests.",
    accent_color: "#7A9CFF",
    cover_url: null,
    feed_url: `/feeds/${slug}.xml`,
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

const DEFAULT_CHAPTERS: ChapterDTO[] = [
  { start_seconds: 0, title: "Intro", toc: true },
  { start_seconds: 30, title: "Main topic", toc: true },
];

export function seedEpisode(seriesId: string, overrides: Partial<Episode> = {}): Episode {
  const number =
    overrides.number ?? store.episodes.filter((e) => e.series_id === seriesId).length + 1;
  const id = overrides.id ?? `episode-${store.nextEpisodeId}`;
  const numericId = Number.parseInt(id.replace(/\D/g, ""), 10);
  store.nextEpisodeId = Number.isFinite(numericId)
    ? Math.max(store.nextEpisodeId, numericId + 1)
    : store.nextEpisodeId + 1;
  const audio =
    overrides.audio === null
      ? null
      : {
          url: `/api/v1/episodes/${id}/audio`,
          content_type: "audio/mpeg",
          size_bytes: 2048,
          duration_seconds: 100,
          original_name: "episode.mp3",
          ...overrides.audio,
        };
  const episode: Episode = {
    id,
    series_id: seriesId,
    number,
    title: `Episode ${number}`,
    description: "",
    status: "published",
    chapters: DEFAULT_CHAPTERS,
    published_at: NOW,
    created_at: NOW,
    updated_at: NOW,
    version: 1,
    ...overrides,
    audio,
  };
  store.episodes.push(episode);
  return episode;
}

export function seedProgress(episodeId: string, overrides: Partial<Progress> = {}): Progress {
  const progress: Progress = {
    episode_id: episodeId,
    position_seconds: 20,
    completed: false,
    updated_at: nextTimestamp(),
    ...overrides,
  };
  store.progress.set(episodeId, progress);
  return progress;
}

function seriesView(series: Series): Series {
  const episodes = store.episodes.filter((episode) => episode.series_id === series.id);
  return {
    ...series,
    episode_count: episodes.length,
    published_count: episodes.filter((episode) => episode.status === "published").length,
    total_duration_seconds: episodes.reduce(
      (total, episode) => total + (episode.audio?.duration_seconds ?? 0),
      0,
    ),
  };
}

export function findSeries(id: string): Series | undefined {
  return store.series.find((series) => series.id === id);
}

export function findEpisode(id: string): Episode | undefined {
  return store.episodes.find((episode) => episode.id === id);
}

function findPublishedEpisode(id: string): Episode | undefined {
  const episode = findEpisode(id);
  return episode?.status === "published" ? episode : undefined;
}

export function episodeSummary(episode: Episode, series: Series): EpisodeSummary {
  return {
    id: episode.id,
    number: episode.number,
    title: episode.title,
    duration_seconds: episode.audio?.duration_seconds ?? 0,
    chapter_count: episode.chapters.length,
    series_id: series.id,
    series_title: series.title,
    series_slug: series.slug,
    accent_color: series.accent_color,
    cover_url: series.cover_url,
  };
}

function apiError(status: number, code: string, message: string) {
  return HttpResponse.json({ code, message }, { status });
}

function record(request: Request, body: unknown): void {
  store.requests.push({
    method: request.method,
    path: new URL(request.url).pathname,
    body,
  });
}

interface ProgressInput {
  position_seconds?: unknown;
  completed?: unknown;
}

export const handlers = [
  http.get("/api/v1/series", () => {
    return HttpResponse.json({ series: store.series.map(seriesView) });
  }),

  http.get("/api/v1/series/:id", ({ params }) => {
    const series = findSeries(String(params.id));
    if (!series) return apiError(404, "SERIES_NOT_FOUND", "The series was not found.");
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
      .sort((left, right) => left.number - right.number);
    return HttpResponse.json({ episodes: list });
  }),

  http.get("/api/v1/episodes/:id", ({ params }) => {
    const episode = findEpisode(String(params.id));
    if (!episode) return apiError(404, "EPISODE_NOT_FOUND", "The episode was not found.");
    return HttpResponse.json(episode);
  }),

  http.get("/api/v1/listening/progress", () => {
    const progress = [...store.progress.values()].sort((left, right) =>
      right.updated_at.localeCompare(left.updated_at),
    );
    return HttpResponse.json({ progress });
  }),

  http.get("/api/v1/listening/continue", ({ request }) => {
    const limitParam = Number(new URL(request.url).searchParams.get("limit") ?? 5);
    const limit = Number.isFinite(limitParam) && limitParam > 0 ? limitParam : 5;
    const rows = [...store.progress.values()]
      .filter((row) => !row.completed && row.position_seconds > 0)
      .sort((left, right) => right.updated_at.localeCompare(left.updated_at));
    const items = [];
    for (const progress of rows) {
      const episode = findPublishedEpisode(progress.episode_id);
      if (!episode) continue;
      const series = findSeries(episode.series_id);
      if (!series) continue;
      items.push({ progress, episode: episodeSummary(episode, series) });
      if (items.length >= limit) break;
    }
    return HttpResponse.json({ items });
  }),

  http.put("/api/v1/listening/progress/:episodeId", async ({ request, params }) => {
    const body = (await request.json()) as ProgressInput;
    record(request, body);
    const episode = findPublishedEpisode(String(params.episodeId));
    if (!episode) return apiError(404, "EPISODE_NOT_FOUND", "The episode was not found.");

    const requested = Number(body.position_seconds);
    if (!Number.isFinite(requested) || requested < 0) {
      return apiError(422, "INVALID_POSITION", "The position must be a positive number.");
    }

    const duration = episode.audio?.duration_seconds ?? 0;
    let position = requested;
    let completed = false;
    if (body.completed === true) {
      position = duration;
      completed = true;
    } else if (body.completed === false) {
      completed = false;
    } else {
      completed = duration > 0 && position >= duration - 2;
    }
    if (duration > 0) position = Math.min(position, duration);

    const progress: Progress = {
      episode_id: episode.id,
      position_seconds: position,
      completed,
      updated_at: nextTimestamp(),
    };
    store.progress.set(episode.id, progress);
    return HttpResponse.json(progress);
  }),

  http.post("/api/v1/listening/progress/:episodeId/played", ({ request, params }) => {
    record(request, null);
    const episode = findPublishedEpisode(String(params.episodeId));
    if (!episode) return apiError(404, "EPISODE_NOT_FOUND", "The episode was not found.");
    const progress: Progress = {
      episode_id: episode.id,
      position_seconds: episode.audio?.duration_seconds ?? 0,
      completed: true,
      updated_at: nextTimestamp(),
    };
    store.progress.set(episode.id, progress);
    return HttpResponse.json(progress);
  }),

  http.delete("/api/v1/listening/progress/:episodeId", ({ request, params }) => {
    record(request, null);
    const episode = findPublishedEpisode(String(params.episodeId));
    if (!episode) return apiError(404, "EPISODE_NOT_FOUND", "The episode was not found.");
    store.progress.delete(episode.id);
    return new HttpResponse(null, { status: 204 });
  }),
];
