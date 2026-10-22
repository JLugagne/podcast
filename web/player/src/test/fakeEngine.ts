import type { MediaEngine, MediaEngineEvent } from "../playback/engine";

interface LoadCall {
  source: string;
  startAt: number;
}

/** Deterministic media engine for tests: records calls and emits events on demand. */
export class FakeMediaEngine implements MediaEngine {
  source: string | null = null;
  startAt = 0;
  currentTime = 0;
  duration = 0;
  paused = true;
  playbackRate = 1;
  loadCalls: LoadCall[] = [];
  playCalls = 0;
  pauseCalls = 0;
  seekCalls: number[] = [];
  destroyed = false;

  private listeners = new Set<(event: MediaEngineEvent) => void>();

  load(source: string, startAt = 0): void {
    this.source = source;
    this.startAt = startAt;
    this.currentTime = startAt;
    this.paused = true;
    this.loadCalls.push({ source, startAt });
  }

  play(): Promise<void> {
    this.playCalls += 1;
    this.paused = false;
    this.emit({ type: "play" });
    return Promise.resolve();
  }

  pause(): void {
    this.pauseCalls += 1;
    this.paused = true;
    this.emit({ type: "pause" });
  }

  seek(seconds: number): void {
    this.seekCalls.push(seconds);
    this.currentTime = seconds;
    this.emit({ type: "time", currentTime: seconds, duration: this.duration });
  }

  setPlaybackRate(rate: number): void {
    this.playbackRate = rate;
  }

  getCurrentTime(): number {
    return this.currentTime;
  }

  getDuration(): number {
    return this.duration;
  }

  isPaused(): boolean {
    return this.paused;
  }

  subscribe(listener: (event: MediaEngineEvent) => void): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  destroy(): void {
    this.destroyed = true;
    this.listeners.clear();
  }

  /** Simulates the audio element reporting a position. */
  emitTime(currentTime: number, duration = this.duration): void {
    this.currentTime = currentTime;
    this.emit({ type: "time", currentTime, duration });
  }

  /** Simulates the episode reaching its end. */
  emitEnded(): void {
    this.currentTime = this.duration;
    this.paused = true;
    this.emit({ type: "ended" });
  }

  private emit(event: MediaEngineEvent): void {
    for (const listener of [...this.listeners]) listener(event);
  }
}
