export const queryKeys = {
  catalog: ["catalog"] as const,
  episode: (id: string) => ["episode", id] as const,
  progress: () => ["listening", "progress"] as const,
  listeningAll: ["listening"] as const,
};
