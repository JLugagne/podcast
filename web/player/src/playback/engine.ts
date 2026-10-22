export type MediaEngineEvent =
  | { type: "time"; currentTime: number; duration: number }
  | { type: "duration"; duration: number }
  | { type: "play" }
  | { type: "pause" }
  | { type: "ended" }
  | { type: "error"; message: string };

export type MediaEngineListener = (event: MediaEngineEvent) => void;

/**
 * The playback surface used by the provider. The default implementation wraps
 * a single HTMLAudioElement; tests inject a deterministic fake.
 */
export interface MediaEngine {
  /** Points the engine at an audio source and prepares to start at `startAtSeconds`. */
  load(source: string, startAtSeconds?: number): void;
  play(): Promise<void>;
  pause(): void;
  seek(seconds: number): void;
  setPlaybackRate(rate: number): void;
  getCurrentTime(): number;
  getDuration(): number;
  isPaused(): boolean;
  subscribe(listener: MediaEngineListener): () => void;
  destroy(): void;
}

function errorMessage(error: unknown): string {
  if (error instanceof Error) return error.message;
  return "The audio could not be played.";
}

export class HtmlAudioEngine implements MediaEngine {
  private readonly audio: HTMLAudioElement;
  private readonly listeners = new Set<MediaEngineListener>();
  private pendingSeek: number | null = null;

  constructor(audio?: HTMLAudioElement) {
    this.audio = audio ?? (typeof Audio !== "undefined" ? new Audio() : ({} as HTMLAudioElement));
    if (typeof Audio !== "undefined") {
      this.audio.preload = "metadata";
      this.audio.addEventListener("timeupdate", this.handleTimeUpdate);
      this.audio.addEventListener("durationchange", this.handleDurationChange);
      this.audio.addEventListener("loadedmetadata", this.handleLoadedMetadata);
      this.audio.addEventListener("play", this.handlePlay);
      this.audio.addEventListener("pause", this.handlePause);
      this.audio.addEventListener("ended", this.handleEnded);
      this.audio.addEventListener("error", this.handleError);
    }
  }

  load(source: string, startAtSeconds = 0): void {
    this.pendingSeek = startAtSeconds > 0 ? startAtSeconds : null;
    if (typeof Audio === "undefined") return;
    this.audio.src = source;
    this.audio.load();
  }

  play(): Promise<void> {
    if (typeof Audio === "undefined") return Promise.resolve();
    try {
      const result = this.audio.play();
      return result instanceof Promise
        ? result.catch((error: unknown) => {
            this.emit({ type: "error", message: errorMessage(error) });
          })
        : Promise.resolve();
    } catch (error) {
      this.emit({ type: "error", message: errorMessage(error) });
      return Promise.resolve();
    }
  }

  pause(): void {
    if (typeof Audio === "undefined") return;
    this.audio.pause();
  }

  seek(seconds: number): void {
    if (typeof Audio === "undefined") return;
    try {
      this.audio.currentTime = seconds;
    } catch {
      this.pendingSeek = seconds;
    }
  }

  setPlaybackRate(rate: number): void {
    if (typeof Audio === "undefined") return;
    this.audio.playbackRate = rate;
  }

  getCurrentTime(): number {
    if (typeof Audio === "undefined") return 0;
    return Number.isFinite(this.audio.currentTime) ? this.audio.currentTime : 0;
  }

  getDuration(): number {
    if (typeof Audio === "undefined") return 0;
    return Number.isFinite(this.audio.duration) ? this.audio.duration : 0;
  }

  isPaused(): boolean {
    if (typeof Audio === "undefined") return true;
    return this.audio.paused;
  }

  subscribe(listener: MediaEngineListener): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  destroy(): void {
    if (typeof Audio === "undefined") return;
    this.audio.removeEventListener("timeupdate", this.handleTimeUpdate);
    this.audio.removeEventListener("durationchange", this.handleDurationChange);
    this.audio.removeEventListener("loadedmetadata", this.handleLoadedMetadata);
    this.audio.removeEventListener("play", this.handlePlay);
    this.audio.removeEventListener("pause", this.handlePause);
    this.audio.removeEventListener("ended", this.handleEnded);
    this.audio.removeEventListener("error", this.handleError);
    this.audio.pause();
    this.audio.removeAttribute("src");
    this.audio.load();
    this.listeners.clear();
  }

  private emit(event: MediaEngineEvent): void {
    for (const listener of this.listeners) listener(event);
  }

  private handleTimeUpdate = (): void => {
    this.emit({
      type: "time",
      currentTime: this.getCurrentTime(),
      duration: this.getDuration(),
    });
  };

  private handleDurationChange = (): void => {
    this.emit({ type: "duration", duration: this.getDuration() });
  };

  private handleLoadedMetadata = (): void => {
    if (this.pendingSeek !== null) {
      const duration = this.getDuration();
      const target =
        duration > 0 ? Math.min(this.pendingSeek, Math.max(0, duration - 0.5)) : this.pendingSeek;
      this.pendingSeek = null;
      try {
        this.audio.currentTime = target;
      } catch {
        // Seeking is best effort; playback still starts from the beginning.
      }
    }
    this.emit({ type: "duration", duration: this.getDuration() });
  };

  private handlePlay = (): void => {
    this.emit({ type: "play" });
  };

  private handlePause = (): void => {
    this.emit({ type: "pause" });
  };

  private handleEnded = (): void => {
    this.emit({ type: "ended" });
  };

  private handleError = (): void => {
    this.emit({ type: "error", message: "The audio could not be played." });
  };
}
