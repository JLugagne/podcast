export const queryKeys = {
  seriesList: ["series", "list"] as const,
  seriesDetail: (id: string) => ["series", "detail", id] as const,
  publishedEpisodes: (seriesId: string) => ["episodes", "published", seriesId] as const,
  episode: (id: string) => ["episodes", "detail", id] as const,
  progress: () => ["listening", "progress"] as const,
  continueListening: (limit: number) => ["listening", "continue", limit] as const,
  listeningAll: ["listening"] as const,
};
