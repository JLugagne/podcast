export interface MediaSessionMetadataInput {
  title: string;
  artist: string;
  album: string;
  artworkUrl?: string | null;
}

export type MediaSessionActionHandlers = Partial<
  Record<MediaSessionAction, MediaSessionActionHandler | null>
>;

function session(): MediaSession | null {
  if (typeof navigator === "undefined" || !("mediaSession" in navigator)) return null;
  return navigator.mediaSession;
}

export function isMediaSessionSupported(): boolean {
  return session() !== null;
}

function absoluteUrl(url: string): string {
  try {
    return new URL(url, window.location.origin).href;
  } catch {
    return url;
  }
}

export function setMediaSessionMetadata(input: MediaSessionMetadataInput): void {
  const mediaSession = session();
  if (!mediaSession) return;
  if (typeof MediaMetadata === "undefined") return;
  const artwork = input.artworkUrl
    ? [{ src: absoluteUrl(input.artworkUrl), sizes: "512x512" }]
    : [];
  try {
    mediaSession.metadata = new MediaMetadata({
      title: input.title,
      artist: input.artist,
      album: input.album,
      artwork,
    });
  } catch {
    // Older browsers can reject metadata; playback continues regardless.
  }
}

export function clearMediaSessionMetadata(): void {
  const mediaSession = session();
  if (!mediaSession) return;
  try {
    mediaSession.metadata = null;
  } catch {
    // Ignore unsupported sessions.
  }
}

export function setMediaSessionHandlers(handlers: MediaSessionActionHandlers): void {
  const mediaSession = session();
  if (!mediaSession) return;
  for (const [action, handler] of Object.entries(handlers)) {
    try {
      mediaSession.setActionHandler(action as MediaSessionAction, handler ?? null);
    } catch {
      // Some browsers reject individual actions; skip them.
    }
  }
}

export function setMediaSessionPositionState(
  duration: number,
  position: number,
  playbackRate: number,
): void {
  const mediaSession = session();
  if (!mediaSession || typeof mediaSession.setPositionState !== "function") return;
  if (!(duration > 0)) return;
  const clampedPosition = Math.min(Math.max(0, position), duration);
  try {
    mediaSession.setPositionState({
      duration,
      position: clampedPosition,
      playbackRate,
    });
  } catch {
    // Position state is best effort.
  }
}
