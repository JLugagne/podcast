import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";
import type { Series, SeriesInput } from "../api/types";
import { queryKeys } from "./keys";

export function useSeriesList(query = "") {
  return useQuery({
    queryKey: queryKeys.seriesList(query),
    queryFn: () => api.listSeries(query),
    placeholderData: (previous) => previous,
  });
}

export function useSeries(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.seriesDetail(id ?? ""),
    queryFn: () => api.getSeries(id as string),
    enabled: Boolean(id),
  });
}

function useStoreSeries() {
  const queryClient = useQueryClient();
  return (series: Series) => {
    queryClient.setQueryData(queryKeys.seriesDetail(series.id), series);
    void queryClient.invalidateQueries({ queryKey: queryKeys.seriesLists });
  };
}

export function useCreateSeries() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: SeriesInput) => api.createSeries(input),
    onSuccess: (series) => {
      queryClient.setQueryData(queryKeys.seriesDetail(series.id), series);
      void queryClient.invalidateQueries({ queryKey: queryKeys.seriesLists });
    },
  });
}

export function useUpdateSeries() {
  const storeSeries = useStoreSeries();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: SeriesInput }) => api.updateSeries(id, input),
    onSuccess: storeSeries,
  });
}

export function useDeleteSeries() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteSeries(id),
    onSuccess: (_result, id) => {
      queryClient.removeQueries({ queryKey: queryKeys.seriesDetail(id) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.seriesLists });
    },
  });
}

export function useUploadCover() {
  const storeSeries = useStoreSeries();
  return useMutation({
    mutationFn: ({ id, file }: { id: string; file: File }) => api.uploadCover(id, file),
    onSuccess: storeSeries,
  });
}

export function useRemoveCover() {
  const storeSeries = useStoreSeries();
  return useMutation({
    mutationFn: (id: string) => api.removeCover(id),
    onSuccess: storeSeries,
  });
}
