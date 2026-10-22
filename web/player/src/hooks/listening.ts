import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";
import type { Progress, ProgressListResponse } from "../api/types";
import { queryKeys } from "./keys";

export function useProgressList() {
  return useQuery({
    queryKey: queryKeys.progress(),
    queryFn: () => api.listProgress(),
  });
}

export function useContinueListening(limit = 5) {
  return useQuery({
    queryKey: queryKeys.continueListening(limit),
    queryFn: () => api.continueListening(limit),
  });
}

/** Progress rows indexed by episode id, for quick lookups in lists. */
export function progressByEpisode(data: ProgressListResponse | undefined): Map<string, Progress> {
  const map = new Map<string, Progress>();
  for (const row of data?.progress ?? []) map.set(row.episode_id, row);
  return map;
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
    mutationFn: (episodeId: string) => api.markPlayed(episodeId),
    onSuccess: invalidate,
  });
}

export function useMarkUnplayed() {
  const invalidate = useInvalidateListening();
  return useMutation({
    mutationFn: (episodeId: string) => api.markUnplayed(episodeId),
    onSuccess: invalidate,
  });
}
