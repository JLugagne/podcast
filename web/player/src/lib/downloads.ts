import { t } from "../i18n";
import { readStorage, writeStorage } from "./storage";

/** Name of the Cache Storage bucket that holds downloaded audio. */
export const AUDIO_CACHE = "podcast-audio";
const INDEX_KEY = "downloads";

export class DownloadError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "DownloadError";
  }
}

export function isOfflineSupported(): boolean {
  return typeof caches !== "undefined";
}

/** Ids of downloaded episodes, mirrored in localStorage so the UI can read them synchronously. */
export function listDownloaded(): string[] {
  const raw = readStorage(INDEX_KEY);
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw) as unknown;
    return Array.isArray(parsed) ? parsed.filter((id): id is string => typeof id === "string") : [];
  } catch {
    return [];
  }
}

export function isDownloaded(episodeId: string): boolean {
  return listDownloaded().includes(episodeId);
}

function writeIndex(ids: string[]): void {
  writeStorage(INDEX_KEY, JSON.stringify(ids));
}

/**
 * Stores the audio of an episode in the Cache API. The request must succeed
 * with CORS, otherwise the bytes cannot be read back for playback.
 */
export async function downloadEpisode(
  episodeId: string,
  audioUrl: string,
  onProgress?: (fraction: number) => void,
): Promise<void> {
  if (!isOfflineSupported()) throw new DownloadError(t("offlineUnavailable"));
  let response: Response;
  try {
    response = await fetch(audioUrl);
  } catch {
    throw new DownloadError(t("downloadCorsBlocked"));
  }
  if (!response.ok || !response.body) {
    throw new DownloadError(t("audioServerAnswered", { status: response.status }));
  }

  const total = Number(response.headers.get("content-length")) || 0;
  const reader = response.body.getReader();
  const chunks: Uint8Array[] = [];
  let received = 0;
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    chunks.push(value);
    received += value.byteLength;
    if (total > 0) onProgress?.(Math.min(1, received / total));
  }

  const bytes = new Uint8Array(received);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  }
  const type = response.headers.get("content-type") ?? "audio/mpeg";
  try {
    const cache = await caches.open(AUDIO_CACHE);
    await cache.put(
      downloadKey(episodeId),
      new Response(bytes, { headers: { "content-type": type } }),
    );
  } catch {
    throw new DownloadError(t("notEnoughStorage"));
  }
  writeIndex([...new Set([...listDownloaded(), episodeId])]);
  onProgress?.(1);
}

export async function removeDownload(episodeId: string): Promise<void> {
  if (isOfflineSupported()) {
    const cache = await caches.open(AUDIO_CACHE);
    await cache.delete(downloadKey(episodeId));
  }
  writeIndex(listDownloaded().filter((id) => id !== episodeId));
}

/** A same-origin pseudo URL used only as the cache key. */
function downloadKey(episodeId: string): string {
  return `/__downloads__/${encodeURIComponent(episodeId)}`;
}

/** An object URL for a downloaded episode, or null when it is not stored. */
export async function downloadedObjectURL(episodeId: string): Promise<string | null> {
  if (!isOfflineSupported() || !isDownloaded(episodeId)) return null;
  const cache = await caches.open(AUDIO_CACHE);
  const response = await cache.match(downloadKey(episodeId));
  if (!response) return null;
  return URL.createObjectURL(await response.blob());
}
