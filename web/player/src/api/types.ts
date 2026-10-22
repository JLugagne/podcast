export type EpisodeStatus = "draft" | "published";

export interface Series {
  id: string;
  slug: string;
  title: string;
  author: string;
  description: string;
  accent_color: string;
  cover_url: string | null;
  feed_url: string;
  episode_count: number;
  published_count: number;
  total_duration_seconds: number;
  created_at: string;
  updated_at: string;
  version: number;
}

export interface SeriesListResponse {
  series: Series[];
}

export interface AudioFileDTO {
  url: string;
  content_type: string;
  size_bytes: number;
  duration_seconds: number;
  original_name: string;
}

export interface ChapterDTO {
  start_seconds: number;
  end_seconds?: number | null;
  title: string;
  url?: string | null;
  image_url?: string | null;
  toc: boolean;
}

export interface Episode {
  id: string;
  series_id: string;
  number: number;
  title: string;
  description: string;
  status: EpisodeStatus;
  audio: AudioFileDTO | null;
  chapters: ChapterDTO[];
  published_at?: string | null;
  created_at: string;
  updated_at: string;
  version: number;
}

export interface EpisodeListResponse {
  episodes: Episode[];
}

/** The identity of the series an episode belongs to, as needed by the player. */
export interface PlaybackSeries {
  id: string;
  slug: string;
  title: string;
  accent_color: string;
  cover_url: string | null;
}

export interface Progress {
  episode_id: string;
  position_seconds: number;
  completed: boolean;
  updated_at: string;
}

export interface ProgressListResponse {
  progress: Progress[];
}

export interface EpisodeSummary {
  id: string;
  number: number;
  title: string;
  duration_seconds: number;
  chapter_count: number;
  series_id: string;
  series_title: string;
  series_slug: string;
  accent_color: string;
  cover_url: string | null;
}

export interface ContinueItem {
  progress: Progress;
  episode: EpisodeSummary;
}

export interface ContinueResponse {
  items: ContinueItem[];
}

export interface ApiErrorBody {
  code: string;
  message: string;
}
