import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import type {
  Catalog,
  ContinueItem,
  Episode,
  PlaybackSeries,
  Progress,
  ProgressListResponse,
} from "../api/types";
import { listProgress, markPlayed, markUnplayed } from "../lib/progress";
import { useCatalog } from "./catalog";
import { queryKeys } from "./keys";

/** Progress is read from this browser's localStorage; no request is made. */
export function useProgressList() {
  return useQuery({
    queryKey: queryKeys.progress(),
    queryFn: async (): Promise<ProgressListResponse> => ({ progress: listProgress() }),
    staleTime: Number.POSITIVE_INFINITY,
  });
}

/** Progress rows indexed by episode id, for quick lookups in lists. */
export function progressByEpisode(data: ProgressListResponse | undefined): Map<string, Progress> {
  const map = new Map<string, Progress>();
  for (const row of data?.progress ?? []) map.set(row.episode_id, row);
  return map;
}

function seriesPlaybackInfo(series: Catalog["series"][number]): PlaybackSeries {
  return {
    slug: series.slug,
    title: series.title,
    accent_color: series.accent_color,
    cover_url: series.cover_url,
  };
}

/** In-progress episodes, most recently updated first. */
export function useContinueListening(limit = 5) {
  const catalogQuery = useCatalog();
  const progressQuery = useProgressList();

  const items = useMemo<ContinueItem[]>(() => {
    const catalog = catalogQuery.data;
    const rows = progressQuery.data?.progress ?? [];
    if (!catalog) return [];

    const result: ContinueItem[] = [];
    for (const progress of rows) {
      if (progress.completed || progress.position_seconds <= 0) continue;
      for (const series of catalog.series) {
        const episode = series.episodes.find((candidate) => candidate.id === progress.episode_id);
        if (!episode) continue;
        result.push({ progress, episode, series: seriesPlaybackInfo(series) });
        break;
      }
      if (result.length >= limit) break;
    }
    return result;
  }, [catalogQuery.data, progressQuery.data, limit]);

  return {
    data: { items },
    isPending: catalogQuery.isPending || progressQuery.isPending,
    isError: catalogQuery.isError || progressQuery.isError,
    error: catalogQuery.error ?? progressQuery.error,
  };
}

function useInvalidateListening() {
  const queryClient = useQueryClient();
  return () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.listeningAll });
  };
}

export function useMarkPlayed() {
  const invalidate = useInvalidateListening();
  return useMutation({
    mutationFn: (episode: Episode) =>
      Promise.resolve(markPlayed(episode.id, episode.duration_seconds)),
    onSuccess: invalidate,
  });
}

export function useMarkUnplayed() {
  const invalidate = useInvalidateListening();
  return useMutation({
    mutationFn: (episodeId: string) => Promise.resolve(markUnplayed(episodeId)),
    onSuccess: invalidate,
  });
}
