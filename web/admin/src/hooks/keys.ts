export const queryKeys = {
  seriesLists: ["series", "list"] as const,
  seriesList: (query: string) => ["series", "list", query] as const,
  seriesDetail: (id: string) => ["series", "detail", id] as const,
  episodes: (seriesId: string) => ["episodes", "list", seriesId] as const,
  episode: (id: string) => ["episodes", "detail", id] as const,
};
