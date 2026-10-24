import { DataError } from "./errors";
import type { Catalog, ChapterDTO, EpisodeDetail } from "./types";

export { DataError, errorMessage } from "./errors";

/** The base path Vite was built with ("/" or "/<repo>/"). */
const BASE = import.meta.env.BASE_URL;

interface RawChapter {
  startTime: number;
  endTime?: number | null;
  title: string;
  url?: string | null;
  img?: string | null;
  toc?: boolean;
}

interface RawEpisodeDetail extends Omit<EpisodeDetail, "chapters"> {
  chapters: RawChapter[];
}

export function catalogURL(): string {
  return `${BASE}data/index.json`;
}

export function episodeURL(id: string): string {
  const [series, slug] = id.split("/");
  if (!series || !slug) throw new DataError(`Invalid episode id ${JSON.stringify(id)}.`);
  return `${BASE}data/episodes/${encodeURIComponent(series)}/${encodeURIComponent(slug)}.json`;
}

/** Resolves an audio URL: absolute URLs are kept, relative ones use the site base. */
export function resolveAudioURL(raw: string): string {
  if (/^https?:\/\//i.test(raw)) return raw;
  return `${BASE}${raw.replace(/^\/+/, "")}`;
}

export async function getCatalog(): Promise<Catalog> {
  return fetchJSON<Catalog>(catalogURL());
}

export async function getEpisode(id: string): Promise<EpisodeDetail> {
  const raw = await fetchJSON<RawEpisodeDetail>(episodeURL(id));
  return {
    ...raw,
    chapters: raw.chapters.map(toChapter),
  };
}

async function fetchJSON<T>(url: string): Promise<T> {
  let response: Response;
  try {
    response = await fetch(url, { headers: { Accept: "application/json" } });
  } catch {
    throw new DataError("Could not reach the site. Check your connection.");
  }
  if (!response.ok) {
    throw new DataError(`The site returned an unexpected response (${response.status}).`);
  }
  try {
    return (await response.json()) as T;
  } catch {
    throw new DataError("The site returned an unexpected response.");
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
