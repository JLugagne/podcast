import { useQuery } from "@tanstack/react-query";
import { api } from "../api/client";
import { queryKeys } from "./keys";

export function useSeriesList() {
  return useQuery({
    queryKey: queryKeys.seriesList,
    queryFn: () => api.listSeries(),
  });
}

export function useSeries(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.seriesDetail(id ?? ""),
    queryFn: () => api.getSeries(id as string),
    enabled: Boolean(id),
  });
}

export function usePublishedEpisodes(seriesId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.publishedEpisodes(seriesId ?? ""),
    queryFn: () => api.listEpisodes(seriesId as string, "published"),
    enabled: Boolean(seriesId),
  });
}

export function useEpisode(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.episode(id ?? ""),
    queryFn: () => api.getEpisode(id as string),
    enabled: Boolean(id),
  });
}
