import { listSubscriptions } from "../lib/library";
import { DataError } from "./errors";
import type { Catalog, ChapterDTO, EpisodeDetail } from "./types";

export { DataError, errorMessage } from "./errors";

/** The base path Vite was built with ("/" or "/<repo>/"). */
const BASE = import.meta.env.BASE_URL;

const SITE = { title: "Podcasts", description: "", author: "" };

interface RawChapter {
  startTime: number;
  endTime?: number | null;
  title: string;
  url?: string | null;
  img?: string | null;
  toc?: boolean;
}

/** Resolves an audio URL: absolute URLs are kept, relative ones use the site base. */
export function resolveAudioURL(raw: string): string {
  if (/^https?:\/\//i.test(raw)) return raw;
  return `${BASE}${raw.replace(/^\/+/, "")}`;
}

/** The catalog is the list of subscriptions saved in this browser's localStorage. */
export async function getCatalog(): Promise<Catalog> {
  return { site: SITE, series: listSubscriptions() };
}

export async function getEpisode(id: string): Promise<EpisodeDetail> {
  const [seriesSlug] = id.split("/");
  const series = listSubscriptions().find((item) => item.slug === seriesSlug);
  const episode = series?.episodes.find((item) => item.id === id);
  if (!episode) throw new DataError(`Unknown episode ${JSON.stringify(id)}.`);
  const { chapter_count: _count, ...rest } = episode;
  return { ...rest, chapters: await fetchChapters(episode.chapters_url) };
}

/** Chapters are optional and come from a Podcasting 2.0 JSON document. */
async function fetchChapters(url: string | null | undefined): Promise<ChapterDTO[]> {
  if (!url) return [];
  try {
    const response = await fetch(url);
    if (!response.ok) return [];
    const body = (await response.json()) as { chapters?: RawChapter[] };
    return (body.chapters ?? []).map(toChapter);
  } catch {
    return [];
  }
}

function toChapter(raw: RawChapter): ChapterDTO {
  return {
    start_seconds: Number(raw.startTime),
    end_seconds: raw.endTime ?? null,
    title: raw.title,
    url: raw.url ?? null,
    image_url: raw.img ?? null,
    toc: raw.toc ?? true,
  };
}
