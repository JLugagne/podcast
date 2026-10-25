const PREFIX = "podcast.player.";

export const PLAYBACK_RATE_KEY = "rate";
export const AUTO_PLAY_NEXT_KEY = "auto-play-next";
export const EPISODE_VIEW_KEY = "episode-view";

export function readStorage(key: string): string | null {
  try {
    return window.localStorage.getItem(`${PREFIX}${key}`);
  } catch {
    return null;
  }
}

export function writeStorage(key: string, value: string): void {
  try {
    window.localStorage.setItem(`${PREFIX}${key}`, value);
  } catch {
    // Storage can be unavailable (private mode, blocked cookies); ignore it.
  }
}

export function readNumber(key: string, fallback: number): number {
  const raw = readStorage(key);
  if (raw === null) return fallback;
  const value = Number(raw);
  return Number.isFinite(value) ? value : fallback;
}

export function writeNumber(key: string, value: number): void {
  writeStorage(key, String(value));
}

export function readBoolean(key: string, fallback: boolean): boolean {
  const raw = readStorage(key);
  if (raw === null) return fallback;
  return raw === "true";
}

export function writeBoolean(key: string, value: boolean): void {
  writeStorage(key, value ? "true" : "false");
}
