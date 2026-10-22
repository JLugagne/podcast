import { vi } from "vitest";

export interface MediaSessionStub {
  handlers: Partial<Record<MediaSessionAction, MediaSessionActionHandler>>;
  metadata: MediaMetadata | null;
  setActionHandler: (action: MediaSessionAction, handler: MediaSessionActionHandler | null) => void;
  setPositionState: ReturnType<typeof vi.fn>;
}

class FakeMediaMetadata {
  title: string;
  artist: string;
  album: string;
  artwork: MediaImage[];

  constructor(
    init: { title?: string; artist?: string; album?: string; artwork?: MediaImage[] } = {},
  ) {
    this.title = init.title ?? "";
    this.artist = init.artist ?? "";
    this.album = init.album ?? "";
    this.artwork = init.artwork ?? [];
  }
}

/** Installs a Media Session API stub on the jsdom navigator. */
export function makeMediaSessionStub(): MediaSessionStub {
  const handlers: Partial<Record<MediaSessionAction, MediaSessionActionHandler>> = {};
  const stub: MediaSessionStub = {
    handlers,
    metadata: null,
    setActionHandler: vi.fn(
      (action: MediaSessionAction, handler: MediaSessionActionHandler | null) => {
        if (handler) handlers[action] = handler;
        else delete handlers[action];
      },
    ),
    setPositionState: vi.fn(),
  };
  Object.defineProperty(navigator, "mediaSession", { configurable: true, value: stub });
  Object.defineProperty(globalThis, "MediaMetadata", {
    configurable: true,
    value: FakeMediaMetadata,
  });
  return stub;
}
