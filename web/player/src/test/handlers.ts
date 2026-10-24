import { HttpResponse, http } from "msw/http";
import type { Catalog, ChapterDTO, Episode, Progress, Series, SiteInfo } from "../api/types";

const NOW = "2026-01-01T00:00:00.000Z";
const PROGRESS_STORAGE_KEY = "podcast.player.progress";

interface TestStore {
  site: SiteInfo;
  series: Series[];
  chapters: Map<string, ChapterDTO[]>;
  clock: number;
}

export const store: TestStore = {
  site: { title: "Podcasts", description: "Test shows", author: "Test Author" },
  series: [],
  chapters: new Map(),
  clock: 0,
};

export function resetStore(): void {
  store.site = { title: "Podcasts", description: "Test shows", author: "Test Author" };
  store.series = [];
  store.chapters.clear();
  store.clock = 0;
}

const DEFAULT_CHAPTERS: ChapterDTO[] = [
  { start_seconds: 0, title: "Intro", toc: true },
  { start_seconds: 30, title: "Main topic", toc: true },
];

export interface SeriesSeed extends Partial<Omit<Series, "episodes">> {
  id?: string;
}

export function seedSeries(seed: SeriesSeed = {}): Series {
  const number = store.series.length + 1;
  const slug = seed.slug ?? seed.id ?? `test-series-${number}`;
  const series: Series = {
    title: "Test Series",
    author: "Test Author",
    description: "A series used by tests.",
    accent_color: "#7A9CFF",
    cover_url: null,
    episode_count: 0,
    total_duration_seconds: 0,
    ...seed,
    slug,
    episodes: [],
  };
  store.series.push(series);
  return series;
}

export interface AudioSeed {
  url: string;
  duration_seconds: number;
}

export interface EpisodeSeed extends Partial<Omit<Episode, "chapters">> {
  id?: string;
  audio?: AudioSeed | null;
  chapters?: ChapterDTO[];
}

export function seedEpisode(seriesSlug: string, seed: EpisodeSeed = {}): Episode {
  const series = store.series.find((item) => item.slug === seriesSlug);
  if (!series) throw new Error(`unknown series ${seriesSlug}`);

  const number = seed.number ?? series.episodes.length + 1;
  const slug = seed.slug ?? seed.id ?? `episode-${number}`;
  const audio =
    seed.audio === undefined ? { url: `/media/${slug}.mp3`, duration_seconds: 100 } : seed.audio;
  const chapters = seed.chapters ?? DEFAULT_CHAPTERS;
  const { audio: _audio, chapters: _chapters, ...rest } = seed;

  const episode: Episode = {
    number,
    title: `Episode ${number}`,
    description: "",
    published_at: NOW,
    duration_seconds: audio?.duration_seconds ?? 0,
    audio_url: audio?.url ?? "",
    ...rest,
    id: `${seriesSlug}/${slug}`,
    slug,
    series_slug: seriesSlug,
    chapter_count: chapters.length,
  };

  store.chapters.set(episode.id, chapters);
  series.episodes.push(episode);
  series.episode_count = series.episodes.length;
  series.total_duration_seconds = series.episodes.reduce(
    (total, item) => total + item.duration_seconds,
    0,
  );
  return episode;
}

export function seedProgress(episodeId: string, overrides: Partial<Progress> = {}): Progress {
  store.clock += 1;
  const progress: Progress = {
    episode_id: episodeId,
    position_seconds: 20,
    completed: false,
    updated_at: new Date(Date.parse(NOW) + store.clock * 1000).toISOString(),
    ...overrides,
  };
  const raw = window.localStorage.getItem(PROGRESS_STORAGE_KEY);
  const map = raw ? (JSON.parse(raw) as Record<string, Progress>) : {};
  map[episodeId] = progress;
  window.localStorage.setItem(PROGRESS_STORAGE_KEY, JSON.stringify(map));
  return progress;
}

function catalogView(): Catalog {
  return {
    site: store.site,
    series: store.series.map((series) => ({
      ...series,
      episode_count: series.episodes.length,
      total_duration_seconds: series.episodes.reduce(
        (total, episode) => total + episode.duration_seconds,
        0,
      ),
      episodes: [...series.episodes].sort((left, right) => left.number - right.number),
    })),
  };
}

function rawChapters(chapters: ChapterDTO[]) {
  return chapters.map((chapter) => ({
    startTime: chapter.start_seconds,
    endTime: chapter.end_seconds ?? undefined,
    title: chapter.title,
    url: chapter.url ?? undefined,
    img: chapter.image_url ?? undefined,
    toc: chapter.toc,
  }));
}

function episodeDetail(episode: Episode) {
  const { chapter_count: _chapterCount, ...rest } = episode;
  return { ...rest, chapters: rawChapters(store.chapters.get(episode.id) ?? []) };
}

export const handlers = [
  http.get("*/data/index.json", () => HttpResponse.json(catalogView())),

  http.get("*/data/episodes/:series/:file", ({ params }) => {
    const series = store.series.find((item) => item.slug === String(params.series));
    const slug = String(params.file).replace(/\.json$/, "");
    const episode = series?.episodes.find((item) => item.slug === slug);
    if (!episode) return new HttpResponse(null, { status: 404 });
    return HttpResponse.json(episodeDetail(episode));
  }),
];
