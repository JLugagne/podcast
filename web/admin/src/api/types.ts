export type EpisodeStatus = "draft" | "published";

export interface Series {
  id: string;
  slug: string;
  feed_url: string;
  title: string;
  author: string;
  description: string;
  accent_color: string;
  cover_url: string | null;
  episode_count: number;
  published_count: number;
  total_duration_seconds: number;
  created_at: string;
  updated_at: string;
  version: number;
}

export interface SeriesInput {
  slug?: string;
  title: string;
  author: string;
  description: string;
  accent_color?: string;
  version?: number;
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

export interface EpisodeInput {
  number?: number;
  title: string;
  description: string;
  version?: number;
}

export interface EpisodeListResponse {
  episodes: Episode[];
}

export interface ApiErrorBody {
  code: string;
  message: string;
}

export interface UploadProgress {
  /** Percentage between 0 and 100. */
  percent: number;
  loaded: number;
  total: number;
}
