import { useQuery } from "@tanstack/react-query";
import { getCatalog, getEpisode } from "../api/client";
import { queryKeys } from "./keys";

/** The whole catalog: one request per page load, cached for the session. */
export function useCatalog() {
  return useQuery({
    queryKey: queryKeys.catalog,
    queryFn: getCatalog,
    staleTime: Number.POSITIVE_INFINITY,
  });
}

/** The full document of one episode, including its chapters. */
export function useEpisode(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.episode(id ?? ""),
    queryFn: () => getEpisode(id as string),
    enabled: Boolean(id),
    staleTime: Number.POSITIVE_INFINITY,
  });
}
