import type { QueryClient } from "@tanstack/react-query";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";
import type { Episode, EpisodeInput, UploadProgress } from "../api/types";
import { queryKeys } from "./keys";

function invalidateEpisode(queryClient: QueryClient, episode: Episode) {
  queryClient.setQueryData(queryKeys.episode(episode.id), episode);
  void queryClient.invalidateQueries({ queryKey: queryKeys.episodes(episode.series_id) });
  void queryClient.invalidateQueries({ queryKey: queryKeys.seriesDetail(episode.series_id) });
  void queryClient.invalidateQueries({ queryKey: queryKeys.seriesLists });
}

export function useEpisodes(seriesId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.episodes(seriesId ?? ""),
    queryFn: () => api.listEpisodes(seriesId as string),
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

export function useCreateEpisode(seriesId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: EpisodeInput) => api.createEpisode(seriesId, input),
    onSuccess: (episode) => {
      queryClient.setQueryData(queryKeys.episode(episode.id), episode);
      void queryClient.invalidateQueries({ queryKey: queryKeys.episodes(seriesId) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.seriesDetail(seriesId) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.seriesLists });
    },
  });
}

export function useUpdateEpisode() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: EpisodeInput }) =>
      api.updateEpisode(id, input),
    onSuccess: (episode) => invalidateEpisode(queryClient, episode),
  });
}

export function useDeleteEpisode() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id }: { id: string; seriesId: string }) => api.deleteEpisode(id),
    onSuccess: (_result, { id, seriesId }) => {
      queryClient.removeQueries({ queryKey: queryKeys.episode(id) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.episodes(seriesId) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.seriesDetail(seriesId) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.seriesLists });
    },
  });
}

export interface UploadAudioVariables {
  id: string;
  file: File;
  durationSeconds: number;
  signal?: AbortSignal;
  onProgress?: (progress: UploadProgress) => void;
}

export function useUploadAudio() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, file, durationSeconds, signal, onProgress }: UploadAudioVariables) =>
      api.uploadAudio(id, file, durationSeconds, { signal, onProgress }),
    onSuccess: (episode) => invalidateEpisode(queryClient, episode),
  });
}

export function useSetChapters() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, raw }: { id: string; raw: string }) => api.setChapters(id, raw),
    onSuccess: (episode) => invalidateEpisode(queryClient, episode),
  });
}

export function usePublishEpisode() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.publishEpisode(id),
    onSuccess: (episode) => invalidateEpisode(queryClient, episode),
  });
}

export function useUnpublishEpisode() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.unpublishEpisode(id),
    onSuccess: (episode) => invalidateEpisode(queryClient, episode),
  });
}
