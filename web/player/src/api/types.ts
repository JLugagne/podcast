/** Site-wide metadata rendered by the listener app. */
export interface SiteInfo {
  title: string;
  description: string;
  author: string;
}

/** One chapter, normalized from the Podcasting 2.0 camelCase fields. */
export interface ChapterDTO {
  start_seconds: number;
  end_seconds?: number | null;
  title: string;
  url?: string | null;
  image_url?: string | null;
  toc: boolean;
}

/** An episode as listed in the catalog index. */
export interface Episode {
  id: string;
  slug: string;
  series_slug: string;
  number: number;
  title: string;
  description: string;
  published_at: string;
  duration_seconds: number;
  chapter_count: number;
  audio_url: string;
  /** Episode artwork, when the feed gives one that differs from the series cover. */
  image_url?: string | null;
  /** Podcasting 2.0 chapters document, fetched lazily when the episode plays. */
  chapters_url?: string | null;
}

/** The full episode document, including its chapters. */
export interface EpisodeDetail extends Omit<Episode, "chapter_count"> {
  chapters: ChapterDTO[];
}

/** A series and its published episodes. */
export interface Series {
  slug: string;
  title: string;
  author: string;
  description: string;
  accent_color: string;
  cover_url: string | null;
  episode_count: number;
  total_duration_seconds: number;
  episodes: Episode[];
}

/** The generated catalog document. */
export interface Catalog {
  site: SiteInfo;
  series: Series[];
}

/** The identity of the series an episode belongs to, as needed by the player. */
export interface PlaybackSeries {
  slug: string;
  title: string;
  accent_color: string;
  cover_url: string | null;
}

/** Listening progress, persisted in this browser only. */
export interface Progress {
  episode_id: string;
  position_seconds: number;
  completed: boolean;
  updated_at: string;
}

export interface ProgressListResponse {
  progress: Progress[];
}

export interface ContinueItem {
  progress: Progress;
  episode: Episode;
  series: PlaybackSeries;
}

export interface ContinueResponse {
  items: ContinueItem[];
}
