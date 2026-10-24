import { useSyncExternalStore } from "react";
import type { Episode } from "../api/types";
import { t } from "../i18n";
import { downloadEpisode, listDownloaded, removeDownload } from "../lib/downloads";

export interface DownloadState {
  downloaded: ReadonlySet<string>;
  /** Fraction (0..1) of in-flight downloads; -1 when the size is unknown. */
  active: ReadonlyMap<string, number>;
  /** Last failure message per episode. */
  errors: ReadonlyMap<string, string>;
}

let state: DownloadState = {
  downloaded: new Set(listDownloaded()),
  active: new Map(),
  errors: new Map(),
};
const listeners = new Set<() => void>();

function update(next: Partial<DownloadState>): void {
  state = { ...state, ...next };
  for (const listener of listeners) listener();
}

function withEntry<V>(map: ReadonlyMap<string, V>, key: string, value: V | null): Map<string, V> {
  const copy = new Map(map);
  if (value === null) copy.delete(key);
  else copy.set(key, value);
  return copy;
}

export async function startDownload(episode: Episode): Promise<void> {
  if (state.active.has(episode.id)) return;
  update({
    active: withEntry(state.active, episode.id, 0),
    errors: withEntry(state.errors, episode.id, null),
  });
  try {
    await downloadEpisode(episode.id, episode.audio_url, (fraction) => {
      update({ active: withEntry(state.active, episode.id, fraction) });
    });
    update({ downloaded: new Set(listDownloaded()) });
  } catch (error) {
    update({
      errors: withEntry(
        state.errors,
        episode.id,
        error instanceof Error ? error.message : t("downloadFailed"),
      ),
    });
  } finally {
    update({ active: withEntry(state.active, episode.id, null) });
  }
}

export async function deleteDownload(episodeId: string): Promise<void> {
  await removeDownload(episodeId);
  update({ downloaded: new Set(listDownloaded()) });
}

/** Test helper: re-reads the persisted index and clears transient state. */
export function resetDownloadState(): void {
  update({ downloaded: new Set(listDownloaded()), active: new Map(), errors: new Map() });
}

export function useDownloads(): DownloadState {
  return useSyncExternalStore(
    (listener) => {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    () => state,
  );
}
